package config

// AzureConfig is representing /etc/kubernetes/azure.json
type AzureConfig struct {
	Cloud             string `json:"cloud" yaml:"cloud"`
	TenantID          string `json:"tenantId" yaml:"tenantId"`
	ClientID          string `json:"aadClientId" yaml:"aadClientId"`
	ClientSecret      string `json:"aadClientSecret" yaml:"aadClientSecret"`
	SubscriptionID    string `json:"subscriptionId" yaml:"subscriptionId"`
	ResourceGroupName string `json:"resourceGroup" yaml:"resourceGroup"`
	SecurityGroupName string `json:"securityGroupName" yaml:"securityGroupName"`
}
