package cloudprovider

import (
	"errors"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
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
	AppendUserIdentity(id string) bool
	RemoveUserIdentity(id string) error
	GetUserIdentityList() []string
}

var (
	errNotAssigned = errors.New("identity is not assigned to the resource")
	errNotFound    = errors.New("user assigned identity not found")
)

// filterUserIdentity provides a common implementation for removing an identity
// from an identity list.
func filterUserIdentity(idType *compute.ResourceIdentityType, idList *[]string, id string) error {
	switch *idType {
	case compute.ResourceIdentityTypeUserAssigned,
		compute.ResourceIdentityTypeSystemAssignedUserAssigned:
	default:
		return nil
	}

	origLen := len(*idList)
	filter(idList, id)

	if len(*idList) >= origLen {
		return errNotAssigned
	}

	if len(*idList) != 0 {
		return nil
	}

	if *idType == compute.ResourceIdentityTypeSystemAssignedUserAssigned {
		*idType = compute.ResourceIdentityTypeSystemAssigned
	} else {
		*idType = compute.ResourceIdentityTypeNone
	}

	return nil
}

func filter(ls *[]string, filter string) {
	if ls == nil {
		return
	}

	for i, v := range *ls {
		if strings.EqualFold(v, filter) {
			copy((*ls)[i:], (*ls)[i+1:])
			*ls = (*ls)[:len(*ls)-1]
			return
		}
	}
}

// appendUserIdentity provides a common implementation for adding a new identity
// to an identity object.
func appendUserIdentity(idType *compute.ResourceIdentityType, idList *[]string, newID string) bool {
	switch *idType {
	case compute.ResourceIdentityTypeUserAssigned, compute.ResourceIdentityTypeSystemAssignedUserAssigned:
		// check if this ID is already in the list
		if checkIfIDInList(*idList, newID) {
			return false
		}
	case compute.ResourceIdentityTypeSystemAssigned:
		*idType = compute.ResourceIdentityTypeSystemAssignedUserAssigned
	default:
		*idType = compute.ResourceIdentityTypeUserAssigned
	}

	*idList = append(*idList, newID)
	return true
}

func checkIfIDInList(idList []string, desiredID string) bool {
	for _, id := range idList {
		if strings.EqualFold(id, desiredID) {
			return true
		}
	}
	return false
}
