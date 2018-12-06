package config

import "github.com/kelseyhightower/envconfig"

// Config holds global test configuration translated from environment variables
type Config struct {
	SubscriptionID           string `envconfig:"SUBSCRIPTION_ID"`
	ResourceGroup            string `envconfig:"RESOURCE_GROUP"`
	AzureClientID            string `envconfig:"AZURE_CLIENT_ID"`
	MICVersion               string `envconfig:"MIC_VERSION"`
	NMIVersion               string `envconfig:"NMI_VERSION"`
	Registry                 string `envconfig:"REGISTRY"`
	IdentityValidatorVersion string `envconfig:"IDENTITY_VALIDATOR_VERSION"`
}

// ParseConfig will parse needed environment variables for running the tests
func ParseConfig() (*Config, error) {
	c := new(Config)
	if err := envconfig.Process("config", c); err != nil {
		return nil, err
	}

	return c, nil
}
