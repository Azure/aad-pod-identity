package azure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/pkg/errors"
)

type Resource = azure.Resource

// UserAssignedIdentity is used to parse user assigned identity data from 'az vm identity show'
type UserAssignedIdentity struct {
	ClientID    string `json:"clientId"`
	PrincipalID string `json:"principalId"`
}

// VMIdentity is used to parse system assigned identity data from 'az vm identity show'
type VMIdentity struct {
	PrincipalID            string           `json:"principalId"`
	TenantID               string           `json:"tenantId"`
	Type                   string           `json:"type"`
	UserAssignedIdentities *json.RawMessage `json:"userAssignedIdentities"`
}

// CreateIdentity will create a user-assigned identity on Azure, assign 'Reader' role
// to the identity and assign 'Managed Identity Operator' role to service principal
func CreateIdentity(subscriptionID, resourceGroup, azureClientID, identityName, keyvaultName string) error {
	fmt.Printf("# Creating user assigned identity on Azure: %s\n", identityName)
	cmd := exec.Command("az", "identity", "create", "-g", resourceGroup, "-n", identityName)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "Failed to create a user-assigned identity on Azure")
	}

	// Assigning 'Reader' role on keyvault to the identity
	_, err = WaitOnReaderRoleAssignment(subscriptionID, resourceGroup, identityName, keyvaultName)
	if err != nil {
		return err
	}

	// Grant identity access to keyvault secret
	identityClientID, err := GetIdentityClientID(resourceGroup, identityName)
	if err != nil {
		return err
	}

	cmd = exec.Command("az", "keyvault", "set-policy", "-n", keyvaultName, "--secret-permissions", "get", "list", "--spn", identityClientID)
	_, err = cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "Failed to grant identity %s access to keyvault secret", identityName)
	}

	// Assign 'Managed Identity Operator' to Service Principal
	identityResourceID := fmt.Sprintf("/subscriptions/%s/resourcegroups/%s/providers/Microsoft.ManagedIdentity/userAssignedIdentities/%s", subscriptionID, resourceGroup, identityName)
	cmd = exec.Command("az", "role", "assignment", "create", "--role", "Managed Identity Operator", "--assignee", azureClientID, "--scope", identityResourceID)
	_, err = cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "Failed to assign 'Managed Identity Operator' role to service principal on Azure")
	}

	return nil
}

// DeleteIdentity will delete a given user-assigned identity on Azure
func DeleteIdentity(resourceGroup, identityName string) error {
	cmd := exec.Command("az", "identity", "delete", "-g", resourceGroup, "-n", identityName)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "Failed to delete the Azure identity from Azure")
	}

	return nil
}

// GetIdentityClientID will return the client id of a user-assigned identity on Azure
func GetIdentityClientID(resourceGroup, identityName string) (string, error) {
	cmd := exec.Command("az", "identity", "show", "-g", resourceGroup, "-n", identityName, "--query", "clientId", "-otsv")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrap(err, "Failed to get the clientID from the identity in Azure")
	}

	return strings.TrimSpace(string(out)), nil
}

// GetIdentityPrincipalID will return the principal id (objecet id) of a user-assigned identity on Azure
func GetIdentityPrincipalID(resourceGroup, identityName string) (string, error) {
	cmd := exec.Command("az", "identity", "show", "-g", resourceGroup, "-n", identityName, "--query", "principalId", "-otsv")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrap(err, "Failed to get the principalID from the identity in Azure")
	}

	return strings.TrimSpace(string(out)), nil
}

