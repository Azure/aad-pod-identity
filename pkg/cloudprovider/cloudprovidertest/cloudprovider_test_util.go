package cloudprovidertest

import (
	"reflect"

	"github.com/Azure/aad-pod-identity/pkg/cloudprovider"
	"github.com/Azure/aad-pod-identity/pkg/config"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/golang/glog"
)

type TestCloudClient struct {
	*cloudprovider.Client
	// testVMClient is test validation purpose.
	testVMClient   *TestVMClient
	testVMSSClient *TestVMSSClient
}

type TestVMClient struct {
	*cloudprovider.VMClient
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
	*cloudprovider.VMSSClient
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
	vmClient := &cloudprovider.VMClient{}

	return &TestVMClient{
		vmClient,
		nodeMap,
		nil,
	}
}

func NewTestVMSSClient() *TestVMSSClient {
	nodeMap := make(map[string]*compute.VirtualMachineScaleSet, 0)
	vmssClient := &cloudprovider.VMSSClient{}

	return &TestVMSSClient{
		vmssClient,
		nodeMap,
		nil,
	}
}

func NewTestCloudClient(cfg config.AzureConfig) *TestCloudClient {
	vmClient := NewTestVMClient()
	vmssClient := NewTestVMSSClient()
	cloudClient := &cloudprovider.Client{
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
