package cloudprovider

import (
	"reflect"
	"testing"

	"github.com/Azure/go-autorest/autorest/azure"
)

func TestParseResourceID(t *testing.T) {
	type testCase struct {
		desc   string
		testID string
		expect azure.Resource
		xErr   bool
	}

	notNested := "/subscriptions/asdf/resourceGroups/qwerty/providers/testCompute/myComputeObjectType/testComputeResource"
	nested := "/subscriptions/asdf/resourceGroups/qwerty/providers/testCompute/myComputeObjectType/testComputeResource/someNestedResource/myNestedResource"

	for _, c := range []testCase{
		{"empty string", "", azure.Resource{}, true},
		{"just a string", "asdf", azure.Resource{}, true},
		{"partial match", "/subscriptions/asdf/resourceGroups/qwery", azure.Resource{}, true},
		{"nested", nested, azure.Resource{
			SubscriptionID: "asdf",
			ResourceGroup:  "qwerty",
			Provider:       "testCompute",
			ResourceName:   "testComputeResource",
			ResourceType:   "myComputeObjectType",
		}, false},
		{"not nested", notNested, azure.Resource{
			SubscriptionID: "asdf",
			ResourceGroup:  "qwerty",
			Provider:       "testCompute",
			ResourceName:   "testComputeResource",
			ResourceType:   "myComputeObjectType",
		}, false},
	} {
		t.Run(c.desc, func(t *testing.T) {
			r, err := ParseResourceID(c.testID)
			if (err != nil) != c.xErr {
				t.Fatalf("expected err==%v, got: %v", c.xErr, err)
			}
			if !reflect.DeepEqual(r, c.expect) {
				t.Fatalf("resource does not match expected:\nexpected:\n\t%+v\ngot:\n\t%+v", c.expect, r)
			}
		})
	}
}
