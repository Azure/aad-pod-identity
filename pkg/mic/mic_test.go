package mic

import (
	"errors"
	"reflect"
	"testing"
	"time"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	"github.com/Azure/aad-pod-identity/pkg/config"

	"github.com/golang/glog"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"

	cp "github.com/Azure/aad-pod-identity/pkg/cloudprovider"
	api "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
)

/****************** CLOUD PROVIDER MOCK ****************************/
type TestCloudClient struct {
	*cp.Client
	// testVMClient is test validation purpose.
	testVMClient   *TestVMClient
	testVMSSClient *TestVMSSClient
}

type TestVMClient struct {
	*cp.VMClient
	nodeMap map[string]*compute.VirtualMachine
	err     *error
}

func (c *TestVMClient) SetError(err error) {
	c.err = &err
}

func (c *TestVMClient) UnSetError() {
	c.err = nil
}

func (c *TestVMClient) Get(rgName string, nodeName string) (ret compute.VirtualMachine, err error) {
	stored := c.nodeMap[nodeName]
	if stored == nil {
		vm := new(compute.VirtualMachine)
		c.nodeMap[nodeName] = vm
		return *vm, nil
	}
	return *stored, nil
}

func (c *TestVMClient) CreateOrUpdate(rg string, nodeName string, vm compute.VirtualMachine) error {
	if c.err != nil {
		return *c.err
	}
	c.nodeMap[nodeName] = &vm
	return nil
}

func (c *TestVMClient) ListMSI() (ret map[string]*[]string) {
	ret = make(map[string]*[]string)

	for key, val := range c.nodeMap {
		ret[key] = val.Identity.IdentityIds
	}
	return ret
}

func (c *TestVMClient) CompareMSI(nodeName string, userIDs []string) bool {
	stored := c.nodeMap[nodeName]
	if stored == nil || stored.Identity == nil {
		return false
	}

	ids := stored.Identity.IdentityIds
	if ids == nil {
		if len(userIDs) == 0 && stored.Identity.Type == compute.ResourceIdentityTypeNone { // Validate that we have reset the resource type as none.
			return true
		}
		return false
	}
	return reflect.DeepEqual(*ids, userIDs)
}

type TestVMSSClient struct {
	*cp.VMSSClient
	nodeMap map[string]*compute.VirtualMachineScaleSet
	err     *error
}

func (c *TestVMSSClient) SetError(err error) {
	c.err = &err
}

func (c *TestVMSSClient) UnSetError() {
	c.err = nil
}

func (c *TestVMSSClient) Get(rgName string, nodeName string) (ret compute.VirtualMachineScaleSet, err error) {
	stored := c.nodeMap[nodeName]
	if stored == nil {
		vm := new(compute.VirtualMachineScaleSet)
		c.nodeMap[nodeName] = vm
		return *vm, nil
	}
	return *stored, nil
}

func (c *TestVMSSClient) CreateOrUpdate(rg string, nodeName string, vm compute.VirtualMachineScaleSet) error {
	if c.err != nil {
		return *c.err
	}
	c.nodeMap[nodeName] = &vm
	return nil
}

func (c *TestVMSSClient) ListMSI() (ret map[string]*[]string) {
	ret = make(map[string]*[]string)

	for key, val := range c.nodeMap {
		ret[key] = val.Identity.IdentityIds
	}
	return ret
}

func (c *TestVMSSClient) CompareMSI(nodeName string, userIDs []string) bool {
	stored := c.nodeMap[nodeName]
	if stored == nil || stored.Identity == nil {
		return false
	}

	ids := stored.Identity.IdentityIds
	if ids == nil {
		if len(userIDs) == 0 && stored.Identity.Type == compute.ResourceIdentityTypeNone { // Validate that we have reset the resource type as none.
			return true
		}
		return false
	}
	return reflect.DeepEqual(*ids, userIDs)
}

func (c *TestCloudClient) ListMSI() (ret map[string]*[]string) {
	if c.Client.Config.VMType == "vmss" {
		return c.testVMSSClient.ListMSI()
	}
	return c.testVMClient.ListMSI()
}

func (c *TestCloudClient) CompareMSI(nodeName string, userIDs []string) bool {
	if c.Client.Config.VMType == "vmss" {
		return c.testVMSSClient.CompareMSI(nodeName, userIDs)
	}
	return c.testVMClient.CompareMSI(nodeName, userIDs)
}

func (c *TestCloudClient) PrintMSI() {
	for key, val := range c.ListMSI() {
		glog.Infof("\nNode name: %s", key)
		if val != nil {
			for i, id := range *val {
				glog.Infof("%d) %s", i, id)
			}
		}
	}
}

