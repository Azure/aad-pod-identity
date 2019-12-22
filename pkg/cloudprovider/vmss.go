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
	"k8s.io/klog"
)

// VMSSClient is used to interact with Azure virtual machine scale sets.
type VMSSClient struct {
	client   compute.VirtualMachineScaleSetsClient
	reporter *metrics.Reporter
}

// VMSSClientInt is the interface used by "cloudprovider" for interacting with Azure vmss
type VMSSClientInt interface {
	CreateOrUpdate(rg, name string, vm compute.VirtualMachineScaleSet) error
	Get(rgName, name string) (compute.VirtualMachineScaleSet, error)
}

// NewVMSSClient creates a new vmss client.
func NewVMSSClient(config config.AzureConfig, spt *adal.ServicePrincipalToken) (c *VMSSClient, e error) {
	client := compute.NewVirtualMachineScaleSetsClient(config.SubscriptionID)

	azureEnv, err := azure.EnvironmentFromName(config.Cloud)
	if err != nil {
		klog.Errorf("Get cloud env error: %+v", err)
		return nil, err
	}
	client.BaseURI = azureEnv.ResourceManagerEndpoint
	client.Authorizer = autorest.NewBearerAuthorizer(spt)
	client.PollingDelay = 5 * time.Second
	client.AddToUserAgent(version.GetUserAgent("MIC", version.MICVersion))

	reporter, err := metrics.NewReporter()
	if err != nil {
		klog.Errorf("New reporter error: %+v", err)
		return nil, err
	}

	return &VMSSClient{
		client:   client,
		reporter: reporter,
	}, nil
}

// CreateOrUpdate creates a new vmss, or if the vmss already exists it updates the existing one.
// This is used by "cloudprovider" to *update* add/remove identities from an already existing vmss.
func (c *VMSSClient) CreateOrUpdate(rg string, vmssName string, vm compute.VirtualMachineScaleSet) error {
	// Set the read-only property of extension to null.
	//vm.Resources = nil
	ctx := context.Background()
	begin := time.Now()
	var err error

	defer func() {
		if err != nil {
			c.reporter.ReportCloudProviderOperationError(metrics.PutVmssOperationName)
			return
		}
		c.reporter.ReportCloudProviderOperationDuration(metrics.PutVmssOperationName, time.Since(begin))
	}()

	future, err := c.client.CreateOrUpdate(ctx, rg, vmssName, vm)
	if err != nil {
		klog.Error(err)
		return err
	}

	err = future.WaitForCompletionRef(ctx, c.client.Client)
	if err != nil {
		klog.Error(err)
		return err
	}
	stats.UpdateCount(stats.TotalPutCalls, 1)
	stats.Update(stats.CloudPut, time.Since(begin))
	return nil
}

// Get gets the passed in vmss.
func (c *VMSSClient) Get(rgName string, vmssName string) (ret compute.VirtualMachineScaleSet, err error) {
	ctx := context.Background()
	begin := time.Now()

	defer func() {
		if err != nil {
			c.reporter.ReportCloudProviderOperationError(metrics.GetVmssOperationName)
			return
		}
		c.reporter.ReportCloudProviderOperationDuration(metrics.GetVmssOperationName, time.Since(begin))
	}()
	vm, err := c.client.Get(ctx, rgName, vmssName)
	if err != nil {
		klog.Error(err)
		return vm, err
	}
	stats.UpdateCount(stats.TotalGetCalls, 1)
	stats.Update(stats.CloudGet, time.Since(begin))
	return vm, nil
}

// vmssIdentityHolder implements `IdentityHolder` for vmss resources.
type vmssIdentityHolder struct {
	vmss *compute.VirtualMachineScaleSet
}

func (h *vmssIdentityHolder) IdentityInfo() IdentityInfo {
	if h.vmss.Identity == nil {
		return nil
	}
	return &vmssIdentityInfo{h.vmss.Identity}
}

func (h *vmssIdentityHolder) ResetIdentity() IdentityInfo {
	h.vmss.Identity = &compute.VirtualMachineScaleSetIdentity{}
	return h.IdentityInfo()
}

type vmssIdentityInfo struct {
	info *compute.VirtualMachineScaleSetIdentity
}

func (i *vmssIdentityInfo) RemoveUserIdentity(id string) error {
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

func (i *vmssIdentityInfo) AppendUserIdentity(id string) bool {
	if i.info.IdentityIds == nil {
		var ids []string
		i.info.IdentityIds = &ids
	}
	return appendUserIdentity(&i.info.Type, i.info.IdentityIds, id)
}

func (i *vmssIdentityInfo) GetUserIdentityList() []string {
	if i.info.IdentityIds == nil {
		return []string{}
	}
	return *i.info.IdentityIds
}
