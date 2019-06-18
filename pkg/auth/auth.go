package auth

import (
	"fmt"

	adal "github.com/Azure/go-autorest/autorest/adal"
)

const (
	activeDirectoryEndpoint = "https://login.microsoftonline.com/"
)

// GetServicePrincipalTokenFromMSIWithUserAssignedID return the token for the assigned user
func GetServicePrincipalTokenFromMSIWithUserAssignedID(clientID, resource string) (*adal.Token, error) {
	// Get the MSI endpoint accoriding with the OS (Linux/Windows)
	msiEndpoint, err := adal.GetMSIVMEndpoint()
	if err != nil {
		return nil, fmt.Errorf("Failed to get the MSI endpoint. Error: %v", err)
	}
	// The ID of the user for whom the token is requested
	userAssignedID := clientID
	// Set up the configuration of the service principal
	spt, err := adal.NewServicePrincipalTokenFromMSIWithUserAssignedID(msiEndpoint, resource, userAssignedID)
	if err != nil {
		return nil, fmt.Errorf("Failed to acquire a token using the MSI VM extension. Error: %v", err)
	}
	// Evectively acqurie the token
	err = spt.Refresh()
	if err != nil {
		return nil, err
	}
	token := spt.Token()

	return &token, nil
}

// GetServicePrincipalToken return the token for the assigned user
func GetServicePrincipalToken(tenantID, clientID, secret, resource string) (*adal.Token, error) {
	oauthConfig, err := adal.NewOAuthConfig(activeDirectoryEndpoint, tenantID)
	if err != nil {
		return nil, fmt.Errorf("creating the OAuth config: %v", err)
	}
	spt, err := adal.NewServicePrincipalToken(
		*oauthConfig,
		clientID,
		secret,
		resource,
	)
	if err != nil {
		return nil, err
	}
	// Evectively acqurie the token
	err = spt.Refresh()
	if err != nil {
		return nil, err
	}
	token := spt.Token()

	return &token, nil
}