// WaitOnReaderRoleAssignment will block until the assignement of 'Reader' role to an identity is executed successfully
func WaitOnReaderRoleAssignment(subscriptionID, resourceGroup, identityName, keyvaultName string) (bool, error) {
	principalID, err := GetIdentityPrincipalID(resourceGroup, identityName)
	if err != nil {
		return false, err
	}
	keyvaultResource := fmt.Sprintf("/subscriptions/%s/resourceGroups/aad-pod-identity-e2e/providers/Microsoft.KeyVault/vaults/%s", subscriptionID, keyvaultName)

	// Need to tight poll the following command because principalID is not
	// immediately available for role assignment after identity creation
	readyChannel, errorChannel := make(chan bool, 1), make(chan error)
	duration, sleep := 100*time.Second, 10*time.Second
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	go func() {
		for {
			select {
			case <-ctx.Done():
				errorChannel <- errors.Errorf("Timeout exceeded (%s) while assigning 'Reader' role to identity on Azure", duration.String())
			default:
				cmd := exec.Command("az", "role", "assignment", "create", "--role", "Reader", "--assignee", principalID, "--scope", keyvaultResource)
				_, err := cmd.CombinedOutput()
				if err == nil {
					readyChannel <- true
					return
				}
				fmt.Printf("# Reader role has not been assigned to the Azure identity. Retrying in %s...\n", sleep.String())
				time.Sleep(sleep)
			}
		}
	}()

	for {
		select {
		case err := <-errorChannel:
			return false, err
		case ready := <-readyChannel:
			return ready, nil
		}
	}
}

// StartVM will start a stopped VM
func StartVM(resourceGroup, vmName string) error {
	fmt.Printf("# Starting a VM...\n")
	cmd := exec.Command("az", "vm", "start", "-g", resourceGroup, "-n", vmName)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "Failed to start the VM")
	}

	return nil
}

// StopVM will stop a running VM
func StopVM(resourceGroup, vmName string) error {
	fmt.Printf("# Stopping a VM...\n")
	cmd := exec.Command("az", "vm", "stop", "-g", resourceGroup, "-n", vmName)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "Failed to stop the VM")
	}

	return nil
}

// StartKubelet will start the kubelet on a given VM
func StartKubelet(resourceGroup, vmName string) error {
	fmt.Printf("# Starting kubelet on %s...\n", vmName)
	cmd := exec.Command("az", "vm", "run-command", "invoke", "-g", resourceGroup, "-n", vmName, "--command-id", "RunShellScript", "--scripts", "sudo systemctl start kubelet && sudo systemctl daemon-reload")
	_, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "Failed to start kubelet on the VM")
	}

	return nil
}

// StopKubelet will stop the kubelet on a given VM
func StopKubelet(resourceGroup, vmName string) error {
	fmt.Printf("# Stopping kubelet on %s...\n", vmName)
	cmd := exec.Command("az", "vm", "run-command", "invoke", "-g", resourceGroup, "-n", vmName, "--command-id", "RunShellScript", "--scripts", "sudo systemctl stop kubelet")
	_, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "Failed to stop kubelet on the VM")
	}

	return nil
}

// EnableUserAssignedIdentityOnVM will enable a user assigned identity to a VM
func EnableUserAssignedIdentityOnVM(resourceGroup, vmName, identityName string) error {
	fmt.Printf("# Assigning user assigned identity '%s' to %s...\n", identityName, vmName)
	cmd := exec.Command("az", "vm", "identity", "assign", "-g", resourceGroup, "-n", vmName, "--identities", identityName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "Failed to assign user assigned identity to VM: %s", string(out))
	}

	return nil
}

// EnableUserAssignedIdentityOnVMSS will enable a user assigned identity to a VM
func EnableUserAssignedIdentityOnVMSS(resourceGroup, vmName, identityName string) error {
	fmt.Printf("# Assigning user assigned identity '%s' to %s...\n", identityName, vmName)
	cmd := exec.Command("az", "vmss", "identity", "assign", "-g", resourceGroup, "-n", vmName, "--identities", identityName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "Failed to assign user assigned identity to VM: %s", string(out))
	}

	return nil
}

