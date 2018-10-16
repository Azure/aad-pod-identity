package pod

import (
	"encoding/json"
	"os/exec"
	"strings"

	"github.com/Azure/aad-pod-identity/test/e2e/util"
	"github.com/pkg/errors"
)

// List is a container that holds all deployment returned from 'kubectl get pods'
type List struct {
	Pods []Pod `json:"items"`
}

// Pod is used to parse data from 'kubectl get pods'
type Pod struct {
	Metadata Metadata `json:"metadata"`
}

// Metadata holds information about a pod
type Metadata struct {
	Name string `json:"name"`
}

// GetNameByPrefix will return the name of the first pod that matches a prefix
func GetNameByPrefix(prefix string) (string, error) {
	cmd := exec.Command("kubectl", "get", "pods", "-ojson")
	util.PrintCommand(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrap(err, "Failed to get pods from the Kubernetes cluster")
	}

	nl := List{}
	err = json.Unmarshal(out, &nl)
	if err != nil {
		return "", errors.Wrap(err, "Failed to unmarshall node json")
	}

	for _, pod := range nl.Pods {
		if strings.HasPrefix(pod.Metadata.Name, prefix) {
			return pod.Metadata.Name, nil
		}
	}

	return "", nil
}
