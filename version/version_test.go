package version

import (
	"fmt"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	BuildDate = "Now"
	GitCommit = "Commit"
	NMIVersion = "NMI version"
	expectedUserAgentStr := fmt.Sprintf("aad-pod-identity/%s/%s/%s/%s", "NMI", NMIVersion, GitCommit, BuildDate)
	gotUserAgentStr := GetUserAgent("NMI", NMIVersion)
	if !strings.EqualFold(expectedUserAgentStr, gotUserAgentStr) {
		t.Fatalf("got unexpected user agent string: %s. Expected: %s.", gotUserAgentStr, expectedUserAgentStr)
	}
}

func TestGetUserAgent(t *testing.T) {
	BuildDate = "now"
	GitCommit = "commit"
	NMIVersion = "version"

	tests := []struct {
		name              string
		customUserAgent   string
		expectedUserAgent string
	}{
		{
			name:              "default NMI user agent",
			customUserAgent:   "",
			expectedUserAgent: "aad-pod-identity/NMI/version/commit/now",
		},
		{
			name:              "default NMI user agent and custom user agent",
			customUserAgent:   "managedBy:aks",
			expectedUserAgent: "aad-pod-identity/NMI/version/commit/now managedBy:aks",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			customUserAgent = &test.customUserAgent
			actualUserAgent := GetUserAgent("NMI", NMIVersion)
			if !strings.EqualFold(test.expectedUserAgent, actualUserAgent) {
				t.Fatalf("got unexpected user agent string: %s. Expected: %s.", test.expectedUserAgent, actualUserAgent)
			}
		})
	}
}
