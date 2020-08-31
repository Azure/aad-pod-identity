// +build e2e

package iptables

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/aad-pod-identity/test/e2e/framework"
	"github.com/Azure/aad-pod-identity/test/e2e/framework/exec"
	"github.com/Azure/aad-pod-identity/test/e2e/framework/pod"

	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	busybox = "busybox"
)

// WaitForRulesInput is the input for WaitForRules.
type WaitForRulesInput struct {
	Creator         framework.Creator
	Getter          framework.Getter
	Lister          framework.Lister
	Namespace       string
	KubeconfigPath  string
	CreateDaemonSet bool
	ShouldExist     bool
}

// WaitForRules waits for iptables rules to exist / get deleted.
func WaitForRules(input WaitForRulesInput) {
	Expect(input.Creator).NotTo(BeNil(), "input.Creator is required for iptables.WaitForRules")
	Expect(input.Getter).NotTo(BeNil(), "input.Getter is required for iptables.WaitForRules")
	Expect(input.Lister).NotTo(BeNil(), "input.Lister is required for iptables.WaitForRules")
	Expect(input.Namespace).NotTo(BeEmpty(), "input.Namespace is required for iptables.WaitForRules")
	Expect(input.KubeconfigPath).NotTo(BeEmpty(), "input.KubeconfigPath is required for iptables.WaitForRules")

	if input.CreateDaemonSet {
		busybox := &appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      busybox,
				Namespace: input.Namespace,
			},
			Spec: appsv1.DaemonSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"component": busybox,
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"component": busybox,
						},
					},
					Spec: corev1.PodSpec{
						HostNetwork:                   true,
						DNSPolicy:                     corev1.DNSClusterFirstWithHostNet,
						TerminationGracePeriodSeconds: to.Int64Ptr(int64(0)),
						Containers: []corev1.Container{
							{
								Name:  busybox,
								Image: "alpine:3.11.5",
								Stdin: true,
								Command: []string{
									"sleep",
									"3600",
								},
								SecurityContext: &corev1.SecurityContext{
									Capabilities: &corev1.Capabilities{
										Add: []corev1.Capability{
											"NET_ADMIN",
										},
									},
								},
							},
						},
						NodeSelector: map[string]string{
							corev1.LabelOSStable: "linux",
						},
					},
				},
			},
		}

		Eventually(func() error {
			return input.Creator.Create(context.TODO(), busybox)
		}, framework.CreateTimeout, framework.CreatePolling).Should(Succeed())
	}

	Eventually(func() (bool, error) {
		// ensure that there is no nmi before checking whether iptables rules are cleaned up
		if !input.ShouldExist {
			nmiPods := pod.List(pod.ListInput{
				Lister:    input.Lister,
				Namespace: framework.NamespaceKubeSystem,
				Labels: map[string]string{
					"app.kubernetes.io/component": "nmi",
				},
			})

			if len(nmiPods.Items) > 0 {
				return false, nil
			}
		}

		ds := &appsv1.DaemonSet{}
		if err := input.Getter.Get(context.TODO(), client.ObjectKey{Name: busybox, Namespace: input.Namespace}, ds); err != nil {
			return false, err
		}

		if ds.Status.NumberReady == 0 || ds.Status.NumberReady != ds.Status.DesiredNumberScheduled {
			return false, nil
		}

		pods := &corev1.PodList{}
		if err := input.Lister.List(context.TODO(), pods, client.InNamespace(input.Namespace)); err != nil {
			return false, err
		}

		for _, p := range pods.Items {
			if input.ShouldExist {
				By(fmt.Sprintf("Checking if iptables rules exist in %s", p.Spec.NodeName))
			} else {
				By(fmt.Sprintf("Checking if iptables rules are removed from %s", p.Spec.NodeName))
			}

			for _, cmd := range []struct {
				command          string
				noError          bool
				expectedErrorMsg string
			}{
				{
					command: "apk add iptables",
					noError: true,
				},
				{
					command:          "iptables -t nat --check PREROUTING -j aad-metadata",
					noError:          input.ShouldExist,
					expectedErrorMsg: "Couldn't load target `aad-metadata':No such file or directory",
				},
				{
					command:          "iptables -t nat -L aad-metadata",
					noError:          input.ShouldExist,
					expectedErrorMsg: "No chain/target/match by that name",
				},
			} {
				stderr, err := exec.KubectlExec(input.KubeconfigPath, p.Name, input.Namespace, strings.Split(cmd.command, " "))
				if cmd.noError {
					Expect(err).To(BeNil())
				} else {
					Expect(err).NotTo(BeNil())
					Expect(strings.Contains(stderr, cmd.expectedErrorMsg)).To(BeTrue())
				}
			}
		}

		return true, nil
	}, framework.WaitTimeout, framework.WaitPolling).Should(BeTrue())
}
