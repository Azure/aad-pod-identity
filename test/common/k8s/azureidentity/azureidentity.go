package azureidentity

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	"github.com/Azure/aad-pod-identity/test/common/util"
	"github.com/pkg/errors"
)

// CreateOnClusterOld will create an Azure Identity on a Kubernetes cluster
func CreateOnClusterOld(subscriptionID, resourceGroup, name, templateOutputPath string) error {
	return CreateOnClusterInternal(subscriptionID, resourceGroup, name, "aadpodidentity-old.yaml", templateOutputPath)
}

// CreateOnCluster will create an Azure Identity on a Kubernetes cluster
func CreateOnCluster(subscriptionID, resourceGroup, name, templateOutputPath string) error {
	return CreateOnClusterInternal(subscriptionID, resourceGroup, name, "aadpodidentity.yaml", templateOutputPath)
}

// CreateOnClusterInternal will create an Azure Identity on a Kubernetes cluster
func CreateOnClusterInternal(subscriptionID, resourceGroup, name, templateInputFile, templateOutputPath string) error {
	clientID, err := GetClientID(resourceGroup, name)
	if err != nil {
		return err
	}

	t, err := template.New(templateInputFile).ParseFiles(path.Join("template", templateInputFile))
	if err != nil {
		return errors.Wrap(err, "Failed to parse aadpodidentity.yaml")
	}

	deployFilePath := path.Join(templateOutputPath, name+".yaml")
	deployFile, err := os.Create(deployFilePath)
	if err != nil {
		return errors.Wrap(err, "Failed to create a deployment file from aadpodidentity.yaml")
	}
	defer deployFile.Close()

	// Go template parameters to be translated in test/e2e/template/aadpodidentity.yaml
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
		return errors.Wrap(err, "Failed to create a deployment file from aadpodidentity.yaml")
	}

	cmd := exec.Command("kubectl", "apply", "-f", deployFilePath)
	util.PrintCommand(cmd)
	_, err = cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "Failed to deploy AzureIdentity to the Kubernetes cluster")
	}

	return nil
}

// DeleteOnCluster will delete an Azure Identity on a Kubernetes cluster
func DeleteOnCluster(name, templateOutputPath string) error {
	cmd := exec.Command("kubectl", "delete", "-f", path.Join(templateOutputPath, name+".yaml"), "--ignore-not-found")
	util.PrintCommand(cmd)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "Failed to delete AzureIdentity from the Kubernetes cluster")
	}

	return nil
}

// GetClientID will return the client id of a user-assigned identity on Azure
func GetClientID(resourceGroup, name string) (string, error) {
	cmd := exec.Command("az", "identity", "show", "-g", resourceGroup, "-n", name, "--query", "clientId", "-otsv")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrapf(err, "Failed to get the clientID of identity '%s' from resource group '%s'", name, resourceGroup)
	}

	return strings.TrimSpace(string(out)), nil
}

// GetPrincipalID will return the principal id (objecet id) of a user-assigned identity on Azure
func GetPrincipalID(resourceGroup, name string) (string, error) {
	cmd := exec.Command("az", "identity", "show", "-g", resourceGroup, "-n", name, "--query", "principalId", "-otsv")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrap(err, "Failed to get the principalID from the identity in Azure")
	}

	return strings.TrimSpace(string(out)), nil
}

// GetAll will return a list of AzureIdentity deployed on a Kubernetes cluster
func GetAll() (*aadpodid.AzureIdentityList, error) {
	cmd := exec.Command("kubectl", "get", "AzureIdentity", "-ojson")
	util.PrintCommand(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get AzureIdentity from the Kubernetes cluster")
	}

	list := aadpodid.AzureIdentityList{}
	if err := json.Unmarshal(out, &list); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshall json")
	}

	return &list, nil
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
