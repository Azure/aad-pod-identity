package cloudprovider

import (
	"context"
	"time"

	"github.com/Azure/aad-pod-identity/pkg/config"
	"github.com/Azure/aad-pod-identity/pkg/metrics"
	"github.com/Azure/aad-pod-identity/pkg/stats"
	"github.com/Azure/aad-pod-identity/version"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
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
	Get(rgName, name string) (compute.VirtualMachineScaleSet, error)
	UpdateIdentities(rg, vmssName string, vmu compute.VirtualMachineScaleSet) error
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

// UpdateIdentities updates the user assigned identities for the provided node
func (c *VMSSClient) UpdateIdentities(rg, vmssName string, vmssIdentities compute.VirtualMachineScaleSet) error {
	var future compute.VirtualMachineScaleSetsUpdateFuture
	var err error
	ctx := context.Background()
	begin := time.Now()

	defer func() {
		if err != nil {
			c.reporter.ReportCloudProviderOperationError(metrics.PutVmssOperationName)
			return
		}
		c.reporter.ReportCloudProviderOperationDuration(metrics.PutVmssOperationName, time.Since(begin))
	}()

	if future, err = c.client.Update(ctx, rg, vmssName, compute.VirtualMachineScaleSetUpdate{
		Identity: vmssIdentities.Identity}); err != nil {
		klog.Errorf("Failed to update VM with error %v", err)
		return err
	}
	if err = future.WaitForCompletionRef(ctx, c.client.Client); err != nil {
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
	h.vmss.Identity = &compute.VirtualMachineScaleSetIdentity{
		Type:                   compute.ResourceIdentityTypeUserAssigned,
		UserAssignedIdentities: make(map[string]*compute.VirtualMachineScaleSetIdentityUserAssignedIdentitiesValue),
	}
	return h.IdentityInfo()
}

type vmssIdentityInfo struct {
	info *compute.VirtualMachineScaleSetIdentity
}

func (i *vmssIdentityInfo) RemoveUserIdentity(id string) bool {
	if i.info == nil {
		return false
	}
	if _, exists := i.info.UserAssignedIdentities[id]; !exists {
		return false
	}
	// set the user assigned id to nil so it can be removed
	i.info.UserAssignedIdentities[id] = nil
	return true
}

func (i *vmssIdentityInfo) AppendUserIdentity(id string) bool {
	if i.info.UserAssignedIdentities == nil {
		i.info.UserAssignedIdentities = make(map[string]*compute.VirtualMachineScaleSetIdentityUserAssignedIdentitiesValue)
		if i.info.Type == compute.ResourceIdentityTypeSystemAssigned {
			i.info.Type = compute.ResourceIdentityTypeSystemAssignedUserAssigned
		} else if i.info.Type == "" || i.info.Type == compute.ResourceIdentityTypeNone {
			i.info.Type = compute.ResourceIdentityTypeUserAssigned
		}
	}
	if _, exists := i.info.UserAssignedIdentities[id]; !exists {
		i.info.UserAssignedIdentities[id] = &compute.VirtualMachineScaleSetIdentityUserAssignedIdentitiesValue{}
		return true
	}
	return false
}

func (i *vmssIdentityInfo) GetUserIdentityList() []string {
	var ids []string
	if i.info == nil {
		return ids
	}
	for id := range i.info.UserAssignedIdentities {
		ids = append(ids, id)
	}
	return ids
}

func (i *vmssIdentityInfo) FinalizeUserIdentityList() {
	if i.info.UserAssignedIdentities == nil {
		i.info.Type = compute.ResourceIdentityTypeNone
		return
	}
	for _, val := range i.info.UserAssignedIdentities {
		if val != nil {
			// even if one identity value is not nil, then the identity type
			// can be user assigned
			return
		}
	}
	// If all the user assigned identities on node are to be deleted,
	// then type should be None and the assigned id list should be nil
	i.info.UserAssignedIdentities = nil

	if i.info.Type == compute.ResourceIdentityTypeSystemAssignedUserAssigned {
		i.info.Type = compute.ResourceIdentityTypeSystemAssigned
	} else {
		i.info.Type = compute.ResourceIdentityTypeNone
	}
}
