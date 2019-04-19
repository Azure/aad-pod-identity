package server

import (
	"testing"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	"github.com/Azure/aad-pod-identity/pkg/k8s"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetTokenForMatchingIDBySP(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()

	secret := &v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "clientSecret"}, StringData: make(map[string]string)}
	secret.StringData["key1"] = "value1"
	fakeClient.CoreV1().Secrets("default").Create(secret)

	kubeClient := &k8s.KubeClient{ClientSet: fakeClient}

	secretRef := v1.SecretReference{
		Name:      "clientSecret",
		Namespace: "default",
	}

	podID := aadpodid.AzureIdentity{
		Spec: aadpodid.AzureIdentitySpec{
			Type:           aadpodid.ServicePrincipal,
			ClientID:       "clientID",
			ClientPassword: secretRef,
		},
	}

	podIDs := []aadpodid.AzureIdentity{podID}
	logger := log.WithError(nil)
	getTokenForMatchingID(kubeClient, logger, "clientID", "rqResource", &podIDs)
}
