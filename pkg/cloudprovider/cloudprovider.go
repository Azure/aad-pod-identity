package cloudprovider

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
	"unicode"

	config "github.com/Azure/aad-pod-identity/pkg/config"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/golang/glog"
	yaml "gopkg.in/yaml.v2"
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
	RemoveUserMSI(userAssignedMSIID string, nodeName string) error
	AssignUserMSI(userAssignedMSIID string, nodeName string) error
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

	switch azureConfig.VMType {
	case "vmss":
		client.VMSSClient, err = NewVMSSClient(azureConfig, spt)
		if err != nil {
			glog.Errorf("Create VM Client error: %+v", err)
			return nil, err
		}
	default:
		client.VMClient, err = NewVirtualMachinesClient(azureConfig, spt)
		if err != nil {
			glog.Errorf("Create VM Client error: %+v", err)
			return nil, err
		}
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
func (c *Client) RemoveUserMSI(userAssignedMSIID string, nodeName string) error {
	idH, updateFunc, err := c.getIdentityResource(nodeName)
	if err != nil {
		return err
	}

	info := idH.IdentityInfo()
	if info == nil {
		glog.Errorf("Identity null for vm: %s ", nodeName)
		return fmt.Errorf("identity null for vm: %s ", nodeName)
	}

	if err := info.RemoveUserIdentity(userAssignedMSIID); err != nil {
		return fmt.Errorf("could not remove identity from node %s: %v", nodeName, err)
	}

	if err := updateFunc(); err != nil {
		glog.Error(err)
		return err
	}

	return nil
}

func (c *Client) AssignUserMSI(userAssignedMSIID string, nodeName string) error {
	// Get the vm using the VmClient
	// Update the assigned identity into the VM using the CreateOrUpdate

	glog.Infof("Find %s in resource group: %s", nodeName, c.Config.ResourceGroupName)
	timeStarted := time.Now()

	idH, updateFunc, err := c.getIdentityResource(nodeName)
	if err != nil {
		return err
	}
	glog.V(6).Infof("Get of %s completed in %s", nodeName, time.Since(timeStarted))

	info := idH.IdentityInfo()
	if info == nil {
		info = idH.ResetIdentity()
	}

	info.AppendUserIdentity(userAssignedMSIID)

	timeStarted = time.Now()
	if err := updateFunc(); err != nil {
		return err
	}

	glog.V(6).Infof("CreateOrUpdate of %s completed in %s", nodeName, time.Since(timeStarted))
	return nil
}

func (c *Client) getIdentityResource(name string) (idH IdentityHolder, update func() error, retErr error) {
	switch c.Config.VMType {
	case "vmss":
		// TODO(@cpuguy83): We are getting a *node* name as an argument to this function, but need the vmss name.
		// For now, assume the name of the node follows <vmssname><nodenumber>, so trimming the node number should give us the vmss name.
		name = strings.TrimRightFunc(name, func(r rune) bool {
			return unicode.IsNumber(r)
		})
		vmss, err := c.VMSSClient.Get(c.Config.ResourceGroupName, name)
		if err != nil {
			return nil, nil, err
		}

		update = func() error {
			return c.VMSSClient.CreateOrUpdate(c.Config.ResourceGroupName, name, vmss)
		}
		idH = &vmssIdentityHolder{&vmss}
	default:
		vm, err := c.VMClient.Get(c.Config.ResourceGroupName, name)
		if err != nil {
			return nil, nil, err
		}
		update = func() error {
			return c.VMClient.CreateOrUpdate(c.Config.ResourceGroupName, name, vm)
		}
		idH = &vmIdentityHolder{&vm}
	}

	return idH, update, nil
}
