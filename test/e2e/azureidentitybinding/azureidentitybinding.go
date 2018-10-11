package azureidentitybinding

import (
	"encoding/json"
	"html/template"
	"log"
	"os"
	"os/exec"
	"path"

	"github.com/Azure/aad-pod-identity/test/e2e/util"
)

// AzureIdentityBinding is used to parse data from 'kubectl get AzureIdentityBinding'
type AzureIdentityBinding struct {
	Metadata Metadata `json:"metadata"`
}

// Metadata holds information about AzureIdentityBinding
type Metadata struct {
	Name        string            `json:"name"`
	Annotations map[string]string `json:"annotations"`
}

// List is a container that holds all AzureIdentityBindings returned from 'kubectl get AzureIdentityBinding'
type List struct {
	AzureIdentityBindings []AzureIdentityBinding `json:"items"`
}

// Create will create an Azure Identity Binding on a Kubernetes cluster
func Create(name, templateOutputPath string) error {
	t, err := template.New("aadpodidentitybinding.yaml").ParseFiles(path.Join("template", "aadpodidentitybinding.yaml"))
	if err != nil {
		return err
	}

	deployFilePath := path.Join(templateOutputPath, name+"-binding.yaml")
	deployFile, err := os.Create(deployFilePath)
	if err != nil {
		return err
	}
	defer deployFile.Close()

	deployData := struct {
		Name string
	}{
		name,
	}
	if err := t.Execute(deployFile, deployData); err != nil {
		return err
	}

	cmd := exec.Command("kubectl", "apply", "-f", deployFilePath)
	util.PrintCommand(cmd)
	_, err = cmd.CombinedOutput()
	if err != nil {
		return err
	}

	return nil
}

// Delete will delete an Azure Identity Binding on a Kubernetes cluster
func Delete(name, templateOutputPath string) error {
	cmd := exec.Command("kubectl", "delete", "-f", path.Join(templateOutputPath, name+"-binding.yaml"), "--ignore-not-found")
	util.PrintCommand(cmd)
	_, err := cmd.CombinedOutput()
	return err
}

// GetAll will return a list of AzureIdentityBinding deployed on a Kubernetes cluster
func GetAll() (*List, error) {
	cmd := exec.Command("kubectl", "get", "AzureIdentityBinding", "-ojson")
	util.PrintCommand(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error trying to run 'kubectl get AzureIdentityBinding':%s", string(out))
		return nil, err
	}

	nl := List{}
	err = json.Unmarshal(out, &nl)
	if err != nil {
		log.Printf("Error unmarshalling nodes json:%s", err)
	}

	return &nl, nil
}
