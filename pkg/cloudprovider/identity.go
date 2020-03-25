package cloudprovider

import (
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-12-01/compute"
)

// IdentityHolder represents a resource that contains an Identity object
// This is used to be able to generically intract with multiple resource types (e.g. VirtualMachine and VirtualMachineScaleSet)
// which each contain an identity.
type IdentityHolder interface {
	IdentityInfo() IdentityInfo
	ResetIdentity() IdentityInfo
}

// IdentityInfo is used to interact with different implementations of Azure compute identities.
// This is needed because different Azure resource types (e.g. VirtualMachine and VirtualMachineScaleSet)
// have different identity types.
// This abstracts those differences.
type IdentityInfo interface {
	GetUserIdentityList() []string
	SetUserIdentities(map[string]bool) bool
}

func checkIfIDInList(idList []string, desiredID string) bool {
	for _, id := range idList {
		if strings.EqualFold(id, desiredID) {
			return true
		}
	}
	return false
}

// getResourceIdentityType returns resource identity type based on current type
func getResourceIdentityType(identityType compute.ResourceIdentityType) compute.ResourceIdentityType {
	switch identityType {
	case "", compute.ResourceIdentityTypeNone:
		return compute.ResourceIdentityTypeUserAssigned
	case compute.ResourceIdentityTypeSystemAssigned:
		return compute.ResourceIdentityTypeSystemAssignedUserAssigned
	default:
		return compute.ResourceIdentityTypeNone
	}
}
