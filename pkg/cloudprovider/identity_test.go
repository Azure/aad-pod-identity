package cloudprovider

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
)

func TestFilterIdentity(t *testing.T) {
	idList := []string{}
	idType := compute.ResourceIdentityTypeNone
	if err := filterUserIdentity(&idType, &idList, "A"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
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

	append := appendUserIdentity(&idType, &idList, "A")
	if !append {
		t.Fatalf("Expecting the id to be not present. But present returned by Append.")
	}
	expect := []string{"A"}
	checkIDList(t, expect, idList)
	if idType != compute.ResourceIdentityTypeUserAssigned {
		t.Fatalf("expected type %s, got: %s", compute.ResourceIdentityTypeUserAssigned, idType)
	}

	// Append the same value again, should not change anything
	append = appendUserIdentity(&idType, &idList, "A")
	if append {
		t.Fatalf("Expecting the id to be not present. But present returned by Append.")
	}
	checkIDList(t, expect, idList)
	if idType != compute.ResourceIdentityTypeUserAssigned {
		t.Fatalf("expected type %s, got: %s", compute.ResourceIdentityTypeUserAssigned, idType)
	}

	append = appendUserIdentity(&idType, &idList, "B")
	if !append {
		t.Fatalf("Expecting the id to be not present. But present returned by Append.")
	}
	expect = []string{"A", "B"}
	checkIDList(t, expect, idList)
	if idType != compute.ResourceIdentityTypeUserAssigned {
		t.Fatalf("expected type %s, got: %s", compute.ResourceIdentityTypeUserAssigned, idType)
	}

	idType = compute.ResourceIdentityTypeSystemAssigned
	idList = []string{"A"}
	expect = []string{"A", "B"}
	append = appendUserIdentity(&idType, &idList, "B")
	if !append {
		t.Fatalf("Expecting the id to be not present. But present returned by Append.")
	}
	checkIDList(t, expect, idList)
	if idType != compute.ResourceIdentityTypeSystemAssignedUserAssigned {
		t.Fatalf("expected type %s, got: %s", compute.ResourceIdentityTypeSystemAssignedUserAssigned, idType)
	}

	idType = compute.ResourceIdentityTypeNone
	idList = []string{}
	expect = []string{"A"}
	append = appendUserIdentity(&idType, &idList, "A")
	if !append {
		t.Fatalf("Expecting the id to be not present. But present returned by Append.")
	}
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
