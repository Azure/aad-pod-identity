package mic

import (
	"fmt"
	"sync"
	"time"

	"github.com/Azure/aad-pod-identity/pkg/stats"
	"github.com/Azure/aad-pod-identity/version"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"

	"github.com/Azure/aad-pod-identity/pkg/cloudprovider"
	"github.com/Azure/aad-pod-identity/pkg/crd"
	"github.com/Azure/aad-pod-identity/pkg/pod"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
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
}

type ClientInt interface {
	Start(exit <-chan struct{})

	ConvertIDListToMap(arr *[]aadpodid.AzureIdentity) (m map[string]aadpodid.AzureIdentity, err error)
	CheckIfInUse(checkAssignedID aadpodid.AzureAssignedIdentity, arr []aadpodid.AzureAssignedIdentity) bool

	Sync(exit <-chan struct{})
	MatchAssignedID(x *aadpodid.AzureAssignedIdentity, y *aadpodid.AzureAssignedIdentity) (ret bool, err error)
	SplitAzureAssignedIDs(old *[]aadpodid.AzureAssignedIdentity, new *[]aadpodid.AzureAssignedIdentity) (retCreate *[]aadpodid.AzureAssignedIdentity, retDelete *[]aadpodid.AzureAssignedIdentity, err error)
	MakeAssignedIDs(azID *aadpodid.AzureIdentity, azBinding *aadpodid.AzureIdentityBinding, podName string, podNameSpace string, nodeName string) (res *aadpodid.AzureAssignedIdentity, err error)
	CreateAssignedIdentityDeps(b *aadpodid.AzureIdentityBinding, id *aadpodid.AzureIdentity,
		podName string, podNameSpace string, nodeName string) error
	RemoveAssignedIDsWithDeps(assignedID *aadpodid.AzureAssignedIdentity, inUse bool) error
	GetAssignedIDName(podName string, podNameSpace string, idName string) string
}

