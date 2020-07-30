// +build e2e

package e2e

import (
	"strings"

	"github.com/Azure/aad-pod-identity/test/e2e/framework/exec"
	"github.com/Azure/aad-pod-identity/test/e2e/framework/pod"
	corev1 "k8s.io/api/core/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("When blocking pods from accessing Instance Metadata Service", func() {
	It("should receive a HTTP 403 response when contacting /metadata/instance endpoint", func() {
		nmiPods := pod.List(pod.ListInput{
			Lister:    kubeClient,
			Namespace: corev1.NamespaceDefault,
			Labels: map[string]string{
				"app.kubernetes.io/component": "nmi",
			},
		})

		for _, nmiPod := range nmiPods.Items {
			cmd := "clean-install wget"
			_, err := exec.KubectlExec(kubeconfigPath, nmiPod.Name, corev1.NamespaceDefault, strings.Split(cmd, " "))
			Expect(err).To(BeNil())

			cmd = "wget 127.0.0.1:2579/metadata/instance"
			stdout, err := exec.KubectlExec(kubeconfigPath, nmiPod.Name, corev1.NamespaceDefault, strings.Split(cmd, " "))
			Expect(err).NotTo(BeNil())
			Expect(strings.Contains(stdout, "ERROR 403: Forbidden")).To(BeTrue())
		}
	})
})
