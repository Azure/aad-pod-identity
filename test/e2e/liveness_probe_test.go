// +build e2e

package e2e

import (
	"fmt"
	"strings"

	"github.com/Azure/aad-pod-identity/test/e2e/framework"
	"github.com/Azure/aad-pod-identity/test/e2e/framework/exec"
	"github.com/Azure/aad-pod-identity/test/e2e/framework/pod"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("When liveness probe is enabled", func() {
	It("should pass liveness probe test", func() {
		nmiPods := pod.List(pod.ListInput{
			Lister:    kubeClient,
			Namespace: framework.NamespaceKubeSystem,
			Labels: map[string]string{
				"app.kubernetes.io/component": "nmi",
			},
		})

		for _, nmiPod := range nmiPods.Items {
			cmd := "clean-install wget"
			_, err := exec.KubectlExec(kubeconfigPath, nmiPod.Name, framework.NamespaceKubeSystem, strings.Split(cmd, " "))
			Expect(err).To(BeNil())

			cmd = "wget http://127.0.0.1:8085/healthz -q -O -"
			stdout, err := exec.KubectlExec(kubeconfigPath, nmiPod.Name, framework.NamespaceKubeSystem, strings.Split(cmd, " "))
			Expect(err).To(BeNil())

			By(fmt.Sprintf("Ensuring that %s's health probe is active", nmiPod.Name))
			Expect(strings.EqualFold(stdout, "Active")).To(BeTrue())
		}
	})
})
