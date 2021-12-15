package auth

import (
	"errors"
	"testing"

	"github.com/Azure/go-autorest/autorest/adal"
)

type testError struct {
	adal.TokenRefreshError
}

func TestIsTokenRefreshError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "not a token refresh error",
			err:  errors.New("some error"),
		},
		{
			name: "token refresh error",
			err:  testError{},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsTokenRefreshError(tt.err)
			if got != tt.want {
				t.Errorf("IsTokenRefreshError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsHealthCheckError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "health check error",
			err:  errors.New("some error"),
			want: true,
		},
		{
			name: "not health check error",
			err:  testError{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsHealthCheckError(tt.err)
			if got != tt.want {
				t.Errorf("IsHealthCheckError() = %v, want %v", got, tt.want)
			}
		})
	}
}
