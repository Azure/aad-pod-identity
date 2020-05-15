package node

import (
	"encoding/json"
	"os/exec"

	"github.com/Azure/aad-pod-identity/test/common/util"

	"github.com/pkg/errors"
)

// List is a container that holds all deployment returned from 'kubectl get pods'
type List struct {
	Nodes []Node `json:"items"`
}

// Node is used to parse data from 'kubectl get pods'
type Node struct {
	Metadata `json:"metadata"`
	Spec     Spec `json:"spec"`
}

// Metadata holds information about a pod
type Metadata struct {
	Name string `json:"name"`
}

// Spec holds the node spec
type Spec struct {
	ProviderID string `json:"providerID"`
}

// GetAll will return a list of pods on a Kubernetes cluster
func GetAll() (*List, error) {
	cmd := exec.Command("kubectl", "get", "nodes", "-ojson")
	util.PrintCommand(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get nodes from the Kubernetes cluster")
	}

	list := List{}
	if err := json.Unmarshal(out, &list); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshall json")
	}

	return &list, nil
}

// Get gets a node
func Get(name string) (*Node, error) {
	cmd := exec.Command("kubectl", "get", "node", "-ojson", name)
	util.PrintCommand(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get nodes from the Kubernetes cluster: %s", string(out))
	}

	var n Node
	if err := json.Unmarshal(out, &n); err != nil {
		return nil, errors.Wrap(err, "error unmarshalling node")
	}
	return &n, nil
}

// Uncordon will uncordon a given node
func Uncordon(nodeName string) error {
	cmd := exec.Command("kubectl", "uncordon", nodeName)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "Failed to uncordon node %s", nodeName)
	}

	return nil
}

// Drain will drain a give node
func Drain(nodeName string) error {
	cmd := exec.Command("kubectl", "drain", nodeName, "--ignore-daemonsets")
	util.PrintCommand(cmd)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "Failed to drain node %s", nodeName)
	}

	return nil
}
