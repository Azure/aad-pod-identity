// +build e2e_new

package azure

import (
	"context"

	"github.com/Azure/aad-pod-identity/test/e2e_new/framework"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/msi/mgmt/msi"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	. "github.com/onsi/gomega"
)

// Client defines the behavior of a type that acts as an intermediary with ARM.
type Client interface {
	// GetIdentityClientID returns the client ID of a user-assigned identity.
	GetIdentityClientID(identityName string) string
}

type client struct {
	config              *framework.Config
	identityClientIDMap map[string]string
	msiClient           msi.UserAssignedIdentitiesClient
}

// NewClient returns an implementation of Client given a test configuration.
func NewClient(config *framework.Config) Client {
	oauthConfig, err := getOAuthConfig(azure.PublicCloud, config.SubscriptionID, config.AzureTenantID)
	Expect(err).To(BeNil())

	armSpt, err := adal.NewServicePrincipalToken(*oauthConfig, config.AzureClientID, config.AzureClientSecret, azure.PublicCloud.ServiceManagementEndpoint)
	Expect(err).To(BeNil())

	c := &client{
		config:              config,
		identityClientIDMap: make(map[string]string),
		msiClient:           msi.NewUserAssignedIdentitiesClient(config.SubscriptionID),
	}

	authorizer := autorest.NewBearerAuthorizer(armSpt)
	c.msiClient.Authorizer = authorizer

	return c
}

// GetIdentityClientID returns the client ID of a user-assigned identity.
func (c *client) GetIdentityClientID(identityName string) string {
	if clientID, ok := c.identityClientIDMap[identityName]; ok {
		return clientID
	}

	result, err := c.msiClient.Get(context.TODO(), c.config.IdentityResourceGroup, identityName)
	if err != nil {
		// Dummy client ID
		return "00000000-0000-0000-0000-000000000000"
	}

	clientID := result.UserAssignedIdentityProperties.ClientID.String()
	c.identityClientIDMap[identityName] = clientID

	return clientID
}

func getOAuthConfig(env azure.Environment, subscriptionID, tenantID string) (*adal.OAuthConfig, error) {
	oauthConfig, err := adal.NewOAuthConfig(env.ActiveDirectoryEndpoint, tenantID)
	if err != nil {
		return nil, err
	}

	return oauthConfig, nil
}
