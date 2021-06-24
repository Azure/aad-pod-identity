package aadpodidentity

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSort(t *testing.T) {
	slice := []AzureIdentityBinding{{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test2",
			Namespace: "test",
		},
	}, {
		ObjectMeta: v1.ObjectMeta{
			Name:      "test1",
			Namespace: "default",
		},
	}, {
		ObjectMeta: v1.ObjectMeta{
			Name:      "test3",
			Namespace: "default",
		},
	}, {
		ObjectMeta: v1.ObjectMeta{
			Name:      "test1",
			Namespace: "test",
		},
	}, {
		ObjectMeta: v1.ObjectMeta{
			Name:      "test2",
			Namespace: "default",
		},
	}}
	expected := []AzureIdentityBinding{{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test1",
			Namespace: "default",
		},
	}, {
		ObjectMeta: v1.ObjectMeta{
			Name:      "test2",
			Namespace: "default",
		},
	}, {
		ObjectMeta: v1.ObjectMeta{
			Name:      "test3",
			Namespace: "default",
		},
	}, {
		ObjectMeta: v1.ObjectMeta{
			Name:      "test1",
			Namespace: "test",
		},
	}, {
		ObjectMeta: v1.ObjectMeta{
			Name:      "test2",
			Namespace: "test",
		},
	}}
	sort.Sort(AzureIdentityBindings(slice))
	assert.Equal(t, slice, expected)
}
