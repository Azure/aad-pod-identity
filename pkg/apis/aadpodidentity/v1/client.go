package aadpodidentity

import (
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	CRDGroup   = "aadpodidentity.k8s.io"
	CRDVersion = "v1"
)

// Client has the required pointers to talk to the api server
// and interact with the CRD related datastructure.
type Client struct {
	CRDClient    *rest.RESTClient
	ClientSet    *kubernetes.Clientset
	K8sInformers informers.SharedInformerFactory
}

func NewAadPodIdentityCrdClient(config *rest.Config) (*Client, error) {
	crdconfig := *config
	crdconfig.GroupVersion = &schema.GroupVersion{Group: CRDGroup, Version: CRDVersion}
	crdconfig.APIPath = "/apis"
	crdconfig.ContentType = runtime.ContentTypeJSON
	s := runtime.NewScheme()
	s.AddKnownTypes(*crdconfig.GroupVersion,
		&AzureIdentity{},
		&AzureIdentityList{},
		&AzureIdentityBinding{},
		&AzureIdentityBindingList{},
		&AzureAssignedIdentity{},
		&AzureAssignedIdentityList{})
	crdconfig.NegotiatedSerializer = serializer.DirectCodecFactory{
		CodecFactory: serializer.NewCodecFactory(s)}

	//Client interacting with our CRDs
	crdClient, err := rest.RESTClientFor(&crdconfig)
	if err != nil {
		return nil, err
	}

	clientSet := kubernetes.NewForConfigOrDie(config)
	k8sInformers := informers.NewSharedInformerFactory(clientSet, time.Minute*5)

	return &Client{
		CRDClient:    crdClient,
		ClientSet:    clientSet,
		K8sInformers: k8sInformers,
	}, nil
}

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

// Adds the list of known types to Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {

	return nil
}

func AssignUserMSI(userAssignedMSIID string, nodeName string) error {
	return nil
}

func (c *Client) AssignIdentity(idName string, podName string) error {
	// Create a new AzureAssignedIdentity which maps the relationship between
	// id and pod
	return nil
}

func (c *Client) ListBindings() (res *AzureIdentityBindingList, err error) {
	err = c.CRDClient.Get().Namespace("default").Resource("azureidentities").Do().Into(res)
	if err != nil {
		return nil, err
	}
	return res, nil

}

// MatchBinding - matches the name of the pod with the bindings. Return back
// the name of the identity which is matching. This name
// will be used to assign the azureidentity to the pod.
func (c *Client) MatchBinding(podName string) (idName string, err error) {
	// List the AzureIdentityBindings and check if the pod name matches
	// any selector.
	bindings, err := c.ListBindings()
	if err != nil {
		return "", err
	}
	for _, v := range bindings.Items {
		if v.Spec.Name == podName {
			return v.Spec.AzureIdRef.Name, nil
		}
	}
	return "", nil
}
