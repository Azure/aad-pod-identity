package k8s

import (
	"testing"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetSecret(t *testing.T) {
	secretName := "clientSecret"

	fakeClient := fake.NewSimpleClientset()

	secret := &v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: secretName}}
	fakeClient.CoreV1().Secrets("default").Create(secret)

	kubeClient := &KubeClient{ClientSet: fakeClient}

	secretRef := &v1.SecretReference{
		Name:      secretName,
		Namespace: "default",
	}
	retrievedSecret, err := kubeClient.GetSecret(secretRef)
	if err != nil {
		t.Fatalf("Error getting secret: %v", err)
	}
	if retrievedSecret.ObjectMeta.Name != secretName {
		t.Fatalf("Incorrect secret name: %v", retrievedSecret.ObjectMeta.Name)
	}
}
