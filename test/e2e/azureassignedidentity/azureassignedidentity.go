package azureassignedidentity

import (
	"encoding/json"
	"log"
	"os/exec"

	"github.com/Azure/aad-pod-identity/test/e2e/util"
)

// TODO: Add comments
type AzureAssignedIdentity struct {
	Metadata Metadata `json:"metadata"`
}

// TODO: Add comments
type Metadata struct {
	Name string `json:"name"`
}

// TODO: Add comments
type List struct {
	AzureAssignedIdentities []AzureAssignedIdentity `json:"items"`
}

// TODO: Add comments
func Delete(name string) error {
	cmd := exec.Command("kubectl", "delete", "AzureAssignedIdentity", name)
	util.PrintCommand(cmd)
	_, err := cmd.CombinedOutput()
	return err
}

// TODO: Add comments
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
