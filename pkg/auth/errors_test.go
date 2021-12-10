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
