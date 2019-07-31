package deploy

import (
	"context"
	"encoding/json"
	"fmt"
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
	Metadata `json:"metadata"`
	Spec     `json:"spec"`
	Status   `json:"status"`
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

// Delete will delete a deployment on a Kubernetes cluster
func Delete(name, templateOutputPath string) error {
	cmd := exec.Command("kubectl", "delete", "-f", path.Join(templateOutputPath, name+"-deployment.yaml"), "--now", "--ignore-not-found")
	util.PrintCommand(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "Failed to delete %v from the Kubernetes cluster: %s", name, out)
	}

	return nil
}

// GetAllDeployments will return a list of deployment on a Kubernetes cluster
func GetAllDeployments() (*List, error) {
	cmd := exec.Command("kubectl", "get", "deploy", "-ojson")
	util.PrintCommand(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get deployment from the Kubernetes cluster")
	}

	list := List{}
	if err := json.Unmarshal(out, &list); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshall json")
	}

	return &list, nil
}

// IsAvailableReplicasMatchDesired will return a boolean that indicate whether the number
// of available replicas of a deployment matches the desired number of replicas
func isAvailableReplicasMatchDesired(name string) (bool, error) {
	dl, err := GetAllDeployments()
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
	duration, sleep := 60*time.Second, 3*time.Second
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	fmt.Printf("# Poll to check if %s deployment is ready...\n", name)
	go func() {
		for {
			select {
			case <-ctx.Done():
				errorChannel <- errors.Errorf("Timeout exceeded (%s) while waiting for deployment (%s) to be available", duration.String(), name)
			default:
				match, err := isAvailableReplicasMatchDesired(name)
				if err != nil {
					errorChannel <- err
					return
				}
				if match {
					successChannel <- true
					return
				}
				fmt.Printf("# %s deployment is not ready yet. Retrying in %s...\n", name, sleep.String())
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