// EnableSystemAssignedIdentityOnVM will enable a system assigned identity to a VM
func EnableSystemAssignedIdentityOnVM(resourceGroup, vmName string) error {
	fmt.Printf("# Assigning system assigned identity to %s...\n", vmName)
	cmd := exec.Command("az", "vm", "identity", "assign", "-g", resourceGroup, "-n", vmName)
	cmdOutput, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("%s\n", cmdOutput)
		return errors.Wrap(err, "Failed to enable identity to VM")
	}

	return nil
}

// EnableSystemAssignedIdentityOnVMSS will enable a system assigned identity to a VM
func EnableSystemAssignedIdentityOnVMSS(resourceGroup, vmName string) error {
	fmt.Printf("# Assigning system assigned identity to %s...\n", vmName)
	cmd := exec.Command("az", "vmss", "identity", "assign", "-g", resourceGroup, "-n", vmName)
	cmdOutput, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("%s\n", cmdOutput)
		return errors.Wrap(err, "Failed to enable identity to VM")
	}

	return nil
}

func UserIdentityAssignedToVMSS(resourceGroup, vmssName, identityName string) (bool, error) {
	cmd := exec.Command("az", "vmss", "identity", "show", "-o", "json", "-g", resourceGroup, "-n", vmssName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, errors.Wrapf(err, "Failed to get user assigned identity from VMSS %q: %s", vmssName, out)
	}

	out = bytes.TrimSpace(out)
	if len(out) == 0 {
		return false, nil
	}

	var id VMIdentity
	if err := json.Unmarshal(out, &id); err != nil {
		return false, errors.Wrap(err, "error unmarshaling json")
	}
	if id.Type == "SystemAssigned" {
		return false, nil
	}

	var userAssignedIdentities map[string]UserAssignedIdentity
	if err := json.Unmarshal(*id.UserAssignedIdentities, &userAssignedIdentities); err != nil {
		return false, errors.Wrap(err, "failed to unmarshall json for user assigned identities")
	}

	for rID := range userAssignedIdentities {
		s := strings.Split(rID, "/")
		if s[len(s)-1] == identityName {
			return true, nil
		}
	}

	return false, nil
}

// GetVMUserAssignedIdentities will return the list of user assigned identity in a given VM
func GetVMUserAssignedIdentities(resourceGroup, vmName string) (map[string]UserAssignedIdentity, error) {
	// Sleep for 30 seconds to allow potential changes to propagate to Azure
	time.Sleep(time.Second * 30)

	cmd := exec.Command("az", "vm", "identity", "show", "-g", resourceGroup, "-n", vmName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get user assigned identity from VM: %s", string(out))
	}

	var vmIdentity VMIdentity
	var userAssignedIdentities map[string]UserAssignedIdentity

	// Return an empty userAssignedIdentities if out slice is empty
	if len(out) == 0 {
		return userAssignedIdentities, nil
	} else if err := json.Unmarshal(out, &vmIdentity); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshall json")
	}

	if vmIdentity.Type == "SystemAssigned" {
		return userAssignedIdentities, nil
	} else if err := json.Unmarshal(*vmIdentity.UserAssignedIdentities, userAssignedIdentities); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshall json")
	}

	return userAssignedIdentities, nil
}

// GetVMSSUserAssignedIdentities will return the list of user assigned identity in a given VM
func GetVMSSUserAssignedIdentities(resourceGroup, name string) (map[string]UserAssignedIdentity, error) {
	// Sleep for 30 seconds to allow potential changes to propagate to Azure
	time.Sleep(time.Second * 30)

	cmd := exec.Command("az", "vmss", "identity", "show", "-g", resourceGroup, "-n", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get user assigned identity from VMSS: %s", string(out))
	}

	var vmIdentity VMIdentity
	var userAssignedIdentities map[string]UserAssignedIdentity

	// Return an empty userAssignedIdentities if out slice is empty
	if len(out) == 0 {
		return userAssignedIdentities, nil
	} else if err := json.Unmarshal(out, &vmIdentity); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshall json")
	}

	if vmIdentity.Type == "SystemAssigned" {
		return userAssignedIdentities, nil
	} else if err := json.Unmarshal(*vmIdentity.UserAssignedIdentities, &userAssignedIdentities); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshall json")
	}

	return userAssignedIdentities, nil
}

