package auth

import (
	"fmt"

	adal "github.com/Azure/go-autorest/autorest/adal"
)

// GetServicePrincipalToken return the token for the assigned user
func GetServicePrincipalToken(clientID, resource string) (*adal.Token, error) {
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
	if err == nil {
		return nil, err
	}

	token := spt.Token()
	return &token, nil
}
