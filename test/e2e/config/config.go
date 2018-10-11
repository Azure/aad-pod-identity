package config

import (
	"github.com/kelseyhightower/envconfig"
)

// TODO: Add comments
type Config struct {
	SubscriptionID string `envconfig:"SUBSCRIPTION_ID"`
	ResourceGroup  string `envconfig:"RESOURCE_GROUP"`
	AzureClientID  string `envconfig:"AZURE_CLIENT_ID"`
}

// TODO: Add comments
func ParseConfig() (*Config, error) {
	c := new(Config)
	if err := envconfig.Process("config", c); err != nil {
		return nil, err
	}

	return c, nil
}
