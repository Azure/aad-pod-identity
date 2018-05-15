package server

import (
	"testing"

	k8s "github.com/Azure/aad-pod-identity/pkg/k8s"
)

func TestServer_Run(t *testing.T) {
	type fields struct {
		KubeClient   k8s.Client
		NMIPort      string
		MetadataIP   string
		MetadataPort string
		Host         string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				KubeClient:   tt.fields.KubeClient,
				NMIPort:      tt.fields.NMIPort,
				MetadataIP:   tt.fields.MetadataIP,
				MetadataPort: tt.fields.MetadataPort,
				Host:         tt.fields.Host,
			}
			if err := s.Run(); (err != nil) != tt.wantErr {
				t.Errorf("Server.Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
