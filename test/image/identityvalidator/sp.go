package main

import (
	"context"
	"fmt"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"k8s.io/klog/v2"
)

// assertWithSystemAssignedIdentity obtains a service principal token with system-assigned identity.
func assertWithSystemAssignedIdentity() error {
	spt, err := adal.NewServicePrincipalTokenFromManagedIdentity(azure.PublicCloud.ResourceManagerEndpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to acquire a service principal token from IMDS, error: %+v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()

	if err := spt.RefreshWithContext(ctx); err != nil {
		return fmt.Errorf("failed to refresh the service principal token, error: %+v", err)
	}

	token := spt.Token()
	if token.IsZero() {
		return fmt.Errorf("%+v is a zero token", token)
	}

	klog.Infof("successfully acquired a service principal token from IMDS")
	return nil
}