func (c *TestCloudClient) SetError(err error) {
	c.testVMClient.SetError(err)
}

func (c *TestCloudClient) UnSetError() {
	c.testVMClient.UnSetError()
}

func NewTestVMClient() *TestVMClient {
	nodeMap := make(map[string]*compute.VirtualMachine, 0)
	vmClient := &cp.VMClient{}

	return &TestVMClient{
		vmClient,
		nodeMap,
		nil,
	}
}

func NewTestVMSSClient() *TestVMSSClient {
	nodeMap := make(map[string]*compute.VirtualMachineScaleSet, 0)
	vmssClient := &cp.VMSSClient{}

	return &TestVMSSClient{
		vmssClient,
		nodeMap,
		nil,
	}
}

func NewTestCloudClient(cfg config.AzureConfig) *TestCloudClient {
	vmClient := NewTestVMClient()
	vmssClient := NewTestVMSSClient()
	cloudClient := &cp.Client{
		Config:     cfg,
		VMClient:   vmClient,
		VMSSClient: vmssClient,
	}

	return &TestCloudClient{
		cloudClient,
		vmClient,
		vmssClient,
	}
}

/****************** POD MOCK ****************************/
type TestPodClient struct {
	pods []*corev1.Pod
}

func NewTestPodClient() *TestPodClient {
	var pods []*corev1.Pod
	return &TestPodClient{
		pods: pods,
	}
}

func (c TestPodClient) Start(exit <-chan struct{}) {
	glog.Info("Start called from the test interface")
}

func (c TestPodClient) GetPods() (pods []*corev1.Pod, err error) {
	//TODO: Add label matching. For now we add only pods which we want to add.
	return c.pods, nil
}

func (c *TestPodClient) AddPod(podName string, podNs string, nodeName string, binding string) {
	labels := make(map[string]string, 0)
	labels[aadpodid.CRDLabelKey] = binding
	pod := &corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:      podName,
			Namespace: podNs,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			NodeName: nodeName,
		},
	}
	c.pods = append(c.pods, pod)
}

func (c *TestPodClient) DeletePod(podName string, podNs string) {
	var newPods []*corev1.Pod
	changed := false
	for _, pod := range c.pods {
		if pod.Name == podName && pod.Namespace == podNs {
			changed = true
			continue
		} else {
			newPods = append(newPods, pod)
		}
	}
	if changed {
		c.pods = newPods
	}
}

/****************** CRD MOCK ****************************/

type TestCrdClient struct {
	*Client
	assignedIDMap map[string]*aadpodid.AzureAssignedIdentity
	bindingMap    map[string]*aadpodid.AzureIdentityBinding
	idMap         map[string]*aadpodid.AzureIdentity
}

func NewTestCrdClient(config *rest.Config) *TestCrdClient {
	return &TestCrdClient{
		assignedIDMap: make(map[string]*aadpodid.AzureAssignedIdentity, 0),
		bindingMap:    make(map[string]*aadpodid.AzureIdentityBinding, 0),
		idMap:         make(map[string]*aadpodid.AzureIdentity, 0),
	}
}

func (c *TestCrdClient) Start(exit <-chan struct{}) {
}

func (c *TestCrdClient) SyncCache(exit <-chan struct{}) {

}

func (c *TestCrdClient) CreateCrdWatchers(eventCh chan aadpodid.EventType) (err error) {
	return nil
}

func (c *TestCrdClient) RemoveAssignedIdentity(assignedIdentity *aadpodid.AzureAssignedIdentity) error {
	delete(c.assignedIDMap, assignedIdentity.Name)
	return nil
}

// This function is not used currently
// TODO: consider remove
func (c *TestCrdClient) CreateAssignedIdentity(assignedIdentity *aadpodid.AzureAssignedIdentity) error {
	assignedIdentityToStore := *assignedIdentity //Make a copy to store in the map.
	c.assignedIDMap[assignedIdentity.Name] = &assignedIdentityToStore
	return nil
}

func (c *TestCrdClient) CreateBinding(bindingName string, idName string, selector string) {
	binding := &aadpodid.AzureIdentityBinding{
		ObjectMeta: v1.ObjectMeta{
			Name: bindingName,
		},
		Spec: aadpodid.AzureIdentityBindingSpec{
			AzureIdentity: idName,
			Selector:      selector,
		},
	}
	c.bindingMap[bindingName] = binding
}

