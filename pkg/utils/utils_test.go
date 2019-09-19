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
		actual := RedactClientID(test.clientID)
		if actual != test.expected {
			t.Fatalf("expected: %s, got %s", test.expected, actual)
		}
	}
}
