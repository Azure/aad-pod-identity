package mic

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	"github.com/Azure/aad-pod-identity/pkg/cloudprovider"
	"github.com/Azure/aad-pod-identity/pkg/crd"
	"github.com/Azure/aad-pod-identity/pkg/pod"
	"github.com/Azure/aad-pod-identity/pkg/stats"
	"github.com/Azure/aad-pod-identity/version"
	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
)

const (
	stopped = int32(0)
	running = int32(1)
)

// NodeGetter ...
type NodeGetter interface {
	Get(name string) (*corev1.Node, error)
	Start(<-chan struct{})
}

// Client has the required pointers to talk to the api server
// and interact with the CRD related datastructure.
type Client struct {
	CRDClient         crd.ClientInt
	CloudClient       cloudprovider.ClientInt
	PodClient         pod.ClientInt
	EventRecorder     record.EventRecorder
	EventChannel      chan aadpodid.EventType
	NodeClient        NodeGetter
	IsNamespaced      bool
	syncRetryInterval time.Duration

	syncing int32 // protect against conucrrent sync's
}

// ClientInt ...
type ClientInt interface {
	Start(exit <-chan struct{})
	Sync(exit <-chan struct{})
}

type trackUserAssignedMSIIds struct {
	addUserAssignedMSIIDs    []string
	removeUserAssignedMSIIDs []string
	assignedIDsToCreate      []aadpodid.AzureAssignedIdentity
	assignedIDsToDelete      []aadpodid.AzureAssignedIdentity
}

// NewMICClient returnes new mic client
func NewMICClient(cloudconfig string, config *rest.Config, isNamespaced bool, syncRetryInterval time.Duration) (*Client, error) {
	glog.Infof("Starting to create the pod identity client. Version: %v. Build date: %v", version.MICVersion, version.BuildDate)

	clientSet := kubernetes.NewForConfigOrDie(config)
	informer := informers.NewSharedInformerFactory(clientSet, 30*time.Second)

	cloudClient, err := cloudprovider.NewCloudProvider(cloudconfig)
	if err != nil {
		return nil, err
	}
	glog.V(1).Infof("Cloud provider initialized")

	eventCh := make(chan aadpodid.EventType, 100)
	crdClient, err := crd.NewCRDClient(config, eventCh)
	if err != nil {
		return nil, err
	}
	glog.V(1).Infof("CRD client initialized")

	podClient := pod.NewPodClient(informer, eventCh)
	glog.V(1).Infof("Pod Client initialized")

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: clientSet.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: aadpodid.CRDGroup})

	return &Client{
		CRDClient:         crdClient,
		CloudClient:       cloudClient,
		PodClient:         podClient,
		EventRecorder:     recorder,
		EventChannel:      eventCh,
		NodeClient:        &NodeClient{informer.Core().V1().Nodes()},
		IsNamespaced:      isNamespaced,
		syncRetryInterval: syncRetryInterval,
	}, nil
}

// Start ...
func (c *Client) Start(exit <-chan struct{}) {
	glog.V(6).Infof("MIC client starting..")

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		c.PodClient.Start(exit)
		glog.V(6).Infof("Pod client started")
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		c.CRDClient.Start(exit)
		glog.V(6).Infof("CRD client started")
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		c.NodeClient.Start(exit)
		glog.V(6).Infof("Node client started")
		wg.Done()
	}()

	wg.Wait()
	go c.Sync(exit)
}

func (c *Client) canSync() bool {
	return atomic.CompareAndSwapInt32(&c.syncing, stopped, running)
}

func (c *Client) setStopped() {
	atomic.StoreInt32(&c.syncing, stopped)
}

