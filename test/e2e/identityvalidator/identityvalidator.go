package main

import (
	"context"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"k8s.io/klog"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-12-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/auth"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/spf13/pflag"
)

var (
	subscriptionID        = pflag.String("subscription-id", "", "subscription id for test")
	identityClientID      = pflag.String("identity-client-id", "", "client id for the msi id")
	resourceGroup         = pflag.String("resource-group", "", "any resource group name with reader permission to the aad object")
	keyvaultName          = pflag.String("keyvault-name", "", "the name of the keyvault to extract the secret from")
	keyvaultSecretName    = pflag.String("keyvault-secret-name", "", "the name of the keyvault secret we are extracting with pod identity")
	keyvaultSecretVersion = pflag.String("keyvault-secret-version", "", "the version of the keyvault secret we are extracting with pod identity")
)

func main() {
	pflag.Parse()

	podname := os.Getenv("E2E_TEST_POD_NAME")
	podnamespace := os.Getenv("E2E_TEST_POD_NAMESPACE")
	podip := os.Getenv("E2E_TEST_POD_IP")

	klog.Infof("Starting identity validator pod %s/%s %s", podnamespace, podname, podip)

	msiEndpoint, err := adal.GetMSIVMEndpoint()
	if err != nil {
		klog.Fatalf("Failed to get msiEndpoint: %+v", err)
	}
	klog.Infof("Successfully obtain MSIEndpoint: %s\n", msiEndpoint)

	if *keyvaultName != "" && *keyvaultSecretName != "" {
		// Test if the pod identity is set up correctly
		if err := testUserAssignedIdentityOnPod(msiEndpoint, *identityClientID, *keyvaultName, *keyvaultSecretName, *keyvaultSecretVersion); err != nil {
			klog.Fatalf("testUserAssignedIdentityOnPod failed, %+v", err)
		}
	} else {
		// Test if the cluster-wide user assigned identity is set up correctly
		if err := testClusterWideUserAssignedIdentity(msiEndpoint, *subscriptionID, *resourceGroup, *identityClientID); err != nil {
			klog.Fatalf("testClusterWideUserAssignedIdentity failed, %+v", err)
		}
	}

	// Test if a service principal token can be obtained when using a system assigned identity
	if t1, err := testSystemAssignedIdentity(msiEndpoint); err != nil || t1 == nil {
		klog.Fatalf("testSystemAssignedIdentity failed, %+v", err)
	}
}

// testClusterWideUserAssignedIdentity will verify whether cluster-wide user assigned identity is working properly
func testClusterWideUserAssignedIdentity(msiEndpoint, subscriptionID, resourceGroup, identityClientID string) error {
	os.Setenv("AZURE_CLIENT_ID", identityClientID)
	defer os.Unsetenv("AZURE_CLIENT_ID")
	token, err := adal.NewServicePrincipalTokenFromMSIWithUserAssignedID(msiEndpoint, azure.PublicCloud.ResourceManagerEndpoint, identityClientID)
	if err != nil {
		return errors.Wrapf(err, "Failed to get service principal token from user assigned identity")
	}

	vmClient := compute.NewVirtualMachinesClient(subscriptionID)
	vmClient.Authorizer = autorest.NewBearerAuthorizer(token)
	vmlist, err := vmClient.List(context.Background(), resourceGroup)
	if err != nil {
		return errors.Wrapf(err, "Failed to verify cluster-wide user assigned identity")
	}

	klog.Infof("Successfully verified cluster-wide user assigned identity. VM count: %d", len(vmlist.Values()))
	return nil
}

// testUserAssignedIdentityOnPod will verify whether a pod identity is working properly
func testUserAssignedIdentityOnPod(msiEndpoint, identityClientID, keyvaultName, keyvaultSecretName, keyvaultSecretVersion string) error {
	// When new authorizer is created, azure-sdk-for-go  tries to create dataplane authorizer using MSI. It checks the AZURE_CLIENT_ID to get the client id
	// for the user assigned identity. If client id not found, then NewServicePrincipalTokenFromMSI is invoked instead of using the actual
	// user assigned identity. Setting this env var ensures we validate GetSecret using the desired user assigned identity.
	os.Setenv("AZURE_CLIENT_ID", identityClientID)
	defer os.Unsetenv("AZURE_CLIENT_ID")

	keyClient := keyvault.New()
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err == nil {
		keyClient.Authorizer = authorizer
	}

	klog.Infof("%s %s %s\n", keyvaultName, keyvaultSecretName, keyvaultSecretVersion)
	secret, err := keyClient.GetSecret(context.Background(), fmt.Sprintf("https://%s.vault.azure.net", keyvaultName), keyvaultSecretName, keyvaultSecretVersion)
	if err != nil || *secret.Value == "" {
		return errors.Wrapf(err, "Failed to verify user assigned identity on pod")
	}

	klog.Infof("Successfully verified user assigned identity on pod")
	return nil
}

// testMSIEndpoint will return a service principal token obtained through a system assigned identity
func testSystemAssignedIdentity(msiEndpoint string) (*adal.Token, error) {
	spt, err := adal.NewServicePrincipalTokenFromMSI(msiEndpoint, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to acquire a token using the MSI VM extension")
	}

	if err := spt.Refresh(); err != nil {
		return nil, errors.Wrapf(err, "Failed to refresh ServicePrincipalTokenFromMSI using the MSI VM extension, msiEndpoint(%s)", msiEndpoint)
	}

	token := spt.Token()
	if token.IsZero() {
		return nil, errors.Errorf("No token found, MSI VM extension, msiEndpoint(%s)", msiEndpoint)
	}

	klog.Infof("Successfully acquired a token using the MSI, msiEndpoint(%s)", msiEndpoint)
	return &token, nil
}
