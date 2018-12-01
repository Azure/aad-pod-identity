package cloudprovider

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
)

func TestFilterIdentity(t *testing.T) {
	idList := []string{}
	idType := compute.ResourceIdentityTypeNone
	if err := filterUserIdentity(&idType, &idList, "A"); err == nil || err != errNotFound {
		t.Fatalf("expected error %q, got: %v", errNotFound, err)
	}

	idType = compute.ResourceIdentityTypeUserAssigned
	if err := filterUserIdentity(&idType, &idList, "A"); err == nil || err != errNotAssigned {
		t.Fatalf("expected error %q, got: %v", errNotAssigned, err)
	}

	idList = []string{"A"}
	if err := filterUserIdentity(&idType, &idList, "A"); err != nil {
		t.Fatal(err)
	}
	expect := []string{}
	checkIDList(t, expect, idList)
	if idType != compute.ResourceIdentityTypeNone {
		t.Fatalf("expected id type to be %q, got: %s", compute.ResourceIdentityTypeNone, idType)
	}

	idList = []string{"A", "B"}
	idType = compute.ResourceIdentityTypeUserAssigned
	if err := filterUserIdentity(&idType, &idList, "A"); err != nil {
		t.Fatal(err)
	}
	expect = []string{"B"}
	checkIDList(t, expect, idList)
	if idType != compute.ResourceIdentityTypeUserAssigned {
		t.Fatalf("expected id type to be %q, got: %s", compute.ResourceIdentityTypeNone, idType)
	}

	idList = []string{"A", "B"}
	idType = compute.ResourceIdentityTypeSystemAssignedUserAssigned
	if err := filterUserIdentity(&idType, &idList, "A"); err != nil {
		t.Fatal(err)
	}
	checkIDList(t, expect, idList)
	if idType != compute.ResourceIdentityTypeSystemAssignedUserAssigned {
		t.Fatalf("expected id type to be %q, got: %s", compute.ResourceIdentityTypeSystemAssignedUserAssigned, idType)
	}

	idList = []string{"A"}
	idType = compute.ResourceIdentityTypeSystemAssignedUserAssigned
	if err := filterUserIdentity(&idType, &idList, "A"); err != nil {
		t.Fatal(err)
	}
	expect = []string{}
	checkIDList(t, expect, idList)
	if idType != compute.ResourceIdentityTypeSystemAssigned {
		t.Fatalf("expected id type to be %q, got: %s", compute.ResourceIdentityTypeSystemAssigned, idType)
	}
}

func TestAppendUserIdentity(t *testing.T) {
	var (
		idList []string
		idType compute.ResourceIdentityType
	)

	appendUserIdentity(&idType, &idList, "A")
	expect := []string{"A"}
	checkIDList(t, expect, idList)
	if idType != compute.ResourceIdentityTypeUserAssigned {
		t.Fatalf("expected type %s, got: %s", compute.ResourceIdentityTypeUserAssigned, idType)
	}

	// Append the same value again, should not change anything
	appendUserIdentity(&idType, &idList, "A")
	checkIDList(t, expect, idList)
	if idType != compute.ResourceIdentityTypeUserAssigned {
		t.Fatalf("expected type %s, got: %s", compute.ResourceIdentityTypeUserAssigned, idType)
	}

	appendUserIdentity(&idType, &idList, "B")
	expect = []string{"A", "B"}
	checkIDList(t, expect, idList)
	if idType != compute.ResourceIdentityTypeUserAssigned {
		t.Fatalf("expected type %s, got: %s", compute.ResourceIdentityTypeUserAssigned, idType)
	}

	idType = compute.ResourceIdentityTypeSystemAssigned
	idList = []string{"A"}
	expect = []string{"A", "B"}
	appendUserIdentity(&idType, &idList, "B")
	checkIDList(t, expect, idList)
	if idType != compute.ResourceIdentityTypeSystemAssignedUserAssigned {
		t.Fatalf("expected type %s, got: %s", compute.ResourceIdentityTypeSystemAssignedUserAssigned, idType)
	}

	idType = compute.ResourceIdentityTypeNone
	idList = []string{}
	expect = []string{"A"}
	appendUserIdentity(&idType, &idList, "A")
	checkIDList(t, expect, idList)
	if idType != compute.ResourceIdentityTypeUserAssigned {
		t.Fatalf("expected type %s, got: %s", compute.ResourceIdentityTypeUserAssigned, idType)
	}
}

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