// Sync ...
func (c *Client) Sync(exit <-chan struct{}) {
	if !c.canSync() {
		panic("concurrent syncs")
	}
	defer c.setStopped()

	ticker := time.NewTicker(c.syncRetryInterval)
	defer ticker.Stop()

	glog.Info("Sync thread started.")
	var event aadpodid.EventType
	for {
		select {
		case <-exit:
			return
		case event = <-c.EventChannel:
			glog.V(6).Infof("Received event: %v", event)
		case <-ticker.C:
			glog.V(6).Infof("Running periodic sync loop")
		}

		stats.Init()
		// This is the only place where the AzureAssignedIdentity creation is initiated.
		begin := time.Now()
		workDone := false

		// List all pods in all namespaces
		systemTime := time.Now()
		listPods, err := c.PodClient.GetPods()
		if err != nil {
			glog.Error(err)
			continue
		}
		listBindings, err := c.CRDClient.ListBindings()
		if err != nil {
			continue
		}
		listIDs, err := c.CRDClient.ListIds()
		if err != nil {
			continue
		}
		idMap, err := c.convertIDListToMap(*listIDs)
		if err != nil {
			glog.Error(err)
			continue
		}

		currentAssignedIDs, err := c.CRDClient.ListAssignedIDs()
		if err != nil {
			continue
		}
		stats.Put(stats.System, time.Since(systemTime))

		beginNewListTime := time.Now()
		newAssignedIDs, nodeRefs, err := c.createDesiredAssignedIdentityList(listPods, listBindings, idMap)
		if err != nil {
			glog.Error(err)
			continue
		}
		stats.Put(stats.CurrentState, time.Since(beginNewListTime))

		// Extract add list and delete list based on existing assigned ids in the system (currentAssignedIDs).
		// and the ones we have arrived at in the volatile list (newAssignedIDs).
		addList, err := c.getAzureAssignedIDsToCreate(*currentAssignedIDs, newAssignedIDs)
		if err != nil {
			glog.Error(err)
			continue
		}
		deleteList, err := c.getAzureAssignedIDsToDelete(currentAssignedIDs, &newAssignedIDs)
		if err != nil {
			glog.Error(err)
			continue
		}
		glog.V(5).Infof("del: %v, add: %v", deleteList, addList)

		nodeMap := make(map[string]trackUserAssignedMSIIds)

		// seperate the add and delete list per node
		c.convertAssignedIDListToMap(addList, deleteList, nodeMap)

		// process the delete and add list
		// determine the list of identities that need to updated, create a node to identity list mapping for add and delete
		if deleteList != nil && len(*deleteList) > 0 {
			workDone = true
			c.getListOfIdsToDelete(*deleteList, newAssignedIDs, nodeMap, nodeRefs)
		}
		if addList != nil && len(*addList) > 0 {
			workDone = true
			c.getListOfIdsToAssign(*addList, nodeMap)
		}

		// one final createorupdate to each node in the map
		c.updateNodeAndDeps(newAssignedIDs, nodeMap, nodeRefs)

		if workDone {
			idsFound := 0
			bindingsFound := 0
			if listIDs != nil {
				idsFound = len(*listIDs)
			}
			if listBindings != nil {
				bindingsFound = len(*listBindings)
			}
			glog.Infof("Found %d pods, %d ids, %d bindings", len(listPods), idsFound, bindingsFound)
			stats.Put(stats.Total, time.Since(begin))
			stats.PrintSync()
		}
	}
}

func (c *Client) convertAssignedIDListToMap(addList, deleteList *[]aadpodid.AzureAssignedIdentity, nodeMap map[string]trackUserAssignedMSIIds) {
	if addList != nil {
		for _, createID := range *addList {
			if trackList, ok := nodeMap[createID.Spec.NodeName]; ok {
				trackList.assignedIDsToCreate = append(trackList.assignedIDsToCreate, createID)
				nodeMap[createID.Spec.NodeName] = trackList
				continue
			}
			nodeMap[createID.Spec.NodeName] = trackUserAssignedMSIIds{assignedIDsToCreate: []aadpodid.AzureAssignedIdentity{createID}}
		}
	}
	if deleteList != nil {
		for _, delID := range *deleteList {
			if trackList, ok := nodeMap[delID.Spec.NodeName]; ok {
				trackList.assignedIDsToDelete = append(trackList.assignedIDsToDelete, delID)
				nodeMap[delID.Spec.NodeName] = trackList
				continue
			}
			nodeMap[delID.Spec.NodeName] = trackUserAssignedMSIIds{assignedIDsToDelete: []aadpodid.AzureAssignedIdentity{delID}}
		}
	}
}

