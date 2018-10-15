package aadpodidentity

import (
	"os"
	"os/exec"
	"path"

	"github.com/Azure/aad-pod-identity/test/e2e/config"
	"github.com/Azure/aad-pod-identity/test/e2e/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

var (
	cfg                config.Config
	templateOutputPath = path.Join("template", "_output")
)

func TestAADPodIdentity(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AAD Pod Identity Suite")
}

var _ = BeforeSuite(func() {
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
	// Uninstall CRDs and delete MIC and NMI
	cmd := exec.Command("kubectl", "delete", "-f", "../../deploy/infra/deployment-rbac.yaml", "--ignore-not-found")
	util.PrintCommand(cmd)
	_, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())

	err = os.RemoveAll(templateOutputPath)
	Expect(err).NotTo(HaveOccurred())
})
