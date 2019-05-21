package config

import "github.com/kelseyhightower/envconfig"

// Config holds global test configuration translated from environment variables
type Config struct {
	SubscriptionID           string `envconfig:"SUBSCRIPTION_ID"`
	ResourceGroup            string `envconfig:"RESOURCE_GROUP"`
	AzureClientID            string `envconfig:"AZURE_CLIENT_ID"`
	KeyvaultName             string `envconfig:"KEYVAULT_NAME"`
	KeyvaultSecretName       string `envconfig:"KEYVAULT_SECRET_NAME"`
	KeyvaultSecretVersion    string `envconfig:"KEYVAULT_SECRET_VERSION"`
	MICVersion               string `envconfig:"MIC_VERSION" default:"1.3"`
	NMIVersion               string `envconfig:"NMI_VERSION" default:"1.4"`
	Registry                 string `envconfig:"REGISTRY" default:"mcr.microsoft.com/k8s/aad-pod-identity"`
	IdentityValidatorVersion string `envconfig:"IDENTITY_VALIDATOR_VERSION" default:"1.4"`
}

// ParseConfig will parse needed environment variables for running the tests
func ParseConfig() (*Config, error) {
	c := new(Config)
	if err := envconfig.Process("config", c); err != nil {
		return nil, err
	}

	return c, nil
}
