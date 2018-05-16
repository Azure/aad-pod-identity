package crd

import (
	"fmt"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
)

const (
	CRDGroup   = "aadpodidentity.k8s.io"
	CRDVersion = "v1"
)

type CrdClient struct {
	rest *rest.RESTClient
}

func NewCRDClient(config *rest.Config) (*CrdClient, error) {
	crdconfig := *config
	crdconfig.GroupVersion = &schema.GroupVersion{Group: CRDGroup, Version: CRDVersion}
	crdconfig.APIPath = "/apis"
	crdconfig.ContentType = runtime.ContentTypeJSON
	s := runtime.NewScheme()
	s.AddKnownTypes(*crdconfig.GroupVersion,
		&aadpodid.AzureIdentity{},
		&aadpodid.AzureIdentityList{},
		&aadpodid.AzureIdentityBinding{},
		&aadpodid.AzureIdentityBindingList{},
		&aadpodid.AzureAssignedIdentity{},
		&aadpodid.AzureAssignedIdentityList{})
	crdconfig.NegotiatedSerializer = serializer.DirectCodecFactory{
		CodecFactory: serializer.NewCodecFactory(s)}

	//Client interacting with our CRDs
	restClient, err := rest.RESTClientFor(&crdconfig)
	if err != nil {
		glog.Error(err)
		return nil, err
	}
	return &CrdClient{
		rest: restClient,
	}, nil
}

func (c *CrdClient) CreateAssignIdentity(idName string, podName string, podNameSpace string, nodeName string) error {
	glog.Infof("Got id %s to assign", idName)
	// Create a new AzureAssignedIdentity which maps the relationship between
	// id and pod
	assignedID := &aadpodid.AzureAssignedIdentity{
		ObjectMeta: v1.ObjectMeta{
			Name: "azureassignedidentities.aadpodidentity.k8s.io",
		},
		Spec: aadpodid.AzureAssignedIdentitySpec{
			AzureIdentityRef: idName,
			Pod:              podName,
			PodNamespace:     podNameSpace,
			NodeName:         nodeName,
		},
		Status: aadpodid.AzureAssignedIdentityStatus{
			AvailableReplicas: 1,
		},
	}

	var res aadpodid.AzureAssignedIdentity
	// TODO: Ensure that the status reflects the corresponding
	err := c.rest.Post().Namespace("default").Resource("azureassignedidentities").Body(assignedID).Do().Into(&res)
	if err != nil {
		glog.Error(err)
		return err
	}

	//TODO: Update the status of the assign identity to indicate that the node assignment got done.
	return nil
}

func (c *CrdClient) ListBindings() (res *aadpodid.AzureIdentityBindingList, err error) {
	var ret aadpodid.AzureIdentityBindingList
	err = c.rest.Get().Namespace("default").Resource("azureidentitybindings").Do().Into(&ret)
	if err != nil {
		glog.Error(err)
		return nil, err
	}
	//glog.Infof("%+v", ret)
	return &ret, nil

}

func (c *CrdClient) Lookup(idName string) (res *aadpodid.AzureIdentity, err error) {
	ids, err := c.ListIds()
	if err != nil {
		return nil, err
	}
	for _, v := range ids.Items {
		glog.Infof("%+v", v)
		glog.Infof("Looking for idName %s in %s", idName, v.Name)
		if v.Name == idName {
			return &v, nil
		}
	}
	return nil, fmt.Errorf("Lookup of %s failed", idName)
}

func (c *CrdClient) ListIds() (res *aadpodid.AzureIdentityList, err error) {
	var ret aadpodid.AzureIdentityList
	err = c.rest.Get().Namespace("default").Resource("azureidentities").Do().Into(&ret)
	if err != nil {
		glog.Error(err)
		return nil, err
	}
	return &ret, nil
}

//GetUserAssignedMSI - given a pod with pod name space
func (c *CrdClient) GetUserAssignedMSI(podns, podname string) (userMSIClientID *string, err error) {
	var azAssignedIDList aadpodid.AzureAssignedIdentityList
	err = c.rest.Get().Namespace("default").Resource("azureassignedidentities").Do().Into(&azAssignedIDList)
	if err != nil {
		glog.Error(err)
		return nil, err
	}

	for _, v := range azAssignedIDList.Items {
		if v.Spec.Pod == podname {
			azID, err := c.Lookup(v.Spec.AzureIdentityRef)
			if err != nil {
				return nil, err
			}
			return &azID.Spec.ClientID, nil
		}
	}
	// We have not yet returned, so pass up an error
	return nil, fmt.Errorf("AzureIdentity not found for pod:%s in namespace:%s", podname, podns)
}
