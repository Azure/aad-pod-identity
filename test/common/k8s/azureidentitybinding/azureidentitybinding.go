package azureidentitybinding

import (
	"encoding/json"
	"html/template"
	"os"
	"os/exec"
	"path"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	"github.com/Azure/aad-pod-identity/test/common/util"
	"github.com/pkg/errors"
)

// CreateOld will create an Azure Identity Binding on a Kubernetes cluster
func CreateOld(name, selector, templateOutputPath string) error {
	return CreateInternal(name, selector, "aadpodidentitybinding-old.yaml", templateOutputPath)
}

// Create will create an Azure Identity Binding on a Kubernetes cluster
func Create(name, selector, templateOutputPath string) error {
	return CreateInternal(name, selector, "aadpodidentitybinding.yaml", templateOutputPath)
}

// CreateInternal will create an Azure Identity Binding on a Kubernetes cluster
func CreateInternal(name, selector, templateInternalFile, templateOutputPath string) error {
	t, err := template.New(templateInternalFile).ParseFiles(path.Join("template", templateInternalFile))
	if err != nil {
		return errors.Wrap(err, "Failed to parse aadpodidentitybinding.yaml")
	}

	deployFilePath := path.Join(templateOutputPath, name+"-binding.yaml")
	deployFile, err := os.Create(deployFilePath)
	if err != nil {
		return errors.Wrap(err, "Failed to create a deployment file from aadpodidentitybinding.yaml")
	}
	defer deployFile.Close()

	// Go template parameters to be translated in test/e2e/template/aadpodidentitybinding.yaml
	deployData := struct {
		Name     string
		Selector string
	}{
		name,
		selector,
	}
	if err := t.Execute(deployFile, deployData); err != nil {
		return errors.Wrap(err, "Failed to create a deployment file from aadpodidentitybinding.yaml")
	}

	cmd := exec.Command("kubectl", "apply", "-f", deployFilePath)
	util.PrintCommand(cmd)
	_, err = cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "Failed to deploy AzureIdentityBinding to the Kubernetes cluster")
	}

	return nil
}

// Delete will delete an Azure Identity Binding on a Kubernetes cluster
func Delete(name, templateOutputPath string) error {
	cmd := exec.Command("kubectl", "delete", "-f", path.Join(templateOutputPath, name+"-binding.yaml"), "--ignore-not-found")
	util.PrintCommand(cmd)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "Failed to delete AzureIdentityBinding from the Kubernetes cluster")
	}

	return nil
}

// GetAll will return a list of AzureIdentityBinding deployed on a Kubernetes cluster
func GetAll() (*aadpodid.AzureIdentityBindingList, error) {
	cmd := exec.Command("kubectl", "get", "AzureIdentityBinding", "-ojson")
	util.PrintCommand(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get AzureIdentityBinding from the Kubernetes cluster")
	}

	list := aadpodid.AzureIdentityBindingList{}
	if err := json.Unmarshal(out, &list); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshall json")
	}

	return &list, nil
}
