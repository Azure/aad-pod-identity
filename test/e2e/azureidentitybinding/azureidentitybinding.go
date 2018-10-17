package azureidentitybinding

import (
	"encoding/json"
	"html/template"
	"os"
	"os/exec"
	"path"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	"github.com/Azure/aad-pod-identity/test/e2e/util"
	"github.com/pkg/errors"
)

// Create will create an Azure Identity Binding on a Kubernetes cluster
func Create(name, templateOutputPath string) error {
	t, err := template.New("aadpodidentitybinding.yaml").ParseFiles(path.Join("template", "aadpodidentitybinding.yaml"))
	if err != nil {
		return errors.Wrap(err, "Failed to parse aadpodidentitybinding.yaml")
	}

	deployFilePath := path.Join(templateOutputPath, name+"-binding.yaml")
	deployFile, err := os.Create(deployFilePath)
	if err != nil {
		return errors.Wrap(err, "Failed to create a deployment file from aadpodidentitybinding.yaml")
	}
	defer deployFile.Close()

	deployData := struct {
		Name string
	}{
		name,
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
	err = json.Unmarshal(out, &list)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshall json")
	}

	return &list, nil
}
