package auth

import (
	"context"
	"crypto/rsa"
	"time"

	"github.com/Azure/aad-pod-identity/pkg/metrics"
	"github.com/Azure/aad-pod-identity/version"

	"github.com/Azure/go-autorest/autorest/adal"
	"golang.org/x/crypto/pkcs12"
	"k8s.io/klog/v2"
)

const (
	defaultActiveDirectoryEndpoint = "https://login.microsoftonline.com/"
)

var reporter *metrics.Reporter

// GetServicePrincipalTokenFromMSI return the token for the assigned user
func GetServicePrincipalTokenFromMSI(resource string) (_ *adal.Token, err error) {
	begin := time.Now()
	defer func() {
		if err != nil {
			merr := reporter.ReportIMDSOperationError(metrics.AdalTokenFromMSIOperationName)
			if merr != nil {
				klog.Warningf("failed to report metrics, error: %+v", merr)
			}
			return
		}
		merr := reporter.ReportIMDSOperationDuration(metrics.AdalTokenFromMSIOperationName, time.Since(begin))
		if merr != nil {
			klog.Warningf("failed to report metrics, error: %+v", merr)
		}
	}()

	spt, err := adal.NewServicePrincipalTokenFromManagedIdentity(resource, nil)
	if err != nil {
		return nil, err
	}
	// obtain a fresh token
	err = spt.Refresh()
	if err != nil {
		return nil, err
	}
	token := spt.Token()
	return &token, nil
}

// GetServicePrincipalTokenFromMSIWithUserAssignedID return the token for the assigned user
func GetServicePrincipalTokenFromMSIWithUserAssignedID(clientID, resource string) (_ *adal.Token, err error) {
	begin := time.Now()
	defer func() {
		if err != nil {
			merr := reporter.ReportIMDSOperationError(metrics.AdalTokenFromMSIWithUserAssignedIDOperationName)
			if merr != nil {
				klog.Warningf("failed to report metrics, error: %+v", merr)
			}
			return
		}
		merr := reporter.ReportIMDSOperationDuration(metrics.AdalTokenFromMSIWithUserAssignedIDOperationName, time.Since(begin))
		if merr != nil {
			klog.Warningf("failed to report metrics, error: %+v", merr)
		}
	}()

	managedIdentityOptions := &adal.ManagedIdentityOptions{ClientID: clientID}
	spt, err := adal.NewServicePrincipalTokenFromManagedIdentity(resource, managedIdentityOptions)
	if err != nil {
		return nil, err
	}

	// obtain a fresh token
	err = spt.Refresh()
	if err != nil {
		return nil, err
	}
	token := spt.Token()
	return &token, nil
}

// GetServicePrincipalToken return the token for the assigned user with client secret
func GetServicePrincipalToken(adEndpointFromSpec, tenantID, clientID, secret, resource string, auxiliaryTenantIDs []string) (_ []*adal.Token, err error) {
	begin := time.Now()
	defer func() {
		if err != nil {
			merr := reporter.ReportIMDSOperationError(metrics.AdalTokenOperationName)
			if merr != nil {
				klog.Warningf("failed to report metrics, error: %+v", merr)
			}
			return
		}
		merr := reporter.ReportIMDSOperationDuration(metrics.AdalTokenOperationName, time.Since(begin))
		if merr != nil {
			klog.Warningf("failed to report metrics, error: %+v", merr)
		}
	}()

	activeDirectoryEndpoint := defaultActiveDirectoryEndpoint
	if adEndpointFromSpec != "" {
		activeDirectoryEndpoint = adEndpointFromSpec
	}

	if len(auxiliaryTenantIDs) != 0 {
		return newMultiTenantServicePrincipalToken(activeDirectoryEndpoint, tenantID, clientID, secret, resource, auxiliaryTenantIDs)
	}
	return newServicePrincipalToken(activeDirectoryEndpoint, tenantID, clientID, secret, resource)
}

// newServicePrincipalToken creates a ServicePrincipalToken from the supplied Service Principal
// credentials scoped to the named resource and tenant
func newServicePrincipalToken(activeDirectoryEndpoint, tenantID, clientID, secret, resource string) ([]*adal.Token, error) {
	oauthConfig, err := adal.NewOAuthConfig(activeDirectoryEndpoint, tenantID)
	if err != nil {
		return nil, err
	}
	spt, err := adal.NewServicePrincipalToken(*oauthConfig, clientID, secret, resource)
	if err != nil {
		return nil, err
	}
	// obtain a fresh token
	err = spt.Refresh()
	if err != nil {
		return nil, err
	}
	token := spt.Token()
	return []*adal.Token{&token}, nil
}

// newMultiTenantServicePrincipalToken creates a new MultiTenantServicePrincipalToken with the specified credentials and resource.
// the first token in the array of tokens returned is the primaryToken
// all tokens [1:] are the auxiliary tokens
func newMultiTenantServicePrincipalToken(activeDirectoryEndpoint, primaryTenantID, clientID, secret, resource string, auxiliaryTenantIDs []string) ([]*adal.Token, error) {
	oauthConfig, err := adal.NewMultiTenantOAuthConfig(activeDirectoryEndpoint, primaryTenantID, auxiliaryTenantIDs, adal.OAuthOptions{})
	if err != nil {
		return nil, err
	}
	spt, err := adal.NewMultiTenantServicePrincipalToken(oauthConfig, clientID, secret, resource)
	if err != nil {
		return nil, err
	}
	err = spt.RefreshWithContext(context.TODO())
	if err != nil {
		return nil, err
	}

	var tokens []*adal.Token
	// add primary token as the first token
	primaryToken := spt.PrimaryToken.Token()
	tokens = append(tokens, &primaryToken)

	// add the auxiliary tokens from [1:]
	for idx := range spt.AuxiliaryTokens {
		auxiliaryToken := spt.AuxiliaryTokens[idx].Token()
		tokens = append(tokens, &auxiliaryToken)
	}
	return tokens, nil
}

// GetServicePrincipalTokenWithCertificate return the token for the assigned user with certificate
func GetServicePrincipalTokenWithCertificate(adEndpointFromSpec, tenantID, clientID string, certificate []byte, password, resource string) (_ *adal.Token, err error) {
	begin := time.Now()
	defer func() {
		if err != nil {
			merr := reporter.ReportIMDSOperationError(metrics.AdalTokenOperationName)
			if merr != nil {
				klog.Warningf("failed to report metrics, error: %+v", merr)
			}
			return
		}
		merr := reporter.ReportIMDSOperationDuration(metrics.AdalTokenOperationName, time.Since(begin))
		if merr != nil {
			klog.Warningf("failed to report metrics, error: %+v", merr)
		}
	}()

	activeDirectoryEndpoint := defaultActiveDirectoryEndpoint
	if adEndpointFromSpec != "" {
		activeDirectoryEndpoint = adEndpointFromSpec
	}
	oauthConfig, err := adal.NewOAuthConfig(activeDirectoryEndpoint, tenantID)
	if err != nil {
		return nil, err
	}

	privateKey, cert, err := pkcs12.Decode(certificate, password)
	if err != nil {
		return nil, err
	}

	spt, err := adal.NewServicePrincipalTokenFromCertificate(*oauthConfig, clientID, cert, privateKey.(*rsa.PrivateKey), resource)
	if err != nil {
		return nil, err
	}
	// obtain a fresh token
	err = spt.Refresh()
	if err != nil {
		return nil, err
	}
	token := spt.Token()
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
