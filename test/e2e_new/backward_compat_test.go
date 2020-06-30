// +build e2e_new

package e2e_new

import (
	"fmt"
	"os"
	"time"

	aadpodv1 "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	"github.com/Azure/aad-pod-identity/test/e2e_new/framework/azureassignedidentity"
	"github.com/Azure/aad-pod-identity/test/e2e_new/framework/azureidentity"
	"github.com/Azure/aad-pod-identity/test/e2e_new/framework/azureidentitybinding"
	"github.com/Azure/aad-pod-identity/test/e2e_new/framework/exec"
	"github.com/Azure/aad-pod-identity/test/e2e_new/framework/helm"
	"github.com/Azure/aad-pod-identity/test/e2e_new/framework/identityvalidator"
	"github.com/Azure/aad-pod-identity/test/e2e_new/framework/namespace"
	corev1 "k8s.io/api/core/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("[PR] When upgrading AAD Pod Identity", func() {
	var (
		specName                 = "backward-compat"
		ns                       *corev1.Namespace
		keyvaultIdentityBinding  = fmt.Sprintf("%s-binding", keyvaultIdentity)
		keyvaultIdentitySelector = fmt.Sprintf("%s-selector", keyvaultIdentity)
	)

	BeforeEach(func() {
		ns = namespace.Create(namespace.CreateInput{
			Creator: kubeClient,
			Name:    specName,
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

	It("should be backward compatible with old and new version of MIC and NMI", func() {
		helm.Uninstall()

		By("Deleting the ConfigMap used to store upgrade information")
		err := exec.KubectlDelete(kubeconfigPath, corev1.NamespaceDefault, []string{
			"--ignore-not-found",
			"cm",
			"aad-pod-identity-config",
		})
		Expect(err).To(BeNil())

		configOldVersion := config.DeepCopy()
		configOldVersion.Registry = "mcr.microsoft.com/k8s/aad-pod-identity"
		configOldVersion.MICVersion = "1.5"
		configOldVersion.NMIVersion = "1.5"
		helm.Install(helm.InstallInput{
			Config: configOldVersion,
		})

		azureIdentityFile, identityClientID := azureidentity.CreateOld(azureidentity.CreateInput{
			Config:       config,
			AzureClient:  azureClient,
			Name:         keyvaultIdentity,
			Namespace:    ns.Name,
			IdentityType: aadpodv1.UserAssignedMSI,
			IdentityName: keyvaultIdentity,
		})
		defer os.Remove(azureIdentityFile)

		err = exec.KubectlApply(kubeconfigPath, ns.Name, []string{"-f", azureIdentityFile})
		Expect(err).To(BeNil())

		azureIdentityBindingFile := azureidentitybinding.CreateOld(azureidentitybinding.CreateInput{
			Name:              keyvaultIdentityBinding,
			Namespace:         ns.Name,
			AzureIdentityName: keyvaultIdentity,
			Selector:          keyvaultIdentitySelector,
		})
		defer os.Remove(azureIdentityBindingFile)

		err = exec.KubectlApply(kubeconfigPath, ns.Name, []string{"-f", azureIdentityBindingFile})
		Expect(err).To(BeNil())

		identityValidator := identityvalidator.Create(identityvalidator.CreateInput{
			Creator:         kubeClient,
			Config:          config,
			Namespace:       ns.Name,
			IdentityBinding: keyvaultIdentitySelector,
		})

		// We won't be able to wait for AzureAssignedIdentity to be Assigned
		// due to change in JSON fields introduced in #398
		// So wait for 60 seconds, which is enough for an identity to be assigned to a VMAS & VMSS node
		By("Waiting for the identity to get assigned to the node")
		time.Sleep(60 * time.Second)

		identityvalidator.Validate(identityvalidator.ValidateInput{
			Getter:           kubeClient,
			Config:           config,
			KubeconfigPath:   kubeconfigPath,
			PodName:          identityValidator.Name,
			Namespace:        ns.Name,
			IdentityClientID: identityClientID,
		})

		helm.Upgrade(config)

		identityvalidator.Validate(identityvalidator.ValidateInput{
			Getter:           kubeClient,
			Config:           config,
			KubeconfigPath:   kubeconfigPath,
			PodName:          identityValidator.Name,
			Namespace:        ns.Name,
			IdentityClientID: identityClientID,
		})
	})
})
