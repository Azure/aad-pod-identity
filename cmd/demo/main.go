package main

import (
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	compute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"k8s.io/klog"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure/auth"
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
	podnamespace := os.Getenv("MY_POD_NAMESPACE")
	podip := os.Getenv("MY_POD_IP")

	klog.Infof("starting demo pod %s/%s %s", podnamespace, podname, podip)

	msiEndpoint, err := adal.GetMSIVMEndpoint()
	if err != nil {
		klog.Fatalf("failed to get msiendpoint, %+v", err)
	}

	for {
		doARMOperations(*subscriptionID, *resourceGroup)

		t1 := testMSIEndpoint(msiEndpoint, *resource)
		if t1 == nil {
			klog.Errorf("testMSIEndpoint failed, %+v", err)
			continue
		}

		t2 := testMSIEndpointFromUserAssignedID(msiEndpoint, *clientID, *resource)
		if t2 == nil {
			klog.Errorf("testMSIEndpointFromUserAssignedID failed, %+v", err)
			continue
		}

		if !strings.EqualFold(t1.AccessToken, t2.AccessToken) {
			klog.Errorf("msi, emsi test failed %+v %+v", t1, t2)
		}

		testInstanceMetadataRequests()

		time.Sleep(time.Duration(*retryWaitTime) * time.Second)
	}
}

func doARMOperations(subscriptionID, resourceGroup string) {
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		klog.Errorf("failed NewAuthorizerFromEnvironment  %+v", err)
		return
	}
	vmClient := compute.NewVirtualMachinesClient(subscriptionID)
	vmClient.Authorizer = authorizer
	vmlist, err := vmClient.List(context.Background(), resourceGroup)
	if err != nil {
		klog.Errorf("failed list all vm %+v", err)
		return
	}

	klog.Infof("successful doARMOperations vm count %d", len(vmlist.Values()))
}

func testMSIEndpoint(msiEndpoint, resource string) *adal.Token {
	spt, err := adal.NewServicePrincipalTokenFromMSI(msiEndpoint, resource)
	if err != nil {
		klog.Errorf("failed to acquire a token using the MSI VM extension, Error: %+v", err)
		return nil
	}
	if err := spt.Refresh(); err != nil {
		klog.Errorf("failed to refresh ServicePrincipalTokenFromMSI using the MSI VM extension, msiEndpoint(%s)", msiEndpoint)
		return nil
	}
	token := spt.Token()
	if token.IsZero() {
		klog.Errorf("zero token found, MSI VM extension, msiEndpoint(%s)", msiEndpoint)
		return nil
	}
	klog.Infof("successfully acquired a token using the MSI, msiEndpoint(%s)", msiEndpoint)
	return &token
}

func testMSIEndpointFromUserAssignedID(msiEndpoint, userAssignedID, resource string) *adal.Token {
	spt, err := adal.NewServicePrincipalTokenFromMSIWithUserAssignedID(msiEndpoint, resource, userAssignedID)
	if err != nil {
		klog.Errorf("failed NewServicePrincipalTokenFromMSIWithUserAssignedID, clientID: %s Error: %+v", userAssignedID, err)
		return nil
	}
	if err := spt.Refresh(); err != nil {
		klog.Errorf("failed to refresh ServicePrincipalToken userAssignedID MSI, msiEndpoint(%s)", msiEndpoint)
		return nil
	}
	token := spt.Token()
	if token.IsZero() {
		klog.Errorf("zero token found, userAssignedID MSI, msiEndpoint(%s) clientID(%s)", msiEndpoint, userAssignedID)
		return nil
	}
	klog.Infof("successfully acquired a token, userAssignedID MSI, msiEndpoint(%s) clientID(%s)", msiEndpoint, userAssignedID)
	return &token
}

// simulates health probe of non MSI instance metadata requests
// e.g. curl -H Metadata:true "http://169.254.169.254/metadata/instance?api-version=2017-08-01" --header "X-Forwarded-For: 192.168.0.2"
func testInstanceMetadataRequests() {
	client := &http.Client{
		Timeout: time.Duration(2) * time.Second,
	}
	req, err := http.NewRequest("GET", "http://169.254.169.254/metadata/instance?api-version=2017-08-01", nil)
	if err != nil {
		klog.Error(err)
		return
	}
	req.Header.Add("Metadata", "true")
	resp, err := client.Do(req)
	if err != nil {
		klog.Error(err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		klog.Error(err)
		return
	}
	klog.Infof("successfully made GET on instance metadata, %s", body)
}
