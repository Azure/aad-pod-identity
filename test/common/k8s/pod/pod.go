package pod

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Azure/aad-pod-identity/test/common/util"
	"github.com/pkg/errors"
)

// List is a container that holds all pods returned from 'kubectl get pods'
type List struct {
	Pods []Pod `json:"items"`
}

// Pod is used to parse data from 'kubectl get pods'
type Pod struct {
	Metadata `json:"metadata"`
	Spec     `json:"spec"`
	Status   `json:"status"`
}

// Metadata holds information about a pod
type Metadata struct {
	Name string `json:"name"`
}

// Spec holds spec about a pod
type Spec struct {
	NodeName string `json:"nodeName"`
}

// Status holds the status about a pod
type Status struct {
	Phase string `json:"phase"`
}

// GetAll will return a list of pods on a Kubernetes cluster
func GetAll() (*List, error) {
	cmd := exec.Command("kubectl", "get", "pods", "-ojson")
	util.PrintCommand(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get pods from the Kubernetes cluster")
	}

	list := List{}
	if err := json.Unmarshal(out, &list); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshall json")
	}

	return &list, nil
}

// GetNameByPrefix will return the name of the first pod that matches a prefix
func GetNameByPrefix(prefix string) (string, error) {
	list, err := GetAll()
	if err != nil {
		return "", err
	}

	for _, pod := range list.Pods {
		if strings.HasPrefix(pod.Metadata.Name, prefix) && pod.Status.Phase == "Running" {
			return pod.Metadata.Name, nil
		}
	}

	return "", nil
}

// GetAllNameByPrefix will return the name of all pods that matches a prefix
func GetAllNameByPrefix(prefix string) ([]string, error) {
	var pods []string
	list, err := GetAll()
	if err != nil {
		return pods, err
	}

	for _, pod := range list.Pods {
		if strings.HasPrefix(pod.Metadata.Name, prefix) && pod.Status.Phase == "Running" {
			pods = append(pods, pod.Metadata.Name)
		}
	}

	return pods, nil
}

// RunCommandInPod runs command with kubectl exec in pod
func RunCommandInPod(execCmd ...string) (string, error) {
	cmd := exec.Command("kubectl", execCmd...)
	util.PrintCommand(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), errors.Wrap(err, fmt.Sprintf("Failed to execute command in pod"))
	}
	return string(out), err
}

// GetNodeName will return the name of the node the pod is running on
func GetNodeName(podName string) (string, error) {
	cmd := exec.Command("kubectl", "get", "pod", podName, "-ojson")
	util.PrintCommand(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("Failed to get pod %s from the Kubernetes cluster", podName))
	}

	pod := Pod{}
	if err := json.Unmarshal(out, &pod); err != nil {
		return "", errors.Wrap(err, "Failed to unmarshall json")
	}

	return pod.Spec.NodeName, nil
}

// WaitOnDeletion will block until there is no pod name that starts with the give prefix
func WaitOnDeletion(prefix string) (bool, error) {
	successChannel, errorChannel := make(chan bool, 1), make(chan error)
	duration, sleep := 120*time.Second, 10*time.Second
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	fmt.Println("# Tight-poll to check if pods are deleted...")
	go func() {
		for {
			select {
			case <-ctx.Done():
				errorChannel <- errors.Errorf("Timeout exceeded (%s) while waiting for pod (%s) deletion to complete", duration.String(), prefix)
			default:
				list, err := GetAll()
				if err != nil {
					errorChannel <- err
					return
				}

				matched := false
				for _, pod := range list.Pods {
					if strings.HasPrefix(pod.Metadata.Name, prefix) {
						matched = true
						break
					}
				}
				if !matched {
					successChannel <- true
					return
				}

				fmt.Printf("# %s pod is not completely deleted yet. Retrying in %s...\n", prefix, sleep.String())
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
