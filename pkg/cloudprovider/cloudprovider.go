package cloudprovider

import (
	"context"
	"fmt"
	"net/http"
	"time"

	config "github.com/Azure/aad-pod-identity/pkg/config"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/golang/glog"
)

type Client struct {
	VMClient  compute.VirtualMachinesClient
	ExtClient compute.VirtualMachineExtensionsClient
}

func NewCloudProvider(conf config.Config) (c *Client, e error) {

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
		VMClient:  virtualMachinesClient,
		ExtClient: extClient,
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

func (c *Client) RemoveUserMSI(userAssignedMSIID string, nodeName string, conf *config.Config) error {
	ctx := context.Background()
	glog.Infof("Find %s in resource group: %s", nodeName, conf.NodeResourceGroup)
	vm, err := c.VMClient.Get(ctx, conf.NodeResourceGroup, nodeName, "")
	if err != nil {
		return err
	}
	//c.VMClient.Client.RequestInspector = withInspection()
	//glog.Infof("Got VM info: %+v. Assign %s\n", vm, userAssignedMSIID)
	var newIds []string
	if vm.Identity != nil {
		//TODO: Handle both the system assigned and user assigned ID being present on one vm.
		if vm.Identity.Type == compute.ResourceIdentityTypeUserAssigned {
			index := 0
			for _, v := range *vm.Identity.IdentityIds {
				if v == userAssignedMSIID {
					glog.Infof("Removing user assigned msi: %s", v)
				} else {
					newIds[index] = v
					index++
				}
			}
			// TODO: Handle more conditions.
			// If the number went down, then we will update the vm.
			if index < len(*vm.Identity.IdentityIds) {
				if index == 0 { // Empty EMSI requires us to reset the type.
					// TODO: Handle the User assigned and regular MSI case.
					vm.Identity.Type = compute.ResourceIdentityTypeNone
					vm.Identity.IdentityIds = nil
				}
				err := c.CreateOrUpdate(conf.NodeResourceGroup, nodeName, vm)
				if err != nil {
					glog.Error(err)
					return err
				}
				return nil
			}
		} else {
			glog.Error("User assigned identity not found for node: %s ", nodeName)
			return fmt.Errorf("User assigned Identity not found for node: %s ", nodeName)
		}
	} else {
		glog.Errorf("Identity null for vm: %s ", nodeName)
		return fmt.Errorf("Identity null for vm: %s ", nodeName)
	}

	if len(newIds) != len(*vm.Identity.IdentityIds)-1 {
		glog.Errorf("Identity %s not found", userAssignedMSIID)
		return fmt.Errorf("Identity %s not found", userAssignedMSIID)
	}
	return nil
}

func (c *Client) CreateOrUpdate(rg string, nodeName string, vm compute.VirtualMachine) error {
	// Set the read-only property of extension to null.
	vm.Resources = nil
	ctx := context.Background()
	future, err := c.VMClient.CreateOrUpdate(ctx, rg, nodeName, vm)
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

func (c *Client) AssignUserMSI(userAssignedMSIID string, nodeName string, conf *config.Config) error {
	// Get the vm using the VmClient
	// Update the assigned identity into the VM using the CreateOrUpdate
	ctx := context.Background()
	glog.Infof("Find %s in resource group: %s", nodeName, conf.NodeResourceGroup)
	vm, err := c.VMClient.Get(ctx, conf.NodeResourceGroup, nodeName, "")
	if err != nil {
		return err
	}
	//c.VMClient.Client.RequestInspector = withInspection()
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
	err = c.CreateOrUpdate(conf.NodeResourceGroup, nodeName, vm)
	if err != nil {
		return err
	}
	return nil
}
