package k8s

import (
	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
)

// FakeClient implements Interface
type FakeClient struct {
}

// NewFakeClient new fake kubernetes api client
func NewFakeClient() (Client, error) {

	fakeClient := &FakeClient{}

	return fakeClient, nil
}

// GetNodeIP returns fake pod ip
func (c *FakeClient) GetNodeIP(hostname string) (nodeip string, err error) {
	return "127.0.0.1", nil
}

// GetPodCidr returns fake pod cidr
func (c *FakeClient) GetPodCidr(hostname string) (podcidr string, err error) {
	return "127.0.0.1/32", nil
}

// GetPodName returns fake pod name
func (c *FakeClient) GetPodName(podip string) (podns, podname string, err error) {
	return "ns", "podname", nil
}

// GetAzureAssignedIdentity returns fake pod name
func (c *FakeClient) GetAzureAssignedIdentity(podns, podname string) (azID *aadpodid.AzureIdentity, err error) {
	azID = &aadpodid.AzureIdentity{}
	err = nil
	return azID, err
}
