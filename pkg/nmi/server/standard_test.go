package server

import (
	"context"
	"encoding/base64"
	"reflect"
	"testing"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity"
	auth "github.com/Azure/aad-pod-identity/pkg/auth"
	"github.com/Azure/aad-pod-identity/pkg/k8s"
	"github.com/Azure/aad-pod-identity/pkg/metrics"
	"github.com/stretchr/testify/assert"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

type TestKubeClient struct {
	k8s.Client
	azureIdentities map[string][]aadpodid.AzureIdentity
	err             error
}

func NewTestKubeClient(azids map[string][]aadpodid.AzureIdentity) *TestKubeClient {
	return &TestKubeClient{
		azureIdentities: azids,
	}
}

func (c *TestKubeClient) setError(err error) {
	c.err = err
}

func (c *TestKubeClient) ListPodIds(podns, podname string) (map[string][]aadpodid.AzureIdentity, error) {
	return c.azureIdentities, c.err
}

func TestGetTokenForMatchingIDBySP(t *testing.T) {
	s := NewServer("default", false)
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
	s.KubeClient = kubeClient
	s.TokenClient = NewStandardTokenClient(kubeClient, 2, 1, 1, false)

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
	s.TokenClient.GetToken(context.Background(), podID.Spec.ClientID, "https://management.azure.com/", podID)
}

func TestGetIdentities(t *testing.T) {
	cases := []struct {
		name                  string
		azureIdentities       map[string][]aadpodid.AzureIdentity
		clientID              string
		expectedErr           bool
		expectedAzureIdentity *aadpodid.AzureIdentity
		isNamespaced          bool
		podName               string
		podNamespace          string
	}{
		{
			name:                  "no azure identities",
			azureIdentities:       make(map[string][]aadpodid.AzureIdentity),
			expectedErr:           true,
			expectedAzureIdentity: nil,
			podName:               "pod1",
			podNamespace:          "default",
		},
		{
			name: "azure identities with old 1.3/1.4, no request client id",
			azureIdentities: map[string][]aadpodid.AzureIdentity{
				"": []aadpodid.AzureIdentity{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "azid1",
							Namespace: "default",
						},
						Spec: aadpodid.AzureIdentitySpec{
							ClientID: "clientid1",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "azid2",
							Namespace: "default",
						},
						Spec: aadpodid.AzureIdentitySpec{
							ClientID: "clientid2",
						},
					},
				},
			},
			expectedErr: false,
			expectedAzureIdentity: &aadpodid.AzureIdentity{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "azid1",
					Namespace: "default",
				},
				Spec: aadpodid.AzureIdentitySpec{
					ClientID: "clientid1",
				},
			},
			podName:      "pod2",
			podNamespace: "default",
		},
		{
			name: "no request client id, found in created state only",
			azureIdentities: map[string][]aadpodid.AzureIdentity{
				aadpodid.AssignedIDCreated: []aadpodid.AzureIdentity{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "azid3",
							Namespace: "default",
						},
						Spec: aadpodid.AzureIdentitySpec{
							ClientID: "clientid3",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "azid4",
							Namespace: "default",
						},
						Spec: aadpodid.AzureIdentitySpec{
							ClientID: "clientid4",
						},
					},
				},
			},
			expectedAzureIdentity: &aadpodid.AzureIdentity{},
			expectedErr:           true,
			podName:               "pod3",
			podNamespace:          "default",
		},
		{
			name: "no request client id, found in assigned state",
			azureIdentities: map[string][]aadpodid.AzureIdentity{
				aadpodid.AssignedIDAssigned: []aadpodid.AzureIdentity{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "azid5",
							Namespace: "default",
						},
						Spec: aadpodid.AzureIdentitySpec{
							ClientID: "clientid5",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "azid6",
							Namespace: "default",
						},
						Spec: aadpodid.AzureIdentitySpec{
							ClientID: "clientid6",
						},
					},
				},
			},
			expectedErr: false,
			expectedAzureIdentity: &aadpodid.AzureIdentity{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "azid5",
					Namespace: "default",
				},
				Spec: aadpodid.AzureIdentitySpec{
					ClientID: "clientid5",
				},
			},
			podName:      "pod4",
			podNamespace: "default",
		},
		{
			name: "client id in request, no identity with same client id in assigned state",
			azureIdentities: map[string][]aadpodid.AzureIdentity{
				aadpodid.AssignedIDCreated: []aadpodid.AzureIdentity{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "azid1",
							Namespace: "default",
						},
						Spec: aadpodid.AzureIdentitySpec{
							ClientID: "clientid1",
						},
					},
				},
				aadpodid.AssignedIDAssigned: []aadpodid.AzureIdentity{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "azid2",
							Namespace: "default",
						},
						Spec: aadpodid.AzureIdentitySpec{
							ClientID: "clientid2",
						},
					},
				},
			},
			expectedErr:           true,
			expectedAzureIdentity: &aadpodid.AzureIdentity{},
			podName:               "pod5",
			podNamespace:          "default",
			clientID:              "clientid1",
		},
		{
			name: "client id in request, identity in same namespace returned with force namespace mode",
			azureIdentities: map[string][]aadpodid.AzureIdentity{
				aadpodid.AssignedIDAssigned: []aadpodid.AzureIdentity{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "azid2",
							Namespace: "testns",
						},
						Spec: aadpodid.AzureIdentitySpec{
							ClientID: "clientid2",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "azid1",
							Namespace: "default",
						},
						Spec: aadpodid.AzureIdentitySpec{
							ClientID: "clientid1",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "azid3",
							Namespace: "default",
						},
						Spec: aadpodid.AzureIdentitySpec{
							ClientID: "clientid3",
						},
					},
				},
			},
			expectedErr: false,
			expectedAzureIdentity: &aadpodid.AzureIdentity{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "azid2",
					Namespace: "testns",
				},
				Spec: aadpodid.AzureIdentitySpec{
					ClientID: "clientid2",
				},
			},
			podName:      "pod7",
			podNamespace: "testns",
			clientID:     "clientid2",
			isNamespaced: true,
		},
		{
			name: "no client id in request, identity in same namespace returned with force namespace mode",
			azureIdentities: map[string][]aadpodid.AzureIdentity{
				aadpodid.AssignedIDAssigned: []aadpodid.AzureIdentity{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "azid2",
							Namespace: "default",
						},
						Spec: aadpodid.AzureIdentitySpec{
							ClientID: "clientid2",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "azid1",
							Namespace: "default",
						},
						Spec: aadpodid.AzureIdentitySpec{
							ClientID: "clientid1",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "azid3",
							Namespace: "testns",
						},
						Spec: aadpodid.AzureIdentitySpec{
							ClientID: "clientid3",
						},
					},
				},
			},
			expectedErr: false,
			expectedAzureIdentity: &aadpodid.AzureIdentity{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "azid3",
					Namespace: "testns",
				},
				Spec: aadpodid.AzureIdentitySpec{
					ClientID: "clientid3",
				},
			},
			podName:      "pod8",
			podNamespace: "testns",
			isNamespaced: true,
		},
	}

	for i, tc := range cases {
		t.Log(i, tc.name)
		tokenClient := NewStandardTokenClient(NewTestKubeClient(tc.azureIdentities), 2, 1, 1, tc.isNamespaced)

		azIdentity, err := tokenClient.GetIdentities(context.Background(), tc.podNamespace, tc.podName, tc.clientID)
		assert.Equal(t, err != nil, tc.expectedErr)
		assert.True(t, reflect.DeepEqual(tc.expectedAzureIdentity, azIdentity))
	}
}
