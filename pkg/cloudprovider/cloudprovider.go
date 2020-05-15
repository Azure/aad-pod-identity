package cloudprovider

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/Azure/aad-pod-identity/pkg/config"
	"github.com/Azure/aad-pod-identity/pkg/utils"
	"github.com/Azure/aad-pod-identity/version"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-12-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	yaml "gopkg.in/yaml.v2"
	"k8s.io/klog"
)

// Client is a cloud provider client
type Client struct {
	VMClient   VMClientInt
	VMSSClient VMSSClientInt
	ExtClient  compute.VirtualMachineExtensionsClient
	Config     config.AzureConfig
	configFile string
}

// ClientInt client interface
type ClientInt interface {
	UpdateUserMSI(addUserAssignedMSIIDs, removeUserAssignedMSIIDs []string, name string, isvmss bool) error
	GetUserMSIs(name string, isvmss bool) ([]string, error)
	Init() error
}

// NewCloudProvider returns a azure cloud provider client
func NewCloudProvider(configFile string) (c *Client, e error) {
	client := &Client{
		configFile: configFile,
	}
	if err := client.Init(); err != nil {
		return nil, err
	}
	return client, nil
}

// Init initializes the cloud provider client based
// on a config path or environment variables
func (c *Client) Init() error {
	c.Config = config.AzureConfig{}
	if c.configFile != "" {
		klog.V(6).Info("Populate AzureConfig from azure.json")
		bytes, err := ioutil.ReadFile(c.configFile)
		if err != nil {
			klog.Errorf("Read file (%s) error: %+v", c.configFile, err)
			return err
		}
		if err = yaml.Unmarshal(bytes, &c.Config); err != nil {
			klog.Errorf("Unmarshall error: %v", err)
			return err
		}
	} else {
		klog.V(6).Info("Populate AzureConfig from secret/environment variables")
		c.Config.Cloud = os.Getenv("CLOUD")
		c.Config.TenantID = os.Getenv("TENANT_ID")
		c.Config.ClientID = os.Getenv("CLIENT_ID")
		c.Config.ClientSecret = os.Getenv("CLIENT_SECRET")
		c.Config.SubscriptionID = os.Getenv("SUBSCRIPTION_ID")
		c.Config.ResourceGroupName = os.Getenv("RESOURCE_GROUP")
		c.Config.VMType = os.Getenv("VM_TYPE")
		c.Config.UseManagedIdentityExtension = strings.EqualFold(os.Getenv("USE_MSI"), "True")
		c.Config.UserAssignedIdentityID = os.Getenv("USER_ASSIGNED_MSI_CLIENT_ID")
	}

	azureEnv, err := azure.EnvironmentFromName(c.Config.Cloud)
	if err != nil {
		klog.Errorf("Get cloud env error: %+v", err)
		return err
	}

	err = adal.AddToUserAgent(version.GetUserAgent("MIC", version.MICVersion))
	if err != nil {
		klog.Errorf("add to user agent error: %+v", err)
		return err
	}

	oauthConfig, err := adal.NewOAuthConfig(azureEnv.ActiveDirectoryEndpoint, c.Config.TenantID)
	if err != nil {
		klog.Errorf("Create OAuth config error: %+v", err)
		return err
	}

	var spt *adal.ServicePrincipalToken
	if c.Config.UseManagedIdentityExtension {
		// MSI endpoint is required for both types of MSI - system assigned and user assigned.
		msiEndpoint, err := adal.GetMSIVMEndpoint()
		if err != nil {
			klog.Errorf("Failed to get MSI endpoint. Error: %+v", err)
			return err
		}
		// UserAssignedIdentityID is empty, so we are going to use system assigned MSI
		if c.Config.UserAssignedIdentityID == "" {
			klog.Infof("MIC using system assigned identity for authentication.")
			spt, err = adal.NewServicePrincipalTokenFromMSI(msiEndpoint, azureEnv.ResourceManagerEndpoint)
			if err != nil {
				klog.Errorf("Get token from system assigned MSI error: %+v", err)
				return err
			}
		} else { // User assigned identity usage.
			klog.Infof("MIC using user assigned identity: %s for authentication.", utils.RedactClientID(c.Config.UserAssignedIdentityID))
			spt, err = adal.NewServicePrincipalTokenFromMSIWithUserAssignedID(msiEndpoint, azureEnv.ResourceManagerEndpoint, c.Config.UserAssignedIdentityID)
			if err != nil {
				klog.Errorf("Get token from user assigned MSI error: %+v", err)
				return err
			}
		}
	} else { // This is the default scenario - use service principal to get the token.
		spt, err = adal.NewServicePrincipalToken(
			*oauthConfig,
			c.Config.ClientID,
			c.Config.ClientSecret,
			azureEnv.ResourceManagerEndpoint,
		)
		if err != nil {
			klog.Errorf("Get service principal token error: %+v", err)
			return err
		}
	}

	extClient := compute.NewVirtualMachineExtensionsClient(c.Config.SubscriptionID)
	extClient.BaseURI = azure.PublicCloud.ResourceManagerEndpoint
	extClient.Authorizer = autorest.NewBearerAuthorizer(spt)
	extClient.PollingDelay = 5 * time.Second

	c.VMSSClient, err = NewVMSSClient(c.Config, spt)
	if err != nil {
		klog.Errorf("Create VMSS Client error: %+v", err)
		return err
	}
	c.VMClient, err = NewVirtualMachinesClient(c.Config, spt)
	if err != nil {
		klog.Errorf("Create VM Client error: %+v", err)
		return err
	}

	return nil
}

