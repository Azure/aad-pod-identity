package azureidentity

import (
	"encoding/json"
	"html/template"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/Azure/aad-pod-identity/test/e2e/util"
)

// TODO: Add comments
type AzureIdentity struct {
	Metadata Metadata `json:"metadata"`
}

// TODO: Add comments
type Metadata struct {
	Name        string            `json:"name"`
	Annotations map[string]string `json:"annotations"`
}

// TODO: Add comments
type List struct {
	AzureIdentities []AzureIdentity `json:"items"`
}

// TODO: Add comments
func CreateOnAzure(subscriptionID, resourceGroup, azureClientID, name string) error {
	cmd := exec.Command("az", "identity", "create", "-g", resourceGroup, "-n", name)
	util.PrintCommand(cmd)
	_, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error while creating an identity on Azure:%s\n", err)
		return err
	}

	// Assign 'Reader' role to the identity
	principalID, err := GetPrincipalID(resourceGroup, name)
	if err != nil {
		return err
	}

	// Need to tight poll the following command because principalID is not
	// immediately available for role assignment after identity creation
	timeout, tick := time.After(100*time.Second), time.Tick(10*time.Second)
tightPoll:
	for {
		select {
		case <-timeout:
			log.Printf("Error while assigning 'Reader' role to the identity on Azure:%s\n", err)
			return err
		case <-tick:
			cmd := exec.Command("az", "role", "assignment", "create", "--role", "Reader", "--assignee-object-id", principalID, "-g", resourceGroup)
			_, err := cmd.CombinedOutput()
			log.Printf("Tight poll command result:%s\n", err)
			if err == nil {
				break tightPoll
			}
		}
	}

	// Assign 'Managed Identity Operator' to Service Principal
	cmd = exec.Command("az", "role", "assignment", "create", "--role", "Managed Identity Operator", "--assignee", azureClientID, "--scope", "/subscriptions/"+subscriptionID+"/resourcegroups/"+resourceGroup+"/providers/Microsoft.ManagedIdentity/userAssignedIdentities/"+name)
	util.PrintCommand(cmd)
	_, err = cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error while assigning 'Managed Identity Operator' role to service principal on Azure:%s\n", err)
		return err
	}

	return nil
}

// TODO: Add comments
func DeleteOnAzure(resourceGroup, name string) error {
	cmd := exec.Command("az", "identity", "delete", "-g", resourceGroup, "-n", name)
	util.PrintCommand(cmd)
	_, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error while deleting AzureIdentity from Azure:%s", err)
	}

	return nil
}

// TODO: Add comments
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

// TODO: Add comments
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

// TODO: Add comments
func GetClientID(resourceGroup, name string) (string, error) {
	cmd := exec.Command("az", "identity", "show", "-g", resourceGroup, "-n", name, "--query", "clientId", "-otsv")
	util.PrintCommand(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error while getting the clientID from identity:%s", err)
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

// TODO: Add comments
func GetPrincipalID(resourceGroup, name string) (string, error) {
	cmd := exec.Command("az", "identity", "show", "-g", resourceGroup, "-n", name, "--query", "principalId", "-otsv")
	util.PrintCommand(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error while getting the principalID from identity:%s", err)
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

// TODO: Add comments
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
