// +build e2e

package e2e

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/aad-pod-identity/test/e2e/framework/exec"
	"github.com/Azure/aad-pod-identity/test/e2e/framework/mic"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("[PR] When liveness probe is enabled", func() {
	It("should pass liveness probe test", func() {
		pods := &corev1.PodList{}
		Eventually(func() (bool, error) {
			if err := kubeClient.List(context.TODO(), pods, client.InNamespace(corev1.NamespaceDefault)); err != nil {
				return false, err
			}

			return true, nil
		}, 10*time.Second, 1*time.Second).Should(BeTrue())

		var micPods, nmiPods []corev1.Pod
		for _, pod := range pods.Items {
			if strings.HasPrefix(pod.Name, "aad-pod-identity-mic") {
				micPods = append(micPods, pod)
			} else if strings.HasPrefix(pod.Name, "aad-pod-identity-nmi") {
				nmiPods = append(nmiPods, pod)
			}
		}

		micLeader := mic.GetLeader(mic.GetLeaderInput{
			Getter: kubeClient,
		})

		for _, micPod := range micPods {
			cmd := "clean-install wget"
			_, err := exec.KubectlExec(kubeconfigPath, micPod.Name, corev1.NamespaceDefault, strings.Split(cmd, " "))
			Expect(err).To(BeNil())

			cmd = "wget http://127.0.0.1:8080/healthz -q -O -"
			stdout, err := exec.KubectlExec(kubeconfigPath, micPod.Name, corev1.NamespaceDefault, strings.Split(cmd, " "))
			Expect(err).To(BeNil())
			if micPod.Name == micLeader.Name {
				By(fmt.Sprintf("Ensuring that %s's health probe is active", micPod.Name))
				Expect(strings.EqualFold(stdout, "Active")).To(BeTrue())
			} else {
				By(fmt.Sprintf("Ensuring that %s's health probe is not active", micPod.Name))
				Expect(strings.EqualFold(stdout, "Not Active")).To(BeTrue())
			}
		}

		for _, nmiPod := range nmiPods {
			cmd := "clean-install wget"
			_, err := exec.KubectlExec(kubeconfigPath, nmiPod.Name, corev1.NamespaceDefault, strings.Split(cmd, " "))
			Expect(err).To(BeNil())

			cmd = "wget http://127.0.0.1:8080/healthz -q -O -"
			stdout, err := exec.KubectlExec(kubeconfigPath, nmiPod.Name, corev1.NamespaceDefault, strings.Split(cmd, " "))
			Expect(err).To(BeNil())

			By(fmt.Sprintf("Ensuring that %s's health probe is active", nmiPod.Name))
			Expect(strings.EqualFold(stdout, "Active")).To(BeTrue())
		}
	})
})
