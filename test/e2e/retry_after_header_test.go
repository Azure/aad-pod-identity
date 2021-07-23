// +build e2e

package e2e

import (
	aadpodv1 "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	"github.com/Azure/aad-pod-identity/test/e2e/framework/azureassignedidentity"
	"github.com/Azure/aad-pod-identity/test/e2e/framework/azureidentity"
	"github.com/Azure/aad-pod-identity/test/e2e/framework/azureidentitybinding"
	"github.com/Azure/aad-pod-identity/test/e2e/framework/helm"
	"github.com/Azure/aad-pod-identity/test/e2e/framework/identityvalidator"
	"github.com/Azure/aad-pod-identity/test/e2e/framework/namespace"

	. "github.com/onsi/ginkgo"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("When SetRetryAfter header is enabled", func() {
	var (
		specName = "retry-after"
		ns       *corev1.Namespace
	)

	BeforeEach(func() {
		ns = namespace.Create(namespace.CreateInput{
			Creator: kubeClient,
			Name:    specName,
		})
		// upgrade pod identity to use the feature flag set-retry-after-header and
		// disable the internal retries.
		c := *config
		c.RetryAttemptsForCreated = 1
		c.RetryAttemptsForAssigned = 1
		c.FindIdentityRetryIntervalInSeconds = 1
		c.SetRetryAfterHeader = true

		helm.Upgrade(helm.UpgradeInput{Config: &c})
	})

	AfterEach(func() {
		Cleanup(CleanupInput{
			Namespace: ns,
			Getter:    kubeClient,
			Lister:    kubeClient,
			Deleter:   kubeClient,
		})
		// reset the feature flag
		helm.Upgrade(helm.UpgradeInput{Config: config})
	})

	It("should pass the identity validation with retries from SDK based on Retry-After header", func() {
		azureIdentity := azureidentity.Create(azureidentity.CreateInput{
			Creator:      kubeClient,
			Config:       config,
			AzureClient:  azureClient,
			Name:         keyvaultIdentity,
			Namespace:    ns.Name,
			IdentityType: aadpodv1.UserAssignedMSI,
			IdentityName: keyvaultIdentity,
		})
		azureIdentityBinding := azureidentitybinding.Create(azureidentitybinding.CreateInput{
			Creator:           kubeClient,
			Name:              keyvaultIdentityBinding,
			Namespace:         ns.Name,
			AzureIdentityName: azureIdentity.Name,
			Selector:          keyvaultIdentitySelector,
		})

		identityValidator := identityvalidator.Create(identityvalidator.CreateInput{
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

		identityvalidator.Validate(identityvalidator.ValidateInput{
			Getter:             kubeClient,
			Config:             config,
			KubeconfigPath:     kubeconfigPath,
			PodName:            identityValidator.Name,
			Namespace:          ns.Name,
			IdentityClientID:   azureIdentity.Spec.ClientID,
			IdentityResourceID: azureIdentity.Spec.ResourceID,
		})
	})
})
