package server

import "github.com/Azure/aad-pod-identity/pkg/nmi"

// ManagedClient implements the TokenClient interface
type ManagedClient struct {
	nmi.TokenClient
	IsNamespaced bool
}

func NewManagedTokenClient()
