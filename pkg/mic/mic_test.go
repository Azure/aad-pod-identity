package mic

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	internalaadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity"
	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	"github.com/Azure/aad-pod-identity/pkg/config"
	"github.com/Azure/aad-pod-identity/pkg/crd"
	"github.com/Azure/aad-pod-identity/pkg/metrics"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"

	cp "github.com/Azure/aad-pod-identity/pkg/cloudprovider"
	api "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
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

	mu       sync.Mutex
	nodeMap  map[string]*compute.VirtualMachine
	err      *error
	identity *compute.VirtualMachineIdentity
}

func (c *TestVMClient) SetError(err error) {
	c.mu.Lock()
	c.err = &err
	c.mu.Unlock()
}

func (c *TestVMClient) UnSetError() {
	c.mu.Lock()
	c.err = nil
	c.mu.Unlock()
}

func (c *TestVMClient) Get(rgName string, nodeName string) (ret compute.VirtualMachine, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	stored := c.nodeMap[nodeName]
	if stored == nil {
		vm := new(compute.VirtualMachine)
		c.nodeMap[nodeName] = vm
		return *vm, nil
	}
	return *stored, nil
}

func (c *TestVMClient) CreateOrUpdate(rg string, nodeName string, vm compute.VirtualMachine) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.err != nil {
		c.nodeMap[nodeName].Identity = c.identity
		return *c.err
	}
	c.nodeMap[nodeName] = &vm
	return nil
}

func (c *TestVMClient) ListMSI() (ret map[string]*[]string) {
	ret = make(map[string]*[]string)

	c.mu.Lock()
	defer c.mu.Unlock()

	for key, val := range c.nodeMap {
		if val.Identity != nil {
			ret[key] = val.Identity.IdentityIds
		}
	}
	return ret
}

func (c *TestVMClient) CompareMSI(nodeName string, userIDs []string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

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

	mu       sync.Mutex
	nodeMap  map[string]*compute.VirtualMachineScaleSet
	err      *error
	identity *compute.VirtualMachineScaleSetIdentity
}

func (c *TestVMSSClient) SetError(err error) {
	c.err = &err
}

func (c *TestVMSSClient) UnSetError() {
	c.err = nil
}

func (c *TestVMSSClient) Get(rgName string, nodeName string) (ret compute.VirtualMachineScaleSet, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	stored := c.nodeMap[nodeName]
	if stored == nil {
		vm := new(compute.VirtualMachineScaleSet)
		c.nodeMap[nodeName] = vm
		return *vm, nil
	}
	return *stored, nil
}

func (c *TestVMSSClient) CreateOrUpdate(rg string, nodeName string, vm compute.VirtualMachineScaleSet) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.err != nil {
		c.nodeMap[nodeName].Identity = c.identity
		return *c.err
	}
	c.nodeMap[nodeName] = &vm
	return nil
}

func (c *TestVMSSClient) ListMSI() (ret map[string]*[]string) {
	ret = make(map[string]*[]string)

	c.mu.Lock()
	defer c.mu.Unlock()

	for key, val := range c.nodeMap {
		ret[key] = val.Identity.IdentityIds
	}
	return ret
}