func (c *Client) createDesiredAssignedIdentityList(
	listPods []*corev1.Pod, listBindings *[]aadpodid.AzureIdentityBinding, idMap map[string]aadpodid.AzureIdentity) ([]aadpodid.AzureAssignedIdentity, map[string]bool, error) {
	//For each pod, check what bindings are matching. For each binding create volatile azure assigned identity.
	//Compare this list with the current list of azure assigned identities.
	//For any new assigned identities found in this volatile list, create assigned identity and assign user assigned msis.
	//For any assigned ids not present the volatile list, proceed with the deletion.
	nodeRefs := make(map[string]bool)
	var newAssignedIDs []aadpodid.AzureAssignedIdentity

	for _, pod := range listPods {
		if pod.Spec.NodeName == "" {
			//Node is not yet allocated. In that case skip the pod
			glog.V(2).Infof("Pod %s/%s has no assigned node yet. it will be ignored", pod.Namespace, pod.Name)
			continue
		}
		crdPodLabelVal := pod.Labels[aadpodid.CRDLabelKey]
		if crdPodLabelVal == "" {
			//No binding mentioned in the label. Just continue to the next pod
			glog.V(2).Infof("Pod %s/%s has correct %s label but with no value. it will be ignored", pod.Namespace, pod.Name, aadpodid.CRDLabelKey)
			continue
		}
		var matchedBindings []aadpodid.AzureIdentityBinding
		for _, allBinding := range *listBindings {
			if allBinding.Spec.Selector == crdPodLabelVal {
				glog.V(5).Infof("Found binding match for pod %s/%s with binding %s", pod.Namespace, pod.Name, allBinding.Name)
				matchedBindings = append(matchedBindings, allBinding)
				nodeRefs[pod.Spec.NodeName] = true
			}
		}

		for _, binding := range matchedBindings {
			glog.V(5).Infof("Looking up id map: %v", binding.Spec.AzureIdentity)
			if azureID, idPresent := idMap[binding.Spec.AzureIdentity]; idPresent {
				// working in Namespaced mode or this specific identity is namespaced
				if c.IsNamespaced || aadpodid.IsNamespacedIdentity(&azureID) {
					// They have to match all
					if !(azureID.Namespace == binding.Namespace && binding.Namespace == pod.Namespace) {
						glog.V(5).Infof("identity %s/%s was matched via binding %s/%s to %s/%s but namespaced identity is enforced, so it will be ignored",
							azureID.Namespace, azureID.Name, binding.Namespace, binding.Name, pod.Name, pod.Namespace)
						continue
					}
				}
				glog.V(5).Infof("identity %s/%s assigned to %s/%s via %s/%s", azureID.Namespace, azureID.Name, pod.Namespace, pod.Name, binding.Namespace, binding.Namespace)
				assignedID, err := c.makeAssignedIDs(&azureID, &binding, pod.Name, pod.Namespace, pod.Spec.NodeName)

				if err != nil {
					glog.Errorf("failed to create assignment for pod %s/%s with identity %s/%s with error %v", pod.Namespace, pod.Name, azureID.Namespace, azureID.Name, err.Error())
					continue
				}
				newAssignedIDs = append(newAssignedIDs, *assignedID)
			} else {
				// This is the case where the identity has been deleted.
				// In such a case, we will skip it from matching binding.
				// This will ensure that the new assigned ids created will not have the
				// one associated with this azure identity.
				glog.V(5).Infof("%s identity not found when using %s binding", binding.Spec.AzureIdentity, binding.Name)
			}
		}
	}
	return newAssignedIDs, nodeRefs, nil
}

