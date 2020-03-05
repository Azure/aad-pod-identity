package cloudprovider

import (
	"testing"
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