func (c *TestVMSSClient) CompareMSI(nodeName string, userIDs []string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

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
		klog.Infof("\nNode name: %s", key)
		if val != nil {
			for i, id := range *val {
				klog.Infof("%d) %s", i, id)
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
	nodeMap := make(map[string]*compute.VirtualMachine)
	vmClient := &cp.VMClient{}
	identity := &compute.VirtualMachineIdentity{IdentityIds: &[]string{}}

	return &TestVMClient{
		VMClient: vmClient,
		nodeMap:  nodeMap,
		identity: identity,
	}
}

func NewTestVMSSClient() *TestVMSSClient {
	nodeMap := make(map[string]*compute.VirtualMachineScaleSet)
	vmssClient := &cp.VMSSClient{}
	identity := &compute.VirtualMachineScaleSetIdentity{IdentityIds: &[]string{}}

	return &TestVMSSClient{
		VMSSClient: vmssClient,
		nodeMap:    nodeMap,
		identity:   identity,
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
	mu   sync.Mutex
	pods []*corev1.Pod
}

func NewTestPodClient() *TestPodClient {
	var pods []*corev1.Pod
	return &TestPodClient{
		pods: pods,
	}
}

func (c *TestPodClient) Start(exit <-chan struct{}) {
	klog.Info("Start called from the test interface")
}

func (c *TestPodClient) GetPods() ([]*corev1.Pod, error) {
	//TODO: Add label matching. For now we add only pods which we want to add.
	c.mu.Lock()
	defer c.mu.Unlock()

	pods := make([]*corev1.Pod, len(c.pods))
	copy(pods, c.pods)

	return pods, nil
}

func (c *TestPodClient) AddPod(podName, podNs, nodeName, binding string) {
	labels := make(map[string]string)
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

	c.mu.Lock()
	defer c.mu.Unlock()
	c.pods = append(c.pods, pod)
}

func (c *TestPodClient) DeletePod(podName string, podNs string) {
	var newPods []*corev1.Pod
	changed := false

	c.mu.Lock()
	defer c.mu.Unlock()

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
	*crd.Client
	mu            sync.Mutex
	assignedIDMap map[string]*internalaadpodid.AzureAssignedIdentity
	bindingMap    map[string]*aadpodid.AzureIdentityBinding
	idMap         map[string]*aadpodid.AzureIdentity
	err           *error
}

func NewTestCrdClient(config *rest.Config) *TestCrdClient {
	return &TestCrdClient{
		assignedIDMap: make(map[string]*internalaadpodid.AzureAssignedIdentity),
		bindingMap:    make(map[string]*aadpodid.AzureIdentityBinding),
		idMap:         make(map[string]*aadpodid.AzureIdentity),
	}
}

func (c *TestCrdClient) Start(exit <-chan struct{}) {
}

func (c *TestCrdClient) SyncCache(exit <-chan struct{}, initial bool, cacheSyncs ...cache.InformerSynced) {

}

func (c *TestCrdClient) CreateCrdWatchers(eventCh chan internalaadpodid.EventType) (err error) {
	return nil
}

func (c *TestCrdClient) RemoveAssignedIdentity(assignedIdentity *internalaadpodid.AzureAssignedIdentity) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.err != nil {
		return *c.err
	}
	delete(c.assignedIDMap, assignedIdentity.Name)
	return nil
}

// This function is not used currently
// TODO: consider remove
func (c *TestCrdClient) CreateAssignedIdentity(assignedIdentity *internalaadpodid.AzureAssignedIdentity) error {
	assignedIdentityToStore := *assignedIdentity //Make a copy to store in the map.
	c.mu.Lock()
	c.assignedIDMap[assignedIdentity.Name] = &assignedIdentityToStore
	c.mu.Unlock()
	return nil
}

func (c *TestCrdClient) UpdateAzureAssignedIdentityStatus(assignedIdentity *internalaadpodid.AzureAssignedIdentity, status string) error {
	assignedIdentity.Status.Status = status
	assignedIdentityToStore := *assignedIdentity //Make a copy to store in the map.
	c.mu.Lock()
	c.assignedIDMap[assignedIdentity.Name] = &assignedIdentityToStore
	c.mu.Unlock()
	return nil
}

func (c *TestCrdClient) CreateBinding(name, ns, idName, selector, resourceVersion string) {
	binding := &aadpodid.AzureIdentityBinding{
		ObjectMeta: v1.ObjectMeta{
			Name:            name,
			Namespace:       ns,
			ResourceVersion: resourceVersion,
		},
		Spec: aadpodid.AzureIdentityBindingSpec{
			AzureIdentity: idName,
			Selector:      selector,
		},
	}
	c.mu.Lock()
	c.bindingMap[getIDKey(ns, name)] = binding
	c.mu.Unlock()
}

func (c *TestCrdClient) CreateID(idName, ns string, t aadpodid.IdentityType, rID, cID string, cp *api.SecretReference, tID, adRID, adEpt, resourceVersion string) {
	id := &aadpodid.AzureIdentity{
		ObjectMeta: v1.ObjectMeta{
			Name:            idName,
			Namespace:       ns,
			ResourceVersion: resourceVersion,
		},
		Spec: aadpodid.AzureIdentitySpec{
			Type:         t,
			ResourceID:   rID,
			ClientID:     cID,
			TenantID:     tID,
			ADResourceID: adRID,
			ADEndpoint:   adEpt,
		},
	}
	c.mu.Lock()
	c.idMap[getIDKey(ns, idName)] = id
	c.mu.Unlock()
}

func (c *TestCrdClient) ListIds() (res *[]internalaadpodid.AzureIdentity, err error) {
	idList := make([]internalaadpodid.AzureIdentity, 0)
	c.mu.Lock()
	for _, v := range c.idMap {
		currID := aadpodid.ConvertV1IdentityToInternalIdentity(*v)
		idList = append(idList, currID)
	}
	c.mu.Unlock()
	return &idList, nil
}

func (c *TestCrdClient) ListBindings() (res *[]internalaadpodid.AzureIdentityBinding, err error) {
	bindingList := make([]internalaadpodid.AzureIdentityBinding, 0)
	c.mu.Lock()
	for _, v := range c.bindingMap {
		newBinding := aadpodid.ConvertV1BindingToInternalBinding(*v)
		bindingList = append(bindingList, newBinding)
	}
	c.mu.Unlock()
	return &bindingList, nil
}

func (c *TestCrdClient) ListAssignedIDs() (res *[]internalaadpodid.AzureAssignedIdentity, err error) {
	assignedIDList := make([]internalaadpodid.AzureAssignedIdentity, 0)
	c.mu.Lock()
	for _, v := range c.assignedIDMap {
		assignedIDList = append(assignedIDList, *v)
	}
	c.mu.Unlock()
	return &assignedIDList, nil
}

func (c *TestCrdClient) ListAssignedIDsInMap() (res map[string]internalaadpodid.AzureAssignedIdentity, err error) {
	assignedIDMap := make(map[string]internalaadpodid.AzureAssignedIdentity)
	c.mu.Lock()
	for k, v := range c.assignedIDMap {
		assignedIDMap[k] = *v
	}
	c.mu.Unlock()
	return assignedIDMap, nil
}

func (c *Client) ListPodIds(podns, podname string) (map[string][]internalaadpodid.AzureIdentity, error) {
	return map[string][]internalaadpodid.AzureIdentity{}, nil
}

// ListPodIdentityExceptions ...
func (c *Client) ListPodIdentityExceptions(ns string) (*[]internalaadpodid.AzurePodIdentityException, error) {
	return nil, nil
}

func (c *TestCrdClient) SetError(err error) {
	c.err = &err
}

func (c *TestCrdClient) UnSetError() {
	c.err = nil
}

/************************ NODE MOCK *************************************/

type TestNodeClient struct {
	mu    sync.Mutex
	nodes map[string]*corev1.Node
}

func NewTestNodeClient() *TestNodeClient {
	return &TestNodeClient{nodes: make(map[string]*corev1.Node)}
}

func (c *TestNodeClient) Get(name string) (*corev1.Node, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	node, exists := c.nodes[name]
	if !exists {
		return nil, errors.New("node not found")
	}
	return node, nil
}

func (c *TestNodeClient) Delete(name string) {
	c.mu.Lock()
	delete(c.nodes, name)
	c.mu.Unlock()
}

func (c *TestNodeClient) Start(<-chan struct{}) {}

func (c *TestNodeClient) AddNode(name string, opts ...func(*corev1.Node)) {
	c.mu.Lock()
	defer c.mu.Unlock()

	n := &corev1.Node{ObjectMeta: v1.ObjectMeta{Name: name}, Spec: corev1.NodeSpec{
		ProviderID: "azure:///subscriptions/testSub/resourceGroups/fakeGroup/providers/Microsoft.Compute/virtualMachines/" + name,
	}}
	for _, o := range opts {
		o(n)
	}
	c.nodes[name] = n
}

/************************ EVENT RECORDER MOCK *************************************/
type LastEvent struct {
	Type    string
	Reason  string
	Message string
}

type TestEventRecorder struct {
	mu        sync.Mutex
	lastEvent *LastEvent

	eventChannel chan bool
}

func (c *TestEventRecorder) WaitForEvents(expectedCount int) bool {
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

func (c *TestEventRecorder) Event(object runtime.Object, t string, r string, message string) {
	c.mu.Lock()

	c.lastEvent.Type = t
	c.lastEvent.Reason = r
	c.lastEvent.Message = message

	c.mu.Unlock()

	c.eventChannel <- true
}

func (c *TestEventRecorder) Validate(e *LastEvent) bool {
	c.mu.Lock()

	t := c.lastEvent.Type
	r := c.lastEvent.Reason
	m := c.lastEvent.Message

	c.mu.Unlock()

	if t != e.Type || r != e.Reason || m != e.Message {
		klog.Errorf("event mismatch. expected - (t:%s, r:%s, m:%s). got - (t:%s, r:%s, m:%s)", e.Type, e.Reason, e.Message, t, r, m)
		return false
	}
	return true
}

func (c *TestEventRecorder) Eventf(object runtime.Object, t string, r string, messageFmt string, args ...interface{}) {

}

func (c *TestEventRecorder) PastEventf(object runtime.Object, timestamp v1.Time, t string, m1 string, messageFmt string, args ...interface{}) {

}

func (c *TestEventRecorder) AnnotatedEventf(object runtime.Object, annotations map[string]string, eventtype, reason, messageFmt string, args ...interface{}) {

}

/************************ MIC MOC *************************************/
func NewMICTestClient(eventCh chan internalaadpodid.EventType,
	cpClient *TestCloudClient,
	crdClient *TestCrdClient,
	podClient *TestPodClient,
	nodeClient *TestNodeClient,
	eventRecorder *TestEventRecorder, isNamespaced bool,
	createDeleteBatch int64,
	immutableUserMSIs map[string]bool) *TestMICClient {

	reporter, _ := metrics.NewReporter()

	realMICClient := &Client{
		CloudClient:          cpClient,
		CRDClient:            crdClient,
		EventRecorder:        eventRecorder,
		PodClient:            podClient,
		EventChannel:         eventCh,
		NodeClient:           nodeClient,
		syncRetryInterval:    120 * time.Second,
		IsNamespaced:         isNamespaced,
		createDeleteBatch:    createDeleteBatch,
		ImmutableUserMSIsMap: immutableUserMSIs,
		Reporter:             reporter,
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
	defaultNS := "default"
	micClient := &TestMICClient{}

	idList := make([]internalaadpodid.AzureIdentity, 0)

	id := new(internalaadpodid.AzureIdentity)
	id.Namespace = "default"
	id.Name = "test-azure-identity"
	id.Namespace = defaultNS

	idList = append(idList, *id)

	id.Namespace = "newns"
	id.Name = "test-akssvcrg-id"

	idList = append(idList, *id)

	idMap, _ := micClient.convertIDListToMap(idList)

	namespace := "default"
	name := "test-azure-identity"
	count := 3
	if azureID, idPresent := idMap[getIDKey(namespace, name)]; idPresent {
		if azureID.Name != name {
			t.Fatalf("id map id value mismatch")
		}
		count = count - 1
	} else {
		t.Fatalf("id %s not found", name)
	}

	namespace = "newns"
	name = "test-akssvcrg-id"
	if azureID, idPresent := idMap[getIDKey(namespace, name)]; idPresent {
		if azureID.Name != name {
			t.Fatalf("id map id value mismatch")
		}
		count = count - 1
	} else {
		t.Fatalf("id %s not found", name)
	}

	namespace = "default"
	name = "test not there"
	if _, idPresent := idMap[getIDKey(namespace, name)]; idPresent {
		t.Fatalf("not present found")
	} else {
		count = count - 1
	}
	if count != 0 {
		t.Fatalf("Test count mismatch. Expected %d, actual %d", 0, count)
	}
}

func (c *TestMICClient) testRunSync() func(t *testing.T) {
	done := make(chan struct{})
	exit := make(chan struct{})
	var closeOnce sync.Once

	go func() {
		c.Sync(exit)
		close(done)
	}()

	return func(t *testing.T) {
		t.Helper()

		closeOnce.Do(func() {
			close(exit)
		})

		timeout := time.NewTimer(30 * time.Second)
		defer timeout.Stop()

		select {
		case <-done:
		case <-timeout.C:
			t.Fatal("timeout waiting for sync to exit")
		}
	}
}

func TestSimpleMICClient(t *testing.T) {
	eventCh := make(chan internalaadpodid.EventType, 100)
	cloudClient := NewTestCloudClient(config.AzureConfig{})
	crdClient := NewTestCrdClient(nil)
	podClient := NewTestPodClient()
	nodeClient := NewTestNodeClient()
	var evtRecorder TestEventRecorder
	evtRecorder.lastEvent = new(LastEvent)
	evtRecorder.eventChannel = make(chan bool, 100)

	micClient := NewMICTestClient(eventCh, cloudClient, crdClient, podClient, nodeClient, &evtRecorder, false, 4, nil)

	crdClient.CreateID("test-id", "default", aadpodid.UserAssignedMSI, "test-user-msi-resourceid", "test-user-msi-clientid", nil, "", "", "", "")
	crdClient.CreateBinding("testbinding", "default", "test-id", "test-select", "")

	nodeClient.AddNode("test-node")
	podClient.AddPod("test-pod", "default", "test-node", "test-select")

	eventCh <- internalaadpodid.PodCreated

	defer micClient.testRunSync()(t)

	evtRecorder.WaitForEvents(1)

	testPass := false
	listAssignedIDs, err := crdClient.ListAssignedIDs()
	if err != nil {
		klog.Error(err)
		t.Errorf("list assigned failed")
	}

	if listAssignedIDs != nil {
		for _, assignedID := range *listAssignedIDs {
			if assignedID.Spec.Pod == "test-pod" && assignedID.Spec.PodNamespace == "default" && assignedID.Spec.NodeName == "test-node" &&
				assignedID.Spec.AzureBindingRef.Name == "testbinding" && assignedID.Spec.AzureIdentityRef.Name == "test-id" {
				testPass = true
				break
			}
		}
	}

	if !testPass {
		t.Fatalf("assigned id mismatch")
	}

	//Test2: Remove assigned id event test
	podClient.DeletePod("test-pod", "default")

	eventCh <- internalaadpodid.PodDeleted
	if !evtRecorder.WaitForEvents(1) {
		t.Fatal("timeout waiting for event sync")
	}

	listAssignedIDs, err = crdClient.ListAssignedIDs()
	if err != nil {
		klog.Error(err)
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

	eventCh <- internalaadpodid.PodCreated
	evtRecorder.WaitForEvents(1)

	listAssignedIDs, err = crdClient.ListAssignedIDs()
	if err != nil {
		klog.Error(err)
		t.Fatalf("list assigned failed")
	}

	if (*listAssignedIDs)[0].Status.Status != aadpodid.AssignedIDCreated {
		t.Fatalf("expected status to be %s, got: %s", aadpodid.AssignedIDCreated, (*listAssignedIDs)[0].Status.Status)
	}

	/*
		testPass = evtRecorder.Validate(&LastEvent{Type: "Warning", Reason: "binding apply error",
			Message: "Applying binding testbinding node test-node for pod test-pod-default-test-id resulted in error error returned from cloud provider"})

		if !testPass {
			t.Errorf("event mismatch")
		} */

	// Test4: Removal error event test
	// Reset the state to add the id.
	cloudClient.UnSetError()

	//podClient.AddPod("test-pod", "default", "test-node", "test-select")
	eventCh <- internalaadpodid.PodCreated

	err = errors.New("remove error returned from cloud provider")
	cloudClient.SetError(err)

	podClient.DeletePod("test-pod", "default")
	eventCh <- internalaadpodid.PodDeleted
	/*
		testPass = evtRecorder.Validate(&LastEvent{Type: "Warning", Reason: "binding remove error",
			Message: "Binding testbinding removal from node test-node for pod test-pod resulted in error remove error returned from cloud provider"})

		if !testPass {
			t.Errorf("event mismatch")
		}
	*/
}

func TestAddDelMICClient(t *testing.T) {
	eventCh := make(chan internalaadpodid.EventType, 100)
	cloudClient := NewTestCloudClient(config.AzureConfig{})
	crdClient := NewTestCrdClient(nil)
	podClient := NewTestPodClient()
	nodeClient := NewTestNodeClient()
	var evtRecorder TestEventRecorder
	evtRecorder.lastEvent = new(LastEvent)
	evtRecorder.eventChannel = make(chan bool, 100)

	micClient := NewMICTestClient(eventCh, cloudClient, crdClient, podClient, nodeClient, &evtRecorder, false, 4, nil)

	// Test to add and delete at the same time.
	// Add a pod, identity and binding.
	crdClient.CreateID("test-id2", "default", aadpodid.UserAssignedMSI, "test-user-msi-resourceid", "test-user-msi-clientid", nil, "", "", "", "")
	crdClient.CreateBinding("testbinding2", "default", "test-id2", "test-select2", "")

	nodeClient.AddNode("test-node2")
	podClient.AddPod("test-pod2", "default", "test-node2", "test-select2")
	podClient.GetPods()

	crdClient.CreateID("test-id4", "default", aadpodid.UserAssignedMSI, "test-user-msi-resourceid", "test-user-msi-clientid", nil, "", "", "", "")
	crdClient.CreateBinding("testbinding4", "default", "test-id4", "test-select4", "")
	podClient.AddPod("test-pod4", "default", "test-node2", "test-select4")
	podClient.GetPods()

	eventCh <- internalaadpodid.PodCreated
	eventCh <- internalaadpodid.PodCreated

	stopSync1 := micClient.testRunSync()
	defer stopSync1(t)

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
		klog.Errorf("Expected len: %d. Got: %d", expectedLen, gotLen)
		t.Fatalf("Add and delete id at same time mismatch")
	}

	// Delete the pod
	podClient.DeletePod("test-pod2", "default")
	podClient.DeletePod("test-pod4", "default")

	//Add a new pod, with different id and binding on the same node.
	crdClient.CreateID("test-id3", "default", aadpodid.UserAssignedMSI, "test-user-msi-resourceid", "test-user-msi-clientid", nil, "", "", "", "")
	crdClient.CreateBinding("testbinding3", "default", "test-id3", "test-select3", "")
	podClient.AddPod("test-pod3", "default", "test-node2", "test-select3")
	podClient.GetPods()

	eventCh <- internalaadpodid.PodCreated
	eventCh <- internalaadpodid.PodDeleted
	eventCh <- internalaadpodid.PodDeleted

	stopSync1(t)
	defer micClient.testRunSync()(t)

	if !evtRecorder.WaitForEvents(3) {
		t.Fatalf("Timeout waiting for mic sync cycles")
	}

	listAssignedIDs, err = crdClient.ListAssignedIDs()
	if err != nil {
		klog.Error(err)
		t.Fatalf("list assigned failed")
	}

	expectedLen = 1
	gotLen = len(*listAssignedIDs)
	//One id should be left around. Rest should be removed
	if gotLen != expectedLen {
		klog.Errorf("Expected len: %d. Got: %d", expectedLen, gotLen)
		t.Fatalf("Add and delete id at same time mismatch")
	} else {
		gotID := (*listAssignedIDs)[0].Name
		expectedID := "test-pod3-default-test-id3"
		if gotID != expectedID {
			klog.Errorf("Expected %s. Got: %s", expectedID, gotID)
			t.Fatalf("Add and delete id at same time. Found wrong id")
		}
	}
}

func TestMicAddDelVMSS(t *testing.T) {
	eventCh := make(chan internalaadpodid.EventType, 100)
	cloudClient := NewTestCloudClient(config.AzureConfig{VMType: "vmss"})
	crdClient := NewTestCrdClient(nil)
	podClient := NewTestPodClient()
	nodeClient := NewTestNodeClient()
	var evtRecorder TestEventRecorder
	evtRecorder.lastEvent = new(LastEvent)
	evtRecorder.eventChannel = make(chan bool, 100)

	micClient := NewMICTestClient(eventCh, cloudClient, crdClient, podClient, nodeClient, &evtRecorder, false, 4, nil)

	// Test to add and delete at the same time.
	// Add a pod, identity and binding.
	crdClient.CreateID("test-id1", "default", aadpodid.UserAssignedMSI, "test-user-msi-resourceid", "test-user-msi-clientid", nil, "", "", "", "")
	crdClient.CreateBinding("testbinding1", "default", "test-id1", "test-select1", "")

	nodeClient.AddNode("test-node1", func(n *corev1.Node) {
		n.Spec.ProviderID = "azure:///subscriptions/fakeSub/resourceGroups/fakeGroup/providers/Microsoft.Compute/virtualMachineScaleSets/testvmss1/virtualMachines/0"
	})

	nodeClient.AddNode("test-node2", func(n *corev1.Node) {
		n.Spec.ProviderID = "azure:///subscriptions/fakeSub/resourceGroups/fakeGroup/providers/Microsoft.Compute/virtualMachineScaleSets/testvmss1/virtualMachines/1"
	})

	nodeClient.AddNode("test-node3", func(n *corev1.Node) {
		n.Spec.ProviderID = "azure:///subscriptions/fakeSub/resourceGroups/fakeGroup/providers/Microsoft.Compute/virtualMachineScaleSets/testvmss2/virtualMachines/0"
	})

	podClient.AddPod("test-pod1", "default", "test-node1", "test-select1")
	podClient.AddPod("test-pod2", "default", "test-node2", "test-select1")
	podClient.AddPod("test-pod3", "default", "test-node3", "test-select1")

	defer micClient.testRunSync()(t)

	eventCh <- internalaadpodid.PodCreated
	eventCh <- internalaadpodid.PodCreated
	eventCh <- internalaadpodid.PodCreated
	if !evtRecorder.WaitForEvents(3) {
		t.Fatalf("Timeout waiting for mic sync cycles")
	}
	listAssignedIDs, err := crdClient.ListAssignedIDs()
	if err != nil {
		klog.Error(err)
		t.Errorf("list assigned failed")
	}
	if !(len(*listAssignedIDs) == 3) {
		t.Fatalf("expected assigned identities len: %d, got: %d", 3, len(*listAssignedIDs))
	}

	if !cloudClient.CompareMSI("testvmss1", []string{"test-user-msi-resourceid"}) {
		t.Fatalf("missing identity: %+v", cloudClient.ListMSI()["testvmss1"])
	}
	if !cloudClient.CompareMSI("testvmss2", []string{"test-user-msi-resourceid"}) {
		t.Fatalf("missing identity: %+v", cloudClient.ListMSI()["testvmss2"])
	}

	podClient.DeletePod("test-pod1", "default")
	eventCh <- internalaadpodid.PodDeleted

	if !evtRecorder.WaitForEvents(1) {
		t.Fatal("Timeout waiting for mic sync cycles")
	}
	listAssignedIDs, err = crdClient.ListAssignedIDs()
	if err != nil {
		klog.Error(err)
		t.Errorf("list assigned failed")
	}
	if !(len(*listAssignedIDs) == 2) {
		t.Fatalf("expected assigned identities len: %d, got: %d", 2, len(*listAssignedIDs))
	}

	if !cloudClient.CompareMSI("testvmss1", []string{"test-user-msi-resourceid"}) {
		t.Fatalf("missing identity: %+v", cloudClient.ListMSI()["testvmss1"])
	}
	if !cloudClient.CompareMSI("testvmss2", []string{"test-user-msi-resourceid"}) {
		t.Fatalf("missing identity: %+v", cloudClient.ListMSI()["testvmss2"])
	}

	podClient.DeletePod("test-pod2", "default")

	eventCh <- internalaadpodid.PodDeleted

	if !evtRecorder.WaitForEvents(1) {
		t.Fatal("Timeout waiting for mic sync cycles")
	}
	listAssignedIDs, err = crdClient.ListAssignedIDs()
	if err != nil {
		klog.Error(err)
		t.Errorf("list assigned failed")
	}
	if !(len(*listAssignedIDs) == 1) {
		t.Fatalf("expected assigned identities len: %d, got: %d", 1, len(*listAssignedIDs))
	}

	if !cloudClient.CompareMSI("testvmss1", []string{}) {
		t.Fatalf("missing identity: %+v", cloudClient.ListMSI()["testvmss1"])
	}
	if !cloudClient.CompareMSI("testvmss2", []string{"test-user-msi-resourceid"}) {
		t.Fatalf("missing identity: %+v", cloudClient.ListMSI()["testvmss2"])
	}
}

func TestMICStateFlow(t *testing.T) {
	eventCh := make(chan internalaadpodid.EventType, 100)
	cloudClient := NewTestCloudClient(config.AzureConfig{})
	crdClient := NewTestCrdClient(nil)
	podClient := NewTestPodClient()
	nodeClient := NewTestNodeClient()
	var evtRecorder TestEventRecorder
	evtRecorder.lastEvent = new(LastEvent)
	evtRecorder.eventChannel = make(chan bool, 100)

	micClient := NewMICTestClient(eventCh, cloudClient, crdClient, podClient, nodeClient, &evtRecorder, false, 4, nil)

	// Add a pod, identity and binding.
	crdClient.CreateID("test-id1", "default", aadpodid.UserAssignedMSI, "test-user-msi-resourceid", "test-user-msi-clientid", nil, "", "", "", "")
	crdClient.CreateBinding("testbinding1", "default", "test-id1", "test-select1", "")

	nodeClient.AddNode("test-node1")
	podClient.AddPod("test-pod1", "default", "test-node1", "test-select1")

	eventCh <- internalaadpodid.PodCreated
	defer micClient.testRunSync()(t)

	if !evtRecorder.WaitForEvents(1) {
		t.Fatalf("Timeout waiting for mic sync cycles")
	}

	listAssignedIDs, err := crdClient.ListAssignedIDs()
	if err != nil {
		klog.Error(err)
		t.Errorf("list assigned failed")
	}
	if !(len(*listAssignedIDs) == 1) {
		t.Fatalf("expected assigned identities len: %d, got: %d", 1, len(*listAssignedIDs))
	}
	if !((*listAssignedIDs)[0].Status.Status == aadpodid.AssignedIDAssigned) {
		t.Fatalf("expected status to be %s, got: %s", aadpodid.AssignedIDAssigned, (*listAssignedIDs)[0].Status.Status)
	}

	// delete the pod, simulate failure in cloud calls on trying to un-assign identity from node
	podClient.DeletePod("test-pod1", "default")
	// SetError sets error in crd client only for remove assigned identity
	cloudClient.SetError(errors.New("error removing identity from node"))
	cloudClient.testVMClient.identity = &compute.VirtualMachineIdentity{IdentityIds: &[]string{"test-user-msi-resourceid"}}

	eventCh <- internalaadpodid.PodDeleted
	if !evtRecorder.WaitForEvents(1) {
		t.Fatalf("Timeout waiting for mic sync cycles")
	}

	listAssignedIDs, err = crdClient.ListAssignedIDs()
	if err != nil {
		klog.Error(err)
		t.Errorf("list assigned failed")
	}
	if !(len(*listAssignedIDs) == 1) {
		t.Fatalf("expected assigned identities len: %d, got: %d", 1, len(*listAssignedIDs))
	}
	if !((*listAssignedIDs)[0].Status.Status == aadpodid.AssignedIDAssigned) {
		t.Fatalf("expected status to be %s, got: %s", aadpodid.AssignedIDAssigned, (*listAssignedIDs)[0].Status.Status)
	}

	cloudClient.UnSetError()
	crdClient.SetError(errors.New("error from crd client"))

	// add new pod, this time the old assigned identity which is in Assigned state should be tried to delete
	// simulate failure on kube api call to delete crd
	crdClient.CreateID("test-id2", "default", aadpodid.UserAssignedMSI, "test-user-msi-resourceid2", "test-user-msi-clientid2", nil, "", "", "", "")
	crdClient.CreateBinding("testbinding2", "default", "test-id2", "test-select2", "")

	nodeClient.AddNode("test-node2")
	podClient.AddPod("test-pod2", "default", "test-node2", "test-select2")

	eventCh <- internalaadpodid.PodCreated
	if !evtRecorder.WaitForEvents(2) {
		t.Fatalf("Timeout waiting for mic sync cycles")
	}
	listAssignedIDs, err = crdClient.ListAssignedIDs()
	if err != nil {
		klog.Error(err)
		t.Errorf("list assigned failed")
	}
	if !(len(*listAssignedIDs) == 2) {
		t.Fatalf("expected assigned identities len: %d, got: %d", 2, len(*listAssignedIDs))
	}
	for _, assignedID := range *listAssignedIDs {
		if assignedID.Spec.Pod == "test-pod1" {
			if assignedID.Status.Status != aadpodid.AssignedIDUnAssigned {
				t.Fatalf("Expected status to be: %s. Got: %s", aadpodid.AssignedIDUnAssigned, assignedID.Status.Status)
			}
		}
		if assignedID.Spec.Pod == "test-pod2" {
			if assignedID.Status.Status != aadpodid.AssignedIDAssigned {
				t.Fatalf("Expected status to be: %s. Got: %s", aadpodid.AssignedIDAssigned, assignedID.Status.Status)
			}
		}
	}
	crdClient.UnSetError()

	// delete pod2 and everything should be cleaned up now
	podClient.DeletePod("test-pod2", "default")
	eventCh <- internalaadpodid.PodDeleted
	if !evtRecorder.WaitForEvents(2) {
		t.Fatalf("Timeout waiting for mic sync cycles")
	}
	listAssignedIDs, err = crdClient.ListAssignedIDs()
	if err != nil {
		klog.Error(err)
		t.Errorf("list assigned failed")
	}
	if !(len(*listAssignedIDs) == 0) {
		t.Fatalf("expected assigned identities len: %d, got: %d", 0, len(*listAssignedIDs))
	}
}

func TestForceNamespaced(t *testing.T) {
	eventCh := make(chan internalaadpodid.EventType, 100)
	cloudClient := NewTestCloudClient(config.AzureConfig{})
	crdClient := NewTestCrdClient(nil)
	podClient := NewTestPodClient()
	nodeClient := NewTestNodeClient()
	var evtRecorder TestEventRecorder
	evtRecorder.lastEvent = new(LastEvent)
	evtRecorder.eventChannel = make(chan bool, 100)

	micClient := NewMICTestClient(eventCh, cloudClient, crdClient, podClient, nodeClient, &evtRecorder, true, 4, nil)

	crdClient.CreateID("test-id1", "default", aadpodid.UserAssignedMSI, "test-user-msi-resourceid", "test-user-msi-clientid", nil, "", "", "", "idrv1")
	crdClient.CreateBinding("testbinding1", "default", "test-id1", "test-select1", "bindingrv1")

	nodeClient.AddNode("test-node1")
	podClient.AddPod("test-pod1", "default", "test-node1", "test-select1")

	eventCh <- internalaadpodid.PodCreated
	defer micClient.testRunSync()(t)

	if !evtRecorder.WaitForEvents(1) {
		t.Fatalf("Timeout waiting for mic sync cycles")
	}
	listAssignedIDs, err := crdClient.ListAssignedIDs()
	if err != nil {
		klog.Error(err)
		t.Errorf("list assigned failed")
	}
	if !(len(*listAssignedIDs) == 1) {
		t.Fatalf("expected assigned identities len: %d, got: %d", 1, len(*listAssignedIDs))
	}
	if !((*listAssignedIDs)[0].Status.Status == aadpodid.AssignedIDAssigned) {
		t.Fatalf("expected status to be %s, got: %s", aadpodid.AssignedIDAssigned, (*listAssignedIDs)[0].Status.Status)
	}

	crdClient.CreateID("test-id1", "default2", aadpodid.UserAssignedMSI, "test-user-msi-resourceid1", "test-user-msi-clientid", nil, "", "", "", "idrv2")
	crdClient.CreateBinding("testbinding1", "default2", "test-id1", "test-select1", "bindingrv2")
	podClient.AddPod("test-pod2", "default2", "test-node1", "test-select1")

	eventCh <- internalaadpodid.IdentityCreated
	eventCh <- internalaadpodid.BindingCreated
	eventCh <- internalaadpodid.PodCreated

	if !evtRecorder.WaitForEvents(1) {
		t.Fatalf("Timeout waiting for mic sync cycles")
	}

	listAssignedIDs, err = crdClient.ListAssignedIDs()
	if err != nil {
		klog.Error(err)
		t.Errorf("list assigned failed")
	}
	if !(len(*listAssignedIDs) == 2) {
		t.Fatalf("expected assigned identities len: %d, got: %d", 1, len(*listAssignedIDs))
	}

	for _, assignedID := range *listAssignedIDs {
		if !(assignedID.Status.Status == aadpodid.AssignedIDAssigned) {
			t.Fatalf("expected status to be %s, got: %s", aadpodid.AssignedIDAssigned, (*listAssignedIDs)[0].Status.Status)
		}
	}
}

func TestSyncRetryLoop(t *testing.T) {
	eventCh := make(chan internalaadpodid.EventType, 100)
	cloudClient := NewTestCloudClient(config.AzureConfig{})
	crdClient := NewTestCrdClient(nil)
	podClient := NewTestPodClient()
	nodeClient := NewTestNodeClient()
	var evtRecorder TestEventRecorder
	evtRecorder.lastEvent = new(LastEvent)
	evtRecorder.eventChannel = make(chan bool, 100)

	micClient := NewMICTestClient(eventCh, cloudClient, crdClient, podClient, nodeClient, &evtRecorder, false, 4, nil)
	syncRetryInterval, err := time.ParseDuration("10s")
	if err != nil {
		t.Errorf("error parsing duration: %v", err)
	}
	micClient.syncRetryInterval = syncRetryInterval

	// Add a pod, identity and binding.
	crdClient.CreateID("test-id1", "default", aadpodid.UserAssignedMSI, "test-user-msi-resourceid", "test-user-msi-clientid", nil, "", "", "", "")
	crdClient.CreateBinding("testbinding1", "default", "test-id1", "test-select1", "")

	nodeClient.AddNode("test-node1")
	podClient.AddPod("test-pod1", "default", "test-node1", "test-select1")

	eventCh <- internalaadpodid.PodCreated
	defer micClient.testRunSync()(t)

	if !evtRecorder.WaitForEvents(1) {
		t.Fatalf("Timeout waiting for mic sync cycles")
	}
	listAssignedIDs, err := crdClient.ListAssignedIDs()
	if err != nil {
		klog.Error(err)
		t.Errorf("list assigned failed")
	}
	if !(len(*listAssignedIDs) == 1) {
		t.Fatalf("expected assigned identities len: %d, got: %d", 1, len(*listAssignedIDs))
	}
	if !((*listAssignedIDs)[0].Status.Status == aadpodid.AssignedIDAssigned) {
		t.Fatalf("expected status to be %s, got: %s", aadpodid.AssignedIDAssigned, (*listAssignedIDs)[0].Status.Status)
	}

	// delete the pod, simulate failure in cloud calls on trying to un-assign identity from node
	podClient.DeletePod("test-pod1", "default")
	cloudClient.SetError(errors.New("error removing identity from node"))
	cloudClient.testVMClient.identity = &compute.VirtualMachineIdentity{IdentityIds: &[]string{"test-user-msi-resourceid"}}

	eventCh <- internalaadpodid.PodDeleted
	if !evtRecorder.WaitForEvents(1) {
		t.Fatalf("Timeout waiting for mic sync cycles")
	}

	listAssignedIDs, err = crdClient.ListAssignedIDs()
	if err != nil {
		klog.Error(err)
		t.Errorf("list assigned failed")
	}
	if !(len(*listAssignedIDs) == 1) {
		t.Fatalf("expected assigned identities len: %d, got: %d", 1, len(*listAssignedIDs))
	}
	if !((*listAssignedIDs)[0].Status.Status == aadpodid.AssignedIDAssigned) {
		t.Fatalf("expected status to be %s, got: %s", aadpodid.AssignedIDAssigned, (*listAssignedIDs)[0].Status.Status)
	}
	cloudClient.UnSetError()

	if !evtRecorder.WaitForEvents(1) {
		t.Fatalf("Timeout waiting for mic sync retry cycle")
	}

	listAssignedIDs, err = crdClient.ListAssignedIDs()
	if err != nil {
		klog.Error(err)
		t.Errorf("list assigned failed")
	}
	if !(len(*listAssignedIDs) == 0) {
		t.Fatalf("expected assigned identities len: %d, got: %d", 0, len(*listAssignedIDs))
	}
}

func TestSyncNodeNotFound(t *testing.T) {
	eventCh := make(chan internalaadpodid.EventType, 100)
	cloudClient := NewTestCloudClient(config.AzureConfig{})
	crdClient := NewTestCrdClient(nil)
	podClient := NewTestPodClient()
	nodeClient := NewTestNodeClient()
	var evtRecorder TestEventRecorder
	evtRecorder.lastEvent = new(LastEvent)
	evtRecorder.eventChannel = make(chan bool, 100)

	micClient := NewMICTestClient(eventCh, cloudClient, crdClient, podClient, nodeClient, &evtRecorder, false, 4, nil)

	// Add a pod, identity and binding.
	crdClient.CreateID("test-id1", "default", aadpodid.UserAssignedMSI, "test-user-msi-resourceid", "test-user-msi-clientid", nil, "", "", "", "")
	crdClient.CreateBinding("testbinding1", "default", "test-id1", "test-select1", "")

	for i := 0; i < 10; i++ {
		nodeClient.AddNode(fmt.Sprintf("test-node%d", i))
		podClient.AddPod(fmt.Sprintf("test-pod%d", i), "default", fmt.Sprintf("test-node%d", i), "test-select1")
		eventCh <- internalaadpodid.PodCreated
	}

	defer micClient.testRunSync()(t)

	if !evtRecorder.WaitForEvents(10) {
		t.Fatalf("Timeout waiting for mic sync cycles")
	}

	listAssignedIDs, err := crdClient.ListAssignedIDs()
	if err != nil {
		klog.Error(err)
		t.Errorf("list assigned failed")
	}
	if !(len(*listAssignedIDs) == 10) {
		t.Fatalf("expected assigned identities len: %d, got: %d", 10, len(*listAssignedIDs))
	}
	for i := range *listAssignedIDs {
		if !((*listAssignedIDs)[i].Status.Status == aadpodid.AssignedIDAssigned) {
			t.Fatalf("expected status to be %s, got: %s", aadpodid.AssignedIDAssigned, (*listAssignedIDs)[i].Status.Status)
		}
	}

	// delete 5 nodes
	for i := 5; i < 10; i++ {
		nodeClient.Delete(fmt.Sprintf("test-node%d", i))
		podClient.DeletePod(fmt.Sprintf("test-pod%d", i), "default")
		eventCh <- internalaadpodid.PodDeleted
	}

	nodeClient.AddNode("test-nodex")
	podClient.AddPod("test-podx", "default", "test-node1", "test-select1")
	eventCh <- internalaadpodid.PodCreated

	if !evtRecorder.WaitForEvents(6) {
		t.Fatalf("Timeout waiting for mic sync cycles")
	}

	listAssignedIDs, err = crdClient.ListAssignedIDs()
	if err != nil {
		klog.Error(err)
		t.Errorf("list assigned failed")
	}
	if !(len(*listAssignedIDs) == 6) {
		t.Fatalf("expected assigned identities len: %d, got: %d", 6, len(*listAssignedIDs))
	}
	for i := range *listAssignedIDs {
		if !((*listAssignedIDs)[i].Status.Status == aadpodid.AssignedIDAssigned) {
			t.Fatalf("expected status to be %s, got: %s", aadpodid.AssignedIDAssigned, (*listAssignedIDs)[i].Status.Status)
		}
	}
}

func TestProcessingTimeForScale(t *testing.T) {
	eventCh := make(chan internalaadpodid.EventType, 20000)
	cloudClient := NewTestCloudClient(config.AzureConfig{})
	crdClient := NewTestCrdClient(nil)
	podClient := NewTestPodClient()
	nodeClient := NewTestNodeClient()
	var evtRecorder TestEventRecorder
	evtRecorder.lastEvent = new(LastEvent)
	evtRecorder.eventChannel = make(chan bool, 20000)

	micClient := NewMICTestClient(eventCh, cloudClient, crdClient, podClient, nodeClient, &evtRecorder, false, 4, nil)

	// Add a pod, identity and binding.
	crdClient.CreateID("test-id1", "default", aadpodid.UserAssignedMSI, "test-user-msi-resourceid", "test-user-msi-clientid", nil, "", "", "", "")
	crdClient.CreateBinding("testbinding1", "default", "test-id1", "test-select1", "")

	nodeClient.AddNode("test-node1")
	for i := 0; i < 20000; i++ {
		podClient.AddPod(fmt.Sprintf("test-pod%d", i), "default", "test-node1", "test-select1")
	}
	eventCh <- internalaadpodid.PodCreated

	defer micClient.testRunSync()(t)

	if !evtRecorder.WaitForEvents(20000) {
		t.Fatalf("Timeout waiting for mic sync cycles")
	}

	listAssignedIDs, err := crdClient.ListAssignedIDs()
	if err != nil {
		klog.Error(err)
		t.Errorf("list assigned failed")
	}
	if !(len(*listAssignedIDs) == 20000) {
		t.Fatalf("expected assigned identities len: %d, got: %d", 20000, len(*listAssignedIDs))
	}

	for i := 10000; i < 20000; i++ {
		podClient.DeletePod(fmt.Sprintf("test-pod%d", i), "default")
	}
	eventCh <- internalaadpodid.PodDeleted

	if !evtRecorder.WaitForEvents(10000) {
		t.Fatalf("Timeout waiting for mic sync cycles")
	}
	listAssignedIDs, err = crdClient.ListAssignedIDs()
	if err != nil {
		klog.Error(err)
		t.Errorf("list assigned failed")
	}
	if !(len(*listAssignedIDs) == 10000) {
		t.Fatalf("expected assigned identities len: %d, got: %d", 10000, len(*listAssignedIDs))
	}
}

func TestSyncExit(t *testing.T) {
	eventCh := make(chan internalaadpodid.EventType)
	cloudClient := NewTestCloudClient(config.AzureConfig{VMType: "vmss"})
	crdClient := NewTestCrdClient(nil)
	podClient := NewTestPodClient()
	nodeClient := NewTestNodeClient()
	var evtRecorder TestEventRecorder
	evtRecorder.lastEvent = new(LastEvent)
	evtRecorder.eventChannel = make(chan bool)

	micClient := NewMICTestClient(eventCh, cloudClient, crdClient, podClient, nodeClient, &evtRecorder, false, 4, nil)

	micClient.testRunSync()(t)
}

func TestMicAddDelVMSSwithImmutableIdentities(t *testing.T) {
	eventCh := make(chan internalaadpodid.EventType, 100)
	cloudClient := NewTestCloudClient(config.AzureConfig{VMType: "vmss"})
	crdClient := NewTestCrdClient(nil)
	podClient := NewTestPodClient()
	nodeClient := NewTestNodeClient()
	var evtRecorder TestEventRecorder
	evtRecorder.lastEvent = new(LastEvent)
	evtRecorder.eventChannel = make(chan bool, 100)
	var immutableUserMSIs = map[string]bool{
		"zero-test":              true,
		"test-user-msi-clientid": true,
	}

	micClient := NewMICTestClient(eventCh, cloudClient, crdClient, podClient, nodeClient, &evtRecorder, false, 4, immutableUserMSIs)

	// Test to add and delete at the same time.
	// Add a pod, identity and binding.
	crdClient.CreateID("test-id1", "default", aadpodid.UserAssignedMSI, "test-user-msi-resourceid", "test-user-msi-clientid", nil, "", "", "", "")
	crdClient.CreateBinding("testbinding1", "default", "test-id1", "test-select1", "")

	nodeClient.AddNode("test-node1", func(n *corev1.Node) {
		n.Spec.ProviderID = "azure:///subscriptions/fakeSub/resourceGroups/fakeGroup/providers/Microsoft.Compute/virtualMachineScaleSets/testvmss1/virtualMachines/0"
	})

	nodeClient.AddNode("test-node2", func(n *corev1.Node) {
		n.Spec.ProviderID = "azure:///subscriptions/fakeSub/resourceGroups/fakeGroup/providers/Microsoft.Compute/virtualMachineScaleSets/testvmss1/virtualMachines/1"
	})

	nodeClient.AddNode("test-node3", func(n *corev1.Node) {
		n.Spec.ProviderID = "azure:///subscriptions/fakeSub/resourceGroups/fakeGroup/providers/Microsoft.Compute/virtualMachineScaleSets/testvmss2/virtualMachines/0"
	})

	podClient.AddPod("test-pod1", "default", "test-node1", "test-select1")
	podClient.AddPod("test-pod2", "default", "test-node2", "test-select1")
	podClient.AddPod("test-pod3", "default", "test-node3", "test-select1")

	defer micClient.testRunSync()(t)

	eventCh <- internalaadpodid.PodCreated
	eventCh <- internalaadpodid.PodCreated
	eventCh <- internalaadpodid.PodCreated
	if !evtRecorder.WaitForEvents(3) {
		t.Fatalf("Timeout waiting for mic sync cycles")
	}
	listAssignedIDs, err := crdClient.ListAssignedIDs()
	if err != nil {
		klog.Error(err)
		t.Errorf("list assigned failed")
	}
	if !(len(*listAssignedIDs) == 3) {
		t.Fatalf("expected assigned identities len: %d, got: %d", 3, len(*listAssignedIDs))
	}

	if !cloudClient.CompareMSI("testvmss1", []string{"test-user-msi-resourceid"}) {
		t.Fatalf("missing identity: %+v", cloudClient.ListMSI()["testvmss1"])
	}
	if !cloudClient.CompareMSI("testvmss2", []string{"test-user-msi-resourceid"}) {
		t.Fatalf("missing identity: %+v", cloudClient.ListMSI()["testvmss2"])
	}

	podClient.DeletePod("test-pod1", "default")
	eventCh <- internalaadpodid.PodDeleted

	if !evtRecorder.WaitForEvents(1) {
		t.Fatal("Timeout waiting for mic sync cycles")
	}
	listAssignedIDs, err = crdClient.ListAssignedIDs()
	if err != nil {
		klog.Error(err)
		t.Errorf("list assigned failed")
	}
	if !(len(*listAssignedIDs) == 2) {
		t.Fatalf("expected assigned identities len: %d, got: %d", 2, len(*listAssignedIDs))
	}

	if !cloudClient.CompareMSI("testvmss1", []string{"test-user-msi-resourceid"}) {
		t.Fatalf("missing identity: %+v", cloudClient.ListMSI()["testvmss1"])
	}
	if !cloudClient.CompareMSI("testvmss2", []string{"test-user-msi-resourceid"}) {
		t.Fatalf("missing identity: %+v", cloudClient.ListMSI()["testvmss2"])
	}

	podClient.DeletePod("test-pod2", "default")

	eventCh <- internalaadpodid.PodDeleted

	if !evtRecorder.WaitForEvents(1) {
		t.Fatal("Timeout waiting for mic sync cycles")
	}
	listAssignedIDs, err = crdClient.ListAssignedIDs()
	if err != nil {
		klog.Error(err)
		t.Errorf("list assigned failed")
	}
	if !(len(*listAssignedIDs) == 1) {
		t.Fatalf("expected assigned identities len: %d, got: %d", 1, len(*listAssignedIDs))
	}

	if !cloudClient.CompareMSI("testvmss1", []string{"test-user-msi-resourceid"}) {
		t.Fatalf("missing identity: %+v", cloudClient.ListMSI()["testvmss1"])
	}
	if !cloudClient.CompareMSI("testvmss2", []string{"test-user-msi-resourceid"}) {
		t.Fatalf("missing identity: %+v", cloudClient.ListMSI()["testvmss2"])
	}
}
