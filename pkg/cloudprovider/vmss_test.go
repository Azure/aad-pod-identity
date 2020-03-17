package cloudprovider

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-12-01/compute"
	"github.com/stretchr/testify/assert"
)

func TestSetUserIdentities(t *testing.T) {
	testIdentityInfo := &vmssIdentityInfo{
		info: &compute.VirtualMachineScaleSetIdentity{},
	}

	// adding id1
	update := testIdentityInfo.SetUserIdentities(map[string]bool{"id1": true})
	assert.True(t, update)
	// adding id2
	update = testIdentityInfo.SetUserIdentities(map[string]bool{"id2": true})
	assert.True(t, update)
	// add id3 and delete id1
	update = testIdentityInfo.SetUserIdentities(map[string]bool{"id3": true, "id4": true, "id1": false})
	assert.True(t, update)
}
