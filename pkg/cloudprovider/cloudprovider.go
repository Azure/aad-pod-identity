package cloudprovider

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"regexp"
	"time"

	config "github.com/Azure/aad-pod-identity/pkg/config"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/golang/glog"
	yaml "gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
)

// Client is a cloud provider client
type Client struct {
	ResourceGroupName string
	VMClient          VMClientInt
	VMSSClient        VMSSClientInt
	ExtClient         compute.VirtualMachineExtensionsClient
	Config            config.AzureConfig
}

type ClientInt interface {
	RemoveUserMSI(userAssignedMSIID string, node *corev1.Node) error
	AssignUserMSI(userAssignedMSIID string, node *corev1.Node) error
}

// NewCloudProvider returns a azure cloud provider client
func NewCloudProvider(configFile string) (c *Client, e error) {
	bytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		glog.Errorf("Read file (%s) error: %+v", configFile, err)
		return nil, err
	}
	azureConfig := config.AzureConfig{}
	if err = yaml.Unmarshal(bytes, &azureConfig); err != nil {
		glog.Errorf("Unmarshall error: %v", err)
		return nil, err
	}
	azureEnv, err := azure.EnvironmentFromName(azureConfig.Cloud)
	if err != nil {
		glog.Errorf("Get cloud env error: %+v", err)
		return nil, err
	}
	oauthConfig, _ := adal.NewOAuthConfig(azureEnv.ActiveDirectoryEndpoint, azureConfig.TenantID)
	if err != nil {
		return nil, fmt.Errorf("creating the OAuth config: %v", err)
	}
	//glog.Info("%+v\n", oauthConfig)
	spt, err := adal.NewServicePrincipalToken(
		*oauthConfig,
		azureConfig.ClientID,
		azureConfig.ClientSecret,
		azureEnv.ResourceManagerEndpoint,
	)
	if err != nil {
		glog.Errorf("Get service principle token error: %+v", err)
		return nil, err
	}

	extClient := compute.NewVirtualMachineExtensionsClient(azureConfig.SubscriptionID)
	extClient.BaseURI = azure.PublicCloud.ResourceManagerEndpoint
	extClient.Authorizer = autorest.NewBearerAuthorizer(spt)
	extClient.PollingDelay = 5 * time.Second

	client := &Client{
		ResourceGroupName: azureConfig.ResourceGroupName,
		Config:            azureConfig,
		ExtClient:         extClient,
	}

	client.VMSSClient, err = NewVMSSClient(azureConfig, spt)
	if err != nil {
		glog.Errorf("Create VMSS Client error: %+v", err)
		return nil, err
	}
	client.VMClient, err = NewVirtualMachinesClient(azureConfig, spt)
	if err != nil {
		glog.Errorf("Create VM Client error: %+v", err)
		return nil, err
	}

	return client, nil
}

func withInspection() autorest.PrepareDecorator {
	return func(p autorest.Preparer) autorest.Preparer {
		return autorest.PreparerFunc(func(r *http.Request) (*http.Request, error) {
			glog.Infof("Inspecting Request: Method: %s \n URL: %s, URI: %s\n", r.Method, r.URL, r.RequestURI)

			return p.Prepare(r)
		})
	}
}

//RemoveUserMSI - Use the underlying cloud api calls and remove the given user assigned MSI from the vm.
func (c *Client) RemoveUserMSI(userAssignedMSIID string, node *corev1.Node) error {
	idH, updateFunc, err := c.getIdentityResource(node)
	if err != nil {
		return err
	}

	info := idH.IdentityInfo()
	if info == nil {
		glog.Errorf("Identity null for vm: %s ", node.Name)
		return fmt.Errorf("identity null for vm: %s ", node.Name)
	}

	if err := info.RemoveUserIdentity(userAssignedMSIID); err != nil {
		return fmt.Errorf("could not remove identity from node %s: %v", node.Name, err)
	}

	if err := updateFunc(); err != nil {
		glog.Error(err)
		return err
	}

	return nil
}

func (c *Client) AssignUserMSI(userAssignedMSIID string, node *corev1.Node) error {
	// Get the vm using the VmClient
	// Update the assigned identity into the VM using the CreateOrUpdate

	glog.Infof("Find %s in resource group: %s", node.Name, c.Config.ResourceGroupName)
	timeStarted := time.Now()

	idH, updateFunc, err := c.getIdentityResource(node)
	if err != nil {
		return err
	}
	glog.V(6).Infof("Get of %s completed in %s", node.Name, time.Since(timeStarted))

	info := idH.IdentityInfo()
	if info == nil {
		info = idH.ResetIdentity()
	}

	info.AppendUserIdentity(userAssignedMSIID)

	timeStarted = time.Now()
	if err := updateFunc(); err != nil {
		return err
	}

	glog.V(6).Infof("CreateOrUpdate of %s completed in %s", node.Name, time.Since(timeStarted))
	return nil
}

func (c *Client) getIdentityResource(node *corev1.Node) (idH IdentityHolder, update func() error, retErr error) {
	name := node.Name // fallback in case parsing the provider spec fails
	rg := c.Config.ResourceGroupName
	rt := c.Config.VMType
	if r, err := ParseResourceID(node.Spec.ProviderID); err == nil {
		name = r.ResourceName
		rg = r.ResourceGroup
		if r.ResourceType == "virtualMachineScaleSets" {
			rt = "vmss"
		}
	}

	switch rt {
	case "vmss":
		vmss, err := c.VMSSClient.Get(rg, name)
		if err != nil {
			return nil, nil, err
		}

		update = func() error {
			return c.VMSSClient.CreateOrUpdate(rg, name, vmss)
		}
		idH = &vmssIdentityHolder{&vmss}
	default:
		vm, err := c.VMClient.Get(rg, name)
		if err != nil {
			return nil, nil, err
		}
		update = func() error {
			return c.VMClient.CreateOrUpdate(rg, name, vm)
		}
		idH = &vmIdentityHolder{&vm}
	}

	return idH, update, nil
}

const nestedResourceIDPatternText = `(?i)subscriptions/(.+)/resourceGroups/(.+)/providers/(.+?)/(.+?)/(.+?)/(.+)`
const resourceIDPatternText = `(?i)subscriptions/(.+)/resourceGroups/(.+)/providers/(.+?)/(.+?)/(.+)`

var (
	nestedResourceIDPattern = regexp.MustCompile(nestedResourceIDPatternText)
	resourceIDPattern       = regexp.MustCompile(resourceIDPatternText)
)

// ParseResourceID is a slightly modified version of https://github.com/Azure/go-autorest/blob/528b76fd0ebec0682f3e3da7c808cd472b999615/autorest/azure/azure.go#L175
// The modification here is to support a nested resource such as is the case for a node resource in a vmss.
func ParseResourceID(resourceID string) (azure.Resource, error) {
	match := nestedResourceIDPattern.FindStringSubmatch(resourceID)
	if len(match) == 0 {
		match = resourceIDPattern.FindStringSubmatch(resourceID)
	}

	if len(match) < 6 {
		return azure.Resource{}, fmt.Errorf("parsing failed for %s: invalid resource id format", resourceID)
	}

	result := azure.Resource{
		SubscriptionID: match[1],
		ResourceGroup:  match[2],
		Provider:       match[3],
		ResourceType:   match[4],
		ResourceName:   path.Base(match[5]),
	}

	return result, nil
}
