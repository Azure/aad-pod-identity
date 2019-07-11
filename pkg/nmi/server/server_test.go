package server

import (
	"encoding/base64"
	"testing"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	"github.com/Azure/aad-pod-identity/pkg/k8s"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetTokenForMatchingIDBySP(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()

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
			ClientID:       "cid",
			ClientPassword: secretRef,
		},
	}
	podIDs := []aadpodid.AzureIdentity{podID}
	logger := log.WithError(nil)
	getTokenForMatchingID(kubeClient, logger, podID.Spec.ClientID, "https://management.azure.com/", podIDs)
}
