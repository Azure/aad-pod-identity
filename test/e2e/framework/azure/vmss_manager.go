// +build e2e

package azure

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/aad-pod-identity/test/e2e/framework"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-12-01/compute"
	. "github.com/onsi/ginkgo"
)

type vmssManager struct {
	config     *framework.Config
	vmssClient compute.VirtualMachineScaleSetsClient
}

func newVMSSManager(config *framework.Config, vmssClient compute.VirtualMachineScaleSetsClient) nodeManager {
	return &vmssManager{
		config:     config,
		vmssClient: vmssClient,
	}
}

// ListUserAssignedIdentities returns a list of user-assigned identities assigned to the node.
func (m *vmssManager) ListUserAssignedIdentities(vmssName string) map[string]bool {
	userAssignedIdentities := make(map[string]bool)
	vmss, err := m.vmssClient.Get(context.TODO(), m.config.ClusterResourceGroup, vmssName)
	if err != nil || vmss.Identity == nil {
		return userAssignedIdentities
	}

	for id := range vmss.Identity.UserAssignedIdentities {
		userAssignedIdentities[strings.ToLower(id)] = true
	}

	return userAssignedIdentities
}

// AssignUserAssignedIdentity assigns a user-assigned identity to a node.
func (m *vmssManager) AssignUserAssignedIdentity(vmssName, identityToAssign string) error {
	vmss, err := m.vmssClient.Get(context.TODO(), m.config.ClusterResourceGroup, vmssName)
	if err != nil {
		return err
	}

	if vmss.Identity == nil {
		vmss.Identity = &compute.VirtualMachineScaleSetIdentity{
			UserAssignedIdentities: map[string]*compute.VirtualMachineScaleSetIdentityUserAssignedIdentitiesValue{},
		}
	}
	if vmss.Identity.UserAssignedIdentities == nil {
		vmss.Identity.UserAssignedIdentities = make(map[string]*compute.VirtualMachineScaleSetIdentityUserAssignedIdentitiesValue)
	}

	identityAssignResourceID := fmt.Sprintf(ResourceIDTemplate, m.config.SubscriptionID, m.config.IdentityResourceGroup, identityToAssign)
	for identity := range vmss.Identity.UserAssignedIdentities {
		// identity already exists and doesn't need to be re-assigned
		if strings.EqualFold(identity, identityAssignResourceID) {
			return nil
		}
	}

	vmss.Identity.UserAssignedIdentities[identityAssignResourceID] = &compute.VirtualMachineScaleSetIdentityUserAssignedIdentitiesValue{}
	switch vmss.Identity.Type {
	case compute.ResourceIdentityTypeSystemAssigned:
		vmss.Identity.Type = compute.ResourceIdentityTypeSystemAssignedUserAssigned
	default:
		vmss.Identity.Type = compute.ResourceIdentityTypeUserAssigned
	}

	By(fmt.Sprintf("Assigning \"%s\" to \"%s\"", identityToAssign, vmssName))
	return m.updateIdentity(vmssName, vmss.Identity)
}

// UnassignUserAssignedIdentity un-assigns a user-assigned identity from a node.
func (m *vmssManager) UnassignUserAssignedIdentity(vmssName, identityToUnassign string) error {
	vmss, err := m.vmssClient.Get(context.TODO(), m.config.ClusterResourceGroup, vmssName)
	if err != nil {
		return err
	}

	if vmss.Identity == nil || len(vmss.Identity.UserAssignedIdentities) == 0 {
		return nil
	}

	for identity := range vmss.Identity.UserAssignedIdentities {
		if s := strings.Split(identity, "/"); strings.EqualFold(s[len(s)-1], identityToUnassign) {
			By(fmt.Sprintf("Un-assigning \"%s\" from \"%s\"", identityToUnassign, vmssName))
			delete(vmss.Identity.UserAssignedIdentities, identity)
			if len(vmss.Identity.UserAssignedIdentities) == 0 {
				vmss.Identity.UserAssignedIdentities = nil
				switch vmss.Identity.Type {
				case compute.ResourceIdentityTypeSystemAssignedUserAssigned:
					vmss.Identity.Type = compute.ResourceIdentityTypeSystemAssigned
				default:
					vmss.Identity.Type = compute.ResourceIdentityTypeNone
				}
			}
			break
		}
	}

	return m.updateIdentity(vmssName, vmss.Identity)
}

// EnableSystemAssignedIdentity enables system-assigned identity for a node.
func (m *vmssManager) EnableSystemAssignedIdentity(vmssName string) error {
	vmss, err := m.vmssClient.Get(context.TODO(), m.config.ClusterResourceGroup, vmssName)
	if err != nil {
		return err
	}

	if vmss.Identity == nil {
		vmss.Identity = &compute.VirtualMachineScaleSetIdentity{
			Type: compute.ResourceIdentityTypeSystemAssigned,
		}
	} else {
		switch vmss.Identity.Type {
		case compute.ResourceIdentityTypeUserAssigned:
			vmss.Identity.Type = compute.ResourceIdentityTypeSystemAssignedUserAssigned
		default:
			vmss.Identity.Type = compute.ResourceIdentityTypeSystemAssigned
		}
	}

	By(fmt.Sprintf("Enabling system-assigned identity for %s", vmssName))
	return m.updateIdentity(vmssName, vmss.Identity)
}

// DisableSystemAssignedIdentity disables system-assigned identity for a node.
func (m *vmssManager) DisableSystemAssignedIdentity(vmssName string) error {
	vmss, err := m.vmssClient.Get(context.TODO(), m.config.ClusterResourceGroup, vmssName)
	if err != nil {
		return err
	}

	if vmss.Identity == nil {
		return nil
	}

	switch vmss.Identity.Type {
	case compute.ResourceIdentityTypeSystemAssignedUserAssigned:
		vmss.Identity.Type = compute.ResourceIdentityTypeUserAssigned
	default:
		vmss.Identity.Type = compute.ResourceIdentityTypeNone
	}

	By(fmt.Sprintf("Disabling system-assigned identity for %s", vmssName))
	return m.updateIdentity(vmssName, vmss.Identity)
}

// GetSystemAssignedIdentityInfo returns the principal ID and tenant ID of the system-assigned identity.
func (m *vmssManager) GetSystemAssignedIdentityInfo(vmssName string) (string, string) {
	vmss, err := m.vmssClient.Get(context.TODO(), m.config.ClusterResourceGroup, vmssName)
	if err != nil {
		return "", ""
	}

	if vmss.Identity == nil || (vmss.Identity.Type != compute.ResourceIdentityTypeSystemAssigned && vmss.Identity.Type != compute.ResourceIdentityTypeSystemAssignedUserAssigned) {
		return "", ""
	}

	return *vmss.Identity.PrincipalID, *vmss.Identity.TenantID
}

func (m *vmssManager) updateIdentity(vmssName string, vmssIdentity *compute.VirtualMachineScaleSetIdentity) error {
	if future, err := m.vmssClient.Update(context.TODO(), m.config.ClusterResourceGroup, vmssName, compute.VirtualMachineScaleSetUpdate{Identity: vmssIdentity}); err != nil {
		return err
	} else {
		if err := future.WaitForCompletionRef(context.TODO(), m.vmssClient.Client); err != nil {
			return err
		}
	}

	return nil
}
