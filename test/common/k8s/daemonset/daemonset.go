package daemonset

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/Azure/aad-pod-identity/test/common/util"
	"github.com/pkg/errors"
)

// List is a container that holds all daemonsets returned from 'kubectl get ds'
type List struct {
	DaemonSets []DaemonSet `json:"items"`
}

// DaemonSet is used to parse data from 'kubectl get ds'
type DaemonSet struct {
	Metadata `json:"metadata"`
	Status   `json:"status"`
}

// Metadata holds information about a daemonset
type Metadata struct {
	Name string `json:"name"`
}

// Status holds the status about a daemonset
type Status struct {
	DesiredNumberScheduled int `json:"desiredNumberScheduled"`
	NumberAvailable        int `json:"numberAvailable"`
	NumberReady            int `json:"numberReady"`
}

// GetAllDaemonSets will return a list of daemonsets on a Kubernetes cluster
func GetAllDaemonSets() (*List, error) {
	cmd := exec.Command("kubectl", "get", "ds", "-ojson")
	util.PrintCommand(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get daemonset from the Kubernetes cluster")
	}

	list := List{}
	if err := json.Unmarshal(out, &list); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshall json")
	}

	return &list, nil
}

// IsAvailableReplicasMatchDesired will return a boolean that indicate whether the number
// of available replicas of a daemonset matches the desired number of replicas
func isAvailableReplicasMatchDesired(name string) (bool, error) {
	dsl, err := GetAllDaemonSets()
	if err != nil {
		return false, err
	}

	for _, ds := range dsl.DaemonSets {
		if ds.Metadata.Name == name {
			return ds.Status.DesiredNumberScheduled == ds.Status.NumberAvailable, nil
		}
	}

	return false, nil
}

// WaitOnReady will block until the number of replicas of a daemonset is equal to the specified amount
func WaitOnReady(name string) (bool, error) {
	successChannel, errorChannel := make(chan bool, 1), make(chan error)
	duration, sleep := 30*time.Second, 3*time.Second
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	fmt.Printf("# Poll to check if %s daemonset is ready...\n", name)
	go func() {
		for {
			select {
			case <-ctx.Done():
				errorChannel <- errors.Errorf("Timeout exceeded (%s) while waiting for daemonset (%s) to be available", duration.String(), name)
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
				fmt.Printf("# %s daemonset is not ready yet. Retrying in %s...\n", name, sleep.String())
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
