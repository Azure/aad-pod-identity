package config

// AzureConfig is representing /etc/kubernetes/azure.json
type AzureConfig struct {
	Cloud             string `json:"cloud"`
	TenantID          string `json:"tenantId"`
	ClientID          string `json:"aadClientId"`
	ClientSecret      string `json:"aadClientSecret"`
	SubscriptionID    string `json:"subscriptionId"`
	ResourceGroupName string `json:"resourceGroup"`
	SecurityGroupName string `json:"securityGroupName"`
}
