package k8s

import (
	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	v1 "k8s.io/api/core/v1"
)

// FakeClient implements Interface
type FakeClient struct {
}

// NewFakeClient new fake kubernetes api client
func NewFakeClient() (Client, error) {

	fakeClient := &FakeClient{}

	return fakeClient, nil
}

// GetPodName returns fake pod name
func (c *FakeClient) GetPodName(podip string) (podns, podname string, err error) {
	return "ns", "podname", nil
}

// ListPodIds for pod
func (c *FakeClient) ListPodIds(podns, podname string) (map[string][]aadpodid.AzureIdentity, error) {
	return nil, nil
}

// GetSecret returns secret the secretRef represents
func (c *FakeClient) GetSecret(secretRef *v1.SecretReference) (*v1.Secret, error) {
	return nil, nil
}
