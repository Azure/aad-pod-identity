package aadpodidentity

// TODO: Add comments
type Config struct {
	SubscriptionID string `envconfig:"SUBSCRIPTION_ID"`
	ResourceGroup  string `envconfig:"RESOURCE_GROUP"`
	AzureClientID  string `envconfig:"AZURE_CLIENT_ID"`
}

// TODO: Add comments
func ParseConfig() {

}
