package azureidentity

import (
	"context"
	"encoding/json"
	"html/template"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/Azure/aad-pod-identity/test/e2e/util"
	"github.com/pkg/errors"
)

// AzureIdentity is used to parse data from 'kubectl get AzureIdentity'
type AzureIdentity struct {
	Metadata Metadata `json:"metadata"`
}

// Metadata holds information about AzureIdentity
type Metadata struct {
	Name        string            `json:"name"`
	Annotations map[string]string `json:"annotations"`
}

// List is a container that holds all AzureIdentities returned from 'kubectl get AzureIdentity'
type List struct {
	AzureIdentities []AzureIdentity `json:"items"`
}

// CreateOnAzure will create an user-assigned identity on Azure, assign 'Reader' role
// to the identity and assign 'Managed Identity Operator' role to service principal
func CreateOnAzure(subscriptionID, resourceGroup, azureClientID, name string) error {
	cmd := exec.Command("az", "identity", "create", "-g", resourceGroup, "-n", name)
	_, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error while creating an identity on Azure:%s\n", err)
		return err
	}

	// Assigning 'Reader' role to the identity
	ok, err := WaitOnReaderRoleAssignment(resourceGroup, name)
	if !ok && err != nil {
		log.Printf("Error while assigning 'Reader' role to identity on Azure:%s\n", err)
		return err
	}

	// Assign 'Managed Identity Operator' to Service Principal
	cmd = exec.Command("az", "role", "assignment", "create", "--role", "Managed Identity Operator", "--assignee", azureClientID, "--scope", "/subscriptions/"+subscriptionID+"/resourcegroups/"+resourceGroup+"/providers/Microsoft.ManagedIdentity/userAssignedIdentities/"+name)
	_, err = cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error while assigning 'Managed Identity Operator' role to service principal on Azure:%s\n", err)
		return err
	}

	return nil
}

// DeleteOnAzure will delete a given user-assigned identity on Azure
func DeleteOnAzure(resourceGroup, name string) error {
	cmd := exec.Command("az", "identity", "delete", "-g", resourceGroup, "-n", name)
	_, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error while deleting AzureIdentity from Azure:%s", err)
	}

	return nil
}

// CreateOnCluster will create an Azure Identity on a Kubernetes cluster
func CreateOnCluster(subscriptionID, resourceGroup, name, templateOutputPath string) error {
	clientID, err := GetClientID(resourceGroup, name)
	if err != nil {
		log.Printf("Error while getting clientID from on Azure:%s\n", err)
		return err
	}

	t, err := template.New("aadpodidentity.yaml").ParseFiles(path.Join("template", "aadpodidentity.yaml"))
	if err != nil {
		return err
	}

	deployFilePath := path.Join(templateOutputPath, name+".yaml")
	deployFile, err := os.Create(deployFilePath)
	if err != nil {
		return err
	}
	defer deployFile.Close()

	deployData := struct {
		SubscriptionID string
		ResourceGroup  string
		ClientID       string
		Name           string
	}{
		subscriptionID,
		resourceGroup,
		clientID,
		name,
	}
	if err := t.Execute(deployFile, deployData); err != nil {
		return err
	}

	cmd := exec.Command("kubectl", "apply", "-f", deployFilePath)
	util.PrintCommand(cmd)
	_, err = cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error while deploying AzureIdentity to k8s cluster:%s", err)
		return err
	}

	return nil
}

// DeleteOnCluster will delete an Azure Identity on a Kubernetes cluster
func DeleteOnCluster(name, templateOutputPath string) error {
	cmd := exec.Command("kubectl", "delete", "-f", path.Join(templateOutputPath, name+".yaml"), "--ignore-not-found")
	util.PrintCommand(cmd)
	_, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error while deleting AzureIdentity from k8s cluster:%s", err)
		return err
	}

	return nil
}

// GetClientID will return the client id of a user-assigned identity on Azure
func GetClientID(resourceGroup, name string) (string, error) {
	cmd := exec.Command("az", "identity", "show", "-g", resourceGroup, "-n", name, "--query", "clientId", "-otsv")
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error while getting the clientID from identity:%s", err)
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

// GetPrincipalID will return the principal id (objecet id) of a user-assigned identity on Azure
func GetPrincipalID(resourceGroup, name string) (string, error) {
	cmd := exec.Command("az", "identity", "show", "-g", resourceGroup, "-n", name, "--query", "principalId", "-otsv")
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error while getting the principalID from identity:%s", err)
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

// GetAll will return a list of AzureIdentity deployed on a Kubernetes cluster
func GetAll() (*List, error) {
	cmd := exec.Command("kubectl", "get", "AzureIdentity", "-ojson")
	util.PrintCommand(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error while trying to run 'kubectl get AzureIdentity':%s", err)
		return nil, err
	}

	nl := List{}
	err = json.Unmarshal(out, &nl)
	if err != nil {
		log.Printf("Error unmarshalling nodes json:%s", err)
		return nil, err
	}

	return &nl, nil
}

// WaitOnReaderRoleAssignment will block until the assignement of 'Reader' role to an identity is executed successfully
func WaitOnReaderRoleAssignment(resourceGroup, name string) (bool, error) {
	principalID, err := GetPrincipalID(resourceGroup, name)
	if err != nil {
		return false, err
	}

	// Need to tight poll the following command because principalID is not
	// immediately available for role assignment after identity creation
	readyChannel, errorChannel := make(chan bool, 1), make(chan error)
	duration := 100 * time.Second
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
				log.Printf("Tight poll command result: %s\n", err)
				if err == nil {
					readyChannel <- true
				}
				time.Sleep(10 * time.Second)
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
