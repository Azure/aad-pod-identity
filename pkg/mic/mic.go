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
	// IdentityCreated status indicates azure assigned identity is created
	IdentityCreated = "Created"
	// IdentityAssigned status indicates identity has been assigned to the node
	IdentityAssigned = "Assigned"
	// IdentityUnassigned status indicates identity has been unassigned from the node
	IdentityUnassigned = "Unassigned"
)

// NodeGetter ...
type NodeGetter interface {
	Get(name string) (*corev1.Node, error)
	Start(<-chan struct{})
}

// Client has the required pointers to talk to the api server
// and interact with the CRD related datastructure.
type Client struct {
	CRDClient     crd.ClientInt
	CloudClient   cloudprovider.ClientInt
	PodClient     pod.ClientInt
	EventRecorder record.EventRecorder
	EventChannel  chan aadpodid.EventType
	NodeClient    NodeGetter
	IsNamespaced  bool

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
	node                     *corev1.Node
}

// NewMICClient returnes new mic client
func NewMICClient(cloudconfig string, config *rest.Config, isNamespaced bool) (*Client, error) {
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
		CRDClient:     crdClient,
		CloudClient:   cloudClient,
		PodClient:     podClient,
		EventRecorder: recorder,
		EventChannel:  eventCh,
		NodeClient:    &NodeClient{informer.Core().V1().Nodes()},
		IsNamespaced:  isNamespaced,
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

	glog.Info("Sync thread started.")
	var event aadpodid.EventType
	for {
		select {
		case <-exit:
			return
		case event = <-c.EventChannel:
		}

		stats.Init()
		// This is the only place where the AzureAssignedIdentity creation is initiated.
		begin := time.Now()
		workDone := false
		glog.V(6).Infof("Received event: %v", event)
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
		// get list of new assigned ids to create
		addList, err := c.getAzureAssignedIDsToCreate(currentAssignedIDs, &newAssignedIDs)
		if err != nil {
			glog.Error(err)
			continue
		}
		// get list of assigned ids to delete
		deleteList, err := c.getAzureAssignedIDsToDelete(currentAssignedIDs, &newAssignedIDs)
		if err != nil {
			glog.Error(err)
			continue
		}
		glog.V(5).Infof("del: %v, add: %v", deleteList, addList)

		nodeMap := make(map[string]trackUserAssignedMSIIds)
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
		// TODO: parallelize this process
		nodeWithErrors := c.updateUserMSI(nodeMap)

		c.handleNodeWithSuccess(nodeWithErrors, addList, deleteList)
		// handle node errors
		if len(nodeWithErrors) > 0 {
			c.handleNodeErrors(nodeWithErrors, addList, deleteList, newAssignedIDs, nodeRefs)
		}

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
				glog.V(5).Infof("Found binding match for pod %s/%s with binding %s", pod.Name, pod.Namespace, allBinding.Name)
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
							azureID.Namespace, azureID.Name, binding.Namespace, binding.Name, pod.Namespace, pod.Name)
						continue
					}
				}
				glog.V(5).Infof("identity %s/%s assigned to %s/%s via %s/%s", azureID.Namespace, azureID.Name, pod.Namespace, pod.Name, binding.Namespace, binding.Namespace)
				assignedID, err := c.makeAssignedIDs(&azureID, &binding, pod.Name, pod.Namespace, pod.Spec.NodeName)

				if err != nil {
					glog.Errorf("failed to create assignment for pod %s/%s with identity %s/%s with error %v", pod.Name, pod.Namespace, azureID.Namespace, azureID.Name, err.Error())
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
		removedBinding := delID.Spec.AzureBindingRef
		isUserAssignedMSI := c.checkIfUserAssignedMSI(id)

		node, err := c.NodeClient.Get(delID.Spec.NodeName)
		if err != nil {
			c.EventRecorder.Event(removedBinding, corev1.EventTypeWarning, "get node error",
				fmt.Sprintf("Lookup of node %s for pod %s resulted in error %v", delID.Spec.NodeName, delID.Name, err))
			continue
		}

		switch delID.Status.Status {
		// this case includes Assigned state and empty state to ensure backward compatability
		case IdentityAssigned, "":
			if !inUse && isUserAssignedMSI {
				c.appendToRemoveListForNode(nodeMap, id.Spec.ResourceID, node)
			}

		case IdentityCreated, IdentityUnassigned:
			// assigned identity is in this state, if the identity was successfully removed from node
			// but api calls to delete crd failed. In this state we just need to delete the assigned identity object.
			err := c.removeAssignedIdentity(&delID)
			if err != nil {
				c.EventRecorder.Event(removedBinding, corev1.EventTypeWarning, "binding remove error",
					fmt.Sprintf("Removing assigned identity binding %s node %s for pod %s resulted in error %v", removedBinding.Name, delID.Spec.NodeName, delID.Name, err))
				glog.Error(err)

				// update the status for the assigned identity to Unassigned as the identity has been successfully removed from node.
				// this will ensure on next sync loop we only try to delete the assigned identity instead of doing everything.
				// update the status to unassigned for assigned identity
				if err := c.updateAssignedIdentityStatus(&delID, IdentityUnassigned); err != nil {
					message := fmt.Sprintf("Updating assigned identity %s status to %s for pod %s failed with error %v", delID.Name, IdentityUnassigned, delID.Spec.Pod, err.Error())
					c.EventRecorder.Event(&delID, corev1.EventTypeWarning, "status update error", message)
					glog.Error(message)
				}
			}
		}
		glog.V(5).Infof("Binding removed: %+v", removedBinding)
	}
}

