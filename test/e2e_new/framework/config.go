// +build e2e_new

package framework

import (
	"strings"

	"github.com/kelseyhightower/envconfig"
)

// Config holds global test configuration translated from environment variables
type Config struct {
	SubscriptionID           string `envconfig:"SUBSCRIPTION_ID"`
	ResourceGroup            string `envconfig:"RESOURCE_GROUP"`
	IdentityResourceGroup    string `envconfig:"IDENTITY_RESOURCE_GROUP"`
	ClusterResourceGroup     string `envconfig:"CLUSTER_RESOURCE_GROUP"`
	AzureClientID            string `envconfig:"AZURE_CLIENT_ID"`
	AzureClientSecret        string `envconfig:"AZURE_CLIENT_SECRET"`
	AzureTenantID            string `envconfig:"AZURE_TENANT_ID"`
	KeyvaultName             string `envconfig:"KEYVAULT_NAME"`
	KeyvaultSecretName       string `envconfig:"KEYVAULT_SECRET_NAME"`
	KeyvaultSecretVersion    string `envconfig:"KEYVAULT_SECRET_VERSION"`
	MICVersion               string `envconfig:"MIC_VERSION" default:"1.6.1"`
	NMIVersion               string `envconfig:"NMI_VERSION" default:"1.6.1"`
	Registry                 string `envconfig:"REGISTRY" default:"mcr.microsoft.com/k8s/aad-pod-identity"`
	IdentityValidatorVersion string `envconfig:"IDENTITY_VALIDATOR_VERSION" default:"1.6.1"`
	SystemMSICluster         bool   `envconfig:"SYSTEM_MSI_CLUSTER" default:"false"`
	EnableScaleFeatures      bool   `envconfig:"ENABLE_SCALE_FEATURES" default:"false"`
	ImmutableUserMSIs        string `envconfig:"IMMUTABLE_IDENTITY_CLIENT_ID"`
	NmiMode                  string `envconfig:"NMI_MODE" default:"standard"`
}

// ParseConfig parses the needed environment variables for running the tests
func ParseConfig() (*Config, error) {
	c := new(Config)
	if err := envconfig.Process("config", c); err != nil {
		return c, err
	}

	if c.IdentityResourceGroup == "" {
		// Assume user-assigned identities are within the cluster resource group
		c.IdentityResourceGroup = c.ResourceGroup
	}
	if c.ClusterResourceGroup == "" {
		c.ClusterResourceGroup = c.ResourceGroup
	}
	c.IdentityResourceGroup = strings.ToLower(c.IdentityResourceGroup)
	c.ClusterResourceGroup = strings.ToLower(c.ClusterResourceGroup)

	return c, nil
}
