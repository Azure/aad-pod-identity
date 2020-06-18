// +build e2e_new

package iptables

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/aad-pod-identity/test/e2e_new/framework"
	"github.com/Azure/aad-pod-identity/test/e2e_new/framework/exec"

	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	createTimeout = 10 * time.Second
	createPolling = 1 * time.Second

	waitTimeout = 1 * time.Minute
	waitPolling = 10 * time.Second
)

// WaitForRulesInput is the input for WaitForRules.
type WaitForRulesInput struct {
	Creator         framework.Creator
	Lister          framework.Lister
	Namespace       string
	KubeconfigPath  string
	CreateDaemonSet bool
	ShouldExist     bool
}

// WaitForRules waits for iptables rules to exist / get deleted.
func WaitForRules(input WaitForRulesInput) {
	Expect(input.Creator).NotTo(BeNil(), "input.Creator is required for iptables.WaitForRules")
	Expect(input.Lister).NotTo(BeNil(), "input.Lister is required for iptables.WaitForRules")
	Expect(input.Namespace).NotTo(BeEmpty(), "input.Namespace is required for iptables.WaitForRules")
	Expect(input.KubeconfigPath).NotTo(BeEmpty(), "input.KubeconfigPath is required for iptables.WaitForRules")

	if input.CreateDaemonSet {
		busybox := &appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "busybox",
				Namespace: input.Namespace,
			},
			Spec: appsv1.DaemonSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"component": "busybox",
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"component": "busybox",
						},
					},
					Spec: corev1.PodSpec{
						HostNetwork: true,
						Containers: []corev1.Container{
							{
								Name:  "busybox",
								Image: "alpine:3.11.5",
								Stdin: true,
								Command: []string{
									"sleep",
									"3600",
								},
								SecurityContext: &corev1.SecurityContext{
									Privileged: to.BoolPtr(true),
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
		}, createTimeout, createPolling).Should(Succeed())
	}

	Eventually(func() (bool, error) {
		pods := &corev1.PodList{}
		if err := input.Lister.List(context.TODO(), pods, client.InNamespace(input.Namespace)); err != nil {
			return false, err
		}

		for _, p := range pods.Items {
			if p.Status.Phase != corev1.PodRunning {
				return false, nil
			}

			if input.ShouldExist {
				By(fmt.Sprintf("Checking if iptables rules exist in %s", p.Spec.NodeName))
			} else {
				By(fmt.Sprintf("Checking if iptables rules are removed from %s", p.Spec.NodeName))
			}

			for _, cmd := range []struct {
				command string
				noError bool
			}{
				{
					command: "apk add iptables",
					noError: true,
				},
				{
					command: "iptables -t nat --check PREROUTING -j aad-metadata",
					noError: input.ShouldExist,
				},
				{
					command: "iptables -t nat -L aad-metadata",
					noError: input.ShouldExist,
				},
			} {
				err := exec.KubectlExec(input.KubeconfigPath, p.Name, input.Namespace, strings.Split(cmd.command, " "))
				if cmd.noError {
					Expect(err).To(BeNil())
				} else {
					Expect(err).NotTo(BeNil())
				}
			}
		}

		return true, nil
	}, waitTimeout, waitPolling).Should(BeTrue())
}
