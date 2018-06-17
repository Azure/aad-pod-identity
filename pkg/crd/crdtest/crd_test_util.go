package crdtest

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	"github.com/Azure/aad-pod-identity/pkg/crd"
	api "k8s.io/api/core/v1"
)

type TestCrdClient struct {
	*crd.Client
	assignedIDMap map[string]*aadpodid.AzureAssignedIdentity
	bindingMap    map[string]*aadpodid.AzureIdentityBinding
	idMap         map[string]*aadpodid.AzureIdentity
}

func NewTestCrdClient(config *rest.Config) *TestCrdClient {
	return &TestCrdClient{
		assignedIDMap: make(map[string]*aadpodid.AzureAssignedIdentity, 0),
		bindingMap:    make(map[string]*aadpodid.AzureIdentityBinding, 0),
		idMap:         make(map[string]*aadpodid.AzureIdentity, 0),
	}
}

func (c *TestCrdClient) Start(exit <-chan struct{}) {
}

func (c *TestCrdClient) SyncCache(exit <-chan struct{}) {

}

func (c *TestCrdClient) CreateCrdWatchers(eventCh chan aadpodid.EventType) (err error) {
	return nil
}

func (c *TestCrdClient) RemoveAssignedIdentity(name string) error {
	delete(c.assignedIDMap, name)
	return nil
}

func (c *TestCrdClient) CreateAssignedIdentity(name string, binding *aadpodid.AzureIdentityBinding, id *aadpodid.AzureIdentity, podName string, podNameSpace string, nodeName string) error {
	assignedID := &aadpodid.AzureAssignedIdentity{
		ObjectMeta: v1.ObjectMeta{
			Name: name,
		},
		Spec: aadpodid.AzureAssignedIdentitySpec{
			Pod:              podName,
			PodNamespace:     podNameSpace,
			NodeName:         nodeName,
			AzureBindingRef:  binding,
			AzureIdentityRef: id,
		},
	}
	c.assignedIDMap[name] = assignedID
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

func (c *TestCrdClient) CreateId(idName string, t aadpodid.IdentityType, rId string, cId string, cp *api.SecretReference, tId string, adRId string, adEpt string) {
	id := &aadpodid.AzureIdentity{
		ObjectMeta: v1.ObjectMeta{
			Name: idName,
		},
		Spec: aadpodid.AzureIdentitySpec{
			Type:       t,
			ResourceID: rId,
			ClientID:   cId,
			//ClientPassword: *cp,
			TenantID:     tId,
			ADResourceID: adRId,
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
	assignedIdList := make([]aadpodid.AzureAssignedIdentity, 0)
	for _, v := range c.assignedIDMap {
		assignedIdList = append(assignedIdList, *v)
	}
	return &assignedIdList, nil
}
