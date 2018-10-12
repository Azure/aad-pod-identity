package aadpodidentity

import (
	"os/exec"

	"github.com/Azure/aad-pod-identity/test/e2e/azureassignedidentity"
	"github.com/Azure/aad-pod-identity/test/e2e/azureidentity"
	"github.com/Azure/aad-pod-identity/test/e2e/azureidentitybinding"
	"github.com/Azure/aad-pod-identity/test/e2e/deploy"
	"github.com/Azure/aad-pod-identity/test/e2e/util"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Kubernetes cluster using aad-pod-identity", func() {
	It("should have an Azure Assigned Identity", func() {
		err := azureidentity.CreateOnCluster(cfg.SubscriptionID, cfg.ResourceGroup, "test-identity", templateOutputPath)
		Expect(err).NotTo(HaveOccurred())

		err = azureidentitybinding.Create("test-identity", templateOutputPath)
		Expect(err).NotTo(HaveOccurred())

		err = deploy.Create(cfg.SubscriptionID, cfg.ResourceGroup, "demo", "test-identity", templateOutputPath)
		Expect(err).NotTo(HaveOccurred())

		ok, err := deploy.WaitOnReady("demo")
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(Equal(true))

		azureAssignedIdentity, err := azureassignedidentity.GetAll()
		Expect(err).NotTo(HaveOccurred())
		Expect(len(azureAssignedIdentity.AzureAssignedIdentities)).To(Equal(1))
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

		err = azureidentity.DeleteOnAzure(cfg.ResourceGroup, "test-identity")
		Expect(err).NotTo(HaveOccurred())

		cmd = exec.Command("kubectl", "delete", "deploy", "--all")
		util.PrintCommand(cmd)
		_, err = cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred())
	})
})
