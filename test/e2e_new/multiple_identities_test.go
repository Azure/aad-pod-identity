// +build e2e_new

package e2e_new

import (
	"fmt"
	"math/rand"
	"time"

	aadpodv1 "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	"github.com/Azure/aad-pod-identity/test/e2e_new/framework/azureassignedidentity"
	"github.com/Azure/aad-pod-identity/test/e2e_new/framework/azureidentity"
	"github.com/Azure/aad-pod-identity/test/e2e_new/framework/azureidentitybinding"
	"github.com/Azure/aad-pod-identity/test/e2e_new/framework/identityvalidator"
	"github.com/Azure/aad-pod-identity/test/e2e_new/framework/namespace"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("[PR] When deploying multiple identities", func() {
	var (
		ns                    *corev1.Namespace
		specName              = "multiple-identities"
		size                  = 5
		azureIdentities       = make([]*aadpodv1.AzureIdentity, size)
		azureIdentityBindings = make([]*aadpodv1.AzureIdentityBinding, size)
	)

	BeforeEach(func() {
		ns = namespace.Create(namespace.CreateInput{
			Creator: kubeClient,
			Name:    specName,
		})

		for i := 0; i < size; i++ {
			azureIdentities[i] = azureidentity.Create(azureidentity.CreateInput{
				Creator:      kubeClient,
				Config:       config,
				AzureClient:  azureClient,
				Name:         fmt.Sprintf("%s-%d", keyvaultIdentity, i),
				Namespace:    ns.Name,
				IdentityType: aadpodv1.UserAssignedMSI,
				IdentityName: fmt.Sprintf("%s-%d", keyvaultIdentity, i),
			})

			azureIdentityBindings[i] = azureidentitybinding.Create(azureidentitybinding.CreateInput{
				Creator:           kubeClient,
				Name:              fmt.Sprintf("%s-binding-%d", keyvaultIdentity, i),
				Namespace:         ns.Name,
				AzureIdentityName: azureIdentities[i].Name,
				Selector:          fmt.Sprintf("%s-selector-%d", keyvaultIdentity, i),
			})
		}
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

	It("should remove the correct identities when adding AzureIdentity and AzureIdentityBinding in order and removing them in random order", func() {
		identityValidators := identityvalidator.CreateBatch(identityvalidator.CreateBatchInput{
			Creator:          kubeClient,
			Config:           config,
			Namespace:        ns.Name,
			IdentityBindings: azureIdentityBindings,
		})

		By("Shuffling the identity validator slice")
		rand.Shuffle(len(identityValidators), func(i, j int) {
			azureIdentities[i], azureIdentities[j] = azureIdentities[j], azureIdentities[i]
			identityValidators[i], identityValidators[j] = identityValidators[j], identityValidators[i]
		})

		for i := 0; i < size; i++ {
			azureassignedidentity.WaitForLen(azureassignedidentity.WaitForLenInput{
				Lister: kubeClient,
				Len:    size - i,
			})

			azureassignedidentity.Wait(azureassignedidentity.WaitInput{
				Getter:            kubeClient,
				PodName:           identityValidators[i].Name,
				Namespace:         ns.Name,
				AzureIdentityName: azureIdentities[i].Name,
				StateToWaitFor:    aadpodv1.AssignedIDAssigned,
			})

			identityvalidator.Validate(identityvalidator.ValidateInput{
				Config:           config,
				KubeconfigPath:   kubeconfigPath,
				PodName:          identityValidators[i].Name,
				Namespace:        ns.Name,
				IdentityClientID: azureIdentities[i].Spec.ClientID,
			})

			// Break when finish checking the entire list
			if i == size-1 {
				break
			}

			azureidentity.Delete(azureidentity.DeleteInput{
				Deleter:       kubeClient,
				AzureIdentity: azureIdentities[i],
			})

			By("Ensuring that non-deleted identities are still working")
			for j := i + 1; j < size; j++ {
				azureassignedidentity.Wait(azureassignedidentity.WaitInput{
					Getter:            kubeClient,
					PodName:           identityValidators[j].Name,
					Namespace:         ns.Name,
					AzureIdentityName: azureIdentities[j].Name,
					StateToWaitFor:    aadpodv1.AssignedIDAssigned,
				})

				identityvalidator.Validate(identityvalidator.ValidateInput{
					Config:           config,
					KubeconfigPath:   kubeconfigPath,
					PodName:          identityValidators[j].Name,
					Namespace:        ns.Name,
					IdentityClientID: azureIdentities[j].Spec.ClientID,
				})
			}
		}
	})

	It("should create AzureAssignedIdentities for 40 pods within 150 seconds", func() {
		expandedAzureIdentityBindings := make([]*aadpodv1.AzureIdentityBinding, 40)
		for i := 0; i < 40; i++ {
			expandedAzureIdentityBindings[i] = azureIdentityBindings[i%size]
		}

		identityValidators := identityvalidator.CreateBatch(identityvalidator.CreateBatchInput{
			Creator:          kubeClient,
			Config:           config,
			Namespace:        ns.Name,
			IdentityBindings: expandedAzureIdentityBindings,
		})

		start := time.Now()
		for i := 0; i < 40; i++ {
			azureassignedidentity.Wait(azureassignedidentity.WaitInput{
				Getter:            kubeClient,
				PodName:           identityValidators[i].Name,
				Namespace:         ns.Name,
				AzureIdentityName: azureIdentities[i%size].Name,
				StateToWaitFor:    aadpodv1.AssignedIDAssigned,
			})

			identityvalidator.Validate(identityvalidator.ValidateInput{
				Config:           config,
				KubeconfigPath:   kubeconfigPath,
				PodName:          identityValidators[i].Name,
				Namespace:        ns.Name,
				IdentityClientID: azureIdentities[i%size].Spec.ClientID,
			})
		}
		Expect(time.Since(start) <= 150*time.Second).To(BeTrue(), "Creation and validation of 40 AzureAssignedIdentities took more than 150 seconds")
	})
})
