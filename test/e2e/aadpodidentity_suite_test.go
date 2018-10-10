package aadpodidentity_test

import (
	"fmt"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestAADPodIdentity(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AAD Pod Identity Suite")
}

func printCommand(cmd *exec.Cmd) {
	fmt.Printf("\n$ %s\n", strings.Join(cmd.Args, " "))
}

var _ = BeforeSuite(func() {
	// Install CRDs and deploy MIC and NMI
	cmd := exec.Command("kubectl", "apply", "-f", "../../deploy/infra/deployment-rbac.yaml")
	printCommand(cmd)
	_, err := cmd.CombinedOutput()
	if err != nil {
		panic(err)
	}
})

var _ = AfterSuite(func() {
	// / uninstall CRDs and delete MIC and NMI
	cmd := exec.Command("kubectl", "delete", "-f", "../../deploy/infra/deployment-rbac.yaml")
	printCommand(cmd)
	_, err := cmd.CombinedOutput()
	if err != nil {
		panic(err)
	}
})
