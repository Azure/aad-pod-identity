package aadpodidentity

import (
	"fmt"
	"os/exec"

	"github.com/Azure/aad-pod-identity/test/e2e/azureassignedidentity"
	"github.com/Azure/aad-pod-identity/test/e2e/azureidentity"
	"github.com/Azure/aad-pod-identity/test/e2e/azureidentitybinding"
	"github.com/Azure/aad-pod-identity/test/e2e/deploy"
	"github.com/Azure/aad-pod-identity/test/e2e/pod"
	"github.com/Azure/aad-pod-identity/test/e2e/util"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Kubernetes cluster using aad-pod-identity", func() {
	Describe("Basic Tests", func() {
		BeforeEach(func() {
			fmt.Println("Setting up the test environment...")

			err := azureidentity.CreateOnCluster(cfg.SubscriptionID, cfg.ResourceGroup, "test-identity", templateOutputPath)
			Expect(err).NotTo(HaveOccurred())

			err = azureidentitybinding.Create("test-identity", templateOutputPath)
			Expect(err).NotTo(HaveOccurred())

			err = deploy.Create(cfg.SubscriptionID, cfg.ResourceGroup, "identity-validator", "test-identity", templateOutputPath)
			Expect(err).NotTo(HaveOccurred())

			ok, err := deploy.WaitOnReady("identity-validator")
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(Equal(true))
		})

		AfterEach(func() {
			fmt.Println("Tearing down the test environment...")

			cmd := exec.Command("kubectl", "delete", "AzureIdentity,AzureIdentityBinding,AzureAssignedIdentity", "--all")
			util.PrintCommand(cmd)
			_, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred())

			ok, err := azureassignedidentity.WaitOnDeletion()
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(Equal(true))
		})

		It("should pass the identity validating test", func() {
			clientID, err := azureidentity.GetClientID(cfg.ResourceGroup, "test-identity")
			Expect(err).NotTo(HaveOccurred())
			Expect(clientID).NotTo(Equal(""))

			podName, err := pod.GetNameByPrefix("identity-validator")
			Expect(err).NotTo(HaveOccurred())
			Expect(podName).NotTo(Equal(""))

			cmd := exec.Command("kubectl", "exec", podName, "--", "identityvalidator", "--subscriptionid", cfg.SubscriptionID, "--clientid", clientID, "--resourcegroup", cfg.ResourceGroup)
			_, err = cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should not pass the identity validating test if the AzureIdentity is deleted", func() {
			err := azureidentity.DeleteOnCluster("test-identity", templateOutputPath)
			Expect(err).NotTo(HaveOccurred())

			list, err := azureidentity.GetAll()
			Expect(err).NotTo(HaveOccurred())
			Expect(list.AzureIdentities).To(HaveLen(0))

			ok, err := azureassignedidentity.WaitOnDeletion()
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(Equal(true))

			clientID, err := azureidentity.GetClientID(cfg.ResourceGroup, "test-identity")
			Expect(err).NotTo(HaveOccurred())
			Expect(clientID).NotTo(Equal(""))

			podName, err := pod.GetNameByPrefix("identity-validator")
			Expect(err).NotTo(HaveOccurred())
			Expect(podName).NotTo(Equal(""))

			cmd := exec.Command("kubectl", "exec", podName, "--", "identityvalidator", "--subscriptionid", cfg.SubscriptionID, "--clientid", clientID, "--resourcegroup", cfg.ResourceGroup)
			_, err = cmd.CombinedOutput()
			Expect(err).To(HaveOccurred())
		})

		It("should not pass the identity validating test if the AzureIdentityBinding is deleted", func() {
			err := azureidentitybinding.Delete("test-identity", templateOutputPath)
			Expect(err).NotTo(HaveOccurred())

			list, err := azureidentitybinding.GetAll()
			Expect(err).NotTo(HaveOccurred())
			Expect(list.AzureIdentityBindings).To(HaveLen(0))

			ok, err := azureassignedidentity.WaitOnDeletion()
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(Equal(true))

			clientID, err := azureidentity.GetClientID(cfg.ResourceGroup, "test-identity")
			Expect(err).NotTo(HaveOccurred())
			Expect(clientID).NotTo(Equal(""))

			podName, err := pod.GetNameByPrefix("identity-validator")
			Expect(err).NotTo(HaveOccurred())
			Expect(podName).NotTo(Equal(""))

			cmd := exec.Command("kubectl", "exec", podName, "--", "identityvalidator", "--subscriptionid", cfg.SubscriptionID, "--clientid", clientID, "--resourcegroup", cfg.ResourceGroup)
			_, err = cmd.CombinedOutput()
			Expect(err).To(HaveOccurred())
		})

		It("should delete the AzureAssignedIdentity if the deployment is deleted", func() {
			err := deploy.Delete("identity-validator", templateOutputPath)
			Expect(err).NotTo(HaveOccurred())

			ok, err := azureassignedidentity.WaitOnDeletion()
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(Equal(true))
		})
	})
})
