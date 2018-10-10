package aadpodidentity

import (
	"os/exec"

	"github.com/Azure/aad-pod-identity/test/e2e/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestAADPodIdentity(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AAD Pod Identity Suite")
}

var _ = BeforeSuite(func() {
	// Install CRDs and deploy MIC and NMI
	cmd := exec.Command("kubectl", "apply", "-f", "../../deploy/infra/deployment-rbac.yaml")
	util.PrintCommand(cmd)
	_, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	// Uninstall CRDs and delete MIC and NMI
	cmd := exec.Command("kubectl", "delete", "-f", "../../deploy/infra/deployment-rbac.yaml")
	util.PrintCommand(cmd)
	_, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())
})
