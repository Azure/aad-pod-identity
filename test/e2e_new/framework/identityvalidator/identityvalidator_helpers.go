// +build e2e_new

package identityvalidator

import (
	"context"
	"fmt"
	"time"

	aadpodv1 "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	"github.com/Azure/aad-pod-identity/test/e2e_new/framework"
	"github.com/Azure/aad-pod-identity/test/e2e_new/framework/exec"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	createTimeout = 10 * time.Second
	createPolling = 1 * time.Second

	deleteTimeout = 10 * time.Second
	deletePolling = 1 * time.Second
)

// CreateInput is the input for Create.
type CreateInput struct {
	Creator         framework.Creator
	Config          *framework.Config
	Namespace       string
	IdentityBinding string
}

// Create creates an identity-validator pod.
func Create(input CreateInput) *corev1.Pod {
	Expect(input.Creator).NotTo(BeNil(), "input.Creator is required for IdentityValidator.Create")
	Expect(input.Config).NotTo(BeNil(), "input.Config is required for IdentityValidator.Create")
	Expect(input.Namespace).NotTo(BeEmpty(), "input.Namespace is required for IdentityValidator.Create")
	Expect(input.IdentityBinding).NotTo(BeEmpty(), "input.IdentityBinding is required for IdentityValidator.Create")

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "identity-validator-",
			Namespace:    input.Namespace,
			Labels: map[string]string{
				aadpodv1.CRDLabelKey: input.IdentityBinding,
			},
		},
		Spec: corev1.PodSpec{
			// InitContainers: []corev1.Container{
			// 	{
			// 		Name:  "init-myservice",
			// 		Image: "microsoft/azure-cli:latest",
			// 		Command: []string{
			// 			"sh",
			// 			"-c",
			// 			"az login --identity",
			// 		},
			// 	},
			// },
			Containers: []corev1.Container{
				{
					Name:  "identity-validator",
					Image: fmt.Sprintf("%s/identityvalidator:%s", input.Config.Registry, input.Config.IdentityValidatorVersion),
					Command: []string{
						"sleep",
						"3600",
					},
					Env: []corev1.EnvVar{
						{
							Name: "E2E_TEST_POD_NAME",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath: "metadata.name",
								},
							},
						},
						{
							Name: "E2E_TEST_POD_NAMESPACE",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath: "metadata.namespace",
								},
							},
						},
						{
							Name: "E2E_TEST_POD_IP",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath: "status.podIP",
								},
							},
						},
					},
				},
			},
		},
	}

	Eventually(func() error {
		return input.Creator.Create(context.TODO(), pod)
	}, createTimeout, createPolling).Should(Succeed())

	return pod
}

// ValidateInput is the input for Validate.
type ValidateInput struct {
	Config           *framework.Config
	KubeconfigPath   string
	PodName          string
	Namespace        string
	IdentityClientID string
}

// Validate performs validation against an identity-validator pod.
func Validate(input ValidateInput) {
	Expect(input.Config).NotTo(BeNil(), "input.Config is required for IdentityValidator.Validate")
	Expect(input.KubeconfigPath).NotTo(BeEmpty(), "input.KubeconfigPath is required for IdentityValidator.Validate")
	Expect(input.PodName).NotTo(BeEmpty(), "input.PodName is required for IdentityValidator.Validate")
	Expect(input.Namespace).NotTo(BeEmpty(), "input.Namespace is required for IdentityValidator.Validate")
	Expect(input.IdentityClientID).NotTo(BeEmpty(), "input.IdentityClientID is required for IdentityValidator.Validate")

	args := []string{
		"identityvalidator",
		"--subscription-id",
		input.Config.SubscriptionID,
		"--resource-group",
		input.Config.IdentityResourceGroup,
		"--identity-client-id",
		input.IdentityClientID,
		"--keyvault-name",
		input.Config.KeyvaultName,
		"--keyvault-secret-name",
		input.Config.KeyvaultSecretName,
		"--keyvault-secret-version",
		input.Config.KeyvaultSecretVersion,
	}
	exec.KubectlExec(input.KubeconfigPath, input.PodName, input.Namespace, args)
}
