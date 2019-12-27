package crd

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity"
	api "k8s.io/api/core/v1"
)

type TestCrdClient struct {
	*Client
	assignedIDMap map[string]*aadpodid.AzureAssignedIdentity
	bindingMap    map[string]*aadpodid.AzureIdentityBinding
	idMap         map[string]*aadpodid.AzureIdentity
}

func NewTestCrdClient(config *rest.Config) *TestCrdClient {
	return &TestCrdClient{
		assignedIDMap: make(map[string]*aadpodid.AzureAssignedIdentity),
		bindingMap:    make(map[string]*aadpodid.AzureIdentityBinding),
		idMap:         make(map[string]*aadpodid.AzureIdentity),
	}
}

func (c *TestCrdClient) Start(exit <-chan struct{}) {
}

func (c *TestCrdClient) SyncCache(exit <-chan struct{}) {

}

func (c *TestCrdClient) CreateCrdWatchers(eventCh chan aadpodid.EventType) (err error) {
	return nil
}

func (c *TestCrdClient) RemoveAssignedIdentity(assignedIdentity *aadpodid.AzureAssignedIdentity) error {
	delete(c.assignedIDMap, assignedIdentity.Name)
	return nil
}

// This function is not used currently
// TODO: consider remove
func (c *TestCrdClient) CreateAssignedIdentity(assignedIdentity *aadpodid.AzureAssignedIdentity) error {
	c.assignedIDMap[assignedIdentity.Name] = assignedIdentity
	return nil
}

func (c *TestCrdClient) CreateBinding(bindingName string, idName string, selector string) {
	binding := &aadpodid.AzureIdentityBinding{
		ObjectMeta: v1.ObjectMeta{
			Name: bindingName,
		},
		Spec: aadpodid.AzureIdentityBindingSpec{
			AzureIdentity: idName,
			Selector:      selector,
		},
	}
	c.bindingMap[bindingName] = binding
}

func (c *TestCrdClient) CreateID(idName string, t aadpodid.IdentityType, rID string, cID string, cp *api.SecretReference, tID string, adRID string, adEpt string) {
	id := &aadpodid.AzureIdentity{
		ObjectMeta: v1.ObjectMeta{
			Name: idName,
		},
		Spec: aadpodid.AzureIdentitySpec{
			Type:       t,
			ResourceID: rID,
			ClientID:   cID,
			//ClientPassword: *cp,
			TenantID:     tID,
			ADResourceID: adRID,
			ADEndpoint:   adEpt,
		},
	}
	c.idMap[idName] = id
}

func (c *TestCrdClient) ListIds() (res *[]aadpodid.AzureIdentity, err error) {
	idList := make([]aadpodid.AzureIdentity, 0)
	for _, v := range c.idMap {
		idList = append(idList, *v)
	}
	return &idList, nil
}

func (c *TestCrdClient) ListBindings() (res *[]aadpodid.AzureIdentityBinding, err error) {
	bindingList := make([]aadpodid.AzureIdentityBinding, 0)
	for _, v := range c.bindingMap {
		bindingList = append(bindingList, *v)
	}
	return &bindingList, nil
}

func (c *TestCrdClient) ListAssignedIDs() (res *[]aadpodid.AzureAssignedIdentity, err error) {
	assignedIDList := make([]aadpodid.AzureAssignedIdentity, 0)
	for _, v := range c.assignedIDMap {
		assignedIDList = append(assignedIDList, *v)
	}
	return &assignedIDList, nil
}
