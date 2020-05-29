package nmi

import (
	"context"
	"reflect"
	"testing"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *TestKubeClient) ListPodIdsWithBinding(podns string, labels map[string]string) ([]aadpodid.AzureIdentity, error) {
	identities, _ := c.azureIdentities.([]aadpodid.AzureIdentity)
	return identities, nil
}

func (c *TestKubeClient) GetPod(ns, name string) (v1.Pod, error) {
	return v1.Pod{}, nil
}

func TestGetIdentitiesManagedClient(t *testing.T) {
	cases := []struct {
		name                  string
		azureIdentities       []aadpodid.AzureIdentity
		clientID              string
		expectedErr           bool
		expectedAzureIdentity *aadpodid.AzureIdentity
		isNamespaced          bool
		podName               string
		podNamespace          string
	}{
		{
			name:                  "no azure identity found",
			azureIdentities:       nil,
			expectedErr:           true,
			expectedAzureIdentity: nil,
			podName:               "pod1",
			podNamespace:          "default",
		},
		{
			name: "clientID in request, but no matching identity",
			azureIdentities: []aadpodid.AzureIdentity{
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
			expectedErr:           true,
			expectedAzureIdentity: nil,
			podName:               "pod2",
			podNamespace:          "default",
			clientID:              "clientid1",
		},
		{
			name: "clientID in request, found matching identity",
			azureIdentities: []aadpodid.AzureIdentity{
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
			expectedErr: false,
			expectedAzureIdentity: &aadpodid.AzureIdentity{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "azid3",
					Namespace: "default",
				},
				Spec: aadpodid.AzureIdentitySpec{
					ClientID: "clientid3",
				},
			},
			podName:      "pod3",
			podNamespace: "default",
			clientID:     "clientid3",
		},
		{
			name: "no clientID in request, first matching identity in namespace returned",
			azureIdentities: []aadpodid.AzureIdentity{
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
						Name:      "azid3",
						Namespace: "default",
					},
					Spec: aadpodid.AzureIdentitySpec{
						ClientID: "clientid3",
					},
				},
			},
			expectedErr: false,
			expectedAzureIdentity: &aadpodid.AzureIdentity{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "azid2",
					Namespace: "default",
				},
				Spec: aadpodid.AzureIdentitySpec{
					ClientID: "clientid2",
				},
			},
			podName:      "pod4",
			podNamespace: "default",
		},
	}

	for i, tc := range cases {
		t.Log(i, tc.name)
		tokenClient, err := NewManagedTokenClient(NewTestKubeClient(tc.azureIdentities), Config{Namespaced: true})
		if err != nil {
			t.Fatalf("expected err to be nil, got: %v", err)
		}

		azIdentity, err := tokenClient.GetIdentities(context.Background(), tc.podNamespace, tc.podName, tc.clientID)
		if tc.expectedErr != (err != nil) {
			t.Fatalf("expected error: %v, got: %v", tc.expectedErr, err)
		}
		if !reflect.DeepEqual(tc.expectedAzureIdentity, azIdentity) {
			t.Fatalf("expected the azure identity to be equal")
		}
	}
}
