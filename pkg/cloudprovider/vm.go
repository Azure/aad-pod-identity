package cloudprovider

import (
	"context"
	"time"

	"github.com/Azure/aad-pod-identity/pkg/config"
	"github.com/Azure/aad-pod-identity/pkg/stats"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/golang/glog"
)

type VMClient struct {
	client compute.VirtualMachinesClient
}

type VMClientInt interface {
	CreateOrUpdate(rg string, nodeName string, vm compute.VirtualMachine) error
	Get(rgName string, nodeName string) (compute.VirtualMachine, error)
}

func NewVirtualMachinesClient(config config.AzureConfig, spt *adal.ServicePrincipalToken) (c *VMClient, e error) {
	client := compute.NewVirtualMachinesClient(config.SubscriptionID)
	client.BaseURI = azure.PublicCloud.ResourceManagerEndpoint
	client.Authorizer = autorest.NewBearerAuthorizer(spt)
	client.PollingDelay = 5 * time.Second
	return &VMClient{
		client: client,
	}, nil
}

func (c *VMClient) CreateOrUpdate(rg string, nodeName string, vm compute.VirtualMachine) error {
	// Set the read-only property of extension to null.
	vm.Resources = nil
	ctx := context.Background()
	begin := time.Now()
	future, err := c.client.CreateOrUpdate(ctx, rg, nodeName, vm)
	if err != nil {
		glog.Error(err)
		return err
	}

	err = future.WaitForCompletion(ctx, c.client.Client)
	if err != nil {
		glog.Error(err)
		return err
	}

	vm, err = future.Result(c.client)
	if err != nil {
		glog.Error(err)
		return err
	}
	stats.Update(stats.CloudPut, time.Since(begin))
	return nil
}

func (c *VMClient) Get(rgName string, nodeName string) (compute.VirtualMachine, error) {
	ctx := context.Background()
	beginGetTime := time.Now()
	vm, err := c.client.Get(ctx, rgName, nodeName, "")
	if err != nil {
		glog.Error(err)
		return vm, err
	}
	stats.Update(stats.CloudGet, time.Since(beginGetTime))
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
	if i.info.Type == compute.ResourceIdentityTypeNone {
		i.info.IdentityIds = nil
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
