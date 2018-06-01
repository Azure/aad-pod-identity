package mic

import (
	"fmt"
	"time"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	cloudprovider "github.com/Azure/aad-pod-identity/pkg/cloudprovider"
	crd "github.com/Azure/aad-pod-identity/pkg/crd"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

// Client has the required pointers to talk to the api server
// and interact with the CRD related datastructure.
type Client struct {
	CRDClient     *crd.Client
	CloudClient   *cloudprovider.Client
	ClientSet     *kubernetes.Clientset
	EventRecorder record.EventRecorder
	PodWatcher    informers.SharedInformerFactory

	EventChannel chan aadpodid.EventType
}

func NewMICClient(cloudconfig string, config *rest.Config) (*Client, error) {
	glog.Infof("Starting to create the pod identity client")
	clientSet := kubernetes.NewForConfigOrDie(config)

	cloudClient, err := cloudprovider.NewCloudProvider(cloudconfig)
	if err != nil {
		return nil, err
	}

	crdClient, err := crd.NewCRDClient(config)
	if err != nil {
		return nil, err
	}

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: clientSet.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: aadpodid.CRDGroup})

	eventCh := make(chan aadpodid.EventType, 100)

	micClient := &Client{
		CRDClient:     crdClient,
		CloudClient:   cloudClient,
		ClientSet:     clientSet,
		EventRecorder: recorder,
		EventChannel:  eventCh,
	}

	err = micClient.CreatePodWatcher(clientSet)
	if err != nil {
		glog.Error(err)
		return nil, err
	}

	err = micClient.CRDClient.CreateCRDWatcher(eventCh)
	if err != nil {
		glog.Error(err)
		return nil, err
	}

	return micClient, nil

}

func (c *Client) Start(exit <-chan struct{}) {
	go c.PodWatcher.Start(exit)
	glog.Info("Pod watcher started !!")
	go c.CRDClient.Start(exit)
	glog.Info("CRD watcher started")
	go c.Sync(exit)
}

func (c *Client) CreatePodWatcher(k8sClient *kubernetes.Clientset) (err error) {
	k8sInformers := informers.NewSharedInformerFactory(k8sClient, time.Second*30)
	if k8sInformers == nil {
		return fmt.Errorf("k8s informers could not be created")
	}
	k8sInformers.Core().V1().Pods().Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				glog.V(6).Infof("Pod Created")
				c.EventChannel <- aadpodid.PodCreated

			},
			DeleteFunc: func(obj interface{}) {
				glog.V(6).Infof("Pod Deleted")
				c.EventChannel <- aadpodid.PodDeleted

			},
			UpdateFunc: func(OldObj, newObj interface{}) {
				glog.V(6).Infof("Pod Updated")
				c.EventChannel <- aadpodid.PodUpdated

			},
		},
	)

	c.PodWatcher = k8sInformers
	return nil
}

func (c *Client) ConvertIDListToMap(arr *[]aadpodid.AzureIdentity) (m map[string]*aadpodid.AzureIdentity, err error) {
	m = make(map[string]*aadpodid.AzureIdentity)
	for _, element := range *arr {
		m[element.Name] = &element
	}
	return m, nil
}

func (c *Client) CheckIfInUse(checkAssignedID aadpodid.AzureAssignedIdentity, arr []aadpodid.AzureAssignedIdentity) bool {
	for _, assignedID := range arr {
		checkID := checkAssignedID.Spec.AzureIdentityRef
		id := assignedID.Spec.AzureIdentityRef
		// If they have the same client id, reside on the same node but the pod name is different, then the
		// assigned id is in use.
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
		// This is the only place where the AzureAssignedIdentity creation is initiated.
		begin := time.Now()
		workDone := false
		glog.V(6).Infof("Received event: %v", event)
		// List all pods in all namespaces
		listPods, err := c.ClientSet.CoreV1().Pods("").List(v1.ListOptions{})
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
		var newAssignedIDs []aadpodid.AzureAssignedIdentity

		//For each pod, check what bindings are matching. For each binding create volatile azure assigned identity.
		//Compare this list with the current list of azure assigned identities.
		//For any new assigned identities found in this volatile list, create assigned identity and assign user assigned msis.
		//For any assigned ids not present the volatile list, proceed with the deletion.
		for _, pod := range listPods.Items {
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
				azureID := idMap[binding.Spec.AzureIdentity]
				if azureID != nil {
					assignedID, err := c.MakeAssignedIDs(azureID, &binding, pod.Name, pod.Namespace, pod.Spec.NodeName)

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
					glog.V(5).Info("%s identity not found when using %s binding", binding.Spec.AzureIdentity, binding.Name)
				}
			}
		}
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
			workDone = true
			for _, delID := range *deleteList {
				glog.V(5).Infof("Deletion of id: ", delID.Name)
				inUse := c.CheckIfInUse(delID, newAssignedIDs)
				// The inUse here checks if there are pods which are using the MSI in the newAssignedIDs.
				err = c.RemoveAssignedIDsWithDeps(&delID, inUse)
				if err != nil {
					glog.Error(err)
					continue
				}
				removedBinding := delID.Spec.AzureBindingRef
				c.EventRecorder.Event(removedBinding, corev1.EventTypeNormal, "binding removed",
					fmt.Sprintf("Binding %s removed from node %s for pod %s", removedBinding.Name, delID.Spec.NodeName, delID.Spec.Pod))
			}
		}
		if addList != nil && len(*addList) > 0 {
			workDone = true
			for _, createID := range *addList {
				id := createID.Spec.AzureIdentityRef
				binding := createID.Spec.AzureBindingRef

				glog.V(5).Infof("Initiating assigned id creation for pod - %s, binding - %s", createID.Spec.Pod, binding.Name)
				err = c.CreateAssignedIdentityDeps(binding, id, createID.Spec.Pod, createID.Spec.PodNamespace, createID.Spec.NodeName)
				if err != nil {
					continue
				}
				appliedBinding := createID.Spec.AzureBindingRef
				c.EventRecorder.Event(appliedBinding, corev1.EventTypeNormal, "binding applied",
					fmt.Sprintf("Binding %s applied on node %s for pod %s", appliedBinding.Name, createID.Spec.NodeName, createID.Name))
			}
		}
		if workDone {
			glog.Infof("Sync took: %s", time.Since(begin).String())
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
			glog.Infof("ok: %v, Delete added: %s", idMatch, oldAssignedID.Name)
			// We are done checking that this new id is not present in the old
			// list. So we will add it to the create list.
			delete = append(create, oldAssignedID)
		}
	}
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
	podName string, podNameSpace string, nodeName string) error {
	name := c.GetAssignedIDName(podName, podNameSpace, id.Name)
	err := c.CRDClient.CreateAssignIdentity(name, b, id, podName, podNameSpace, nodeName)
	if err != nil {
		glog.Error(err)
		return err
	}
	err = c.CloudClient.AssignUserMSI(id.Spec.ResourceID, nodeName)
	if err != nil {
		//TODO: If we have not applied the user id, but created the assigned identity, we need to either
		// go back and remove it or have state in the assigned id which we need to retry.
		glog.Error(err)
		return err
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
		err = c.CloudClient.RemoveUserMSI(id.Spec.ResourceID, assignedID.Spec.NodeName)
		if err != nil {
			glog.Error(err)
			return err
		}
	}
	return nil
}

func (c *Client) GetAssignedIDName(podName string, podNameSpace string, idName string) string {
	return podName + "-" + podNameSpace + "-" + idName
}
