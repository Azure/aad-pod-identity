package main

import (
	"context"
	"fmt"
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
	managedIdentityOptions := &adal.ManagedIdentityOptions{IdentityResourceID: kvt.identityResourceID}
	spt, err := adal.NewServicePrincipalTokenFromManagedIdentity(keyvaultResource, managedIdentityOptions)
	if err != nil {
		return nil, err
	}
	err = spt.Refresh()
	if err != nil {
		return nil, err
	}
	token := spt.Token()
	return &token, nil
}
