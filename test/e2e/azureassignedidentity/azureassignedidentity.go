package azureassignedidentity

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"time"

	"github.com/Azure/aad-pod-identity/test/e2e/util"
	"github.com/pkg/errors"
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

// WaitOnDeletion
func WaitOnDeletion() (bool, error) {
	successChannel, errorChannel := make(chan bool, 1), make(chan error)
	duration := 60 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	fmt.Println("# Tight-poll to check if the Azure Assigned Identity is deleted...")
	go func() {
		for {
			select {
			case <-ctx.Done():
				errorChannel <- errors.Errorf("Timeout exceeded (%s) while waiting for AzureAssignedIdentity deletion to complete", duration.String())
			default:
				list, err := GetAll()
				if err != nil {
					errorChannel <- err
				}
				if len(list.AzureAssignedIdentities) == 0 {
					successChannel <- true
				}
				time.Sleep(10 * time.Second)
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
