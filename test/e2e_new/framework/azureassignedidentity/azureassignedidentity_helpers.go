// +build e2e_new

package azureassignedidentity

import (
	"context"
	"fmt"
	"time"

	aadpodv1 "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	"github.com/Azure/aad-pod-identity/test/e2e_new/framework"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	waitTimeout = 5 * time.Minute
	waitPolling = 5 * time.Second
)

// WaitInput is the input for Wait.
type WaitInput struct {
	Getter            framework.Getter
	PodName           string
	Namespace         string
	AzureIdentityName string
	StateToWaitFor    string
}

// Wait waits for an AzureAssignedIdentity to reach a desired state.
func Wait(input WaitInput) {
	Expect(input.Getter).NotTo(BeNil(), "input.Getter is required for AzureAssignedIdentity.Wait")
	Expect(input.PodName).NotTo(BeEmpty(), "input.PodName is required for AzureAssignedIdentity.Wait")
	Expect(input.Namespace).NotTo(BeEmpty(), "input.Namespace is required for AzureAssignedIdentity.Wait")
	Expect(input.AzureIdentityName).NotTo(BeEmpty(), "input.AzureIdentityName is required for AzureAssignedIdentity.Wait")
	Expect(input.StateToWaitFor).NotTo(BeEmpty(), "input.StateToWaitFor is required for AzureAssignedIdentity.Wait")

	name := fmt.Sprintf("%s-%s-%s", input.PodName, input.Namespace, input.AzureIdentityName)

	By(fmt.Sprintf("Ensuring that AzureAssignedIdentity \"%s\" is in %s state", name, input.StateToWaitFor))

	Eventually(func() (bool, error) {
		azureAssignedIdentity := &aadpodv1.AzureAssignedIdentity{}

		// AzureAssignedIdentity is always in default namespace unless MIC is in namespaced mode
		if err := input.Getter.Get(context.TODO(), client.ObjectKey{Name: name, Namespace: "default"}, azureAssignedIdentity); err != nil {
			return false, err
		}
		if azureAssignedIdentity.Status.Status == input.StateToWaitFor {
			return true, nil
		}
		return false, nil
	}, waitTimeout, waitPolling).Should(BeTrue())
}
