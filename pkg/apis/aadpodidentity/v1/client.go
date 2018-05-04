package aadpodidentity

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2017-12-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
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

type Config struct {
	Cloud             string `json:"cloud" yaml:"cloud"`
	TenantID          string `json:"tenantId" yaml:"tenantId"`
	SubscriptionID    string `json:"subscriptionId" yaml:"subscriptionId"`
	NodeResourceGroup string `json:"resourceGroup" yaml:"nodeResourceGroup"`
	AADClientID       string `json:"aadClientId" yaml:"aadClientId"`
	AADClientSecret   string `json:"aadClientSecret" yaml:"aadClientSecret"`
}

// Client has the required pointers to talk to the api server
// and interact with the CRD related datastructure.
type Client struct {
	CRDClient    *rest.RESTClient
	ClientSet    *kubernetes.Clientset
	K8sInformers informers.SharedInformerFactory
	CredConfig   Config
	VMClient     compute.VirtualMachinesClient
}

func Cleanup() {

}

func NewAadPodIdentityCrdClient(config *rest.Config, credConfigFile string) (*Client, error) {
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

	var conf Config
	f, err := os.Open(credConfigFile)
	if err != nil {
		Cleanup()
		return nil, err
	}

	jsonStream := json.NewDecoder(f)
	err = jsonStream.Decode(&conf)
	if err != nil {
		return nil, err
	}
	glog.Infof("%v", conf)

	oauthConfig, err := adal.NewOAuthConfig(azure.PublicCloud.ActiveDirectoryEndpoint, conf.TenantID)
	if err != nil {
		return nil, fmt.Errorf("creating the OAuth config: %v", err)
	}

	spt, err := adal.NewServicePrincipalToken(
		*oauthConfig,
		conf.AADClientID,
		conf.AADClientSecret,
		azure.PublicCloud.ServiceManagementEndpoint)
	if err != nil {
		Cleanup()
		return nil, err
	}

	virtualMachinesClient := compute.NewVirtualMachinesClient(conf.SubscriptionID)
	virtualMachinesClient.BaseURI = azure.PublicCloud.ResourceManagerEndpoint
	virtualMachinesClient.Authorizer = autorest.NewBearerAuthorizer(spt)
	virtualMachinesClient.PollingDelay = 5 * time.Second

	return &Client{
		CRDClient:    crdClient,
		ClientSet:    clientSet,
		K8sInformers: k8sInformers,
		CredConfig:   conf,
		VMClient:     virtualMachinesClient,
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

func (c *Client) AssignUserMSI(userAssignedMSIID string, nodeName string) error {
	// Get the vm using the VmClient
	// Update the assigned identity into the VM using the CreateOrUpdate

	return nil
}

func (c *Client) AssignIdentity(idName string, podName string, nodeName string) error {
	// Create a new AzureAssignedIdentity which maps the relationship between
	// id and pod
	assignedID := &AzureAssignedIdentity{
		ObjectMeta: v1.ObjectMeta{
			Name: idName,
		},
		Spec: AzureAssignedIdentitySpec{
			AzureIdentityRef: idName,
			Pod:              podName,
			NodeName:         nodeName,
		},
		Status: AzureAssignedIdentityStatus{
			AvailableReplicas: 1,
		},
	}
	var res AzureAssignedIdentity
	// TODO: Ensure that the status reflects the corresponding
	err := c.CRDClient.Post().Namespace("default").Resource("azureassignedidentities").Body(assignedID).Do().Into(&res)
	if err != nil {
		return err
	}

	id, err := c.Lookup(idName)
	if err == nil {
		return err
	}
	err = c.AssignUserMSI(id.Spec.ID, nodeName)
	if err != nil {
		glog.Error(err)
		return err
	}

	//TODO: Update the status of the assign identity to indicate that the node assignment got done.
	return nil
}

func (c *Client) ListBindings() (res *AzureIdentityBindingList, err error) {
	err = c.CRDClient.Get().Namespace("default").Resource("azureidentitybindings").Do().Into(res)
	if err != nil {
		return nil, err
	}
	return res, nil

}

func (c *Client) Lookup(idName string) (res *AzureIdentity, err error) {
	ids, err := c.ListIds()
	if err != nil {
		return nil, err
	}
	for _, v := range ids.Items {
		glog.Infof("Looking for idName %s in ")
		if v.Spec.Name == idName {
			return &v, nil
		}
	}
	return nil, fmt.Errorf("Lookup of %s failed", idName)
}

func (c *Client) ListIds() (res *AzureIdentityList, err error) {
	err = c.CRDClient.Get().Namespace("default").Resource("azureidentities").Do().Into(res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// MatchBinding - matches the name of the pod with the bindings. Return back
// the name of the identity which is matching. This name
// will be used to assign the azureidentity to the pod.
func (c *Client) Bind(podName string, nodeName string) (err error) {
	// List the AzureIdentityBindings and check if the pod name matches
	// any selector.
	bindings, err := c.ListBindings()
	if err != nil {
		return err
	}
	for _, v := range bindings.Items {
		glog.Infof("Matching pod name %s with binding name %s", podName, v.Spec.Name)
		if v.Spec.MatchName == podName {
			return c.AssignIdentity(v.Spec.AzureIdentityRef, podName, nodeName)
		}
	}
	return nil
}
