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
	corev1 "k8s.io/api/core/v1"
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
		if err := input.Getter.Get(context.TODO(), client.ObjectKey{Name: name, Namespace: corev1.NamespaceDefault}, azureAssignedIdentity); err != nil {
			return false, err
		}
		if azureAssignedIdentity.Status.Status == input.StateToWaitFor {
			return true, nil
		}
		return false, nil
	}, waitTimeout, waitPolling).Should(BeTrue())
}

// WaitForLenInput is the input for WaitForLen.
type WaitForLenInput struct {
	Lister framework.Lister
	Len    int
}

// WaitForLen waits for the number of AzureAssignedIdentities to reach a desired length.
func WaitForLen(input WaitForLenInput) {
	Expect(input.Lister).NotTo(BeNil(), "input.Lister is required for AzureAssignedIdentity.WaitForLen")
	Expect(input.Len >= 0).To(BeTrue(), "input.Len must be positive for AzureAssignedIdentity.WaitForLen")

	By(fmt.Sprintf("Ensuring that there exists %d AzureAssignedIdentity", input.Len))

	Eventually(func() (bool, error) {
		azureAssignedIdentityList := &aadpodv1.AzureAssignedIdentityList{}

		// AzureAssignedIdentity is always in default namespace unless MIC is in namespaced mode
		if err := input.Lister.List(context.TODO(), azureAssignedIdentityList, client.InNamespace(corev1.NamespaceDefault)); err != nil {
			return false, err
		}
		if len(azureAssignedIdentityList.Items) == input.Len {
			return true, nil
		}
		return false, nil
	}, waitTimeout, waitPolling).Should(BeTrue())
}
