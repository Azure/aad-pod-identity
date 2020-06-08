// +build e2e_new

package e2e_new

import (
	aadpodv1 "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	"github.com/Azure/aad-pod-identity/test/e2e_new/framework/azureassignedidentity"
	"github.com/Azure/aad-pod-identity/test/e2e_new/framework/azureidentity"
	"github.com/Azure/aad-pod-identity/test/e2e_new/framework/azureidentitybinding"
	"github.com/Azure/aad-pod-identity/test/e2e_new/framework/identityvalidator"
	"github.com/Azure/aad-pod-identity/test/e2e_new/framework/namespace"

	. "github.com/onsi/ginkgo"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("AAD Pod Identity in standard mode", func() {
	var (
		ns *corev1.Namespace
	)

	BeforeEach(func() {
		ns = namespace.Create(namespace.CreateInput{
			Creator: clusterProxy.GetClient(),
			Name:    "standard",
		})
	})

	AfterEach(func() {
		namespace.Delete(namespace.DeleteInput{
			Deleter:   clusterProxy.GetClient(),
			Getter:    clusterProxy.GetClient(),
			Namespace: ns,
		})
	})

	It("should create an AzureAssignedIdentity", func() {
		azureIdentity := azureidentity.Create(azureidentity.CreateInput{
			Creator:      clusterProxy.GetClient(),
			Config:       config,
			AzureClient:  azureClient,
			Name:         "test",
			Namespace:    ns.Name,
			IdentityType: aadpodv1.UserAssignedMSI,
			IdentityName: "keyvault-identity",
		})

		azureIdentityBinding := azureidentitybinding.Create(azureidentitybinding.CreateInput{
			Creator:           clusterProxy.GetClient(),
			Name:              "test-binding",
			Namespace:         ns.Name,
			AzureIdentityName: azureIdentity.Name,
			Selector:          "selector",
		})

		identityValidator := identityvalidator.Create(identityvalidator.CreateInput{
			Creator:         clusterProxy.GetClient(),
			Config:          config,
			Namespace:       ns.Name,
			IdentityBinding: azureIdentityBinding.Spec.Selector,
		})

		azureassignedidentity.Wait(azureassignedidentity.WaitInput{
			Getter:            clusterProxy.GetClient(),
			PodName:           identityValidator.Name,
			Namespace:         ns.Name,
			AzureIdentityName: azureIdentity.Name,
			StateToWaitFor:    aadpodv1.AssignedIDAssigned,
		})

		identityvalidator.Validate(identityvalidator.ValidateInput{
			Config:           config,
			KubeconfigPath:   clusterProxy.GetKubeconfigPath(),
			PodName:          identityValidator.Name,
			Namespace:        ns.Name,
			IdentityClientID: azureIdentity.Spec.ClientID,
		})
	})
})