// getListOfIdsToAssign will first create the assigned identity resource for all id in addList
// then it will add the id to the append list for node if it's user assigned identity
func (c *Client) getListOfIdsToAssign(addList []aadpodid.AzureAssignedIdentity, nodeMap map[string]trackUserAssignedMSIIds) {
loop:
	for _, createID := range addList {
		id := createID.Spec.AzureIdentityRef
		binding := createID.Spec.AzureBindingRef
		isUserAssignedMSI := c.checkIfUserAssignedMSI(id)

		node, err := c.NodeClient.Get(createID.Spec.NodeName)
		if err != nil {
			c.EventRecorder.Event(binding, corev1.EventTypeWarning, "get node error",
				fmt.Sprintf("Lookup of node %s for pod %s resulted in error %v", createID.Spec.NodeName, createID.Name, err))
			continue
		}

		switch createID.Status.Status {
		case "":
			// this is the state when the azure assigned identity is yet to be created
			glog.V(5).Infof("Initiating assigned id creation for pod - %s, binding - %s", createID.Spec.Pod, binding.Name)

			createID.Status.Status = IdentityCreated
			if err = c.createAssignedIdentity(&createID); err != nil {
				c.EventRecorder.Event(binding, corev1.EventTypeWarning, "binding apply error",
					fmt.Sprintf("Creating assigned identity for pod %s resulted in error %v", createID.Name, err))
				glog.Error(err)
				break loop
			}
			// append the id to list of ids to be assigned for node
			if isUserAssignedMSI {
				c.appendToAddListForNode(nodeMap, id.Spec.ResourceID, node)
			}

		case IdentityCreated:
			// this is the state when the assigned identity was successfully created
			// but failed to assign the identity to the node
			if isUserAssignedMSI {
				c.appendToAddListForNode(nodeMap, id.Spec.ResourceID, node)
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

func (c *Client) getAzureAssignedIDsToCreate(old *[]aadpodid.AzureAssignedIdentity, new *[]aadpodid.AzureAssignedIdentity) (*[]aadpodid.AzureAssignedIdentity, error) {
	// everything in new needs to be created
	if old == nil || len(*old) == 0 {
		return new, nil
	}

	create := make([]aadpodid.AzureAssignedIdentity, 0)
	var err error
	idMatch := false
	begin := time.Now()
	// TODO: We should be able to optimize the many for loops.
	for _, newAssignedID := range *new {
		idMatch = false
		for _, oldAssignedID := range *old {
			idMatch, err = c.matchAssignedID(&newAssignedID, &oldAssignedID)
			if err != nil {
				glog.Error(err)
				continue
			}
			if idMatch {
				// if the old assigned id is in created state, then the identity assignment to the node
				// is not done. Adding to the list will ensure we retry identity assignment to node for
				// this assigned identity.
				if oldAssignedID.Status.Status == IdentityCreated {
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
		// evantually this should be identity namespace
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

func (c *Client) appendToRemoveListForNode(nodeMap map[string]trackUserAssignedMSIIds, resourceID string, node *corev1.Node) {
	if trackList, ok := nodeMap[node.Name]; ok {
		if trackList.removeUserAssignedMSIIDs != nil {
			trackList.removeUserAssignedMSIIDs = append(trackList.removeUserAssignedMSIIDs, resourceID)
			trackList.node = node
		} else {
			trackList.removeUserAssignedMSIIDs = []string{resourceID}
			trackList.node = node
		}
		nodeMap[node.Name] = trackList
	} else {
		nodeMap[node.Name] = trackUserAssignedMSIIds{removeUserAssignedMSIIDs: []string{resourceID}, node: node}
	}
}

func (c *Client) appendToAddListForNode(nodeMap map[string]trackUserAssignedMSIIds, resourceID string, node *corev1.Node) {
	if trackList, ok := nodeMap[node.Name]; ok {
		if trackList.addUserAssignedMSIIDs != nil {
			trackList.addUserAssignedMSIIDs = append(trackList.addUserAssignedMSIIDs, resourceID)
			trackList.node = node
		} else {
			trackList.addUserAssignedMSIIDs = []string{resourceID}
			trackList.node = node
		}
		nodeMap[node.Name] = trackList
	} else {
		nodeMap[node.Name] = trackUserAssignedMSIIds{addUserAssignedMSIIDs: []string{resourceID}, node: node}
	}
}

// handleNodeWithSuccess will generate a success event for all new assigned identities created on the successful nodes and
// remove the assigned identity in delete list for all successfull nodes
func (c *Client) handleNodeWithSuccess(nodeWithErrors []string, addList, deleteList *[]aadpodid.AzureAssignedIdentity) {
	isNodeSuccess := func(node string) bool {
		for _, n := range nodeWithErrors {
			if n == node {
				return false
			}
		}
		return true
	}

	if addList != nil && len(*addList) > 0 {
		for _, createID := range *addList {
			binding := createID.Spec.AzureBindingRef

			if isNodeSuccess(createID.Spec.NodeName) {
				// the identity was successfully assigned to the node
				c.EventRecorder.Event(binding, corev1.EventTypeNormal, "binding applied",
					fmt.Sprintf("Binding %s applied on node %s for pod %s", binding.Name, createID.Spec.NodeName, createID.Name))

				// update the status to assigned for assigned identity
				if err := c.updateAssignedIdentityStatus(&createID, IdentityAssigned); err != nil {
					message := fmt.Sprintf("Updating assigned identity %s status to %s for pod %s failed with error %v", createID.Name, IdentityAssigned, createID.Spec.Pod, err.Error())
					c.EventRecorder.Event(&createID, corev1.EventTypeWarning, "status update error", message)
					glog.Error(message)
				}
			}
		}
	}

	if deleteList != nil && len(*deleteList) > 0 {
		for _, delID := range *deleteList {
			removedBinding := delID.Spec.AzureBindingRef

			if isNodeSuccess(delID.Spec.NodeName) {
				// remove assigned identity crd from cluster
				err := c.removeAssignedIdentity(&delID)
				if err != nil {
					c.EventRecorder.Event(removedBinding, corev1.EventTypeWarning, "binding remove error",
						fmt.Sprintf("Removing assigned identity binding %s node %s for pod %s resulted in error %v", removedBinding.Name, delID.Spec.NodeName, delID.Name, err))
					glog.Error(err)

					// update the status for the assigned identity to Unassigned as the identity has been successfully removed from node.
					// this will ensure on next sync loop we only try to delete the assigned identity instead of doing everything.
					// update the status to unassigned for assigned identity
					if err := c.updateAssignedIdentityStatus(&delID, IdentityUnassigned); err != nil {
						message := fmt.Sprintf("Updating assigned identity %s status to %s for pod %s failed with error %v", delID.Name, IdentityUnassigned, delID.Spec.Pod, err.Error())
						c.EventRecorder.Event(&delID, corev1.EventTypeWarning, "status update error", message)
						glog.Error(message)
					}
					continue
				}

				// the identity was successfully removed from node
				c.EventRecorder.Event(removedBinding, corev1.EventTypeNormal, "binding removed",
					fmt.Sprintf("Binding %s removed from node %s for pod %s", removedBinding.Name, delID.Spec.NodeName, delID.Spec.Pod))
			}
		}
	}
}

// handleNodeErrors is a redundant check performed for nodes that errored out during UpdateUserMSI call
// check is performed to ensure the identity exists on the node for assigned identity in add list and identity doesn't
// exist for assigned identity in delete list.
func (c *Client) handleNodeErrors(nodesWithError []string, addList, deleteList *[]aadpodid.AzureAssignedIdentity, newAssignedIDs []aadpodid.AzureAssignedIdentity, nodeRefs map[string]bool) error {
	nodeIdentityList := make(map[string][]string)
	for _, n := range nodesWithError {
		// get the list of user assigned identities currently on node
		node, err := c.NodeClient.Get(n)
		if err != nil {
			glog.Errorf("Lookup of node %s resulted in error %v", n, err)
			continue
		}
		idList, err := c.getUserMSIListForNode(node)
		if err != nil {
			glog.Errorf("Getting list of msi's from node %s resulted in error %v", n, err)
			// don't add the node to the map if we fail to get list of identities
			continue
		}
		nodeIdentityList[n] = idList
	}

	isNodeErrored := func(node string) bool {
		for _, n := range nodesWithError {
			if n == node {
				return true
			}
		}
		return false
	}

	// for all assigned identities in add list
	//	- if assigned identity spec node doesn't have any error, then continue
	// 	- if assigned identity spec node errored during UpdateUserMSI call, but the identity exists on the node, then continue
	// 	- if the user assigned identity doesn't exist on the node, then assigned identity is removed and will be retried in next sync cycle
	if addList != nil && len(*addList) > 0 {
		for _, createID := range *addList {
			id := createID.Spec.AzureIdentityRef
			binding := createID.Spec.AzureBindingRef
			isUserAssignedMSI := c.checkIfUserAssignedMSI(id)
			idExistsOnNode := c.checkIfMSIExistsOnNode(id, createID.Spec.NodeName, nodeIdentityList)
			nodeHasErrors := isNodeErrored(createID.Spec.NodeName)

			// node has no errors, events for this success are already generated in handleSuccessNodes
			if !nodeHasErrors {
				continue
			}
			// if the node has errors and we failed to get the list of identities currently on node,
			// then make no changes to the state of the assigned identity as we need to retry in next sync cycle.
			_, ok := nodeIdentityList[createID.Spec.NodeName]
			if (!ok && nodeHasErrors) || (isUserAssignedMSI && !idExistsOnNode) {
				message := fmt.Sprintf("Applying binding %s node %s for pod %s resulted in error", binding.Name, createID.Spec.NodeName, createID.Name)
				c.EventRecorder.Event(binding, corev1.EventTypeWarning, "binding apply error", message)
				glog.Error(message)
				continue
			}
			// the identity was successfully assigned to the node
			c.EventRecorder.Event(binding, corev1.EventTypeNormal, "binding applied",
				fmt.Sprintf("Binding %s applied on node %s for pod %s", binding.Name, createID.Spec.NodeName, createID.Name))

			// Identity is successfully assigned to node, so update the status of assigned identity to assigned
			if err := c.updateAssignedIdentityStatus(&createID, IdentityAssigned); err != nil {
				message := fmt.Sprintf("Updating assigned identity %s status to %s for pod %s failed with error %v", createID.Name, IdentityAssigned, createID.Spec.Pod, err.Error())
				c.EventRecorder.Event(&createID, corev1.EventTypeWarning, "status update error", message)
				glog.Error(message)
			}
		}
	}

	if deleteList != nil && len(*deleteList) > 0 {
		for _, delID := range *deleteList {
			nodeHasErrors := isNodeErrored(delID.Spec.NodeName)
			id := delID.Spec.AzureIdentityRef
			removedBinding := delID.Spec.AzureBindingRef
			isUserAssignedMSI := c.checkIfUserAssignedMSI(id)
			idExistsOnNode := c.checkIfMSIExistsOnNode(id, delID.Spec.NodeName, nodeIdentityList)

			// if node has no errors, then the assigned identity is deleted in handleSuccessNodes
			if !nodeHasErrors {
				continue
			}
			// if the node has errors and we failed to get the list of identities currently on node,
			// then make no changes to the state of the assigned identity as we need to retry in next sync cycle.
			if _, ok := nodeIdentityList[delID.Spec.NodeName]; !ok && nodeHasErrors {
				message := fmt.Sprintf("Binding %s removal from node %s for pod %s failed", removedBinding.Name, delID.Spec.NodeName, delID.Spec.Pod)
				c.EventRecorder.Event(removedBinding, corev1.EventTypeWarning, "binding remove error", message)
				glog.Error(message)
				continue
			}
			vmssGroups, err := getVMSSGroups(c.NodeClient, nodeRefs)
			if err != nil {
				glog.Error(err)
				continue
			}
			inUse, err := c.checkIfInUse(delID, newAssignedIDs, vmssGroups)
			if err != nil {
				glog.Error(err)
				continue
			}
			// the identity still exists on node, which means removing the identity from the node failed
			if isUserAssignedMSI && !inUse && idExistsOnNode {
				message := fmt.Sprintf("Binding %s removal from node %s for pod %s failed", removedBinding.Name, delID.Spec.NodeName, delID.Spec.Pod)
				c.EventRecorder.Event(removedBinding, corev1.EventTypeWarning, "binding remove error", message)
				glog.Error(message)
				continue
			}
			// remove assigned identity crd from cluster as the identity has successfully been removed from the node
			err = c.removeAssignedIdentity(&delID)
			if err != nil {
				c.EventRecorder.Event(removedBinding, corev1.EventTypeWarning, "binding remove error",
					fmt.Sprintf("Removing assigned identity binding %s node %s for pod %s resulted in error %v", removedBinding.Name, delID.Spec.NodeName, delID.Name, err))
				glog.Error(err)

				// update the status for the assigned identity to Unassigned as the identity has been successfully removed from node.
				// this will ensure on next sync loop we only try to delete the assigned identity instead of doing everything.
				// update the status to unassigned for assigned identity
				if err := c.updateAssignedIdentityStatus(&delID, IdentityUnassigned); err != nil {
					message := fmt.Sprintf("Updating assigned identity %s status to %s for pod %s failed with error %v", delID.Name, IdentityUnassigned, delID.Spec.Pod, err.Error())
					c.EventRecorder.Event(&delID, corev1.EventTypeWarning, "status update error", message)
					glog.Error(message)
				}
				continue
			}

			// the identity was successfully removed from node
			c.EventRecorder.Event(removedBinding, corev1.EventTypeNormal, "binding removed",
				fmt.Sprintf("Binding %s removed from node %s for pod %s", removedBinding.Name, delID.Spec.NodeName, delID.Spec.Pod))
		}
	}
	return nil
}

func (c *Client) checkIfUserAssignedMSI(id *aadpodid.AzureIdentity) bool {
	return id.Spec.Type == aadpodid.UserAssignedMSI
}

func (c *Client) getAssignedIDName(podName, podNameSpace, idName string) string {
	return podName + "-" + podNameSpace + "-" + idName
}

func (c *Client) checkIfMSIExistsOnNode(id *aadpodid.AzureIdentity, nodeName string, nodeMSIList map[string][]string) bool {
	if nList, ok := nodeMSIList[nodeName]; ok {
		for _, userAssignedMSI := range nList {
			if userAssignedMSI == id.Spec.ResourceID {
				return true
			}
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

func (c *Client) updateUserMSI(nodeMap map[string]trackUserAssignedMSIIds) []string {
	var nodeWithErrors []string
	for _, n := range nodeMap {
		beginAdding := time.Now()

		// generate unique list so we don't make multiple calls to assign/remove same id
		n.addUserAssignedMSIIDs = c.getUniqueIDs(n.addUserAssignedMSIIDs)
		n.removeUserAssignedMSIIDs = c.getUniqueIDs(n.removeUserAssignedMSIIDs)

		err := c.CloudClient.UpdateUserMSI(n.addUserAssignedMSIIDs, n.removeUserAssignedMSIIDs, n.node)
		if err != nil {
			// check which all identity assignment failed
			// remove those assigned identities
			glog.Errorf("UpdateUserMSI failed for node %s with error %v", n.node.Name, err)
			nodeWithErrors = append(nodeWithErrors, n.node.Name)
		}
		stats.Put(stats.TotalCreateOrUpdate, time.Since(beginAdding))
	}
	return nodeWithErrors
}
