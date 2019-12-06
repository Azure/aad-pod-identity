package auth

import (
	"fmt"
	"time"

	"github.com/Azure/aad-pod-identity/pkg/metrics"
	"github.com/Azure/aad-pod-identity/version"
	adal "github.com/Azure/go-autorest/autorest/adal"
)

const (
	activeDirectoryEndpoint                         = "https://login.microsoftonline.com/"
	adalTokenFromMSIOperationName                   = "adal_token_msi"
	adalTokenFromMSIWithUserAssignedIDOperationName = "adal_token_msi_userassignedid"
	adalTokenOperationName                          = "adal_token"
)

var reporter *metrics.Reporter

// GetServicePrincipalTokenFromMSI return the token for the assigned user
func GetServicePrincipalTokenFromMSI(resource string) (*adal.Token, error) {
	begin := time.Now()
	// Get the MSI endpoint accoriding with the OS (Linux/Windows)
	msiEndpoint, err := adal.GetMSIVMEndpoint()
	if err != nil {
		recordError(adalTokenFromMSIOperationName)
		return nil, fmt.Errorf("Failed to get the MSI endpoint. Error: %v", err)
	}
	// Set up the configuration of the service principal
	spt, err := adal.NewServicePrincipalTokenFromMSI(msiEndpoint, resource)
	if err != nil {
		recordError(adalTokenFromMSIOperationName)
		return nil, fmt.Errorf("Failed to acquire a token for MSI. Error: %v", err)
	}
	// Effectively acquire the token
	err = spt.Refresh()
	if err != nil {
		recordError(adalTokenFromMSIOperationName)
		return nil, err
	}
	token := spt.Token()
	recordDuration(adalTokenFromMSIOperationName, time.Since(begin))
	return &token, nil
}

// GetServicePrincipalTokenFromMSIWithUserAssignedID return the token for the assigned user
func GetServicePrincipalTokenFromMSIWithUserAssignedID(clientID, resource string) (*adal.Token, error) {
	begin := time.Now()
	// Get the MSI endpoint accoriding with the OS (Linux/Windows)
	msiEndpoint, err := adal.GetMSIVMEndpoint()
	if err != nil {
		recordError(adalTokenFromMSIWithUserAssignedIDOperationName)
		return nil, fmt.Errorf("Failed to get the MSI endpoint. Error: %v", err)
	}
	// The ID of the user for whom the token is requested
	userAssignedID := clientID
	// Set up the configuration of the service principal
	spt, err := adal.NewServicePrincipalTokenFromMSIWithUserAssignedID(msiEndpoint, resource, userAssignedID)
	if err != nil {
		recordError(adalTokenFromMSIWithUserAssignedIDOperationName)
		return nil, fmt.Errorf("Failed to acquire a token using the MSI VM extension. Error: %v", err)
	}

	// Effectively acquire the token
	err = spt.Refresh()
	if err != nil {
		recordError(adalTokenFromMSIWithUserAssignedIDOperationName)
		return nil, err
	}
	token := spt.Token()
	recordDuration(adalTokenFromMSIWithUserAssignedIDOperationName, time.Since(begin))
	return &token, nil
}

// GetServicePrincipalToken return the token for the assigned user
func GetServicePrincipalToken(tenantID, clientID, secret, resource string) (*adal.Token, error) {
	begin := time.Now()
	oauthConfig, err := adal.NewOAuthConfig(activeDirectoryEndpoint, tenantID)
	if err != nil {
		recordError(adalTokenOperationName)
		return nil, fmt.Errorf("creating the OAuth config: %v", err)
	}
	spt, err := adal.NewServicePrincipalToken(
		*oauthConfig,
		clientID,
		secret,
		resource,
	)
	if err != nil {
		recordError(adalTokenOperationName)
		return nil, err
	}
	// Evectively acqurie the token
	err = spt.Refresh()
	if err != nil {
		recordError(adalTokenOperationName)
		return nil, err
	}
	token := spt.Token()
	recordDuration(adalTokenOperationName, time.Since(begin))
	return &token, nil
}

func init() {
	err := adal.AddToUserAgent(version.GetUserAgent("NMI", version.NMIVersion))
	if err != nil {
		// shouldn't fail ever
		panic(err)
	}
}

// InitReporter initialize the reporter with given reporter
func InitReporter(reporterInstance *metrics.Reporter) {
	reporter = reporterInstance
}

// recordError records the error in appropriate metric
func recordError(operation string) {
	if reporter != nil {
		reporter.ReportOperation(
			operation,
			metrics.CloudProviderOperationsErrorsCountM.M(1))
	}
}

// recordDuration records the duration in appropriate metric
func recordDuration(operation string, duration time.Duration) {
	if reporter != nil {
		reporter.ReportOperation(
			operation,
			metrics.CloudProviderOperationsDurationM.M(duration.Seconds()))
	}
}
