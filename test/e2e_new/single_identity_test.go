// +build e2e_new

package e2e_new

import (
	"fmt"

	aadpodv1 "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	"github.com/Azure/aad-pod-identity/test/e2e_new/framework/azureassignedidentity"
	"github.com/Azure/aad-pod-identity/test/e2e_new/framework/azureidentity"
	"github.com/Azure/aad-pod-identity/test/e2e_new/framework/azureidentitybinding"
	"github.com/Azure/aad-pod-identity/test/e2e_new/framework/identityvalidator"
	"github.com/Azure/aad-pod-identity/test/e2e_new/framework/namespace"

	. "github.com/onsi/ginkgo"
	corev1 "k8s.io/api/core/v1"
)

var (
	keyvaultIdentityBinding  = fmt.Sprintf("%s-binding", keyvaultIdentity)
	keyvaultIdentitySelector = fmt.Sprintf("%s-selector", keyvaultIdentity)
)

var _ = Describe("[PR] When deploying one identity", func() {
	var (
		specName             = "single-identity"
		ns                   *corev1.Namespace
		azureIdentity        *aadpodv1.AzureIdentity
		azureIdentityBinding *aadpodv1.AzureIdentityBinding
		identityValidator    *corev1.Pod
	)

	BeforeEach(func() {
		ns = namespace.Create(namespace.CreateInput{
			Creator: kubeClient,
			Name:    specName,
		})

		azureIdentity = azureidentity.Create(azureidentity.CreateInput{
			Creator:      kubeClient,
			Config:       config,
			AzureClient:  azureClient,
			Name:         keyvaultIdentity,
			Namespace:    ns.Name,
			IdentityType: aadpodv1.UserAssignedMSI,
			IdentityName: keyvaultIdentity,
		})

		azureIdentityBinding = azureidentitybinding.Create(azureidentitybinding.CreateInput{
			Creator:           kubeClient,
			Name:              keyvaultIdentityBinding,
			Namespace:         ns.Name,
			AzureIdentityName: azureIdentity.Name,
			Selector:          keyvaultIdentitySelector,
		})

		identityValidator = identityvalidator.Create(identityvalidator.CreateInput{
			Creator:         kubeClient,
			Config:          config,
			Namespace:       ns.Name,
			IdentityBinding: azureIdentityBinding.Spec.Selector,
		})

		azureassignedidentity.Wait(azureassignedidentity.WaitInput{
			Getter:            kubeClient,
			PodName:           identityValidator.Name,
			Namespace:         ns.Name,
			AzureIdentityName: azureIdentity.Name,
			StateToWaitFor:    aadpodv1.AssignedIDAssigned,
		})
	})

	AfterEach(func() {
		namespace.Delete(namespace.DeleteInput{
			Deleter:   kubeClient,
			Getter:    kubeClient,
			Namespace: ns,
		})

		azureassignedidentity.WaitForLen(azureassignedidentity.WaitForLenInput{
			Lister: kubeClient,
			Len:    0,
		})
	})

	It("should pass the identity validation", func() {
		identityvalidator.Validate(identityvalidator.ValidateInput{
			Config:           config,
			KubeconfigPath:   kubeconfigPath,
			PodName:          identityValidator.Name,
			Namespace:        ns.Name,
			IdentityClientID: azureIdentity.Spec.ClientID,
		})
	})

	It("should delete the AzureAssignedIdentity if the pod is deleted", func() {
		identityvalidator.Delete(identityvalidator.DeleteInput{
			Deleter:           kubeClient,
			IdentityValidator: identityValidator,
		})

		azureassignedidentity.WaitForLen(azureassignedidentity.WaitForLenInput{
			Lister: kubeClient,
			Len:    0,
		})
	})

	It("should not pass the identity validation if the AzureIdentity is deleted", func() {
		azureidentity.Delete(azureidentity.DeleteInput{
			Deleter:       kubeClient,
			AzureIdentity: azureIdentity,
		})

		azureassignedidentity.WaitForLen(azureassignedidentity.WaitForLenInput{
			Lister: kubeClient,
			Len:    0,
		})

		identityvalidator.Validate(identityvalidator.ValidateInput{
			Config:           config,
			KubeconfigPath:   kubeconfigPath,
			PodName:          identityValidator.Name,
			Namespace:        ns.Name,
			IdentityClientID: azureIdentity.Spec.ClientID,
			ExpectError:      true,
		})
	})

	It("should not pass the identity validation if the AzureIdentityBinding is deleted", func() {
		azureidentitybinding.Delete(azureidentitybinding.DeleteInput{
			Deleter:              kubeClient,
			AzureIdentityBinding: azureIdentityBinding,
		})

		azureassignedidentity.WaitForLen(azureassignedidentity.WaitForLenInput{
			Lister: kubeClient,
			Len:    0,
		})

		identityvalidator.Validate(identityvalidator.ValidateInput{
			Config:           config,
			KubeconfigPath:   kubeconfigPath,
			PodName:          identityValidator.Name,
			Namespace:        ns.Name,
			IdentityClientID: azureIdentity.Spec.ClientID,
			ExpectError:      true,
		})
	})

	It("should establish a new AzureAssignedIdentity and remove the old one when draining the node containing identity validator", func() {

	})

	It("should pass liveness probe test", func() {

	})

	It("should pass multiple identity validating test even when MIC is failing over", func() {

	})

	It("should assign identity with init containers", func() {

	})

	It("should assign new identity and remove old when AzureIdentity updated", func() {

	})
})
