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

type ClientInt interface {
	Start(exit <-chan struct{})

	Sync(exit <-chan struct{})
}

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

		nodeRefs := make(map[string]bool)

		var newAssignedIDs []aadpodid.AzureAssignedIdentity
		beginNewListTime := time.Now()
		//For each pod, check what bindings are matching. For each binding create volatile azure assigned identity.
		//Compare this list with the current list of azure assigned identities.
		//For any new assigned identities found in this volatile list, create assigned identity and assign user assigned msis.
		//For any assigned ids not present the volatile list, proceed with the deletion.
		for _, pod := range listPods {
			//Node is not yet allocated. In that case skip the pod
			if pod.Spec.NodeName == "" {
				glog.V(2).Infof("Pod %s/%s has no assigned node yet. it will be ignored", pod.Name, pod.Namespace)
				continue
			}
			crdPodLabelVal := pod.Labels[aadpodid.CRDLabelKey]
			if crdPodLabelVal == "" {
				//No binding mentioned in the label. Just continue to the next pod
				glog.V(2).Infof("Pod %s/%s has correct %s label but with no value. it will be ignored", pod.Name, pod.Namespace, aadpodid.CRDLabelKey)
				continue
			}
			//glog.Infof("Found label with our CRDKey %s for pod: %s", crdPodLabelVal, pod.Name)
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
		stats.Put(stats.CurrentState, time.Since(beginNewListTime))

		// Extract add list and delete list based on existing assigned ids in the system (currentAssignedIDs).
		// and the ones we have arrived at in the volatile list (newAssignedIDs).
		// TODO: Separate this into two methods.
		addList, deleteList, err := c.splitAzureAssignedIDs(currentAssignedIDs, &newAssignedIDs)
		if err != nil {
			glog.Error(err)
			continue
		}

		glog.V(5).Infof("del: %v, add: %v", deleteList, addList)

		if deleteList != nil && len(*deleteList) > 0 {
			beginDeletion := time.Now()
			workDone = true

			vmssGroups, err := getVMSSGroups(c.NodeClient, nodeRefs)
			if err != nil {
				glog.Error(err)
				continue
			}

			for _, delID := range *deleteList {
				glog.V(5).Infof("Deletion of id: %s", delID.Name)
				inUse, err := c.checkIfInUse(delID, newAssignedIDs, vmssGroups)
				if err != nil {
					glog.Error(err)
					continue
				}
				removedBinding := delID.Spec.AzureBindingRef
				// The inUse here checks if there are pods which are using the MSI in the newAssignedIDs.
				err = c.removeAssignedIDsWithDeps(&delID, inUse)
				if err != nil {
					// Since k8s event has only Info and Warning, using Warning.
					message := fmt.Sprintf("Binding %s removal from node %s for pod %s resulted in error %v", removedBinding.Name, delID.Spec.NodeName, delID.Spec.Pod, err.Error())
					c.EventRecorder.Event(removedBinding, corev1.EventTypeWarning, "binding remove error", message)
					glog.Error(message)
					continue
				}
				eventRecordStart := time.Now()
				glog.V(5).Infof("Binding removed: %+v", removedBinding)
				c.EventRecorder.Event(removedBinding, corev1.EventTypeNormal, "binding removed",
					fmt.Sprintf("Binding %s removed from node %s for pod %s", removedBinding.Name, delID.Spec.NodeName, delID.Spec.Pod))
				stats.Update(stats.EventRecord, time.Since(eventRecordStart))
			}
			stats.Update(stats.TotalIDDel, time.Since(beginDeletion))
		}

		if addList != nil && len(*addList) > 0 {
			beginAdding := time.Now()
			workDone = true
			for _, createID := range *addList {
				id := createID.Spec.AzureIdentityRef
				binding := createID.Spec.AzureBindingRef

				node, err := c.NodeClient.Get(createID.Spec.NodeName)
				if err != nil {
					c.EventRecorder.Event(binding, corev1.EventTypeWarning, "get node error",
						fmt.Sprintf("Lookup of node %s for pod %s resulted in error %v", createID.Spec.NodeName, createID.Name, err))
					continue
				}

				glog.V(5).Infof("Initiating assigned id creation for pod - %s, binding - %s", createID.Spec.Pod, binding.Name)

				err = c.createAssignedIdentityDeps(&createID, id, node)
				if err != nil {
					// Since k8s event has only Info and Warning, using Warning.
					c.EventRecorder.Event(binding, corev1.EventTypeWarning, "binding apply error",
						fmt.Sprintf("Applying binding %s node %s for pod %s resulted in error %v", binding.Name, createID.Spec.NodeName, createID.Name, err))
					glog.Error(err)
					continue
				}
				appliedBinding := createID.Spec.AzureBindingRef
				eventRecordStart := time.Now()
				glog.V(5).Infof("Binding applied: %+v", appliedBinding)
				c.EventRecorder.Event(appliedBinding, corev1.EventTypeNormal, "binding applied",
					fmt.Sprintf("Binding %s applied on node %s for pod %s", appliedBinding.Name, createID.Spec.NodeName, createID.Name))
				stats.Update(stats.EventRecord, time.Since(eventRecordStart))
			}
			stats.Put(stats.TotalIDAdd, time.Since(beginAdding))
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

func (c *Client) splitAzureAssignedIDs(old *[]aadpodid.AzureAssignedIdentity, new *[]aadpodid.AzureAssignedIdentity) (retCreate *[]aadpodid.AzureAssignedIdentity, retDelete *[]aadpodid.AzureAssignedIdentity, err error) {

	if old == nil || len(*old) == 0 {
		return new, nil, nil
	}

	create := make([]aadpodid.AzureAssignedIdentity, 0)
	delete := make([]aadpodid.AzureAssignedIdentity, 0)

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
			//glog.Infof("Match %s %s %v", newAssignedID.Name, oldAssignedID.Name, idMatch)
			if idMatch {
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

	begin = time.Now()
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

	//	glog.Info("Time taken to split create/delete list: %s", time.Since(begin).String())
	return &create, &delete, nil
}

func (c *Client) makeAssignedIDs(azID *aadpodid.AzureIdentity, azBinding *aadpodid.AzureIdentityBinding, podName string, podNameSpace string, nodeName string) (res *aadpodid.AzureAssignedIdentity, err error) {
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

func (c *Client) createAssignedIdentityDeps(assignedID *aadpodid.AzureAssignedIdentity, id *aadpodid.AzureIdentity, node *corev1.Node) error {
	err := c.CRDClient.CreateAssignedIdentity(assignedID)
	if err != nil {
		glog.Error(err)
		return err
	}

	if id.Spec.Type == aadpodid.UserAssignedMSI {

		err = c.CloudClient.AssignUserMSI(id.Spec.ResourceID, node)
		if err != nil {
			glog.Error(err)
			newErr := c.CRDClient.RemoveAssignedIdentity(assignedID)
			if newErr != nil {
				glog.Errorf("Error when removing assigned identity in create error path err: %v", newErr)
			}
			return err
		}
	}
	return nil
}

func (c *Client) removeAssignedIDsWithDeps(assignedID *aadpodid.AzureAssignedIdentity, inUse bool) error {
	err := c.CRDClient.RemoveAssignedIdentity(assignedID)
	if err != nil {
		glog.Error(err)
		return nil
	}
	if !inUse {
		id := assignedID.Spec.AzureIdentityRef
		if id.Spec.Type == aadpodid.UserAssignedMSI {

			node, err := c.NodeClient.Get(assignedID.Spec.NodeName)
			if err != nil {
				return err
			}

			err = c.CloudClient.RemoveUserMSI(id.Spec.ResourceID, node)
			if err != nil {
				glog.Error(err)
				return err
			}
		}
	}
	return nil
}

func (c *Client) getAssignedIDName(podName string, podNameSpace string, idName string) string {
	return podName + "-" + podNameSpace + "-" + idName
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
