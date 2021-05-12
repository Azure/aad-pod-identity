package main

import (
	"context"
	"fmt"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"k8s.io/klog/v2"
)

type servicePrincipalTester struct {
	imdsTokenEndpoint string
}

// assertWithSystemAssignedIdentity obtains a service principal token with system-assigned identity.
func (spt *servicePrincipalTester) assertWithSystemAssignedIdentity() error {
	spToken, err := adal.NewServicePrincipalTokenFromMSI(spt.imdsTokenEndpoint, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return fmt.Errorf("failed to acquire a service principal token with %s, error: %+v", spt.imdsTokenEndpoint, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()

	if err := spToken.RefreshWithContext(ctx); err != nil {
		return fmt.Errorf("failed to refresh the service principal token, error: %+v", err)
	}

	token := spToken.Token()
	if token.IsZero() {
		return fmt.Errorf("%+v is a zero token", token)
	}

	klog.Infof("successfully acquired a service principal token from %s", spt.imdsTokenEndpoint)
	return nil
}
