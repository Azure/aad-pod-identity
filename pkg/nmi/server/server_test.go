package server

import (
	"encoding/base64"
	"testing"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	auth "github.com/Azure/aad-pod-identity/pkg/auth"
	"github.com/Azure/aad-pod-identity/pkg/k8s"
	"github.com/Azure/aad-pod-identity/pkg/metrics"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetTokenForMatchingIDBySP(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	reporter, err := metrics.NewReporter()
	if err != nil {
		t.Fatalf("expected nil error, got: %+v", err)
	}
	auth.InitReporter(reporter)

	secret := &v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "clientSecret"}, Data: make(map[string][]byte)}
	val, _ := base64.StdEncoding.DecodeString("YWJjZA==")
	secret.Data["key1"] = val
	fakeClient.CoreV1().Secrets("default").Create(secret)

	kubeClient := &k8s.KubeClient{ClientSet: fakeClient}

	secretRef := v1.SecretReference{
		Name:      "clientSecret",
		Namespace: "default",
	}

	podID := aadpodid.AzureIdentity{
		Spec: aadpodid.AzureIdentitySpec{
			Type:           aadpodid.ServicePrincipal,
			TenantID:       "tid",
			ClientID:       "aabc0000-a83v-9h4m-000j-2c0a66b0c1f9",
			ClientPassword: secretRef,
		},
	}
	podIDs := []aadpodid.AzureIdentity{podID}
	getTokenForMatchingID(kubeClient, podID.Spec.ClientID, "https://management.azure.com/", podIDs)
}