// getListOfIdsToDelete will go over the delete list to determine if the id is required to be deleted
// only user assigned identity not in use are added to the remove list for the node
func (c *Client) getListOfIdsToDelete(deleteList, newAssignedIDs []aadpodid.AzureAssignedIdentity, nodeMap map[string]trackUserAssignedMSIIds, nodeRefs map[string]bool) {
	vmssGroups, err := getVMSSGroups(c.NodeClient, nodeRefs)
	if err != nil {
		glog.Error(err)
		return
	}
	for _, delID := range deleteList {
		glog.V(5).Infof("Deletion of id: %s", delID.Name)
		inUse, err := c.checkIfInUse(delID, newAssignedIDs, vmssGroups)
		if err != nil {
			glog.Error(err)
			continue
		}
		id := delID.Spec.AzureIdentityRef
		isUserAssignedMSI := c.checkIfUserAssignedMSI(id)

		// this case includes Assigned state and empty state to ensure backward compatability
		if delID.Status.Status == aadpodid.AssignedIDAssigned || delID.Status.Status == "" {
			if !inUse && isUserAssignedMSI {
				c.appendToRemoveListForNode(id.Spec.ResourceID, delID.Spec.NodeName, nodeMap)
			}
		}
		glog.V(5).Infof("Binding removed: %+v", delID.Spec.AzureBindingRef)
	}
}

// getListOfIdsToAssign will add the id to the append list for node if it's user assigned identity
func (c *Client) getListOfIdsToAssign(addList []aadpodid.AzureAssignedIdentity, nodeMap map[string]trackUserAssignedMSIIds) {
	for _, createID := range addList {
		id := createID.Spec.AzureIdentityRef
		isUserAssignedMSI := c.checkIfUserAssignedMSI(id)

		if createID.Status.Status == "" || createID.Status.Status == aadpodid.AssignedIDCreated {
			if isUserAssignedMSI {
				c.appendToAddListForNode(id.Spec.ResourceID, createID.Spec.NodeName, nodeMap)
			}
		}
		glog.V(5).Infof("Binding applied: %+v", createID.Spec.AzureBindingRef)
	}
}

func (c *Client) matchAssignedID(x *aadpodid.AzureAssignedIdentity, y *aadpodid.AzureAssignedIdentity) (ret bool, err error) {
	bindingX := x.Spec.AzureBindingRef
	bindingY := y.Spec.AzureBindingRef

	idX := x.Spec.AzureIdentityRef
	idY := y.Spec.AzureIdentityRef

	if bindingX.Name == bindingY.Name && bindingX.ResourceVersion == bindingY.ResourceVersion &&
		idX.Name == idY.Name && idX.ResourceVersion == idY.ResourceVersion &&
		x.Spec.Pod == y.Spec.Pod && x.Spec.PodNamespace == y.Spec.PodNamespace && x.Spec.NodeName == y.Spec.NodeName {
		return true, nil
	}
	return false, nil
}

func (c *Client) getAzureAssignedIDsToCreate(old, new []aadpodid.AzureAssignedIdentity) (*[]aadpodid.AzureAssignedIdentity, error) {
	// everything in new needs to be created
	if old == nil || len(old) == 0 {
		return &new, nil
	}

	create := make([]aadpodid.AzureAssignedIdentity, 0)
	var err error
	idMatch := false
	begin := time.Now()
	// TODO: We should be able to optimize the many for loops.
	for _, newAssignedID := range new {
		idMatch = false
		for _, oldAssignedID := range old {
			idMatch, err = c.matchAssignedID(&newAssignedID, &oldAssignedID)
			if err != nil {
				glog.Error(err)
				continue
			}
			if idMatch {
				// if the old assigned id is in created state, then the identity assignment to the node
				// is not done. Adding to the list will ensure we retry identity assignment to node for
				// this assigned identity.
				if oldAssignedID.Status.Status == aadpodid.AssignedIDCreated {
					create = append(create, oldAssignedID)
				}
				break
			}
		}
		if !idMatch {
			glog.V(5).Infof("ok: %v, Create added: %s", idMatch, newAssignedID.Name)
			// We are done checking that this new id is not present in the old
			// list. So we will add it to the create list.
			create = append(create, newAssignedID)
		}
	}
	stats.Put(stats.FindAssignedIDCreate, time.Since(begin))
	return &create, nil
}

