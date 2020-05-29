package azurepodidentityexception

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

// Create will create an Azure Identity Binding on a Kubernetes cluster
func Create(name, templateOutputPath string, podLabels map[string]string) error {
	t, err := template.New("aadpodidentityexception.yaml").ParseFiles(path.Join("template", "aadpodidentityexception.yaml"))
	if err != nil {
		return errors.Wrap(err, "Failed to parse aadpodidentityexception.yaml")
	}

	deployFilePath := path.Join(templateOutputPath, name+"-exception.yaml")
	deployFile, err := os.Create(deployFilePath)
	if err != nil {
		return errors.Wrap(err, "Failed to create a deployment file from aadpodidentityexception.yaml")
	}
	defer deployFile.Close()

	// Go template parameters to be translated in test/e2e/template/aadpodidentityexception.yaml
	deployData := struct {
		Name      string
		PodLabels map[string]string
	}{
		name,
		podLabels,
	}
	if err := t.Execute(deployFile, deployData); err != nil {
		return errors.Wrap(err, "Failed to create a deployment file from aadpodidentityexception.yaml")
	}

	cmd := exec.Command("kubectl", "apply", "-f", deployFilePath)
	util.PrintCommand(cmd)
	_, err = cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "Failed to deploy AzurePodIdentityException to the Kubernetes cluster")
	}

	return nil
}

// Delete will delete an AzurePodIdentityException from Kubernetes cluster
func Delete(name, templateOutputPath string) error {
	cmd := exec.Command("kubectl", "delete", "-f", path.Join(templateOutputPath, name+"-exception.yaml"), "--ignore-not-found")
	util.PrintCommand(cmd)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "Failed to delete AzurePodIdentityException from the Kubernetes cluster")
	}

	return nil
}

// GetAll will return a list of AzurePodIdentityException deployed on a Kubernetes cluster
func GetAll() (*aadpodid.AzurePodIdentityExceptionList, error) {
	cmd := exec.Command("kubectl", "get", "AzurePodIdentityException", "-ojson")
	util.PrintCommand(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get AzurePodIdentityException from the Kubernetes cluster")
	}

	list := aadpodid.AzurePodIdentityExceptionList{}
	if err := json.Unmarshal(out, &list); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshall json")
	}

	return &list, nil
}
