package nmi

import (
	"context"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	"github.com/Azure/go-autorest/autorest/adal"
)

// OperationMode is the mode in which NMI is operating
// allowed values: standard
type OperationMode string

const (
	// StandardMode ...
	StandardMode OperationMode = "standard"
)

// TokenClient ...
type TokenClient interface {
	//GetIdentities gets the list of identities which match the
	// given pod in the form of AzureIdentity.
	GetIdentities(ctx context.Context, podns, podname, clientID string) (*aadpodid.AzureIdentity, error)
	// GetToken acquires a token by using the AzureIdentity.
	GetToken(ctx context.Context, clientID, resource string, podID aadpodid.AzureIdentity) (token *adal.Token, err error)
}