func NewMICClient(cloudconfig string, config *rest.Config) (*Client, error) {
	glog.Infof("Starting to create the pod identity client. Version: %v. Build date: %v", version.Version, version.BuildDate)

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

func (c *Client) ConvertIDListToMap(arr *[]aadpodid.AzureIdentity) (m map[string]aadpodid.AzureIdentity, err error) {
	m = make(map[string]aadpodid.AzureIdentity)
	for _, element := range *arr {
		m[element.Name] = element
	}
	return m, nil
}

func (c *Client) CheckIfInUse(checkAssignedID aadpodid.AzureAssignedIdentity, arr []aadpodid.AzureAssignedIdentity) bool {
	for _, assignedID := range arr {
		checkID := checkAssignedID.Spec.AzureIdentityRef
		id := assignedID.Spec.AzureIdentityRef
		// If they have the same client id, reside on the same node but the pod name is different, then the
		// assigned id is in use.
		//This is applicable only for user assigned MSI since that is node specific. Ignore other cases.
		if checkID.Spec.Type != aadpodid.UserAssignedMSI {
			continue
		}
		if checkID.Spec.ClientID == id.Spec.ClientID && checkAssignedID.Spec.NodeName == assignedID.Spec.NodeName &&
			checkAssignedID.Spec.Pod != assignedID.Spec.Pod {
			return true
		}
	}
	return false
}

func (c *Client) Sync(exit <-chan struct{}) {
	glog.Info("Sync thread started\n")
	for event := range c.EventChannel {
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
		idMap, err := c.ConvertIDListToMap(listIDs)
		if err != nil {
			glog.Error(err)
			continue
		}

		currentAssignedIDs, err := c.CRDClient.ListAssignedIDs()
		if err != nil {
			continue
		}
		stats.Put(stats.System, time.Since(systemTime))

		var newAssignedIDs []aadpodid.AzureAssignedIdentity
		beginNewListTime := time.Now()
		//For each pod, check what bindings are matching. For each binding create volatile azure assigned identity.
		//Compare this list with the current list of azure assigned identities.
		//For any new assigned identities found in this volatile list, create assigned identity and assign user assigned msis.
		//For any assigned ids not present the volatile list, proceed with the deletion.
		for _, pod := range listPods {
			//Node is not yet allocated. In that case skip the pod
			if pod.Spec.NodeName == "" {
				continue
			}
			crdPodLabelVal := pod.Labels[aadpodid.CRDLabelKey]
			if crdPodLabelVal == "" {
				//No binding mentioned in the label. Just continue to the next pod
				continue
			}
			//glog.Infof("Found label with our CRDKey %s for pod: %s", crdPodLabelVal, pod.Name)
			var matchedBindings []aadpodid.AzureIdentityBinding
			for _, allBinding := range *listBindings {
				if allBinding.Spec.Selector == crdPodLabelVal {
					glog.V(5).Infof("Found binding match for pod %s with binding %s", pod.Name, allBinding.Name)
					matchedBindings = append(matchedBindings, allBinding)
				}
			}

			for _, binding := range matchedBindings {
				glog.V(5).Infof("Looking up id map: %v", binding.Spec.AzureIdentity)
				if azureID, idPresent := idMap[binding.Spec.AzureIdentity]; idPresent {
					glog.V(5).Infof("Id %s got for assigning", azureID.Name)
					assignedID, err := c.MakeAssignedIDs(&azureID, &binding, pod.Name, pod.Namespace, pod.Spec.NodeName)

					if err != nil {
						glog.Error(err)
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
		addList, deleteList, err := c.SplitAzureAssignedIDs(currentAssignedIDs, &newAssignedIDs)
		if err != nil {
			glog.Error(err)
			continue
		}

		glog.V(5).Infof("del: %v, add: %v", deleteList, addList)

		if deleteList != nil && len(*deleteList) > 0 {
			beginDeletion := time.Now()
			workDone = true
			for _, delID := range *deleteList {
				glog.V(5).Infof("Deletion of id: %s", delID.Name)
				inUse := c.CheckIfInUse(delID, newAssignedIDs)
				removedBinding := delID.Spec.AzureBindingRef
				// The inUse here checks if there are pods which are using the MSI in the newAssignedIDs.
				err = c.RemoveAssignedIDsWithDeps(&delID, inUse)
				if err != nil {
					// Since k8s event has only Info and Warning, using Warning.
					c.EventRecorder.Event(removedBinding, corev1.EventTypeWarning, "binding remove error",
						fmt.Sprintf("Binding %s removal from node %s for pod %s resulted in error %v", removedBinding.Name, delID.Spec.NodeName, delID.Spec.Pod, err))
					glog.Error(err)
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

				err = c.CreateAssignedIdentityDeps(binding, id, createID.Spec.Pod, createID.Spec.PodNamespace, node)
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

func (c *Client) MatchAssignedID(x *aadpodid.AzureAssignedIdentity, y *aadpodid.AzureAssignedIdentity) (ret bool, err error) {
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

func (c *Client) SplitAzureAssignedIDs(old *[]aadpodid.AzureAssignedIdentity, new *[]aadpodid.AzureAssignedIdentity) (retCreate *[]aadpodid.AzureAssignedIdentity, retDelete *[]aadpodid.AzureAssignedIdentity, err error) {

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
			idMatch, err = c.MatchAssignedID(&newAssignedID, &oldAssignedID)
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
			idMatch, err = c.MatchAssignedID(&newAssignedID, &oldAssignedID)
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
			// We are done checking that this new id is not present in the old
			// list. So we will add it to the create list.
			delete = append(create, oldAssignedID)
		}
	}
	stats.Put(stats.FindAssignedIDDel, time.Since(begin))

	//	glog.Info("Time taken to split create/delete list: %s", time.Since(begin).String())
	return &create, &delete, nil
}

func (c *Client) MakeAssignedIDs(azID *aadpodid.AzureIdentity, azBinding *aadpodid.AzureIdentityBinding, podName string, podNameSpace string, nodeName string) (res *aadpodid.AzureAssignedIdentity, err error) {
	assignedID := &aadpodid.AzureAssignedIdentity{
		ObjectMeta: v1.ObjectMeta{
			Name: c.GetAssignedIDName(podName, podNameSpace, azID.Name),
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
	glog.V(5).Infof("Making assigned ID: %v", assignedID)
	return assignedID, nil
}

func (c *Client) CreateAssignedIdentityDeps(b *aadpodid.AzureIdentityBinding, id *aadpodid.AzureIdentity,
	podName string, podNameSpace string, node *corev1.Node) error {
	name := c.GetAssignedIDName(podName, podNameSpace, id.Name)
	err := c.CRDClient.CreateAssignedIdentity(name, b, id, podName, podNameSpace, node.Name)
	if err != nil {
		glog.Error(err)
		return err
	}

	if id.Spec.Type == aadpodid.UserAssignedMSI {
		err = c.CloudClient.AssignUserMSI(id.Spec.ResourceID, node)
		if err != nil {
			glog.Error(err)
			newErr := c.CRDClient.RemoveAssignedIdentity(name)
			if newErr != nil {
				glog.Errorf("Error when removing assigned identity in create error path err: %v", newErr)
			}
			return err
		}
	}
	return nil
}

func (c *Client) RemoveAssignedIDsWithDeps(assignedID *aadpodid.AzureAssignedIdentity, inUse bool) error {
	err := c.CRDClient.RemoveAssignedIdentity(assignedID.Name)
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

func (c *Client) GetAssignedIDName(podName string, podNameSpace string, idName string) string {
	return podName + "-" + podNameSpace + "-" + idName
}