func (c *TestCrdClient) CreateId(idName string, t aadpodid.IdentityType, rId string, cId string, cp *api.SecretReference, tId string, adRId string, adEpt string) {
	id := &aadpodid.AzureIdentity{
		ObjectMeta: v1.ObjectMeta{
			Name: idName,
		},
		Spec: aadpodid.AzureIdentitySpec{
			Type:       t,
			ResourceID: rId,
			ClientID:   cId,
			//ClientPassword: *cp,
			TenantID:     tId,
			ADResourceID: adRId,
			ADEndpoint:   adEpt,
		},
	}
	c.idMap[idName] = id
}

func (c *TestCrdClient) ListIds() (res *[]aadpodid.AzureIdentity, err error) {
	idList := make([]aadpodid.AzureIdentity, 0)
	for _, v := range c.idMap {
		idList = append(idList, *v)
	}
	return &idList, nil
}

func (c *TestCrdClient) ListBindings() (res *[]aadpodid.AzureIdentityBinding, err error) {
	bindingList := make([]aadpodid.AzureIdentityBinding, 0)
	for _, v := range c.bindingMap {
		bindingList = append(bindingList, *v)
	}
	return &bindingList, nil
}

func (c *TestCrdClient) ListAssignedIDs() (res *[]aadpodid.AzureAssignedIdentity, err error) {
	assignedIdList := make([]aadpodid.AzureAssignedIdentity, 0)
	for _, v := range c.assignedIDMap {
		assignedIdList = append(assignedIdList, *v)
	}
	return &assignedIdList, nil
}
func (c *Client) ListPodIds(podns, podname string) (*[]aadpodid.AzureIdentity, error) {
	return &[]aadpodid.AzureIdentity{}, nil
}

/************************ NODE MOCK *************************************/

type TestNodeClient struct {
	nodes map[string]*corev1.Node
}

func NewTestNodeClient() *TestNodeClient {
	return &TestNodeClient{nodes: make(map[string]*corev1.Node)}
}

func (c *TestNodeClient) Get(name string) (*corev1.Node, error) {
	node, exists := c.nodes[name]
	if !exists {
		return nil, errors.New("node not found")
	}
	return node, nil
}

func (c *TestNodeClient) Start(<-chan struct{}) {}

func (c *TestNodeClient) AddNode(name string) {
	c.nodes[name] = &corev1.Node{ObjectMeta: v1.ObjectMeta{Name: name}}
}

/************************ EVENT RECORDER MOCK *************************************/
type LastEvent struct {
	Type    string
	Reason  string
	Message string
}

type TestEventRecorder struct {
	lastEvent    *LastEvent
	eventChannel chan bool
}

func (c TestEventRecorder) WaitForEvents(expectedCount int) bool {
	count := 0
	for {
		select {
		case <-c.eventChannel:
			count++
			if expectedCount == count {
				return true
			}
		case <-time.After(2 * time.Minute):
			return false
		}
	}
}

func (c TestEventRecorder) Event(object runtime.Object, t string, r string, message string) {
	c.lastEvent.Type = t
	c.lastEvent.Reason = r
	c.lastEvent.Message = message
	c.eventChannel <- true
}

func (c TestEventRecorder) Validate(e *LastEvent) bool {
	t := c.lastEvent.Type
	r := c.lastEvent.Reason
	m := c.lastEvent.Message

	if t != e.Type || r != e.Reason || m != e.Message {
		glog.Errorf("event mismatch. expected - (t:%s, r:%s, m:%s). got - (t:%s, r:%s, m:%s)", e.Type, e.Reason, e.Message, t, r, m)
		return false
	}
	return true
}

func (c TestEventRecorder) Eventf(object runtime.Object, t string, r string, messageFmt string, args ...interface{}) {

}

func (c TestEventRecorder) PastEventf(object runtime.Object, timestamp v1.Time, t string, m1 string, messageFmt string, args ...interface{}) {

}

func (c TestEventRecorder) AnnotatedEventf(object runtime.Object, annotations map[string]string, eventtype, reason, messageFmt string, args ...interface{}) {

}

/************************ MIC MOC *************************************/
func NewMICTestClient(eventCh chan aadpodid.EventType, cpClient *TestCloudClient, crdClient *TestCrdClient, podClient *TestPodClient, nodeClient *TestNodeClient, eventRecorder *TestEventRecorder) *TestMICClient {

	realMICClient := &Client{
		CloudClient:   cpClient,
		CRDClient:     crdClient,
		EventRecorder: eventRecorder,
		PodClient:     podClient,
		EventChannel:  eventCh,
		NodeClient:    nodeClient,
	}

	return &TestMICClient{
		realMICClient,
	}
}

type TestMICClient struct {
	*Client
}

/************************ UNIT TEST *************************************/

