package cloudprovidertest

import (
	"reflect"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/golang/glog"
)

type TestCloudClient struct {
	nodeMap map[string]*[]string
}

func (c *TestCloudClient) Get(rgName string, nodeName string) (ret compute.VirtualMachine, err error) {
	vm := new(compute.VirtualMachine)
	return *vm, nil
}

func (c *TestCloudClient) ListMSI() (ret map[string]*[]string) {
	return c.nodeMap
}

func (c *TestCloudClient) CompareMSI(nodeName string, userIDs []string) bool {
	stored := c.nodeMap[nodeName]
	if stored == nil && len(userIDs) > 0 {
		return false
	}
	return reflect.DeepEqual(*stored, userIDs)

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

func (c *TestCloudClient) RemoveUserMSI(userAssignedMSIID string, nodeName string) error {
	listMSIs := c.nodeMap[nodeName]

	if listMSIs == nil {
		glog.Warningf("Could not find MSI %s on %s", userAssignedMSIID, nodeName)
		return nil
	}

	newList := make([]string, 0)
	for _, msi := range *listMSIs {
		if msi != userAssignedMSIID {
			newList = append(newList, msi)
		}
	}
	c.nodeMap[nodeName] = &newList
	return nil
}

func (c *TestCloudClient) CreateOrUpdate(rg string, nodeName string, vm compute.VirtualMachine) error {
	return nil
}

func (c *TestCloudClient) AssignUserMSI(userAssignedMSIID string, nodeName string) error {
	listMSIs := c.nodeMap[nodeName]

	if listMSIs == nil {
		// List is not allocated yet.
		array := []string{userAssignedMSIID}
		c.nodeMap[nodeName] = &array
		return nil
	}

	for _, msi := range *listMSIs {

		if userAssignedMSIID == msi {
			//We found that MSI is already present.
			//return without doing any work.
			return nil
		}
	}

	// We need to append the given user assigned msi to
	newList := append(*listMSIs, userAssignedMSIID)
	c.nodeMap[nodeName] = &newList
	return nil
}

func NewTestCloudClient() *TestCloudClient {
	nodeMap := make(map[string]*[]string, 0)
	return &TestCloudClient{
		nodeMap: nodeMap,
	}
}
