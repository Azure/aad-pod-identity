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
			Getter:           kubeClient,
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
			Getter:           kubeClient,
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
			Getter:           kubeClient,
			Config:           config,
			KubeconfigPath:   kubeconfigPath,
			PodName:          identityValidator.Name,
			Namespace:        ns.Name,
			IdentityClientID: azureIdentity.Spec.ClientID,
			ExpectError:      true,
		})
	})

	It("should update AzureAssignedIdentity and when AzureIdentity fields are updated", func() {
		azureIdentity = azureidentity.Update(azureidentity.UpdateInput{
			Updater:             kubeClient,
			Config:              config,
			AzureClient:         azureClient,
			AzureIdentity:       azureIdentity,
			UpdatedIdentityName: fmt.Sprintf("%s-0", keyvaultIdentity),
		})

		azureassignedidentity.Wait(azureassignedidentity.WaitInput{
			Getter:            kubeClient,
			PodName:           identityValidator.Name,
			Namespace:         ns.Name,
			AzureIdentityName: azureIdentity.Name,
			StateToWaitFor:    aadpodv1.AssignedIDAssigned,
		})

		identityvalidator.Validate(identityvalidator.ValidateInput{
			Getter:           kubeClient,
			Config:           config,
			KubeconfigPath:   kubeconfigPath,
			PodName:          identityValidator.Name,
			Namespace:        ns.Name,
			IdentityClientID: azureIdentity.Spec.ClientID,
		})
	})

	It("should pass identity validation with correct identity and fail with wrong identity", func() {
		// This test is to ensure when 2 identities for the pod exist, the
		// correct identity is used based on the client id in the request.
		// keyvault-identity has the right permissions to get and list secret
		// keyvault-identity-5 only has permission to list and should fail to get secret.
		keyvaultIdentity5 := fmt.Sprintf("%s-5", keyvaultIdentity)
		azureIdentityWithoutGetPermission := azureidentity.Create(azureidentity.CreateInput{
			Creator:      kubeClient,
			Config:       config,
			AzureClient:  azureClient,
			Name:         keyvaultIdentity5,
			Namespace:    ns.Name,
			IdentityType: aadpodv1.UserAssignedMSI,
			IdentityName: keyvaultIdentity5,
		})

		azureidentitybinding.Create(azureidentitybinding.CreateInput{
			Creator:           kubeClient,
			Name:              fmt.Sprintf("%s-binding", keyvaultIdentity5),
			Namespace:         ns.Name,
			AzureIdentityName: azureIdentityWithoutGetPermission.Name,
			Selector:          keyvaultIdentitySelector,
		})

		azureassignedidentity.Wait(azureassignedidentity.WaitInput{
			Getter:            kubeClient,
			PodName:           identityValidator.Name,
			Namespace:         ns.Name,
			AzureIdentityName: azureIdentityWithoutGetPermission.Name,
			StateToWaitFor:    aadpodv1.AssignedIDAssigned,
		})

		identityvalidator.Validate(identityvalidator.ValidateInput{
			Getter:           kubeClient,
			Config:           config,
			KubeconfigPath:   kubeconfigPath,
			PodName:          identityValidator.Name,
			Namespace:        ns.Name,
			IdentityClientID: azureIdentity.Spec.ClientID,
		})

		identityvalidator.Validate(identityvalidator.ValidateInput{
			Getter:           kubeClient,
			Config:           config,
			KubeconfigPath:   kubeconfigPath,
			PodName:          identityValidator.Name,
			Namespace:        ns.Name,
			IdentityClientID: azureIdentityWithoutGetPermission.Spec.ClientID,
			ExpectError:      true,
		})
	})
})
