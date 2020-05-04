package cloudprovider

import (
	"context"
	"time"

	"github.com/Azure/aad-pod-identity/pkg/config"
	"github.com/Azure/aad-pod-identity/pkg/metrics"
	"github.com/Azure/aad-pod-identity/pkg/stats"
	"github.com/Azure/aad-pod-identity/version"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-12-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"k8s.io/klog"
)

// VMClient client for VirtualMachines
type VMClient struct {
	client   compute.VirtualMachinesClient
	reporter *metrics.Reporter
}

// VMClientInt is the interface used by "cloudprovider" for interacting with Azure vmas
type VMClientInt interface {
	Get(rgName string, nodeName string) (compute.VirtualMachine, error)
	UpdateIdentities(rg, nodeName string, vmu compute.VirtualMachine) error
}

// NewVirtualMachinesClient creates a new vm client.
func NewVirtualMachinesClient(config config.AzureConfig, spt *adal.ServicePrincipalToken) (c *VMClient, e error) {
	client := compute.NewVirtualMachinesClient(config.SubscriptionID)

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

	return &VMClient{
		client:   client,
		reporter: reporter,
	}, nil
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
		klog.Error(err)
		return vm, err
	}
	stats.UpdateCount(stats.TotalGetCalls, 1)
	stats.Update(stats.CloudGet, time.Since(begin))
	return vm, nil
}

// UpdateIdentities updates the user assigned identities for the provided node
func (c *VMClient) UpdateIdentities(rg, nodeName string, vm compute.VirtualMachine) error {
	var future compute.VirtualMachinesUpdateFuture
	var err error
	ctx := context.Background()
	begin := time.Now()

	defer func() {
		if err != nil {
			c.reporter.ReportCloudProviderOperationError(metrics.PutVMOperationName)
			return
		}
		c.reporter.ReportCloudProviderOperationDuration(metrics.PutVMOperationName, time.Since(begin))
	}()

	if future, err = c.client.Update(ctx, rg, nodeName, compute.VirtualMachineUpdate{
		Identity: vm.Identity}); err != nil {
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

func (i *vmIdentityInfo) GetUserIdentityList() []string {
	var ids []string
	if i.info == nil {
		return ids
	}
	for id := range i.info.UserAssignedIdentities {
		ids = append(ids, id)
	}
	return ids
}

func (i *vmIdentityInfo) SetUserIdentities(ids map[string]bool) bool {
	if i.info.UserAssignedIdentities == nil {
		i.info.UserAssignedIdentities = make(map[string]*compute.VirtualMachineIdentityUserAssignedIdentitiesValue)
	}

	nodeList := make(map[string]bool)
	// add all current existing ids
	for id := range i.info.UserAssignedIdentities {
		nodeList[id] = true
	}

	// add and remove the new list of identities keeping the same type as before
	userAssignedIdentities := make(map[string]*compute.VirtualMachineIdentityUserAssignedIdentitiesValue)
	for id, add := range ids {
		_, exists := nodeList[id]
		// already exists on node and want to remove existing identity
		if exists && !add {
			userAssignedIdentities[id] = nil
			delete(nodeList, id)
		}
		// doesn't exist on the node and want to add new identity
		if !exists && add {
			userAssignedIdentities[id] = &compute.VirtualMachineIdentityUserAssignedIdentitiesValue{}
			nodeList[id] = true
		}
		// exists and add - will already be in the nodeList and no need to patch for it
		// not exists and delete - no need to patch it as it already doesn't exist
	}

	// all identities are the node are to be removed
	if len(nodeList) == 0 {
		i.info.UserAssignedIdentities = nil
		if i.info.Type == compute.ResourceIdentityTypeSystemAssignedUserAssigned {
			i.info.Type = compute.ResourceIdentityTypeSystemAssigned
		} else {
			i.info.Type = compute.ResourceIdentityTypeNone
		}
		return true
	}

	i.info.Type = getUpdatedResourceIdentityType(i.info.Type)
	i.info.UserAssignedIdentities = userAssignedIdentities
	return len(i.info.UserAssignedIdentities) > 0
}