func TestMapMICClient(t *testing.T) {
	micClient := &TestMICClient{}

	idList := make([]aadpodid.AzureIdentity, 0)

	id := new(aadpodid.AzureIdentity)
	id.Name = "test-azure-identity"

	idList = append(idList, *id)

	id.Name = "test-akssvcrg-id"
	idList = append(idList, *id)

	idMap, _ := micClient.convertIDListToMap(idList)

	name := "test-azure-identity"
	count := 3
	if azureID, idPresent := idMap[name]; idPresent {
		if azureID.Name != name {
			t.Errorf("id map id value mismatch")
		}
		count = count - 1
	}

	name = "test-akssvcrg-id"
	if azureID, idPresent := idMap[name]; idPresent {
		if azureID.Name != name {
			t.Errorf("id map id value mismatch")
		}
		count = count - 1
	}

	name = "test not there"
	if _, idPresent := idMap[name]; idPresent {
		t.Errorf("not present found")
	} else {
		count = count - 1
	}
	if count != 0 {
		t.Errorf("Test count mismatch")
	}

}

func TestSimpleMICClient(t *testing.T) {

	exit := make(<-chan struct{}, 0)
	eventCh := make(chan aadpodid.EventType, 100)
	cloudClient := NewTestCloudClient(config.AzureConfig{})
	crdClient := NewTestCrdClient(nil)
	podClient := NewTestPodClient()
	nodeClient := NewTestNodeClient()
	var evtRecorder TestEventRecorder
	evtRecorder.lastEvent = new(LastEvent)
	evtRecorder.eventChannel = make(chan bool, 1)

	micClient := NewMICTestClient(eventCh, cloudClient, crdClient, podClient, nodeClient, &evtRecorder)

	crdClient.CreateId("test-id", aadpodid.UserAssignedMSI, "test-user-msi-resourceid", "test-user-msi-clientid", nil, "", "", "")
	crdClient.CreateBinding("testbinding", "test-id", "test-select")

	nodeClient.AddNode("test-node")
	podClient.AddPod("test-pod", "default", "test-node", "test-select")

	eventCh <- aadpodid.PodCreated
	go micClient.Sync(exit)
	evtRecorder.WaitForEvents(1)

	testPass := false
	listAssignedIDs, err := crdClient.ListAssignedIDs()
	if err != nil {
		glog.Error(err)
		t.Errorf("list assigned failed")
	}

	if listAssignedIDs != nil {
		for _, assignedID := range *listAssignedIDs {
			if assignedID.Spec.Pod == "test-pod" && assignedID.Spec.PodNamespace == "default" && assignedID.Spec.NodeName == "test-node" &&
				assignedID.Spec.AzureBindingRef.Name == "testbinding" && assignedID.Spec.AzureIdentityRef.Name == "test-id" {
				testPass = true
				/*
					testPass = evtRecorder.Validate(&LastEvent{Type: "Normal", Reason: "binding applied",
						Message: "Binding testbinding applied on node test-node for pod test-pod-default-test-id"})
					if !testPass {
						t.Errorf("event mismatch")
					}
				*/
				break
			}
		}
	}

	if !testPass {
		t.Fatalf("assigned id mismatch")
	}

	//Test2: Remove assigned id event test
	podClient.DeletePod("test-pod", "default")

	eventCh <- aadpodid.PodDeleted
	time.Sleep(5 * time.Second)

	listAssignedIDs, err = crdClient.ListAssignedIDs()
	if err != nil {
		glog.Error(err)
		t.Fatalf("list assigned failed")
	}

	if len(*listAssignedIDs) != 0 {
		t.Fatalf("Assigned id not deleted")
	}

	/*
		testPass = evtRecorder.Validate(&LastEvent{Type: "Normal", Reason: "binding removed",
			Message: "Binding testbinding removed from node test-node for pod test-pod"})

		if !testPass {
			t.Errorf("event mismatch")
		}
	*/

	// Test3: Error from cloud provider event test
	err = errors.New("error returned from cloud provider")
	cloudClient.SetError(err)

	podClient.AddPod("test-pod", "default", "test-node", "test-select")
	eventCh <- aadpodid.PodCreated
	evtRecorder.WaitForEvents(1)

	listAssignedIDs, err = crdClient.ListAssignedIDs()
	if err != nil {
		glog.Error(err)
		t.Fatalf("list assigned failed")
	}

	if len(*listAssignedIDs) != 0 {
		t.Fatalf("ID assigned")
	}

	/*
		testPass = evtRecorder.Validate(&LastEvent{Type: "Warning", Reason: "binding apply error",
			Message: "Applying binding testbinding node test-node for pod test-pod-default-test-id resulted in error error returned from cloud provider"})

		if !testPass {
			t.Errorf("event mismatch")
		} */

	// Test4: Removal error event test
	//Reset the state to add the id.
	cloudClient.UnSetError()

	//podClient.AddPod("test-pod", "default", "test-node", "test-select")
	eventCh <- aadpodid.PodCreated

	err = errors.New("remove error returned from cloud provider")
	cloudClient.SetError(err)

	podClient.DeletePod("test-pod", "default")
	eventCh <- aadpodid.PodDeleted
	time.Sleep(5 * time.Second)
	/*
		testPass = evtRecorder.Validate(&LastEvent{Type: "Warning", Reason: "binding remove error",
			Message: "Binding testbinding removal from node test-node for pod test-pod resulted in error remove error returned from cloud provider"})

		if !testPass {
			t.Errorf("event mismatch")
		}
	*/
}

