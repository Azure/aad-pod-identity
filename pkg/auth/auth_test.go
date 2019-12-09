package auth

import (
	"testing"

	"github.com/Azure/aad-pod-identity/pkg/metrics"
)

func TestGetServicePrincipalToken(t *testing.T) {
	reporter, err := metrics.NewReporter()
	if err != nil {
		t.Fatalf("expected nil error, got: %+v", err)
	}
	InitReporter(reporter)
	_, err = GetServicePrincipalToken("tid", "cid", "", "")
	if err == nil {
		t.Fatal("should be error with empty secret")
	}
}