func (c *Client) getAzureAssignedIDsToDelete(old *[]aadpodid.AzureAssignedIdentity, new *[]aadpodid.AzureAssignedIdentity) (*[]aadpodid.AzureAssignedIdentity, error) {
	// nothing to delete
	if old == nil || len(*old) == 0 {
		return nil, nil
	}
	// delete everything as nothing in new
	if new == nil || len(*new) == 0 {
		return old, nil
	}

	delete := make([]aadpodid.AzureAssignedIdentity, 0)
	var err error
	idMatch := false

	begin := time.Now()
	for _, oldAssignedID := range *old {
		idMatch = false
		for _, newAssignedID := range *new {
			idMatch, err = c.matchAssignedID(&newAssignedID, &oldAssignedID)
			if err != nil {
				glog.Error(err)
				continue
			}
			//glog.Infof("Match %s %s %v", newAssignedID.Name, oldAssignedID.Name, idMatch)
			if idMatch {
				break
			}
		}
		if !idMatch {
			glog.V(5).Infof("ok: %v, Delete added: %s", idMatch, oldAssignedID.Name)
			// We are done checking that this old id is not present in the new
			// list. So we will add it to the delete list.
			delete = append(delete, oldAssignedID)
		}
	}
	stats.Put(stats.FindAssignedIDDel, time.Since(begin))
	return &delete, nil
}

func (c *Client) makeAssignedIDs(azID *aadpodid.AzureIdentity, azBinding *aadpodid.AzureIdentityBinding, podName, podNameSpace, nodeName string) (res *aadpodid.AzureAssignedIdentity, err error) {
	assignedID := &aadpodid.AzureAssignedIdentity{
		ObjectMeta: v1.ObjectMeta{
			Name: c.getAssignedIDName(podName, podNameSpace, azID.Name),
		},
		Spec: aadpodid.AzureAssignedIdentitySpec{
			AzureIdentityRef: azID,
			AzureBindingRef:  azBinding,
			Pod:              podName,
			PodNamespace:     podNameSpace,
			NodeName:         nodeName,
		},
		Status: aadpodid.AzureAssignedIdentityStatus{
			AvailableReplicas: 1,
		},
	}
	// if we are in namespaced mode (or az identity is namespaced)
	if c.IsNamespaced || aadpodid.IsNamespacedIdentity(azID) {
		assignedID.Namespace = azID.Namespace
	} else {
		// eventually this should be identity namespace
		// but to maintain back compat we will use existing
		// behavior
		assignedID.Namespace = "default"
	}

	glog.V(5).Infof("Making assigned ID: %v", assignedID)
	return assignedID, nil
}

func (c *Client) createAssignedIdentity(assignedID *aadpodid.AzureAssignedIdentity) error {
	err := c.CRDClient.CreateAssignedIdentity(assignedID)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) removeAssignedIdentity(assignedID *aadpodid.AzureAssignedIdentity) error {
	err := c.CRDClient.RemoveAssignedIdentity(assignedID)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) appendToRemoveListForNode(resourceID, nodeName string, nodeMap map[string]trackUserAssignedMSIIds) {
	if trackList, ok := nodeMap[nodeName]; ok {
		trackList.removeUserAssignedMSIIDs = append(trackList.removeUserAssignedMSIIDs, resourceID)
		nodeMap[nodeName] = trackList
		return
	}
	nodeMap[nodeName] = trackUserAssignedMSIIds{removeUserAssignedMSIIDs: []string{resourceID}}
}

func (c *Client) appendToAddListForNode(resourceID, nodeName string, nodeMap map[string]trackUserAssignedMSIIds) {
	if trackList, ok := nodeMap[nodeName]; ok {
		trackList.addUserAssignedMSIIDs = append(trackList.addUserAssignedMSIIDs, resourceID)
		nodeMap[nodeName] = trackList
		return
	}
	nodeMap[nodeName] = trackUserAssignedMSIIds{addUserAssignedMSIIDs: []string{resourceID}}
}

