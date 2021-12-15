package auth

import "github.com/Azure/go-autorest/autorest/adal"

// IsTokenRefreshError returns true if the error is a TokenRefreshError.
// This method can be used to distinguish health check errors from token refresh errors.
func IsTokenRefreshError(err error) bool {
	_, ok := err.(adal.TokenRefreshError)
	return ok
}

// IsHealthCheckError returns true if the error is not a token refresh error.
func IsHealthCheckError(err error) bool {
	return !IsTokenRefreshError(err)
}
