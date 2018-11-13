package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// Identity TODO
type Identity struct {
	ClientID    string `json:"clientId"`
	PrincipalID string `json:"principalId"`
}

// CreateIdentity will create a user-assigned identity on Azure, assign 'Reader' role
// to the identity and assign 'Managed Identity Operator' role to service principal
func CreateIdentity(subscriptionID, resourceGroup, azureClientID, name string) error {
	cmd := exec.Command("az", "identity", "create", "-g", resourceGroup, "-n", name)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "Failed to create a user-assigned identity on Azure")
	}

	// Assigning 'Reader' role to the identity
	_, err = WaitOnReaderRoleAssignment(resourceGroup, name)
	if err != nil {
		return err
	}

	// Assign 'Managed Identity Operator' to Service Principal
	cmd = exec.Command("az", "role", "assignment", "create", "--role", "Managed Identity Operator", "--assignee", azureClientID, "--scope", "/subscriptions/"+subscriptionID+"/resourcegroups/"+resourceGroup+"/providers/Microsoft.ManagedIdentity/userAssignedIdentities/"+name)
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
func WaitOnReaderRoleAssignment(resourceGroup, identityName string) (bool, error) {
	principalID, err := GetIdentityPrincipalID(resourceGroup, identityName)
	if err != nil {
		return false, err
	}

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
				cmd := exec.Command("az", "role", "assignment", "create", "--role", "Reader", "--assignee-object-id", principalID, "-g", resourceGroup)
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
	// "az vm stop" does not actually cause the pod to re-schedule in another vm
	cmd := exec.Command("az", "vm", "deallocate", "-g", resourceGroup, "-n", vmName)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "Failed to stop the VM")
	}

	return nil
}

// AssignIdentityToVM will assign a user assigned identity to a VM
func AssignIdentityToVM(resourceGroup, vmName, identityName string) error {
	fmt.Printf("# Assigning %s to %s...\n", identityName, vmName)
	// "az vm stop" does not actually cause the pod to re-schedule in another vm
	cmd := exec.Command("az", "vm", "identity", "assign", "-g", resourceGroup, "-n", vmName, "--identities", identityName)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "Failed to assign identity to VM")
	}

	return nil
}

// GetUserAssignedIdentities will return the list of user assigned identity in a given VM
func GetUserAssignedIdentities(resourceGroup, vmName string) (*map[string]Identity, error) {
	cmd := exec.Command("az", "vm", "identity", "show", "-g", resourceGroup, "-n", vmName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to stop the VM")
	}

	var vmIdentities map[string]*json.RawMessage
	if err := json.Unmarshal(out, &vmIdentities); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshall json")
	}

	var userAssignedIdentities map[string]Identity
	if err := json.Unmarshal(*vmIdentities["userAssignedIdentities"], &userAssignedIdentities); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshall json")
	}

	return &userAssignedIdentities, nil
}

// RemoveIdentityFromVM will remove a user assigned identity to a VM
func RemoveIdentityFromVM(resourceGroup, vmName, identityName string) error {
	fmt.Printf("# Removing identity '%s' to VM...\n", identityName)
	// "az vm stop" does not actually cause the pod to re-schedule in another vm
	cmd := exec.Command("az", "vm", "identity", "remove", "-g", resourceGroup, "-n", vmName, "--identities", identityName)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "Failed to assign identity to VM")
	}

	return nil
}
