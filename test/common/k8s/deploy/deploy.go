package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/Azure/aad-pod-identity/test/common/util"
	"github.com/pkg/errors"
)

// List is a container that holds all deployment returned from 'kubectl get deploy'
type List struct {
	Deploys []Deploy `json:"items"`
}

// Deploy is used to parse data from 'kubectl get deploy'
type Deploy struct {
	Metadata Metadata `json:"metadata"`
	Spec     Spec     `json:"spec"`
	Status   Status   `json:"status"`
}

// Metadata holds information about a deployment
type Metadata struct {
	Name string `json:"name"`
}

// Spec holds the spec about a deployment
type Spec struct {
	Replicas int `json:"replicas"`
}

// Status holds the status about a deployment
type Status struct {
	AvailableReplicas int `json:"availableReplicas"`
}

// CreateInfra will deploy all the infrastructure components (nmi and mic) on a Kubernetes cluster
func CreateInfra(namespace, registry, nmiVersion, micVersion, templateOutputPath string) error {
	t, err := template.New("deployment-rbac.yaml").ParseFiles(path.Join("template", "deployment-rbac.yaml"))
	if err != nil {
		return errors.Wrap(err, "Failed to parse deployment-rbac.yaml")
	}

	deployFilePath := path.Join(templateOutputPath, namespace+"-deployment-rbac.yaml")
	deployFile, err := os.Create(deployFilePath)
	if err != nil {
		return errors.Wrap(err, "Failed to create a deployment file from deployment-rbac.yaml")
	}
	defer deployFile.Close()

	deployData := struct {
		Namespace  string
		Registry   string
		NMIVersion string
		MICVersion string
	}{
		namespace,
		registry,
		nmiVersion,
		micVersion,
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

// CreateIdentityValidator will create an identity validator deployment on a Kubernetes cluster
func CreateIdentityValidator(subscriptionID, resourceGroup, registryName, name, identityBinding, identityValidatorVersion, templateOutputPath string) error {
	t, err := template.New("deployment.yaml").ParseFiles(path.Join("template", "deployment.yaml"))
	if err != nil {
		return errors.Wrap(err, "Failed to parse deployment.yaml")
	}

	deployFilePath := path.Join(templateOutputPath, name+"-deployment.yaml")
	deployFile, err := os.Create(deployFilePath)
	if err != nil {
		return errors.Wrap(err, "Failed to create a deployment file from deployment.yaml")
	}
	defer deployFile.Close()

	deployData := struct {
		Name                     string
		IdentityBinding          string
		Registry                 string
		IdentityValidatorVersion string
	}{
		name,
		identityBinding,
		registryName,
		identityValidatorVersion,
	}
	if err := t.Execute(deployFile, deployData); err != nil {
		return errors.Wrap(err, "Failed to create a deployment file from deployment.yaml")
	}

	cmd := exec.Command("kubectl", "apply", "-f", deployFilePath)
	util.PrintCommand(cmd)
	_, err = cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "Failed to deploy AzureIdentityBinding to the Kubernetes cluster")
	}

	return nil
}

// Delete will delete a deployment on a Kubernetes cluster
func Delete(name, templateOutputPath string) error {
	cmd := exec.Command("kubectl", "delete", "-f", path.Join(templateOutputPath, name), "--now", "--ignore-not-found")
	util.PrintCommand(cmd)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "Failed to delete %v from the Kubernetes cluster", name)
	}

	return nil
}

// GetAll will return a list of deployment on a Kubernetes cluster
func GetAll() (*List, error) {
	cmd := exec.Command("kubectl", "get", "deploy", "-ojson")
	util.PrintCommand(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get deployment from the Kubernetes cluster")
	}

	list := List{}
	err = json.Unmarshal(out, &list)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshall json")
	}

	return &list, nil
}

// IsAvailableReplicasMatchDesired will return a boolean that indicate whether the number
// of available replicas of a deployment matches the desired number of replicas
func isAvailableReplicasMatchDesired(name string) (bool, error) {
	dl, err := GetAll()
	if err != nil {
		return false, err
	}

	for _, deploy := range dl.Deploys {
		if deploy.Metadata.Name == name {
			return deploy.Status.AvailableReplicas == deploy.Spec.Replicas, nil
		}
	}

	return false, nil
}

// WaitOnReady will block until the number of replicas of a deployment is equal to the specified amount
func WaitOnReady(name string) (bool, error) {
	successChannel, errorChannel := make(chan bool, 1), make(chan error)
	duration, sleep := 30*time.Second, 3*time.Second
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	fmt.Println("# Tight-poll to check if the deployment is ready...")
	go func() {
		for {
			select {
			case <-ctx.Done():
				errorChannel <- errors.Errorf("Timeout exceeded (%s) while waiting for deployment (%s) to be available", duration.String(), name)
			default:
				match, err := isAvailableReplicasMatchDesired(name)
				if err != nil {
					errorChannel <- err
				}
				if match {
					successChannel <- true
					return
				}
				fmt.Printf("# The deployment is not ready yet. Retrying in %s...\n", sleep.String())
				time.Sleep(sleep)
			}
		}
	}()

	for {
		select {
		case err := <-errorChannel:
			return false, err
		case success := <-successChannel:
			return success, nil
		}
	}
}
