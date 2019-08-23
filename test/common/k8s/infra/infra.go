package infra

import (
	"os"
	"os/exec"
	"path"
	"text/template"

	"github.com/Azure/aad-pod-identity/test/common/util"
	"github.com/pkg/errors"
)

// CreateInfra will deploy all the infrastructure components (nmi and mic) on a Kubernetes cluster
func CreateInfra(namespace, registry, nmiVersion, micVersion, templateOutputPath string, old bool) error {
	var err error
	var t *template.Template
	if !old {
		t, err = template.New("deployment-rbac.yaml").ParseFiles(path.Join("template", "deployment-rbac.yaml"))
	} else {
		t, err = template.New("deployment-rbac-old.yaml").ParseFiles(path.Join("template", "deployment-rbac-old.yaml"))
	}
	if err != nil {
		return errors.Wrap(err, "Failed to parse deployment-rbac.yaml")
	}

	deployFilePath := path.Join(templateOutputPath, namespace+"-deployment.yaml")
	deployFile, err := os.Create(deployFilePath)
	if err != nil {
		return errors.Wrap(err, "Failed to create a deployment file")
	}
	defer deployFile.Close()

	// this arg is required only for these specific versions
	// we can remove this after next release
	var micArg, nmiArg bool
	micArg = micVersion == "1.3"
	nmiArg = nmiVersion == "1.4"

	deployData := struct {
		Namespace  string
		Registry   string
		NMIVersion string
		MICVersion string
		MICArg     bool
		NMIArg     bool
	}{
		namespace,
		registry,
		nmiVersion,
		micVersion,
		micArg,
		nmiArg,
	}
	if err := t.Execute(deployFile, deployData); err != nil {
		return errors.Wrap(err, "Failed to create a deployment file from deployment-rbac.yaml")
	}

	cmd := exec.Command("kubectl", "apply", "-f", deployFilePath)
	util.PrintCommand(cmd)
	_, err = cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "Failed to deploy Infrastructure to the Kubernetes cluster")
	}

	return nil
}

// TemplateData is the data passed to the deployment template
type IdentityValidatorTemplateData struct {
	Name                     string
	IdentityBinding          string
	Registry                 string
	IdentityValidatorVersion string
	NodeName                 string
	Replicas                 string
	DeploymentLabels         map[string]string
	InitContainer            bool
}

// CreateIdentityValidator will create an identity validator deployment on a Kubernetes cluster
func CreateIdentityValidator(subscriptionID, resourceGroup, templateOutputPath string, deployTmplData IdentityValidatorTemplateData) error {
	t, err := template.New("deployment.yaml").ParseFiles(path.Join("template", "deployment.yaml"))
	if err != nil {
		return errors.Wrap(err, "Failed to parse deployment.yaml")
	}

	deployFilePath := path.Join(templateOutputPath, deployTmplData.Name+"-deployment.yaml")
	deployFile, err := os.Create(deployFilePath)
	if err != nil {
		return errors.Wrap(err, "Failed to create a deployment file from deployment.yaml")
	}
	defer deployFile.Close()

	if err := t.Execute(deployFile, deployTmplData); err != nil {
		return errors.Wrap(err, "Failed to create a deployment file from deployment.yaml")
	}

	cmd := exec.Command("kubectl", "apply", "-f", deployFilePath)
	util.PrintCommand(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "Failed to deploy AzureIdentityBinding to the Kubernetes cluster: %s", out)
	}

	return nil
}
