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
	retryWaitTime  = pflag.Int("retry-wait-time", 20, "retry wait time in seconds")
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
		logger.Fatalf("failed to get msiendpoint, %+v", err)
	}

	for {
		doARMOperations(logger, *subscriptionID, *resourceGroup)

		t1 := testMSIEndpoint(logger, msiEndpoint, *resource)
		if t1 == nil {
			logger.Errorf("testMSIEndpoint failed, %+v", err)
			continue
		}

		t2 := testMSIEndpointFromUserAssignedID(logger, msiEndpoint, *clientID, *resource)
		if t2 == nil {
			logger.Errorf("testMSIEndpointFromUserAssignedID failed, %+v", err)
			continue
		}

		if !strings.EqualFold(t1.AccessToken, t2.AccessToken) {
			logger.Errorf("msi, emsi test failed %+v %+v", t1, t2)
		}

		testInstanceMetadataRequests(logger)

		time.Sleep(time.Duration(*retryWaitTime) * time.Second)
	}
}

func doARMOperations(logger *log.Entry, subscriptionID, resourceGroup string) {
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		logger.Errorf("failed NewAuthorizerFromEnvironment  %+v", authorizer)
		return
	}
	vmClient := compute.NewVirtualMachinesClient(subscriptionID)
	vmClient.Authorizer = authorizer
	vmlist, err := vmClient.List(context.Background(), resourceGroup)
	if err != nil {
		logger.Errorf("failed list all vm %+v", err)
		return
	}

	logger.Infof("succesfull doARMOperations vm count %d", len(vmlist.Values()))
}

func testMSIEndpoint(logger *log.Entry, msiEndpoint, resource string) *adal.Token {
	spt, err := adal.NewServicePrincipalTokenFromMSI(msiEndpoint, resource)
	if err != nil {
		logger.Errorf("failed to acquire a token using the MSI VM extension, Error: %+v", err)
		return nil
	}
	if err := spt.Refresh(); err != nil {
		logger.Errorf("failed to refresh ServicePrincipalTokenFromMSI using the MSI VM extension, msiEndpoint(%s)", msiEndpoint)
		return nil
	}
	token := spt.Token()
	if token.IsZero() {
		logger.Errorf("zero token found, MSI VM extension, msiEndpoint(%s)", msiEndpoint)
		return nil
	}
	logger.Infof("succesfully acquired a token using the MSI, msiEndpoint(%s)", msiEndpoint)
	return &token
}

func testMSIEndpointFromUserAssignedID(logger *log.Entry, msiEndpoint, userAssignedID, resource string) *adal.Token {
	spt, err := adal.NewServicePrincipalTokenFromMSIWithUserAssignedID(msiEndpoint, resource, userAssignedID)
	if err != nil {
		logger.Errorf("failed NewServicePrincipalTokenFromMSIWithUserAssignedID, clientID: %s Error: %+v", userAssignedID, err)
		return nil
	}
	if err := spt.Refresh(); err != nil {
		logger.Errorf("failed to refresh ServicePrincipalToken userAssignedID MSI, msiEndpoint(%s)", msiEndpoint)
		return nil
	}
	token := spt.Token()
	if token.IsZero() {
		logger.Errorf("zero token found, userAssignedID MSI, msiEndpoint(%s) clientID(%s)", msiEndpoint, userAssignedID)
		return nil
	}
	logger.Infof("succesfully acquired a token, userAssignedID MSI, msiEndpoint(%s) clientID(%s)", msiEndpoint, userAssignedID)
	return &token
}

// simulates health probe of non MSI instance metadata requests
// e.g. curl -H Metadata:true "http://169.254.169.254/metadata/instance?api-version=2017-08-01" --header "X-Forwarded-For: 192.168.0.2"
func testInstanceMetadataRequests(logger *log.Entry) {
	client := &http.Client{
		Timeout: time.Duration(2) * time.Second,
	}
	req, err := http.NewRequest("GET", "http://169.254.169.254/metadata/instance?api-version=2017-08-01", nil)
	req.Header.Add("Metadata", "true")
	resp, err := client.Do(req)
	if err != nil {
		logger.Errorf("failed, GET on instance metadata")
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	logger.Infof("succesfully made GET on instance metadata, %s", body)
}
