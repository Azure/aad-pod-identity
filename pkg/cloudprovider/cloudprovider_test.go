package cloudprovider

import (
	"flag"
	"reflect"
	"testing"

	"github.com/Azure/aad-pod-identity/pkg/config"
	"github.com/Azure/go-autorest/autorest/azure"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/golang/glog"
)

func TestParseResourceID(t *testing.T) {
	type testCase struct {
		desc   string
		testID string
		expect azure.Resource
		xErr   bool
	}

	notNested := "/subscriptions/asdf/resourceGroups/qwerty/providers/testCompute/myComputeObjectType/testComputeResource"
	nested := "/subscriptions/asdf/resourceGroups/qwerty/providers/testCompute/myComputeObjectType/testComputeResource/someNestedResource/myNestedResource"

	for _, c := range []testCase{
		{"empty string", "", azure.Resource{}, true},
		{"just a string", "asdf", azure.Resource{}, true},
		{"partial match", "/subscriptions/asdf/resourceGroups/qwery", azure.Resource{}, true},
		{"nested", nested, azure.Resource{
			SubscriptionID: "asdf",
			ResourceGroup:  "qwerty",
			Provider:       "testCompute",
			ResourceName:   "testComputeResource",
			ResourceType:   "myComputeObjectType",
		}, false},
		{"not nested", notNested, azure.Resource{
			SubscriptionID: "asdf",
			ResourceGroup:  "qwerty",
			Provider:       "testCompute",
			ResourceName:   "testComputeResource",
			ResourceType:   "myComputeObjectType",
		}, false},
	} {
		t.Run(c.desc, func(t *testing.T) {
			r, err := ParseResourceID(c.testID)
			if (err != nil) != c.xErr {
				t.Fatalf("expected err==%v, got: %v", c.xErr, err)
			}
			if !reflect.DeepEqual(r, c.expect) {
				t.Fatalf("resource does not match expected:\nexpected:\n\t%+v\ngot:\n\t%+v", c.expect, r)
			}
		})
	}
}
func TestSimple(t *testing.T) {
	flag.Set("logtostderr", "true")
	flag.Set("v", "3")
	flag.Parse()

	for _, cfg := range []config.AzureConfig{
		config.AzureConfig{},
		config.AzureConfig{VMType: "vmss"},
	} {
		desc := cfg.VMType
		if desc == "" {
			desc = "default"
		}
		t.Run(desc, func(t *testing.T) {
			cloudClient := NewTestCloudClient(cfg)

			node0 := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node0"}}
			node1 := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node1"}}
			node2 := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node2"}}

			cloudClient.AssignUserMSI("ID0", node0)
			cloudClient.AssignUserMSI("ID0", node0)
			cloudClient.AssignUserMSI("ID0again", node0)
			cloudClient.AssignUserMSI("ID1", node1)
			cloudClient.AssignUserMSI("ID2", node2)

			testMSI := []string{"ID0", "ID0again"}
			if !cloudClient.CompareMSI("node0", testMSI) {
				t.Fatal("MSI mismatch")
			}

			cloudClient.RemoveUserMSI("ID0", node0)
			cloudClient.RemoveUserMSI("ID2", node2)
			testMSI = []string{"ID0again"}
			if !cloudClient.CompareMSI(node0.Name, testMSI) {
				t.Fatal("MSI mismatch")
			}
			testMSI = []string{}
			if !cloudClient.CompareMSI(node2.Name, testMSI) {
				t.Fatal("MSI mismatch")
			}
		})
	}
}

type TestCloudClient struct {
	*Client
	// testVMClient is test validation purpose.
	testVMClient   *TestVMClient
	testVMSSClient *TestVMSSClient
}

type TestVMClient struct {
	*VMClient
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
	*VMSSClient
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
	vmClient := &VMClient{}

	return &TestVMClient{
		vmClient,
		nodeMap,
		nil,
	}
}

func NewTestVMSSClient() *TestVMSSClient {
	nodeMap := make(map[string]*compute.VirtualMachineScaleSet, 0)
	vmssClient := &VMSSClient{}

	return &TestVMSSClient{
		vmssClient,
		nodeMap,
		nil,
	}
}

func NewTestCloudClient(cfg config.AzureConfig) *TestCloudClient {
	vmClient := NewTestVMClient()
	vmssClient := NewTestVMSSClient()
	cloudClient := &Client{
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
