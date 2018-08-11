package cloudprovider

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

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

	vmClient, err := NewVirtualMachinesClient(azureConfig, spt)
	if err != nil {
		glog.Errorf("Create VM Client error: %+v", err)
		return nil, err
	}

	return &Client{
		ResourceGroupName: azureConfig.ResourceGroupName,
		VMClient:          vmClient,
		ExtClient:         extClient,
		Config:            azureConfig,
	}, nil
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
	vm, err := c.VMClient.Get(c.Config.ResourceGroupName, nodeName)
	if err != nil {
		return err
	}

	//c.VMClient.Client.RequestInspector = withInspection()
	//glog.Infof("Got VM info: %+v. Assign %s\n", vm, userAssignedMSIID)
	var newIds []string
	if vm.Identity != nil { // In case of null identity, we don't have anything to remove. Error condition.
		if vm.Identity.Type == compute.ResourceIdentityTypeUserAssigned ||
			vm.Identity.Type == compute.ResourceIdentityTypeSystemAssignedUserAssigned {
			index := 0
			for _, v := range *vm.Identity.IdentityIds {
				if v == userAssignedMSIID {
					glog.Infof("Remove user id %s from volatile list", v)
				} else {
					newIds = append(newIds, v)
					index++
				}
			}
			// TODO: Handle more conditions.
			// If the number went down, then we will update the vm.
			if index < len(*vm.Identity.IdentityIds) {
				if index == 0 { // Empty EMSI requires us to reset the type.
					if vm.Identity.Type == compute.ResourceIdentityTypeSystemAssignedUserAssigned {
						vm.Identity.Type = compute.ResourceIdentityTypeSystemAssigned
					} else {
						vm.Identity.Type = compute.ResourceIdentityTypeNone
					}
					vm.Identity.IdentityIds = nil
				} else {
					// Regular update on removal. No change required for type since there is atleast one
					// user assigned MSI in the array.
					vm.Identity.IdentityIds = &newIds
				}
				err := c.VMClient.CreateOrUpdate(c.Config.ResourceGroupName, nodeName, vm)
				if err != nil {
					glog.Error(err)
					return err
				}
				return nil
			}
		} else {
			glog.Errorf("User assigned identity not found for node: %s ", nodeName)
			return fmt.Errorf("User assigned Identity not found for node: %s ", nodeName)
		}
	} else {
		glog.Errorf("Identity null for vm: %s ", nodeName)
		return fmt.Errorf("Identity null for vm: %s ", nodeName)
	}
	return fmt.Errorf("Identity %s not removed from node %s", userAssignedMSIID, nodeName)
}

func (c *Client) AssignUserMSI(userAssignedMSIID string, nodeName string) error {
	// Get the vm using the VmClient
	// Update the assigned identity into the VM using the CreateOrUpdate

	glog.Infof("Find %s in resource group: %s", nodeName, c.Config.ResourceGroupName)
	timeStarted := time.Now()
	vm, err := c.VMClient.Get(c.Config.ResourceGroupName, nodeName)
	if err != nil {
		return err
	}
	glog.V(6).Infof("Get of %s completed in %s", nodeName, time.Since(timeStarted))
	found := false
	if vm.Identity != nil &&
		(vm.Identity.Type == compute.ResourceIdentityTypeUserAssigned ||
			vm.Identity.Type == compute.ResourceIdentityTypeSystemAssignedUserAssigned) &&
		vm.Identity.IdentityIds != nil {
		// Update the User Assigned Identity
		for _, id := range *vm.Identity.IdentityIds {
			if id == userAssignedMSIID {
				glog.Infof("ID: %s already found in vm identities", userAssignedMSIID)
				found = true
				break
			}
		}
		if !found {
			vmIDs := *vm.Identity.IdentityIds
			vmIDs = append(vmIDs, userAssignedMSIID)
			vm.Identity.IdentityIds = &vmIDs
		}
	} else { // No ids found yet.
		//c.VMClient.Client.RequestInspector = withInspection()
		var idType compute.ResourceIdentityType
		//glog.Infof("Got VM info: %+v. Assign %s\n", vm, userAssignedMSIID)
		if vm.Identity != nil && vm.Identity.Type == compute.ResourceIdentityTypeSystemAssigned {
			idType = compute.ResourceIdentityTypeSystemAssignedUserAssigned
		} else {
			idType = compute.ResourceIdentityTypeUserAssigned
		}
		vm.Identity = &compute.VirtualMachineIdentity{
			Type:        idType,
			IdentityIds: &[]string{userAssignedMSIID},
		}
	}

	timeStarted = time.Now()
	err = c.VMClient.CreateOrUpdate(c.Config.ResourceGroupName, nodeName, vm)
	if err != nil {
		return err
	}
	glog.V(6).Infof("CreateOrUpdate of %s completed in %s", nodeName, time.Since(timeStarted))
	return nil
}
