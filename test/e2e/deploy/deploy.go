package deploy

import (
	"html/template"
	"os"
	"os/exec"
	"path"

	"github.com/Azure/aad-pod-identity/test/e2e/azureidentity"
	"github.com/Azure/aad-pod-identity/test/e2e/util"
)

func Create(subscriptionID, resourceGroup, name, identityBinding, templateOutputPath string) error {
	clientID, err := azureidentity.GetClientID(resourceGroup, identityBinding)
	if err != nil {
		return err
	}

	t, err := template.New("deployment.yaml").ParseFiles(path.Join("template", "deployment.yaml"))
	if err != nil {
		return err
	}

	deployFilePath := path.Join(templateOutputPath, name+"-deployment.yaml")
	deployFile, err := os.Create(deployFilePath)
	if err != nil {
		return err
	}
	defer deployFile.Close()

	deployData := struct {
		SubscriptionID  string
		ResourceGroup   string
		ClientID        string
		Name            string
		IdentityBinding string
	}{
		subscriptionID,
		resourceGroup,
		clientID,
		name,
		identityBinding,
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

func Delete(name, templateOutputPath string) error {
	cmd := exec.Command("kubectl", "delete", "-f", path.Join(templateOutputPath, name+"-deployment.yaml"), "--ignore-not-found")
	util.PrintCommand(cmd)
	_, err := cmd.CombinedOutput()
	return err
}