// GetUserMSIs will return a list of all identities on the node or vmss based on value of isvmss
func (c *Client) GetUserMSIs(name string, isvmss bool) ([]string, error) {
	idH, _, err := c.getIdentityResource(name, isvmss)
	if err != nil {
		klog.Errorf("GetUserMSIs: get identity resource failed with error %v", err)
		return nil, err
	}
	info := idH.IdentityInfo()
	if info == nil {
		return nil, fmt.Errorf("identity info is nil")
	}
	idList := info.GetUserIdentityList()
	return idList, nil
}

// UpdateUserMSI will batch process the removal and addition of ids
func (c *Client) UpdateUserMSI(addUserAssignedMSIIDs, removeUserAssignedMSIIDs []string, name string, isvmss bool) error {
	idH, updateFunc, err := c.getIdentityResource(name, isvmss)
	if err != nil {
		return err
	}

	info := idH.IdentityInfo()
	if info == nil {
		info = idH.ResetIdentity()
	}

	ids := make(map[string]bool)
	// remove msi ids from the list
	for _, userAssignedMSIID := range removeUserAssignedMSIIDs {
		ids[userAssignedMSIID] = false
	}
	// add new ids to the list
	// add is done after setting del ids in the map to ensure an identity if in
	// both add and del list is not deleted
	for _, userAssignedMSIID := range addUserAssignedMSIIDs {
		ids[userAssignedMSIID] = true
	}
	requiresUpdate := info.SetUserIdentities(ids)

	if requiresUpdate {
		klog.Infof("Updating user assigned MSIs on %s, assign [%d], unassign [%d]", name, len(addUserAssignedMSIIDs), len(removeUserAssignedMSIIDs))
		timeStarted := time.Now()
		if err := updateFunc(); err != nil {
			return err
		}
		klog.V(6).Infof("UpdateUserMSI of %s completed in %s", name, time.Since(timeStarted))
	}
	return nil
}

func (c *Client) getIdentityResource(name string, isvmss bool) (idH IdentityHolder, update func() error, retErr error) {
	rg := c.Config.ResourceGroupName

	if isvmss {
		vmss, err := c.VMSSClient.Get(rg, name)
		if err != nil {
			return nil, nil, err
		}

		update = func() error {
			return c.VMSSClient.UpdateIdentities(rg, name, vmss)
		}
		idH = &vmssIdentityHolder{&vmss}
		return idH, update, nil
	}

	vm, err := c.VMClient.Get(rg, name)
	if err != nil {
		return nil, nil, err
	}
	update = func() error {
		return c.VMClient.UpdateIdentities(rg, name, vm)
	}
	idH = &vmIdentityHolder{&vm}
	return idH, update, nil
}

const nestedResourceIDPatternText = `(?i)subscriptions/(.+)/resourceGroups/(.+)/providers/(.+?)/(.+?)/(.+?)/(.+)`
const resourceIDPatternText = `(?i)subscriptions/(.+)/resourceGroups/(.+)/providers/(.+?)/(.+?)/(.+)`

var (
	nestedResourceIDPattern = regexp.MustCompile(nestedResourceIDPatternText)
	resourceIDPattern       = regexp.MustCompile(resourceIDPatternText)
)

const (
	// VMResourceType virtual machine resource type
	VMResourceType = "virtualMachines"
	// VMSSResourceType virtual machine scale sets resource type
	VMSSResourceType = "virtualMachineScaleSets"
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
