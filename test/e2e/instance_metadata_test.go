//go:build e2e
// +build e2e

package e2e

import (
	"strings"

	"github.com/Azure/aad-pod-identity/test/e2e/framework"
	"github.com/Azure/aad-pod-identity/test/e2e/framework/exec"
	"github.com/Azure/aad-pod-identity/test/e2e/framework/iptables"
	"github.com/Azure/aad-pod-identity/test/e2e/framework/pod"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("When sending a request to Instance Metadata Service", func() {
	It("should receive proper HTTP status code when contacting various endpoints", func() {
		nmiPods := pod.List(pod.ListInput{
			Lister:    kubeClient,
			Namespace: framework.NamespaceKubeSystem,
			Labels: map[string]string{
				"app.kubernetes.io/component": "nmi",
			},
		})

		for _, nmiPod := range nmiPods.Items {
			busyboxPod, namespace := iptables.GetBusyboxPodByNode(iptables.GetBusyboxPodByNodeInput{
				NodeName: nmiPod.Spec.NodeName,
			})

			cmd := "clean-install curl"
			_, err := exec.KubectlExec(kubeconfigPath, busyboxPod.Name, namespace, strings.Split(cmd, " "))
			Expect(err).To(BeNil())

			cmd = "curl -f 127.0.0.1:2579/doesnotexist -H Metadata:true"
			stdout, err := exec.KubectlExec(kubeconfigPath, busyboxPod.Name, namespace, strings.Split(cmd, " "))
			Expect(err).NotTo(BeNil())
			Expect(strings.Contains(stdout, "404 Not Found")).To(BeTrue())

			if config.MetadataHeaderRequired {
				cmd = "curl 127.0.0.1:2579/doesnotexist"
				stdout, _ := exec.KubectlExec(kubeconfigPath, busyboxPod.Name, namespace, strings.Split(cmd, " "))
				Expect(strings.Contains(stdout, "Required metadata header not specified")).To(BeTrue())
			}

			if config.BlockInstanceMetadata {
				cmd = "curl -f -H 'Metadata:true' 127.0.0.1:2579/metadata/instance"
				stdout, err := exec.KubectlExec(kubeconfigPath, busyboxPod.Name, namespace, strings.Split(cmd, " "))
				Expect(err).NotTo(BeNil())
				Expect(strings.Contains(stdout, "403 Forbidden")).To(BeTrue())
			}
		}
	})
})
