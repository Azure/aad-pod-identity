package cloudprovider

import (
	"context"
	"time"

	"github.com/Azure/aad-pod-identity/pkg/config"
	"github.com/Azure/aad-pod-identity/pkg/metrics"
	"github.com/Azure/aad-pod-identity/pkg/stats"
	"github.com/Azure/aad-pod-identity/version"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/golang/glog"
)

// VMClient client for VirtualMachines
type VMClient struct {
	client   compute.VirtualMachinesClient
	reporter *metrics.Reporter
}

// VMClientInt is the interface used by "cloudprovider" for interacting with Azure vmas
type VMClientInt interface {
	CreateOrUpdate(rg string, nodeName string, vm compute.VirtualMachine) error
	Get(rgName string, nodeName string) (compute.VirtualMachine, error)
}

// NewVirtualMachinesClient creates a new vm client.
func NewVirtualMachinesClient(config config.AzureConfig, spt *adal.ServicePrincipalToken) (c *VMClient, e error) {
	client := compute.NewVirtualMachinesClient(config.SubscriptionID)

	azureEnv, err := azure.EnvironmentFromName(config.Cloud)
	if err != nil {
		glog.Errorf("Get cloud env error: %+v", err)
		return nil, err
	}
	client.BaseURI = azureEnv.ResourceManagerEndpoint
	client.Authorizer = autorest.NewBearerAuthorizer(spt)
	client.PollingDelay = 5 * time.Second
	client.AddToUserAgent(version.GetUserAgent("MIC", version.MICVersion))

	reporter, err := metrics.NewReporter()
	if err != nil {
		glog.Errorf("New reporter error: %+v", err)
		return nil, err
	}

	return &VMClient{
		client:   client,
		reporter: reporter,
	}, nil
}

// CreateOrUpdate creates a new vm, or if the vm already exists it updates the existing one.
// This is used by "cloudprovider" to *update* add/remove identities from an already existing vm.
func (c *VMClient) CreateOrUpdate(rg string, nodeName string, vm compute.VirtualMachine) error {
	// Set the read-only property of extension to null.
	vm.Resources = nil
	ctx := context.Background()
	begin := time.Now()
	var err error

	defer func() {
		if err != nil {
			c.reporter.ReportCloudProviderOperationError(metrics.PutVMOperationName)
			return
		}
		c.reporter.ReportCloudProviderOperationDuration(metrics.PutVMOperationName, time.Since(begin))
	}()

	future, err := c.client.CreateOrUpdate(ctx, rg, nodeName, vm)
	if err != nil {
		glog.Error(err)
		return err
	}

	err = future.WaitForCompletionRef(ctx, c.client.Client)
	if err != nil {
		glog.Error(err)
		return err
	}
	stats.UpdateCount(stats.TotalPutCalls, 1)
	stats.Update(stats.CloudPut, time.Since(begin))
	return nil
}

// Get gets the passed in vm.
func (c *VMClient) Get(rgName string, nodeName string) (compute.VirtualMachine, error) {
	ctx := context.Background()
	begin := time.Now()
	var err error

	defer func() {
		if err != nil {
			c.reporter.ReportCloudProviderOperationError(metrics.GetVMOperationName)
			return
		}
		c.reporter.ReportCloudProviderOperationDuration(metrics.GetVMOperationName, time.Since(begin))
	}()

	vm, err := c.client.Get(ctx, rgName, nodeName, "")
	if err != nil {
		glog.Error(err)
		return vm, err
	}
	stats.UpdateCount(stats.TotalGetCalls, 1)
	stats.Update(stats.CloudGet, time.Since(begin))
	return vm, nil
}

type vmIdentityHolder struct {
	vm *compute.VirtualMachine
}

func (h *vmIdentityHolder) IdentityInfo() IdentityInfo {
	if h.vm.Identity == nil {
		return nil
	}
	return &vmIdentityInfo{h.vm.Identity}
}

func (h *vmIdentityHolder) ResetIdentity() IdentityInfo {
	h.vm.Identity = &compute.VirtualMachineIdentity{}
	return h.IdentityInfo()
}

type vmIdentityInfo struct {
	info *compute.VirtualMachineIdentity
}

func (i *vmIdentityInfo) RemoveUserIdentity(id string) error {
	if err := filterUserIdentity(&i.info.Type, i.info.IdentityIds, id); err != nil {
		return err
	}
	// If we have either no identity assigned or have the system assigned identity only, then we need to set the
	// IdentityIds list as nil.
	if i.info.Type == compute.ResourceIdentityTypeNone || i.info.Type == compute.ResourceIdentityTypeSystemAssigned {
		i.info.IdentityIds = nil
	}
	// if the identityids is nil and identity type is not set, then set it to ResourceIdentityTypeNone
	if i.info.IdentityIds == nil && i.info.Type == "" {
		i.info.Type = compute.ResourceIdentityTypeNone
	}
	return nil
}

func (i *vmIdentityInfo) AppendUserIdentity(id string) bool {
	if i.info.IdentityIds == nil {
		var ids []string
		i.info.IdentityIds = &ids
	}
	return appendUserIdentity(&i.info.Type, i.info.IdentityIds, id)
}

func (i *vmIdentityInfo) GetUserIdentityList() []string {
	if i.info.IdentityIds == nil {
		return []string{}
	}
	return *i.info.IdentityIds
}
