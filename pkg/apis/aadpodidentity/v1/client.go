package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
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
	NodeResourceGroup string `json:"nodeResourceGroup" yaml:"nodeResourceGroup"`
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
	ExtClient    compute.VirtualMachineExtensionsClient
}

func Cleanup() {

}

func withInspection() autorest.PrepareDecorator {
	return func(p autorest.Preparer) autorest.Preparer {
		return autorest.PreparerFunc(func(r *http.Request) (*http.Request, error) {
			glog.Infof("Inspecting Request: Method: %s \n URL: %s, URI: %s\n", r.Method, r.URL, r.RequestURI)

			return p.Prepare(r)
		})
	}
}

func NewCRDClient(config *rest.Config, credConfigFile string) (*rest.RESTClient, error) {
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
		glog.Error(err)
		return nil, err
	}
	return crdClient, nil
}

func NewAadPodIdentityCrdClient(config *rest.Config, credConfigFile string) (*Client, error) {
	glog.Infof("Starting to create the pod identity client")

	crdClient, err := NewCRDClient(config, credConfigFile)
	if err != nil {
		return nil, err
	}

	clientSet := kubernetes.NewForConfigOrDie(config)
	k8sInformers := informers.NewSharedInformerFactory(clientSet, time.Minute*5)

	glog.Infof("Going to open the file: %s", credConfigFile)
	var conf Config
	f, err := os.Open(credConfigFile)
	if err != nil {
		Cleanup()
		glog.Error(err)
		return nil, err
	}

	glog.Infof("Going to decode: %+v\n", f)
	jsonStream := json.NewDecoder(f)
	err = jsonStream.Decode(&conf)
	if err != nil {
		glog.Error(err)
		return nil, err
	}
	glog.Infof("%+v\n", conf)

	oauthConfig, err := adal.NewOAuthConfig(azure.PublicCloud.ActiveDirectoryEndpoint, conf.TenantID)
	if err != nil {
		return nil, fmt.Errorf("creating the OAuth config: %v", err)
	}
	glog.Info("%+v\n", oauthConfig)
	spt, err := adal.NewServicePrincipalToken(
		*oauthConfig,
		conf.AADClientID,
		conf.AADClientSecret,
		azure.PublicCloud.ServiceManagementEndpoint)
	if err != nil {
		Cleanup()
		return nil, err
	}

	extClient := compute.NewVirtualMachineExtensionsClient(conf.SubscriptionID)
	extClient.BaseURI = azure.PublicCloud.ResourceManagerEndpoint
	extClient.Authorizer = autorest.NewBearerAuthorizer(spt)
	extClient.PollingDelay = 5 * time.Second

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
		ExtClient:    extClient,
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
	ctx := context.Background()
	glog.Infof("Find %s in resource group: %s", nodeName, c.CredConfig.NodeResourceGroup)
	vm, err := c.VMClient.Get(ctx, c.CredConfig.NodeResourceGroup, nodeName, "")
	if err != nil {
		return err
	}

	c.VMClient.Client.RequestInspector = withInspection()
	glog.Infof("Got VM info: %+v. Assign %s\n", vm, userAssignedMSIID)
	/*
		location := "eastus"
		ctx = context.Background()
		extFuture, err := c.ExtClient.CreateOrUpdate(ctx, c.CredConfig.NodeResourceGroup, nodeName, "msiextension", compute.VirtualMachineExtension{
			Location: &location,
			VirtualMachineExtensionProperties: &compute.VirtualMachineExtensionProperties{
				Publisher:               to.StringPtr("Microsoft.ManagedIdentity"),
				Type:                    to.StringPtr("ManagedIdentityExtensionForLinux"),
				TypeHandlerVersion:      to.StringPtr("1.0"),
				AutoUpgradeMinorVersion: to.BoolPtr(true),
				Settings: &map[string]interface{}{
					"port": "50342",
				},
			},
		})

		err = extFuture.WaitForCompletion(ctx, c.ExtClient.Client)
		if err != nil {
			glog.Error(err)
			return err
		}
		ext, err := extFuture.Result(c.ExtClient)
		if err != nil {
			glog.Error(err)
			return err
		}
		glog.Info("After update the ext info: %+v", ext)
	*/
	vm.Identity = &compute.VirtualMachineIdentity{
		Type:        compute.ResourceIdentityTypeUserAssigned,
		IdentityIds: &[]string{userAssignedMSIID},
	}

	// Set the read-only property of extension to null.
	vm.Resources = nil
	ctx = context.Background()
	future, err := c.VMClient.CreateOrUpdate(ctx, c.CredConfig.NodeResourceGroup, nodeName, vm)
	if err != nil {
		glog.Error(err)
		return err
	}
	err = future.WaitForCompletion(ctx, c.VMClient.Client)
	if err != nil {
		glog.Error(err)
		return err
	}
	vm, err = future.Result(c.VMClient)
	if err != nil {
		glog.Error(err)
		return err
	}
	glog.Info("After update the vm info: %+v", vm)
	return nil
}

func (c *Client) AssignIdentity(idName string, podName string, nodeName string) error {
	glog.Infof("Got id %s to assign", idName)
	// Create a new AzureAssignedIdentity which maps the relationship between
	// id and pod
	assignedID := &AzureAssignedIdentity{
		ObjectMeta: v1.ObjectMeta{
			Name: "azureassignedidentities.aadpodidentity.k8s.io",
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
		glog.Error(err)
		return err
	}
	glog.Infof("Looking up id: %s", idName)
	id, err := c.Lookup(idName)
	if err != nil {
		glog.Error(err)
		return err
	}
	glog.Infof("Assigning MSI ID: %s to node %s", id.Spec.ID, nodeName)
	err = c.AssignUserMSI(id.Spec.ID, nodeName)
	if err != nil {
		glog.Error(err)
		return err
	}

	//TODO: Update the status of the assign identity to indicate that the node assignment got done.
	return nil
}

func (c *Client) ListBindings() (res *AzureIdentityBindingList, err error) {
	var ret AzureIdentityBindingList
	err = c.CRDClient.Get().Namespace("default").Resource("azureidentitybindings").Do().Into(&ret)
	if err != nil {
		glog.Error(err)
		return nil, err
	}
	//glog.Infof("%+v", ret)
	return &ret, nil

}

func (c *Client) Lookup(idName string) (res *AzureIdentity, err error) {
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

func (c *Client) ListIds() (res *AzureIdentityList, err error) {
	var ret AzureIdentityList
	err = c.CRDClient.Get().Namespace("default").Resource("azureidentities").Do().Into(&ret)
	if err != nil {
		glog.Error(err)
		return nil, err
	}
	return &ret, nil
}

// MatchBinding - matches the name of the pod with the bindings. Return back
// the name of the identity which is matching. This name
// will be used to assign the azureidentity to the pod.
func (c *Client) Bind(podName string, nodeName string) (err error) {
	// List the AzureIdentityBindings and check if the pod name matches
	// any selector.
	glog.Infof("Created pod with Name: %s", podName)
	bindings, err := c.ListBindings()
	if err != nil {
		glog.Error(err)
		return err
	}
	for _, v := range bindings.Items {
		glog.Infof("Matching pod name %s with binding name %s", podName, v.Spec.MatchName)
		if v.Spec.MatchName == podName {
			glog.Infof("%+v", v.Spec)
			return c.AssignIdentity(v.Spec.AzureIdentityRef, podName, nodeName)
		}
	}
	return nil
}
