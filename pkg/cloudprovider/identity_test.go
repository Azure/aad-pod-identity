package cloudprovider

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-12-01/compute"
)

func checkIDList(t *testing.T, expect, actual []string) {
	t.Helper()
	if len(actual) != len(expect) {
		t.Fatalf("expected %v, got: %v", expect, actual)
	}
	for i, v := range expect {
		if actual[i] != v {
			t.Fatalf("expected entry %d to be %q, got: %s", i, v, actual[i])
		}
	}
}

func TestGetUpdatedResourceIdentityType(t *testing.T) {
	cases := []struct {
		current  compute.ResourceIdentityType
		expected compute.ResourceIdentityType
	}{
		{
			current:  "",
			expected: compute.ResourceIdentityTypeUserAssigned,
		},
		{
			current:  compute.ResourceIdentityTypeNone,
			expected: compute.ResourceIdentityTypeUserAssigned,
		},
		{
			current:  compute.ResourceIdentityTypeUserAssigned,
			expected: compute.ResourceIdentityTypeUserAssigned,
		},
		{
			current:  compute.ResourceIdentityTypeSystemAssigned,
			expected: compute.ResourceIdentityTypeSystemAssignedUserAssigned,
		},
		{
			current:  compute.ResourceIdentityTypeSystemAssignedUserAssigned,
			expected: compute.ResourceIdentityTypeSystemAssignedUserAssigned,
		},
	}

	for _, tc := range cases {
		actual := getUpdatedResourceIdentityType(tc.current)
		if tc.expected != actual {
			t.Fatalf("expected: %v, got: %v", tc.expected, actual)
		}
	}
}
