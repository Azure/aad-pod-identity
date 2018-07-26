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
	Get(rgName string, nodeName string) (ret compute.VirtualMachine, err error)
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

func (c *VMClient) Get(rgName string, nodeName string) (ret compute.VirtualMachine, err error) {
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
