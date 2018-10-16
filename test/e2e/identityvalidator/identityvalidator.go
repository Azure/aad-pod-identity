package main

import (
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

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

	log.Infof("starting demo pod %s/%s %s", podnamespace, podname, podip)

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

	// Test 1
	if err := doARMOperations(logger, *subscriptionID, *resourceGroup); err != nil {
		logger.Fatalf("doARMOperations failed, %+v", err)
	}

	// Test 2
	t1, err := testMSIEndpoint(logger, msiEndpoint, *resource)
	if err != nil || t1 == nil {
		logger.Fatalf("testMSIEndpoint failed, %+v", err)
	}

	// Test 3
	t2, err := testMSIEndpointFromUserAssignedID(logger, msiEndpoint, *clientID, *resource)
	if err != nil || t2 == nil {
		logger.Fatalf("testMSIEndpointFromUserAssignedID failed, %+v", err)
	}

	// Check if the above two tokens are the same
	if !strings.EqualFold(t1.AccessToken, t2.AccessToken) {
		logger.Fatalf("msi, emsi test failed %+v %+v", t1, t2)
	}

	// Test 4
	if err := testInstanceMetadataRequests(logger); err != nil {
		logger.Fatalf("testInstanceMetadataRequests failed, %+v", err)
	}
}

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

// Simulates health probe of non MSI instance metadata requests
// e.g. curl -H Metadata:true "http://169.254.169.254/metadata/instance?api-version=2017-08-01" --header "X-Forwarded-For: 192.168.0.2"
func testInstanceMetadataRequests(logger *log.Entry) error {
	client := &http.Client{
		Timeout: time.Duration(2) * time.Second,
	}
	req, err := http.NewRequest("GET", "http://169.254.169.254/metadata/instance?api-version=2017-08-01", nil)
	req.Header.Add("Metadata", "true")
	resp, err := client.Do(req)
	if err != nil {
		logger.Errorf("failed, GET on instance metadata")
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	logger.Infof("succesfully made GET on instance metadata, %s", body)
	return nil
}
