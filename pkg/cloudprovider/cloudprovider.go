package cloudprovider

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	config "github.com/Azure/aad-pod-identity/pkg/config"
	"github.com/Azure/aad-pod-identity/version"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/golang/glog"
	yaml "gopkg.in/yaml.v2"
)

// Client is a cloud provider client
type Client struct {
	VMClient   VMClientInt
	VMSSClient VMSSClientInt
	ExtClient  compute.VirtualMachineExtensionsClient
	Config     config.AzureConfig
}

// ClientInt client interface
type ClientInt interface {
	RemoveUserMSI(userAssignedMSIID, name string, isvmss bool) error
	AssignUserMSI(userAssignedMSIID, name string, isvmss bool) error
	UpdateUserMSI(addUserAssignedMSIIDs, removeUserAssignedMSIIDs []string, name string, isvmss bool) error
	GetUserMSIs(name string, isvmss bool) ([]string, error)
}

// NewCloudProvider returns a azure cloud provider client
func NewCloudProvider(configFile string) (c *Client, e error) {
	azureConfig := config.AzureConfig{}
	if configFile != "" {
		glog.V(6).Info("Populate AzureConfig from azure.json")
		bytes, err := ioutil.ReadFile(configFile)
		if err != nil {
			glog.Errorf("Read file (%s) error: %+v", configFile, err)
			return nil, err
		}
		if err = yaml.Unmarshal(bytes, &azureConfig); err != nil {
			glog.Errorf("Unmarshall error: %v", err)
			return nil, err
		}
	} else {
		glog.V(6).Info("Populate AzureConfig from secret/environment variables")
		azureConfig.Cloud = os.Getenv("CLOUD")
		azureConfig.TenantID = os.Getenv("TENANT_ID")
		azureConfig.ClientID = os.Getenv("CLIENT_ID")
		azureConfig.ClientSecret = os.Getenv("CLIENT_SECRET")
		azureConfig.SubscriptionID = os.Getenv("SUBSCRIPTION_ID")
		azureConfig.ResourceGroupName = os.Getenv("RESOURCE_GROUP")
		azureConfig.VMType = os.Getenv("VM_TYPE")
		azureConfig.UseManagedIdentityExtension = strings.EqualFold(os.Getenv("USE_MSI"), "True")
		azureConfig.UserAssignedIdentityID = os.Getenv("USER_ASSIGNED_MSI_CLIENT_ID")
	}

	azureEnv, err := azure.EnvironmentFromName(azureConfig.Cloud)
	if err != nil {
		glog.Errorf("Get cloud env error: %+v", err)
		return nil, err
	}

	err = adal.AddToUserAgent(version.GetUserAgent("MIC", version.MICVersion))
	if err != nil {
		return nil, err
	}

	oauthConfig, err := adal.NewOAuthConfig(azureEnv.ActiveDirectoryEndpoint, azureConfig.TenantID)
	if err != nil {
		glog.Errorf("Create OAuth config error: %+v", err)
		return nil, err
	}

	var spt *adal.ServicePrincipalToken
	if azureConfig.UseManagedIdentityExtension {
		// MSI endpoint is required for both types of MSI - system assigned and user assigned.
		msiEndpoint, err := adal.GetMSIVMEndpoint()
		if err != nil {
			glog.Errorf("Failed to get MSI endpoint. Error: %+v", err)
			return nil, err
		}
		// UserAssignedIdentityID is empty, so we are going to use system assigned MSI
		if azureConfig.UserAssignedIdentityID == "" {
			glog.Infof("MIC using system assigned identity for authentication.")
			spt, err = adal.NewServicePrincipalTokenFromMSI(msiEndpoint, azureEnv.ResourceManagerEndpoint)
			if err != nil {
				glog.Errorf("Get token from system assigned MSI error: %+v", err)
				return nil, err
			}
		} else { // User assigned identity usage.
			glog.Infof("MIC using user assigned identity: %s for authentication.", azureConfig.UserAssignedIdentityID)
			spt, err = adal.NewServicePrincipalTokenFromMSIWithUserAssignedID(msiEndpoint, azureEnv.ResourceManagerEndpoint, azureConfig.UserAssignedIdentityID)
			if err != nil {
				glog.Errorf("Get token from user assigned MSI error: %+v", err)
				return nil, err
			}
		}
	} else { // This is the default scenario - use service principal to get the token.
		spt, err = adal.NewServicePrincipalToken(
			*oauthConfig,
			azureConfig.ClientID,
			azureConfig.ClientSecret,
			azureEnv.ResourceManagerEndpoint,
		)
		if err != nil {
			glog.Errorf("Get service principal token error: %+v", err)
			return nil, err
		}
	}

	extClient := compute.NewVirtualMachineExtensionsClient(azureConfig.SubscriptionID)
	extClient.BaseURI = azure.PublicCloud.ResourceManagerEndpoint
	extClient.Authorizer = autorest.NewBearerAuthorizer(spt)
	extClient.PollingDelay = 5 * time.Second

	client := &Client{
		Config:    azureConfig,
		ExtClient: extClient,
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

// GetUserMSIs will return a list of all identities on the node or vmss based on value of isvmss
func (c *Client) GetUserMSIs(name string, isvmss bool) ([]string, error) {
	idH, _, err := c.getIdentityResource(name, isvmss)
	if err != nil {
		glog.Errorf("GetUserMSIs: get identity resource failed with error %v", err)
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

	requiresUpdate := false
	// remove msi ids from the list
	for _, userAssignedMSIID := range removeUserAssignedMSIIDs {
		requiresUpdate = true
		if err := info.RemoveUserIdentity(userAssignedMSIID); err != nil {
			return fmt.Errorf("could not remove identity from node %s: %v", name, err)
		}
	}
	// add new ids to the list
	for _, userAssignedMSIID := range addUserAssignedMSIIDs {
		addedToList := info.AppendUserIdentity(userAssignedMSIID)
		if !addedToList {
			glog.V(6).Infof("Identity %s already assigned to node %s. Skipping assignment.", userAssignedMSIID, name)
		}
		requiresUpdate = requiresUpdate || addedToList
	}
	if requiresUpdate {
		glog.Infof("Updating user assigned MSIs on %s", name)
		timeStarted := time.Now()
		if err := updateFunc(); err != nil {
			return err
		}
		glog.V(6).Infof("UpdateUserMSI of %s completed in %s", name, time.Since(timeStarted))
	}
	return nil
}

//RemoveUserMSI - Use the underlying cloud api calls and remove the given user assigned MSI from the vm.
func (c *Client) RemoveUserMSI(userAssignedMSIID, name string, isvmss bool) error {
	idH, updateFunc, err := c.getIdentityResource(name, isvmss)
	if err != nil {
		return err
	}

	info := idH.IdentityInfo()
	if info == nil {
		glog.Errorf("Identity null for vm: %s ", name)
		return fmt.Errorf("identity null for vm: %s ", name)
	}

	if err := info.RemoveUserIdentity(userAssignedMSIID); err != nil {
		return fmt.Errorf("could not remove identity from node %s: %v", name, err)
	}

	if err := updateFunc(); err != nil {
		glog.Error(err)
		return err
	}

	return nil
}

// AssignUserMSI - Use the underlying cloud api call and add the given user assigned MSI to the vm
func (c *Client) AssignUserMSI(userAssignedMSIID, name string, isvmss bool) error {
	// Get the vm using the VmClient
	// Update the assigned identity into the VM using the CreateOrUpdate

	glog.Infof("Find %s in resource group: %s", name, c.Config.ResourceGroupName)
	timeStarted := time.Now()

	idH, updateFunc, err := c.getIdentityResource(name, isvmss)
	if err != nil {
		return err
	}
	glog.V(6).Infof("Get of %s completed in %s", name, time.Since(timeStarted))

	info := idH.IdentityInfo()
	if info == nil {
		info = idH.ResetIdentity()
	}

	if info.AppendUserIdentity(userAssignedMSIID) {
		timeStarted = time.Now()
		if err := updateFunc(); err != nil {
			return err
		}
		glog.V(6).Infof("CreateOrUpdate of %s completed in %s", name, time.Since(timeStarted))
	} else {
		glog.V(6).Infof("Identity %s already assigned to node %s. Skipping assignment.", userAssignedMSIID, name)
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
			return c.VMSSClient.CreateOrUpdate(rg, name, vmss)
		}
		idH = &vmssIdentityHolder{&vmss}
		return idH, update, nil
	}

	vm, err := c.VMClient.Get(rg, name)
	if err != nil {
		return nil, nil, err
	}
	update = func() error {
		return c.VMClient.CreateOrUpdate(rg, name, vm)
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

func vmTypeOrDefault(r *azure.Resource, val string) string {
	switch r.ResourceType {
	case VMResourceType:
		return "vm"
	case VMSSResourceType:
		return "vmss"
	}
	return val
}

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
