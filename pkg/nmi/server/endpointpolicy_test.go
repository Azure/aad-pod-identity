// +build windows

package server

import(
	"testing"
	"errors"
)

func TestApplyEndpointRoutePolicy(t *testing.T) {
	cases := []struct {
		name            string
		podIP           string
		metadataIP      string
		metadataPort    string
		nmiIP           string
		nmiPort         string
		expectedError   error
	}{
		{
			name: "Fail with missing Pod IP",
			podIP: "",
			metadataIP: "169.254.169.254",
			metadataPort: "80",
			nmiIP: "127.10.0.23",
			nmiPort: "8329",
			expectedError: errors.New("Missing IP Address"),
		},
		{
			name: "Success with non-existant pipe",
			podIP: "127.10.0.152",
			metadataIP: "169.254.169.254",
			metadataPort: "80",
			nmiIP: "127.10.0.23",
			nmiPort: "8329",
			expectedError: errors.New(`No endpoint found for Pod IP - 127.10.0.152. Error: open \\.\pipe\hnspipe: The system cannot find the file specified.`),
		},
	}

	for i, tc := range cases {
		t.Log(i, tc.name)
		err := ApplyEndpointRoutePolicy(tc.podIP, tc.metadataIP, tc.metadataPort, tc.nmiIP, tc.nmiPort)
		
		if(err != nil) {
			if(tc.expectedError == nil) {
				t.Fatalf("no error expected, but found - %s", err)
			} else if(tc.expectedError.Error() != err.Error()) {
				t.Fatalf("expected error to be - %s, but found - %s", tc.expectedError, err)
			}
		}
	}
}

func TestDeleteEndpointRoutePolicy(t *testing.T) {
	cases := []struct {
		name            string
		podIP           string
		metadataIP      string
		expectedError   error
	}{
		{
			name: "Fail with missing Pod IP",
			podIP: "",
			metadataIP: "169.254.169.254",
			expectedError: errors.New("Missing IP Address"),
		},
		{
			name: "Success with non-existant pipe",
			podIP: "127.10.0.152",
			metadataIP: "169.254.169.254",
			expectedError: errors.New(`No endpoint found for Pod IP - 127.10.0.152. Error: open \\.\pipe\hnspipe: The system cannot find the file specified.`),
		},
	}

	for i, tc := range cases {
		t.Log(i, tc.name)
		err := DeleteEndpointRoutePolicy(tc.podIP, tc.metadataIP)
		
		if(err != nil) {
			if(tc.expectedError == nil) {
				t.Fatalf("no error expected, but found - %s", err)
			} else if(tc.expectedError.Error() != err.Error()) {
				t.Fatalf("expected error to be - %s, but found - %s", tc.expectedError, err)
			}
		}
	}
}
