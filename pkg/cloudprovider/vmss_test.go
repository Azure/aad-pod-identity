package cloudprovider

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-12-01/compute"
	"github.com/stretchr/testify/assert"
)

func TestSetUserIdentitiesVMSS(t *testing.T) {
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

func TestRemoveUserIdentityVMSS(t *testing.T) {
	testIdentityInfo := &vmssIdentityInfo{
		info: &compute.VirtualMachineScaleSetIdentity{
			UserAssignedIdentities: map[string]*compute.VirtualMachineScaleSetIdentityUserAssignedIdentitiesValue{
				"ID1": nil,
				"iD2": nil,
			},
		},
	}

	// removing id1 (should be case-insensitive)
	removed := testIdentityInfo.RemoveUserIdentity("id1")
	assert.True(t, removed)
	assert.Len(t, testIdentityInfo.info.UserAssignedIdentities, 1)

	// removing id2 (should be case-insensitive)
	removed = testIdentityInfo.RemoveUserIdentity("id2")
	assert.True(t, removed)
	assert.Len(t, testIdentityInfo.info.UserAssignedIdentities, 0)

	removed = testIdentityInfo.RemoveUserIdentity("id2")
	assert.False(t, removed)
	assert.Len(t, testIdentityInfo.info.UserAssignedIdentities, 0)
}