func (c *Client) checkIfUserAssignedMSI(id *aadpodid.AzureIdentity) bool {
	return id.Spec.Type == aadpodid.UserAssignedMSI
}

func (c *Client) getAssignedIDName(podName, podNameSpace, idName string) string {
	return podName + "-" + podNameSpace + "-" + idName
}

func (c *Client) checkIfMSIExistsOnNode(id *aadpodid.AzureIdentity, nodeName string, nodeMSIList []string) bool {
	for _, userAssignedMSI := range nodeMSIList {
		if userAssignedMSI == id.Spec.ResourceID {
			return true
		}
	}
	return false
}

func (c *Client) getUserMSIListForNode(node *corev1.Node) ([]string, error) {
	return c.CloudClient.GetUserMSIs(node)
}

func (c *Client) convertIDListToMap(arr []aadpodid.AzureIdentity) (m map[string]aadpodid.AzureIdentity, err error) {
	m = make(map[string]aadpodid.AzureIdentity, len(arr))
	for _, element := range arr {
		m[element.Name] = element
	}
	return m, nil
}

func (c *Client) checkIfInUse(checkAssignedID aadpodid.AzureAssignedIdentity, arr []aadpodid.AzureAssignedIdentity, vmssGroups *vmssGroupList) (bool, error) {
	for _, assignedID := range arr {
		checkID := checkAssignedID.Spec.AzureIdentityRef
		id := assignedID.Spec.AzureIdentityRef
		// If they have the same client id, reside on the same node but the pod name is different, then the
		// assigned id is in use.
		// This is applicable only for user assigned MSI since that is node specific. Ignore other cases.
		if checkID.Spec.Type != aadpodid.UserAssignedMSI {
			continue
		}

		if checkAssignedID.Spec.Pod == assignedID.Spec.Pod {
			// No need to do the rest of the checks in this case, since it's the same assignment
			// The same identity won't be assigned to a pod twice, so it's the same reference.
			continue
		}

		if checkID.Spec.ClientID != id.Spec.ClientID {
			continue
		}

		if checkAssignedID.Spec.NodeName == assignedID.Spec.NodeName {
			return true, nil
		}

		vmss, err := getVMSSGroupFromPossiblyUnreferencedNode(c.NodeClient, vmssGroups, checkAssignedID.Spec.NodeName)
		if err != nil {
			return false, err
		}

		// check if this identity is used on another node in the same vmss
		// This check is needed because vmss identities currently operate on all nodes
		// in the vmss not just a single node.
		if vmss != nil && vmss.hasNode(assignedID.Spec.NodeName) {
			return true, nil
		}
	}

	return false, nil
}

func (c *Client) getUniqueIDs(idList []string) []string {
	idSet := make(map[string]struct{})
	var uniqueList []string

	for _, id := range idList {
		idSet[id] = struct{}{}
	}
	for id := range idSet {
		uniqueList = append(uniqueList, id)
	}
	return uniqueList
}

func (c *Client) updateAssignedIdentityStatus(assignedID *aadpodid.AzureAssignedIdentity, status string) error {
	return c.CRDClient.UpdateAzureAssignedIdentityStatus(assignedID, status)
}

func (c *Client) updateNodeAndDeps(newAssignedIDs []aadpodid.AzureAssignedIdentity, nodeMap map[string]trackUserAssignedMSIIds, nodeRefs map[string]bool) {
	var wg sync.WaitGroup
	for nodeName, nodeTrackList := range nodeMap {
		wg.Add(1)
		go c.updateUserMSI(newAssignedIDs, nodeName, nodeTrackList, nodeRefs, &wg)
	}
	wg.Wait()
}

