// +build e2e

package e2e

import (
	aadpodv1 "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	"github.com/Azure/aad-pod-identity/test/e2e/framework/azureassignedidentity"
	"github.com/Azure/aad-pod-identity/test/e2e/framework/azureidentity"
	"github.com/Azure/aad-pod-identity/test/e2e/framework/azureidentitybinding"
	"github.com/Azure/aad-pod-identity/test/e2e/framework/identityvalidator"
	"github.com/Azure/aad-pod-identity/test/e2e/framework/namespace"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

// e2e test for regression https://github.com/Azure/aad-pod-identity/issues/1065.
var _ = Describe("When AAD Pod Identity operations are disrupted", func() {
	var (
		specName = "name-conflict"
		ns1      *corev1.Namespace
		ns2      *corev1.Namespace
	)

	BeforeEach(func() {
		ns1 = namespace.Create(namespace.CreateInput{
			Creator: kubeClient,
			Name:    specName,
		})

		ns2 = namespace.Create(namespace.CreateInput{
			Creator: kubeClient,
			Name:    specName,
		})

		// assume that ns1's name is alphabetically smaller than ns2's name
		if ns1.Name >= ns2.Name {
			ns1, ns2 = ns2, ns1
		}
	})

	AfterEach(func() {
		for _, ns := range []*corev1.Namespace{ns1, ns2} {
			Cleanup(CleanupInput{
				Namespace: ns,
				Getter:    kubeClient,
				Lister:    kubeClient,
				Deleter:   kubeClient,
			})
		}
	})

	It("should pass the identity validation even when two AzureIdentities and AzureIdentityBindings with the same name are deployed across different namespaces", func() {
		azureIdentity := azureidentity.Create(azureidentity.CreateInput{
			Creator:      kubeClient,
			Config:       config,
			AzureClient:  azureClient,
			Name:         keyvaultIdentity,
			Namespace:    ns1.Name,
			IdentityType: aadpodv1.UserAssignedMSI,
			IdentityName: keyvaultIdentity,
		})
		azureIdentityBinding := azureidentitybinding.Create(azureidentitybinding.CreateInput{
			Creator:           kubeClient,
			Name:              keyvaultIdentityBinding,
			Namespace:         ns1.Name,
			AzureIdentityName: azureIdentity.Name,
			Selector:          keyvaultIdentitySelector,
		})

		// Create AzureIdentity and AzureIdentityBiniding in ns2 with the same name
		_ = azureidentity.Create(azureidentity.CreateInput{
			Creator:      kubeClient,
			Config:       config,
			AzureClient:  azureClient,
			Name:         keyvaultIdentity,
			Namespace:    ns2.Name,
			IdentityType: aadpodv1.UserAssignedMSI,
			IdentityName: keyvaultIdentity,
		})
		_ = azureidentitybinding.Create(azureidentitybinding.CreateInput{
			Creator:           kubeClient,
			Name:              keyvaultIdentityBinding,
			Namespace:         ns2.Name,
			AzureIdentityName: azureIdentity.Name,
			Selector:          keyvaultIdentitySelector,
		})

		identityValidator := identityvalidator.Create(identityvalidator.CreateInput{
			Creator:         kubeClient,
			Config:          config,
			Namespace:       ns1.Name,
			IdentityBinding: azureIdentityBinding.Spec.Selector,
		})

		azureAssignedIdentity := azureassignedidentity.Wait(azureassignedidentity.WaitInput{
			Getter:            kubeClient,
			PodName:           identityValidator.Name,
			Namespace:         ns1.Name,
			AzureIdentityName: azureIdentity.Name,
			StateToWaitFor:    aadpodv1.AssignedIDAssigned,
		})

		identityvalidator.Validate(identityvalidator.ValidateInput{
			Getter:             kubeClient,
			Config:             config,
			KubeconfigPath:     kubeconfigPath,
			PodName:            identityValidator.Name,
			Namespace:          ns1.Name,
			IdentityClientID:   azureIdentity.Spec.ClientID,
			IdentityResourceID: azureIdentity.Spec.ResourceID,
		})

		// ensure that the AzureAssignedIdentity is referencing the AzureIdentity and AzureIdentityBinding from ns1
		Expect(azureAssignedIdentity.Spec.AzureIdentityRef.Namespace).Should(Equal(ns1.Name))
		Expect(azureAssignedIdentity.Spec.AzureBindingRef.Namespace).Should(Equal(ns1.Name))
	})
})
