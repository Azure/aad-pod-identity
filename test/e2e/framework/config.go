// +build e2e

package framework

import (
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config holds global test configuration translated from environment variables
type Config struct {
	SubscriptionID                     string        `envconfig:"SUBSCRIPTION_ID"`
	ResourceGroup                      string        `envconfig:"RESOURCE_GROUP"`
	IdentityResourceGroup              string        `envconfig:"IDENTITY_RESOURCE_GROUP"`
	NodeResourceGroup                  string        `envconfig:"NODE_RESOURCE_GROUP"`
	AzureClientID                      string        `envconfig:"AZURE_CLIENT_ID"`
	AzureClientSecret                  string        `envconfig:"AZURE_CLIENT_SECRET"`
	AzureTenantID                      string        `envconfig:"AZURE_TENANT_ID"`
	KeyvaultName                       string        `envconfig:"KEYVAULT_NAME"`
	KeyvaultSecretName                 string        `envconfig:"KEYVAULT_SECRET_NAME"`
	KeyvaultSecretVersion              string        `envconfig:"KEYVAULT_SECRET_VERSION"`
	MICVersion                         string        `envconfig:"MIC_VERSION" default:"v1.8.1"`
	NMIVersion                         string        `envconfig:"NMI_VERSION" default:"v1.8.1"`
	Registry                           string        `envconfig:"REGISTRY" default:"mcr.microsoft.com/oss/azure/aad-pod-identity"`
	IdentityValidatorVersion           string        `envconfig:"IDENTITY_VALIDATOR_VERSION" default:"v1.8.1"`
	EnableScaleFeatures                bool          `envconfig:"ENABLE_SCALE_FEATURES" default:"true"`
	ImmutableUserMSIs                  string        `envconfig:"IMMUTABLE_IDENTITY_CLIENT_ID"`
	NMIMode                            string        `envconfig:"NMI_MODE" default:"standard"`
	BlockInstanceMetadata              bool          `envconfig:"BLOCK_INSTANCE_METADATA" default:"true"`
	IsSoakTest                         bool          `envconfig:"IS_SOAK_TEST" default:"false"`
	IdentityReconcileInterval          time.Duration `envconfig:"IDENTITY_RECONCILE_INTERVAL" default:"2m"`
	ServicePrincipalClientID           string        `envconfig:"SERVICE_PRINCIPAL_CLIENT_ID"`
	ServicePrincipalClientSecret       string        `envconfig:"SERVICE_PRINCIPAL_CLIENT_SECRET"`
	MICSyncInterval                    time.Duration `envconfig:"MIC_SYNC_INTERVAL" default:"30s"`
	SetRetryAfterHeader                bool          `envconfig:"SET_RETRY_AFTER_HEADER" default:"false"`
	RetryAttemptsForCreated            int           `envconfig:"RETRY_ATTEMPTS_FOR_CREATED" default:"16"`
	RetryAttemptsForAssigned           int           `envconfig:"RETRY_ATTEMPTS_FOR_ASSIGNED" default:"4"`
	FindIdentityRetryIntervalInSeconds int           `envconfig:"FIND_IDENTITY_RETRY_INTERVAL_IN_SECONDS" default:"5"`
}

func (c *Config) DeepCopy() *Config {
	copy := new(Config)
	copy.SubscriptionID = c.SubscriptionID
	copy.ResourceGroup = c.ResourceGroup
	copy.IdentityResourceGroup = c.IdentityResourceGroup
	copy.NodeResourceGroup = c.NodeResourceGroup
	copy.AzureClientID = c.AzureClientID
	copy.AzureClientSecret = c.AzureClientSecret
	copy.AzureTenantID = c.AzureTenantID
	copy.KeyvaultName = c.KeyvaultName
	copy.KeyvaultSecretName = c.KeyvaultSecretName
	copy.KeyvaultSecretVersion = c.KeyvaultSecretVersion
	copy.MICVersion = c.MICVersion
	copy.NMIVersion = c.NMIVersion
	copy.Registry = c.Registry
	copy.IdentityValidatorVersion = c.IdentityValidatorVersion
	copy.EnableScaleFeatures = c.EnableScaleFeatures
	copy.ImmutableUserMSIs = c.ImmutableUserMSIs
	copy.NMIMode = c.NMIMode
	copy.BlockInstanceMetadata = c.BlockInstanceMetadata
	copy.ServicePrincipalClientID = c.ServicePrincipalClientID
	copy.ServicePrincipalClientSecret = c.ServicePrincipalClientSecret
	copy.MICSyncInterval = c.MICSyncInterval
	copy.SetRetryAfterHeader = c.SetRetryAfterHeader
	copy.RetryAttemptsForCreated = c.RetryAttemptsForCreated
	copy.RetryAttemptsForAssigned = c.RetryAttemptsForAssigned
	copy.FindIdentityRetryIntervalInSeconds = c.FindIdentityRetryIntervalInSeconds

	return copy
}

// ParseConfig parses the needed environment variables for running the tests
func ParseConfig() (*Config, error) {
	c := new(Config)
	if err := envconfig.Process("config", c); err != nil {
		return c, err
	}

	if c.IdentityResourceGroup == "" {
		// Assume user-assigned identities are within the node resource group
		c.IdentityResourceGroup = c.ResourceGroup
	}
	if c.NodeResourceGroup == "" {
		c.NodeResourceGroup = c.ResourceGroup
	}
	c.IdentityResourceGroup = strings.ToLower(c.IdentityResourceGroup)
	c.NodeResourceGroup = strings.ToLower(c.NodeResourceGroup)

	return c, nil
}
