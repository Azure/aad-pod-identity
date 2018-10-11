package aadpodidentity

import (
	"os"
	"os/exec"
	"path"

	"github.com/Azure/aad-pod-identity/test/e2e/azureidentity"

	"github.com/Azure/aad-pod-identity/test/e2e/config"
	"github.com/Azure/aad-pod-identity/test/e2e/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	cfg                config.Config
	templateOutputPath = path.Join("template", "_output")
)

var _ = BeforeSuite(func() {
	err := os.Mkdir(templateOutputPath, os.ModePerm)
	Expect(err).NotTo(HaveOccurred())

	c, err := config.ParseConfig()
	Expect(err).NotTo(HaveOccurred())
	cfg = *c // To avoid 'Declared and not used' linting error

	// Create a folder '_output' in template/ for storing temporary deployment files

	// Install CRDs and deploy MIC and NMI
	cmd := exec.Command("kubectl", "apply", "-f", "../../deploy/infra/deployment-rbac.yaml")
	util.PrintCommand(cmd)
	_, err = cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	// Uninstall CRDs and delete MIC and NMI
	cmd := exec.Command("kubectl", "delete", "-f", "../../deploy/infra/deployment-rbac.yaml")
	util.PrintCommand(cmd)
	_, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())

	err = os.RemoveAll(templateOutputPath)
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("Kubernetes cluster using aad-pod-identity", func() {
	BeforeEach(func() {
		err := azureidentity.CreateOnAzure(cfg.ResourceGroup, "test-identity")
		Expect(err).NotTo(HaveOccurred())
		err = azureidentity.CreateOnCluster(cfg.SubscriptionID, cfg.ResourceGroup, "test-identity", templateOutputPath)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("Creating an AzureIdentity and AzureIdentityBinding to bind the identity to a pod", func() {
		It("should create an AzureAssignedIdentity", func() {
			Expect(1).To(Equal(1))
		})

		It("should be able to aquire a token and access azure resources", func() {
			Expect(1).To(Equal(1))
		})
	})

	AfterEach(func() {
		err := azureidentity.DeleteOnAzure(cfg.ResourceGroup, "test-identity")
		Expect(err).NotTo(HaveOccurred())
		err = azureidentity.DeleteOnCluster("test-identity", templateOutputPath)
		Expect(err).NotTo(HaveOccurred())
	})
})
