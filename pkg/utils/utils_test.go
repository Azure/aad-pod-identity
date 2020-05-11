package utils

import (
	"testing"
)

func TestRedactClientID(t *testing.T) {
	tests := []struct {
		name     string
		clientID string
		expected string
	}{
		{
			name:     "should redact client id",
			clientID: "aabc0000-a83v-9h4m-000j-2c0a66b0c1f9",
			expected: "aabc##### REDACTED #####c1f9",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := RedactClientID(test.clientID)
			if actual != test.expected {
				t.Fatalf("expected: %s, got %s", test.expected, actual)
			}
		})
	}
}

func TestIsValidResourceID(t *testing.T) {
	tests := []struct {
		name        string
		resourceID  string
		expectedErr bool
	}{
		{
			name:        "invalid resource id 0",
			resourceID:  "invalidresid",
			expectedErr: true,
		},
		{
			name:        "invalid resource id 1",
			resourceID:  "/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourcegroups/0000/providers/Microsoft.ManagedIdentity/keyvault-identity-0",
			expectedErr: true,
		},
		{
			name:        "valid resource id",
			resourceID:  "/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/resourcegroups/0000/providers/Microsoft.ManagedIdentity/userAssignedIdentities/keyvault-identity-0",
			expectedErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateResourceID(test.resourceID)
			actualErr := err != nil
			if actualErr != test.expectedErr {
				t.Fatalf("expected error: %v, got error: %v", test.expectedErr, err)
			}
		})
	}
}
