// +build e2e_new

package azureidentitybinding

import (
	"context"
	"fmt"
	"time"

	aadpodv1 "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	"github.com/Azure/aad-pod-identity/test/e2e_new/framework"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	createTimeout = 10 * time.Second
	createPolling = 1 * time.Second

	deleteTimeout = 10 * time.Second
	deletePolling = 1 * time.Second
)

// CreateInput is the input for Create.
type CreateInput struct {
	Creator           framework.Creator
	Name              string
	Namespace         string
	AzureIdentityName string
	Selector          string
}

// Create creates an AzureIdentityBinding resource.
func Create(input CreateInput) *aadpodv1.AzureIdentityBinding {
	Expect(input.Creator).NotTo(BeNil(), "input.Creator is required for AzureIdentityBinding.Create")
	Expect(input.Name).NotTo(BeEmpty(), "input.Name is required for AzureIdentityBinding.Create")
	Expect(input.Namespace).NotTo(BeEmpty(), "input.Namespace is required for AzureIdentityBinding.Create")
	Expect(input.AzureIdentityName).NotTo(BeEmpty(), "input.AzureIdentityName is required for AzureIdentityBinding.Create")
	Expect(input.Selector).NotTo(BeEmpty(), "input.Selector is required for AzureIdentityBinding.Create")

	By(fmt.Sprintf("Creating AzureIdentityBinding \"%s\"", input.Name))

	azureIdentityBinding := &aadpodv1.AzureIdentityBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      input.Name,
			Namespace: input.Namespace,
		},
		Spec: aadpodv1.AzureIdentityBindingSpec{
			AzureIdentity: input.AzureIdentityName,
			Selector:      input.Selector,
		},
	}

	Eventually(func() error {
		return input.Creator.Create(context.TODO(), azureIdentityBinding)
	}, createTimeout, createPolling).Should(Succeed())

	return azureIdentityBinding
}

// DeleteInput is the input for Delete.
type DeleteInput struct {
	Deleter              framework.Deleter
	AzureIdentityBinding *aadpodv1.AzureIdentityBinding
}

// Delete deletes an AzureIdentityBinding resource.
func Delete(input DeleteInput) {
	Expect(input.Deleter).NotTo(BeNil(), "input.Deleter is required for AzureIdentityBinding.Delete")
	Expect(input.AzureIdentityBinding).NotTo(BeNil(), "input.AzureIdentityBinding is required for AzureIdentityBinding.Delete")

	Eventually(func() error {
		return input.Deleter.Delete(context.TODO(), input.AzureIdentityBinding)
	}, deleteTimeout, deletePolling).Should(Succeed())
}
