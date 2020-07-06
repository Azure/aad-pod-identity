// +build e2e_new

package azurepodidentityexception

import (
	"context"
	"fmt"

	aadpodv1 "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	"github.com/Azure/aad-pod-identity/test/e2e_new/framework"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateInput is the input for Create.
type CreateInput struct {
	Creator   framework.Creator
	Name      string
	Namespace string
	PodLabels map[string]string
}

// Create creates an AzurePodIdentityException resource.
func Create(input CreateInput) *aadpodv1.AzurePodIdentityException {
	Expect(input.Creator).NotTo(BeNil(), "input.Creator is required for AzurePodIdentityException.Create")
	Expect(input.Name).NotTo(BeEmpty(), "input.Name is required for AzurePodIdentityException.Create")
	Expect(input.Namespace).NotTo(BeEmpty(), "input.Namespace is required for AzurePodIdentityException.Create")
	Expect(len(input.PodLabels) == 0).NotTo(BeTrue(), "input.PodLabels is required for AzurePodIdentityException.Create")

	By(fmt.Sprintf("Creating AzurePodIdentityException \"%s\"", input.Name))

	azurePodIdentityException := &aadpodv1.AzurePodIdentityException{
		ObjectMeta: metav1.ObjectMeta{
			Name:      input.Name,
			Namespace: input.Namespace,
		},
		Spec: aadpodv1.AzurePodIdentityExceptionSpec{
			PodLabels: input.PodLabels,
		},
	}

	Eventually(func() error {
		return input.Creator.Create(context.TODO(), azurePodIdentityException)
	}, framework.CreateTimeout, framework.CreatePolling).Should(Succeed())

	return azurePodIdentityException
}
