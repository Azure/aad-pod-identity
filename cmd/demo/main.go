package main

import (
	"context"
	"flag"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/adal"
	"k8s.io/klog/v2"
)

const (
	timeout = 80 * time.Second
)

var (
	period           time.Duration
	resourceName     string
	subscriptionID   string
	resourceGroup    string
	identityClientID string
)

func main() {
	flag.DurationVar(&period, "period", 100*time.Second, "The period that the demo is being executed")
	flag.StringVar(&resourceName, "resource-name", "https://management.azure.com/", "The resource name to grant the access token")
	flag.StringVar(&subscriptionID, "subscription-id", "", "The Azure subscription ID")
	flag.StringVar(&resourceGroup, "resource-group", "", "The resource group name which the user-assigned identity read access to")
	flag.StringVar(&identityClientID, "identity-client-id", "", "The user-assigned identity client ID")
	flag.Parse()

	imdsTokenEndpoint, err := adal.GetMSIVMEndpoint()
	if err != nil {
		klog.Fatalf("failed to get IMDS token endpoint, error: %+v", err)
	}

	ticker := time.NewTicker(period)
	defer ticker.Stop()

	for ; true; <-ticker.C {
		curlIMDSMetadataInstanceEndpoint()
		t1 := getTokenFromIMDS(imdsTokenEndpoint)
		t2 := getTokenFromIMDSWithUserAssignedID(imdsTokenEndpoint)
		if t1 == nil || t2 == nil || !strings.EqualFold(t1.AccessToken, t2.AccessToken) {
			klog.Error("Tokens acquired from IMDS with and without identity client ID do not match")
		}
		klog.Infof("Try decoding your token %s at https://jwt.io", t1.AccessToken)
	}
}

func getTokenFromIMDS(imdsTokenEndpoint string) *adal.Token {
	spt, err := adal.NewServicePrincipalTokenFromMSIWithUserAssignedID(imdsTokenEndpoint, resourceName, identityClientID)
	if err != nil {
		klog.Errorf("failed to acquire a token from IMDS using user-assigned identity, error: %+v", err)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := spt.RefreshWithContext(ctx); err != nil {
		klog.Errorf("failed to refresh the service principal token, error: %+v", err)
		return nil
	}

	token := spt.Token()
	if token.IsZero() {
		klog.Errorf("%+v is a zero token", token)
		return nil
	}

	klog.Infof("successfully acquired a service principal token from %s", imdsTokenEndpoint)
	return &token
}

func getTokenFromIMDSWithUserAssignedID(imdsTokenEndpoint string) *adal.Token {
	spt, err := adal.NewServicePrincipalTokenFromMSIWithUserAssignedID(imdsTokenEndpoint, resourceName, identityClientID)
	if err != nil {
		klog.Errorf("failed to acquire a token from IMDS using user-assigned identity, error: %+v", err)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := spt.RefreshWithContext(ctx); err != nil {
		klog.Errorf("failed to refresh the service principal token, error: %+v", err)
		return nil
	}

	token := spt.Token()
	if token.IsZero() {
		klog.Errorf("%+v is a zero token", token)
		return nil
	}

	klog.Infof("successfully acquired a service principal token from %s using a user-assigned identity (%s)", imdsTokenEndpoint, identityClientID)
	return &token
}

func curlIMDSMetadataInstanceEndpoint() {
	client := &http.Client{
		Timeout: timeout,
	}
	req, err := http.NewRequest("GET", "http://169.254.169.254/metadata/instance?api-version=2017-08-01", nil)
	if err != nil {
		klog.Errorf("failed to create a new HTTP request, error: %+v", err)
		return
	}
	req.Header.Add("Metadata", "true")

	resp, err := client.Do(req)
	if err != nil {
		klog.Error(err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		klog.Errorf("failed to read response body, error: %+v", err)
		return
	}

	klog.Infof(`curl -H Metadata:true "http://169.254.169.254/metadata/instance?api-version=2017-08-01": %s`, body)
}
