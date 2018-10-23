package azureassignedidentity

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	"github.com/Azure/aad-pod-identity/test/common/util"
	"github.com/pkg/errors"
)

// GetAll will return a list of AzureAssignedIdentity deployed on a Kubernetes cluster
func GetAll() (*aadpodid.AzureAssignedIdentityList, error) {
	cmd := exec.Command("kubectl", "get", "AzureAssignedIdentity", "-ojson")
	util.PrintCommand(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get AzureAssignedIdentity from the Kubernetes cluster")
	}

	list := aadpodid.AzureAssignedIdentityList{}
	err = json.Unmarshal(out, &list)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshall json")
	}

	return &list, nil
}

// WaitOnDeletion will block until the Azure Assigned Identity is deleted
func WaitOnDeletion() (bool, error) {
	successChannel, errorChannel := make(chan bool, 1), make(chan error)
	duration, sleep := 60*time.Second, 10*time.Second
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
				if len(list.Items) == 0 {
					successChannel <- true
					return
				}
				fmt.Printf("# Azure Assigned Identity is not completely deleted yet. Retrying in %s...\n", sleep.String())
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
