package main

import (
	"context"
	"os"
	"strings"

	compute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

var (
	resource       = pflag.String("aad-resourcename", "https://management.azure.com/", "name of resource to grant token")
	subscriptionID = pflag.String("subscriptionid", "", "subscription id for test")
	clientID       = pflag.String("clientid", "", "client id for the msi id")
	resourceGroup  = pflag.String("resourcegroup", "", "any resource group name with reader permission to the aad object")
)

func main() {
	pflag.Parse()

	podname := os.Getenv("MY_POD_NAME")
	podnamespace := os.Getenv("MY_POD_NAME")
	podip := os.Getenv("MY_POD_IP")

	log.Infof("starting identity validator pod %s/%s %s", podnamespace, podname, podip)

	logger := log.WithFields(log.Fields{
		"podnamespace": podnamespace,
		"podname":      podname,
		"podip":        podip,
	})

	msiEndpoint, err := adal.GetMSIVMEndpoint()
	if err != nil {
		logger.Fatalf("Failed to get msiEndpoint: %+v", err)
	}
	logger.Infof("Successfully obtain msiEndpoint: %s", msiEndpoint)

	// Test if an ARM operation can be executed successfully through Managed Service Identity
	if err := doARMOperations(logger, *subscriptionID, *resourceGroup); err != nil {
		logger.Fatalf("doARMOperations failed, %+v", err)
	}

	// Test if a service principal token can be obtained when using a system assigned identity
	t1, err := testMSIEndpoint(logger, msiEndpoint, *resource)
	if err != nil || t1 == nil {
		logger.Fatalf("testMSIEndpoint failed, %+v", err)
	}

	// Test if a service principal token can be obtained when using a user assigned identity
	t2, err := testMSIEndpointFromUserAssignedID(logger, msiEndpoint, *clientID, *resource)
	if err != nil || t2 == nil {
		logger.Fatalf("testMSIEndpointFromUserAssignedID failed, %+v", err)
	}

	// Check if the above two tokens are the same
	if !strings.EqualFold(t1.AccessToken, t2.AccessToken) {
		logger.Fatalf("msi, emsi test failed %+v %+v", t1, t2)
	}
}

// doARMOperations will count how many Azure virtual machines are deployed in the resource group
func doARMOperations(logger *log.Entry, subscriptionID, resourceGroup string) error {
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		logger.Errorf("failed NewAuthorizerFromEnvironment  %+v", authorizer)
		return err
	}
	vmClient := compute.NewVirtualMachinesClient(subscriptionID)
	vmClient.Authorizer = authorizer
	vmlist, err := vmClient.List(context.Background(), resourceGroup)
	if err != nil {
		logger.Errorf("failed to list all vm %+v", err)
		return err
	}

	logger.Infof("succesfull doARMOperations vm count %d", len(vmlist.Values()))
	return nil
}

// testMSIEndpoint will return a service principal token obtained through a system assigned identity
func testMSIEndpoint(logger *log.Entry, msiEndpoint, resource string) (*adal.Token, error) {
	spt, err := adal.NewServicePrincipalTokenFromMSI(msiEndpoint, resource)
	if err != nil {
		logger.Errorf("failed to acquire a token using the MSI VM extension, Error: %+v", err)
		return nil, err
	}

	if err := spt.Refresh(); err != nil {
		logger.Errorf("failed to refresh ServicePrincipalTokenFromMSI using the MSI VM extension, msiEndpoint(%s)", msiEndpoint)
		return nil, err
	}

	token := spt.Token()
	if token.IsZero() {
		logger.Errorf("zero token found, MSI VM extension, msiEndpoint(%s)", msiEndpoint)
		return nil, err
	}

	logger.Infof("succesfully acquired a token using the MSI, msiEndpoint(%s)", msiEndpoint)
	return &token, nil
}

// testMSIEndpointFromUserAssignedID will return a service principal token obtained through a user assigned identity
func testMSIEndpointFromUserAssignedID(logger *log.Entry, msiEndpoint, userAssignedID, resource string) (*adal.Token, error) {
	spt, err := adal.NewServicePrincipalTokenFromMSIWithUserAssignedID(msiEndpoint, resource, userAssignedID)
	if err != nil {
		logger.Errorf("failed NewServicePrincipalTokenFromMSIWithUserAssignedID, clientID: %s Error: %+v", userAssignedID, err)
		return nil, err
	}

	if err := spt.Refresh(); err != nil {
		logger.Errorf("failed to refresh ServicePrincipalToken userAssignedID MSI, msiEndpoint(%s)", msiEndpoint)
		return nil, err
	}

	token := spt.Token()
	if token.IsZero() {
		logger.Errorf("zero token found, userAssignedID MSI, msiEndpoint(%s) clientID(%s)", msiEndpoint, userAssignedID)
		return nil, err
	}

	logger.Infof("succesfully acquired a token, userAssignedID MSI, msiEndpoint(%s) clientID(%s)", msiEndpoint, userAssignedID)
	return &token, err
}
