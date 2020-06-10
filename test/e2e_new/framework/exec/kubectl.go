// +build e2e_new

package exec

import (
	"fmt"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// KubectlApply executes "kubectl apply" given a list of arguments.
func KubectlApply() {
	// TODO
}

// KubectlExec executes "kubectl exec" given a list of arguments.
func KubectlExec(kubeconfigPath, podName, namespace string, args []string) {
	args = append([]string{
		"exec",
		fmt.Sprintf("--kubeconfig=%s", kubeconfigPath),
		fmt.Sprintf("--namespace=%s", namespace),
		podName,
		"--",
	}, args...)

	By(fmt.Sprintf("kubectl %s", strings.Join(args, " ")))

	cmd := exec.Command("kubectl", args...)
	stdoutStderr, err := cmd.CombinedOutput()
	fmt.Printf("%s", stdoutStderr)

	Expect(err).To(BeNil())
}