func TestAddDelMICClient(t *testing.T) {
	exit := make(<-chan struct{}, 0)
	eventCh := make(chan aadpodid.EventType, 100)
	cloudClient := NewTestCloudClient(config.AzureConfig{})
	crdClient := NewTestCrdClient(nil)
	podClient := NewTestPodClient()
	nodeClient := NewTestNodeClient()
	var evtRecorder TestEventRecorder
	evtRecorder.lastEvent = new(LastEvent)
	evtRecorder.eventChannel = make(chan bool, 1)

	micClient := NewMICTestClient(eventCh, cloudClient, crdClient, podClient, nodeClient, &evtRecorder)

	// Test to add and delete at the same time.
	// Add a pod, identity and binding.
	crdClient.CreateId("test-id2", aadpodid.UserAssignedMSI, "test-user-msi-resourceid", "test-user-msi-clientid", nil, "", "", "")
	crdClient.CreateBinding("testbinding2", "test-id2", "test-select2")

	nodeClient.AddNode("test-node2")
	podClient.AddPod("test-pod2", "default", "test-node2", "test-select2")
	podClient.GetPods()

	crdClient.CreateId("test-id4", aadpodid.UserAssignedMSI, "test-user-msi-resourceid", "test-user-msi-clientid", nil, "", "", "")
	crdClient.CreateBinding("testbinding4", "test-id4", "test-select4")
	podClient.AddPod("test-pod4", "default", "test-node2", "test-select4")
	podClient.GetPods()

	eventCh <- aadpodid.PodCreated
	go micClient.Sync(exit)

	if !evtRecorder.WaitForEvents(2) {
		t.Fatalf("Timeout waiting for mic sync cycles")
	}

	listAssignedIDs, err := crdClient.ListAssignedIDs()
	if err != nil {
		t.Fatalf("error from list assigned ids")
	}
	expectedLen := 2
	gotLen := len(*listAssignedIDs)

	//One id should be left around. Rest should be removed
	if gotLen != expectedLen {
		glog.Errorf("Expected len: %d. Got: %d", expectedLen, gotLen)
		t.Fatalf("Add and delete id at same time mismatch")
	}

	//Delete the pod
	podClient.DeletePod("test-pod2", "default")
	podClient.DeletePod("test-pod4", "default")

	//Add a new pod, with different id and binding on the same node.
	crdClient.CreateId("test-id3", aadpodid.UserAssignedMSI, "test-user-msi-resourceid", "test-user-msi-clientid", nil, "", "", "")
	crdClient.CreateBinding("testbinding3", "test-id3", "test-select3")
	podClient.AddPod("test-pod3", "default", "test-node2", "test-select3")
	podClient.GetPods()

	eventCh <- aadpodid.PodCreated
	go micClient.Sync(exit)

	if !evtRecorder.WaitForEvents(3) {
		t.Fatalf("Timeout waiting for mic sync cycles")
	}

	listAssignedIDs, err = crdClient.ListAssignedIDs()
	if err != nil {
		glog.Error(err)
		t.Fatalf("list assigned failed")
	}

	expectedLen = 1
	gotLen = len(*listAssignedIDs)
	//One id should be left around. Rest should be removed
	if gotLen != expectedLen {
		glog.Errorf("Expected len: %d. Got: %d", expectedLen, gotLen)
		t.Fatalf("Add and delete id at same time mismatch")
	} else {
		gotID := (*listAssignedIDs)[0].Name
		expectedID := "test-pod3-default-test-id3"
		if gotID != expectedID {
			glog.Errorf("Expected %s. Got: %s", expectedID, gotID)
			t.Fatalf("Add and delete id at same time. Found wrong id")
		}
	}
}