// RemoveUserAssignedIdentityFromVM will remove a user assigned identity to a VM
func RemoveUserAssignedIdentityFromVM(resourceGroup, vmName, identityName string) error {
	fmt.Printf("# Removing identity '%s' from %s...\n", identityName, vmName)
	cmd := exec.Command("az", "vm", "identity", "remove", "-g", resourceGroup, "-n", vmName, "--identities", identityName)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "Failed to remove user assigned identity to VM")
	}

	return nil
}

// RemoveUserAssignedIdentityFromVMSS will remove a user assigned identity to a VMSS
func RemoveUserAssignedIdentityFromVMSS(resourceGroup, vmName, identityName string) error {
	fmt.Printf("# Removing identity '%s' from %s...\n", identityName, vmName)
	cmd := exec.Command("az", "vmss", "identity", "remove", "-g", resourceGroup, "-n", vmName, "--identities", identityName)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "Failed to remove user assigned identity to VMSS")
	}

	return nil
}

// GetVMSystemAssignedIdentity will return the principal ID and tenant ID of a system assigned identity
func GetVMSystemAssignedIdentity(resourceGroup, vmName string) (string, string, error) {
	cmd := exec.Command("az", "vm", "identity", "show", "-g", resourceGroup, "-n", vmName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", "", errors.Wrap(err, "Failed to get system assigned identity from VM")
	}

	var systemAssignedIdentity VMIdentity
	if err := json.Unmarshal(out, &systemAssignedIdentity); err != nil {
		return "", "", errors.Wrap(err, "Failed to unmarshall json")
	}

	if strings.Contains(systemAssignedIdentity.Type, "SystemAssigned") {
		return systemAssignedIdentity.PrincipalID, systemAssignedIdentity.TenantID, nil
	}

	return "", "", nil
}

// GetVMSSSystemAssignedIdentity will return the principal ID and tenant ID of a system assigned identity
func GetVMSSSystemAssignedIdentity(resourceGroup, name string) (string, string, error) {
	cmd := exec.Command("az", "vmss", "identity", "show", "-g", resourceGroup, "-n", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", "", errors.Wrap(err, "Failed to get system assigned identity from VMSS")
	}

	var systemAssignedIdentity VMIdentity
	if err := json.Unmarshal(out, &systemAssignedIdentity); err != nil {
		return "", "", errors.Wrap(err, "Failed to unmarshall json")
	}

	if strings.Contains(systemAssignedIdentity.Type, "SystemAssigned") {
		return systemAssignedIdentity.PrincipalID, systemAssignedIdentity.TenantID, nil
	}

	return "", "", nil
}

// RemoveSystemAssignedIdentityFromVM will remove the system assigned identity to a VM
func RemoveSystemAssignedIdentityFromVM(resourceGroup, vmName string) error {
	fmt.Printf("# Removing system assigned identity from %s...\n", vmName)
	cmd := exec.Command("az", "vm", "identity", "remove", "-g", resourceGroup, "-n", vmName)
	cmdOutput, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("%s\n", cmdOutput)
		return errors.Wrap(err, "Failed to remove system assigned identity to VM")
	}

	return nil
}

// RemoveSystemAssignedIdentityFromVMSS will remove the system assigned identity to a VMSS
func RemoveSystemAssignedIdentityFromVMSS(resourceGroup, name string) error {
	fmt.Printf("# Removing system assigned identity from %s...\n", name)
	cmd := exec.Command("az", "vmss", "identity", "remove", "-g", resourceGroup, "-n", name)
	cmdOutput, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("%s\n", cmdOutput)
		return errors.Wrap(err, "Failed to remove system assigned identity to VMSS")
	}

	return nil
}
