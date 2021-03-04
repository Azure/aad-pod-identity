package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/auth"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"k8s.io/klog/v2"
)

const (
	keyvaultResource = "https://vault.azure.net"
)

type keyvaultTester struct {
	client             keyvault.BaseClient
	subscriptionID     string
	identityClientID   string
	identityResourceID string
	keyvaultName       string
	secretName         string
	secretVersion      string
	secretValue        string
	imdsTokenEndpoint  string
}

// assertWithIdentityClientID obtains the secret value from a keyvault using
// aad-pod-identity and check if is the same as the expected secret values.
func (kvt *keyvaultTester) assertWithIdentityClientID() error {
	if kvt.identityClientID == "" {
		return nil
	}

	// When new authorizer is created, azure-sdk-for-go  tries to create data plane authorizer using MSI. It checks the AZURE_CLIENT_ID to get the client id
	// for the user assigned identity. If client id not found, then NewServicePrincipalTokenFromMSI is invoked instead of using the actual
	// user assigned identity. Setting this env var ensures we validate GetSecret using the desired user assigned identity.
	if err := os.Setenv("AZURE_CLIENT_ID", kvt.identityClientID); err != nil {
		return fmt.Errorf("failed to set AZURE_CLIENT_ID environment variable, error: %+v", err)
	}
	defer os.Unsetenv("AZURE_CLIENT_ID")

	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return fmt.Errorf("failed to generate a new authorizer from environment, error: %+v", err)
	}

	klog.Infof("added authorizer with clientID: %s", kvt.identityClientID)

	secret, err := kvt.getSecret(authorizer)
	if err != nil {
		return err
	}

	if err := kvt.assertSecret(secret); err != nil {
		return err
	}

	klog.Info("successfully verified user-assigned identity on pod with identity client ID")
	return nil
}

// assertWithIdentityResourceID obtains the secret value from a keyvault using
// aad-pod-identity and check if is the same as the expected secret values.
func (kvt *keyvaultTester) assertWithIdentityResourceID() error {
	if kvt.identityResourceID == "" {
		return nil
	}

	token, err := kvt.getADALTokenWithIdentityResourceID()
	if err != nil {
		return fmt.Errorf("failed to get ADAL token with identity resource ID, error: %+v", err)
	}

	klog.Infof("added authorizer with resource ID: %s", kvt.identityResourceID)

	secret, err := kvt.getSecret(autorest.NewBearerAuthorizer(token))
	if err != nil {
		return err
	}

	if err := kvt.assertSecret(secret); err != nil {
		return err
	}

	klog.Info("successfully verified user-assigned identity on pod with identity resource ID")
	return nil
}

// assertSecret checks if kvt.secretValue == actualSecret.
func (kvt *keyvaultTester) assertSecret(actualSecret string) error {
	if kvt.secretValue != actualSecret {
		return fmt.Errorf("expected %s to be equal to %s", actualSecret, kvt.secretValue)
	}

	return nil
}

// getSecret returns the secret value with a specific autorest authorizer.
func (kvt *keyvaultTester) getSecret(authorizer autorest.Authorizer) (string, error) {
	kvt.client.Authorizer = authorizer

	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()

	secret, err := kvt.client.GetSecret(ctx, kvt.getKeyvaultURL(), kvt.secretName, kvt.secretVersion)
	if err != nil {
		return "", fmt.Errorf("failed to get secret, error: %+v", err)
	}

	return *secret.Value, nil
}

// getKeyvaultURL returns the FQDN of the Azure Key Vault.
func (kvt *keyvaultTester) getKeyvaultURL() string {
	return fmt.Sprintf("https://%s.vault.azure.net", kvt.keyvaultName)
}

// getADALTokenWithIdentityResourceID returns an ADAL token
// using the resource ID of a user-assigned identity.
func (kvt *keyvaultTester) getADALTokenWithIdentityResourceID() (*adal.Token, error) {
	// Create HTTP request for a managed services for Azure resources token to access Azure Resource Manager
	imdsTokenURL, err := url.Parse(kvt.imdsTokenEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse IMDS token endpoint (%s), error: %+v", kvt.imdsTokenEndpoint, err)
	}

	v := url.Values{}
	v.Set("resource", keyvaultResource)
	v.Set("msi_res_id", kvt.identityResourceID)
	imdsTokenURL.RawQuery = v.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", imdsTokenURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create a new HTTP request, error: %+v", err)
	}
	req.Header.Add("Metadata", "true")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send a HTTP request, error: %+v", err)
	}
	defer resp.Body.Close()

	responseBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read the response body, error: %+v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Status Code = '%d'. Response body: %s", resp.StatusCode, string(responseBytes))
	}

	var token adal.Token
	err = json.Unmarshal(responseBytes, &token)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s, error: %+v", string(responseBytes), err)
	}

	return &token, nil
}
