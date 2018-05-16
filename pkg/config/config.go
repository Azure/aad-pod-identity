package config

type Config struct {
	Cloud             string `json:"cloud" yaml:"cloud"`
	TenantID          string `json:"tenantId" yaml:"tenantId"`
	SubscriptionID    string `json:"subscriptionId" yaml:"subscriptionId"`
	NodeResourceGroup string `json:"nodeResourceGroup" yaml:"nodeResourceGroup"`
	AADClientID       string `json:"aadClientId" yaml:"aadClientId"`
	AADClientSecret   string `json:"aadClientSecret" yaml:"aadClientSecret"`
}
