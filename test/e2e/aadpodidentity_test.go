package aadpodidentity

import (
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
	It("should pass the identity validating test", func() {
		err := azureidentity.CreateOnCluster(cfg.SubscriptionID, cfg.ResourceGroup, "test-identity", templateOutputPath)
		Expect(err).NotTo(HaveOccurred())

		err = azureidentitybinding.Create("test-identity", templateOutputPath)
		Expect(err).NotTo(HaveOccurred())

		err = deploy.Create(cfg.SubscriptionID, cfg.ResourceGroup, "identity-validator", "test-identity", templateOutputPath)
		Expect(err).NotTo(HaveOccurred())

		ok, err := deploy.WaitOnReady("identity-validator")
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(Equal(true))

		azureAssignedIdentity, err := azureassignedidentity.GetAll()
		Expect(err).NotTo(HaveOccurred())
		Expect(len(azureAssignedIdentity.AzureAssignedIdentities)).To(Equal(1))

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

	AfterEach(func() {
		cmd := exec.Command("kubectl", "delete", "AzureIdentity", "--all")
		util.PrintCommand(cmd)
		_, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred())

		cmd = exec.Command("kubectl", "delete", "AzureIdentityBinding", "--all")
		util.PrintCommand(cmd)
		_, err = cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred())

		cmd = exec.Command("kubectl", "delete", "deploy", "--all")
		util.PrintCommand(cmd)
		_, err = cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred())
	})
})