func (c *Client) updateUserMSI(newAssignedIDs []aadpodid.AzureAssignedIdentity, nodeName string, nodeTrackList trackUserAssignedMSIIds, nodeRefs map[string]bool, wg *sync.WaitGroup) {
	defer wg.Done()
	beginAdding := time.Now()
	glog.V(5).Infof("Processing node %s", nodeName)

	for _, createID := range nodeTrackList.assignedIDsToCreate {
		if createID.Status.Status == "" {
			binding := createID.Spec.AzureBindingRef

			// this is the state when the azure assigned identity is yet to be created
			glog.V(5).Infof("Initiating assigned id creation for pod - %s, binding - %s", createID.Spec.Pod, binding.Name)

			createID.Status.Status = aadpodid.AssignedIDCreated
			err := c.createAssignedIdentity(&createID)
			if err != nil {
				c.EventRecorder.Event(binding, corev1.EventTypeWarning, "binding apply error",
					fmt.Sprintf("Creating assigned identity for pod %s resulted in error %v", createID.Name, err))
				glog.Error(err)
			}
		}
	}
	// generate unique list so we don't make multiple calls to assign/remove same id
	addUserAssignedMSIIDs := c.getUniqueIDs(nodeTrackList.addUserAssignedMSIIDs)
	removeUserAssignedMSIIDs := c.getUniqueIDs(nodeTrackList.removeUserAssignedMSIIDs)

	node, err := c.NodeClient.Get(nodeName)
	if err != nil {
		glog.Errorf("Lookup of node %s resulted in error %v", nodeName, err)
		return
	}

	err = c.CloudClient.UpdateUserMSI(addUserAssignedMSIIDs, removeUserAssignedMSIIDs, node)
	if err != nil {
		glog.Errorf("Updating msi's on node %s, add [%d], del [%d] failed with error %v", nodeName, len(nodeTrackList.assignedIDsToCreate), len(nodeTrackList.assignedIDsToDelete), err)
		idList, getErr := c.getUserMSIListForNode(node)
		if getErr != nil {
			glog.Errorf("Getting list of msi's from node %s resulted in error %v", nodeName, getErr)
			return
		}

		for _, createID := range nodeTrackList.assignedIDsToCreate {
			id := createID.Spec.AzureIdentityRef
			binding := createID.Spec.AzureBindingRef

			isUserAssignedMSI := c.checkIfUserAssignedMSI(id)
			idExistsOnNode := c.checkIfMSIExistsOnNode(id, createID.Spec.NodeName, idList)

			if isUserAssignedMSI && !idExistsOnNode {
				message := fmt.Sprintf("Applying binding %s node %s for pod %s resulted in error %v", binding.Name, createID.Spec.NodeName, createID.Name, err.Error())
				c.EventRecorder.Event(binding, corev1.EventTypeWarning, "binding apply error", message)
				glog.Error(message)
				continue
			}
			// the identity was successfully assigned to the node
			c.EventRecorder.Event(binding, corev1.EventTypeNormal, "binding applied",
				fmt.Sprintf("Binding %s applied on node %s for pod %s", binding.Name, createID.Spec.NodeName, createID.Name))

			// Identity is successfully assigned to node, so update the status of assigned identity to assigned
			if updateErr := c.updateAssignedIdentityStatus(&createID, aadpodid.AssignedIDAssigned); updateErr != nil {
				message := fmt.Sprintf("Updating assigned identity %s status to %s for pod %s failed with error %v", createID.Name, aadpodid.AssignedIDAssigned, createID.Spec.Pod, updateErr.Error())
				c.EventRecorder.Event(&createID, corev1.EventTypeWarning, "status update error", message)
				glog.Error(message)
			}
		}

		for _, delID := range nodeTrackList.assignedIDsToDelete {
			id := delID.Spec.AzureIdentityRef
			removedBinding := delID.Spec.AzureBindingRef
			isUserAssignedMSI := c.checkIfUserAssignedMSI(id)
			idExistsOnNode := c.checkIfMSIExistsOnNode(id, delID.Spec.NodeName, idList)
			vmssGroups, getErr := getVMSSGroups(c.NodeClient, nodeRefs)
			if getErr != nil {
				glog.Error(getErr)
				continue
			}
			inUse, checkErr := c.checkIfInUse(delID, newAssignedIDs, vmssGroups)
			if checkErr != nil {
				glog.Error(checkErr)
				continue
			}
			// the identity still exists on node, which means removing the identity from the node failed
			if isUserAssignedMSI && !inUse && idExistsOnNode {
				message := fmt.Sprintf("Binding %s removal from node %s for pod %s resulted in error %v", removedBinding.Name, delID.Spec.NodeName, delID.Spec.Pod, err.Error())
				c.EventRecorder.Event(removedBinding, corev1.EventTypeWarning, "binding remove error", message)
				glog.Error(message)
				continue
			}
			// remove assigned identity crd from cluster as the identity has successfully been removed from the node
			err = c.removeAssignedIdentity(&delID)
			if err != nil {
				c.EventRecorder.Event(removedBinding, corev1.EventTypeWarning, "binding remove error",
					fmt.Sprintf("Removing assigned identity binding %s node %s for pod %s resulted in error %v", removedBinding.Name, delID.Spec.NodeName, delID.Name, err.Error()))
				glog.Error(err)
				continue
			}
			// the identity was successfully removed from node
			c.EventRecorder.Event(removedBinding, corev1.EventTypeNormal, "binding removed",
				fmt.Sprintf("Binding %s removed from node %s for pod %s", removedBinding.Name, delID.Spec.NodeName, delID.Spec.Pod))
		}
		stats.Put(stats.TotalCreateOrUpdate, time.Since(beginAdding))
		return
	}

	for _, createID := range nodeTrackList.assignedIDsToCreate {
		binding := createID.Spec.AzureBindingRef
		// update the status to assigned for assigned identity as identity was successfully assigned to node.
		err = c.updateAssignedIdentityStatus(&createID, aadpodid.AssignedIDAssigned)
		if err != nil {
			message := fmt.Sprintf("Updating assigned identity %s status to %s for pod %s failed with error %v", createID.Name, aadpodid.AssignedIDAssigned, createID.Spec.Pod, err.Error())
			c.EventRecorder.Event(&createID, corev1.EventTypeWarning, "status update error", message)
			glog.Error(message)
			continue
		}
		c.EventRecorder.Event(binding, corev1.EventTypeNormal, "binding applied",
			fmt.Sprintf("Binding %s applied on node %s for pod %s", binding.Name, createID.Spec.NodeName, createID.Name))
	}

	for _, delID := range nodeTrackList.assignedIDsToDelete {
		removedBinding := delID.Spec.AzureBindingRef

		// update the status for the assigned identity to Unassigned as the identity has been successfully removed from node.
		// this will ensure on next sync loop we only try to delete the assigned identity instead of doing everything.
		err = c.updateAssignedIdentityStatus(&delID, aadpodid.AssignedIDUnAssigned)
		if err != nil {
			message := fmt.Sprintf("Updating assigned identity %s status to %s for pod %s failed with error %v", delID.Name, aadpodid.AssignedIDUnAssigned, delID.Spec.Pod, err.Error())
			c.EventRecorder.Event(&delID, corev1.EventTypeWarning, "status update error", message)
			glog.Error(message)
			continue
		}
		// remove assigned identity crd from cluster as the identity has successfully been removed from the node
		err = c.removeAssignedIdentity(&delID)
		if err != nil {
			c.EventRecorder.Event(removedBinding, corev1.EventTypeWarning, "binding remove error",
				fmt.Sprintf("Removing assigned identity binding %s node %s for pod %s resulted in error %v", removedBinding.Name, delID.Spec.NodeName, delID.Name, err))
			glog.Error(err)
			continue
		}
		// the identity was successfully removed from node
		c.EventRecorder.Event(removedBinding, corev1.EventTypeNormal, "binding removed",
			fmt.Sprintf("Binding %s removed from node %s for pod %s", removedBinding.Name, delID.Spec.NodeName, delID.Spec.Pod))
	}
	stats.Put(stats.TotalCreateOrUpdate, time.Since(beginAdding))
}
