package azureassignedidentity

import (
	"encoding/json"
	"log"
	"os/exec"

	"github.com/Azure/aad-pod-identity/test/e2e/util"
)

// AzureAssignedIdentity is used to parse data from 'kubectl get AzureAssignedIdentity'
type AzureAssignedIdentity struct {
	Metadata Metadata `json:"metadata"`
}

// Metadata holds information about AzureAssignedIdentity
type Metadata struct {
	Name string `json:"name"`
}

// List is a container that holds all AzureAssignedIdentity returned from 'kubectl get AzureAssignedIdentity'
type List struct {
	AzureAssignedIdentities []AzureAssignedIdentity `json:"items"`
}

// GetAll will return a list of AzureAssignedIdentity deployed on a Kubernetes cluster
func GetAll() (*List, error) {
	cmd := exec.Command("kubectl", "get", "AzureAssignedIdentity", "-ojson")
	util.PrintCommand(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error trying to run 'kubectl get AzureAssignedIdentity':%s", string(out))
		return nil, err
	}

	nl := List{}
	err = json.Unmarshal(out, &nl)
	if err != nil {
		log.Printf("Error unmarshalling nodes json:%s", err)
		return nil, err
	}

	return &nl, nil
}
