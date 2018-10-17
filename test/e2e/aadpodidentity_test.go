package aadpodidentity

import (
	"fmt"
	"os"
	"os/exec"
	"path"

	"github.com/Azure/aad-pod-identity/test/e2e/azureassignedidentity"
	"github.com/Azure/aad-pod-identity/test/e2e/azureidentity"
	"github.com/Azure/aad-pod-identity/test/e2e/azureidentitybinding"
	"github.com/Azure/aad-pod-identity/test/e2e/config"
	"github.com/Azure/aad-pod-identity/test/e2e/deploy"
	"github.com/Azure/aad-pod-identity/test/e2e/pod"
	"github.com/Azure/aad-pod-identity/test/e2e/util"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	cfg                config.Config
	templateOutputPath = path.Join("template", "_output")
)

var _ = BeforeSuite(func() {
	fmt.Println("Setting up the test suite environment...")

	// Create a folder '_output' in template/ for storing temporary deployment files
	err := os.Mkdir(templateOutputPath, os.ModePerm)
	Expect(err).NotTo(HaveOccurred())

	c, err := config.ParseConfig()
	Expect(err).NotTo(HaveOccurred())
	cfg = *c // To avoid 'Declared and not used' linting error

	// Install CRDs and deploy MIC and NMI
	cmd := exec.Command("kubectl", "apply", "-f", "../../deploy/infra/deployment-rbac.yaml")
	util.PrintCommand(cmd)
	_, err = cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	fmt.Println("Tearing down the test suite environment...")

	// Uninstall CRDs and delete MIC and NMI
	cmd := exec.Command("kubectl", "delete", "-f", "../../deploy/infra/deployment-rbac.yaml", "--ignore-not-found")
	util.PrintCommand(cmd)
	_, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())

	err = os.RemoveAll(templateOutputPath)
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("Kubernetes cluster using aad-pod-identity", func() {
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

		list, err := azureassignedidentity.GetAll()
		Expect(err).NotTo(HaveOccurred())
		Expect(list.Items).To(HaveLen(1))
		Expect(list.Items[0].Spec.Pod).NotTo(Equal(""))

		cmd := exec.Command("kubectl", "exec", list.Items[0].Spec.Pod, "--", "identityvalidator", "--subscriptionid", cfg.SubscriptionID, "--clientid", clientID, "--resourcegroup", cfg.ResourceGroup)
		_, err = cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred())
	})

	It("should not pass the identity validating test if the AzureIdentity is deleted", func() {
		err := azureidentity.DeleteOnCluster("test-identity", templateOutputPath)
		Expect(err).NotTo(HaveOccurred())

		list, err := azureidentity.GetAll()
		Expect(err).NotTo(HaveOccurred())
		Expect(list.Items).To(HaveLen(0))

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
		Expect(list.Items).To(HaveLen(0))

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
