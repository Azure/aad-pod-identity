package cloudprovidertest

import (
	"flag"
	"testing"

	"github.com/Azure/aad-pod-identity/pkg/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSimple(t *testing.T) {
	flag.Set("logtostderr", "true")
	flag.Set("v", "3")
	flag.Parse()

	for _, cfg := range []config.AzureConfig{
		config.AzureConfig{},
		config.AzureConfig{VMType: "vmss"},
	} {
		desc := cfg.VMType
		if desc == "" {
			desc = "default"
		}
		t.Run(desc, func(t *testing.T) {
			cloudClient := NewTestCloudClient(cfg)

			node0 := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node0"}}
			node1 := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node1"}}
			node2 := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node2"}}

			cloudClient.AssignUserMSI("ID0", node0)
			cloudClient.AssignUserMSI("ID0", node0)
			cloudClient.AssignUserMSI("ID0again", node0)
			cloudClient.AssignUserMSI("ID1", node1)
			cloudClient.AssignUserMSI("ID2", node2)

			testMSI := []string{"ID0", "ID0again"}
			if !cloudClient.CompareMSI("node0", testMSI) {
				t.Fatal("MSI mismatch")
			}

			cloudClient.RemoveUserMSI("ID0", node0)
			cloudClient.RemoveUserMSI("ID2", node2)
			testMSI = []string{"ID0again"}
			if !cloudClient.CompareMSI(node0.Name, testMSI) {
				t.Fatal("MSI mismatch")
			}
			testMSI = []string{}
			if !cloudClient.CompareMSI(node2.Name, testMSI) {
				t.Fatal("MSI mismatch")
			}
		})
	}
}

func TestVMLevelIdentity(t *testing.T) {
	cloudClient := NewTestCloudClient()
	cloudClient.UserAssignedIdentitiesOnVM["node0"] = make(map[string]bool)
	cloudClient.UserAssignedIdentitiesOnVM["node0"]["ID0"] = true

	cloudClient.AssignUserMSI("ID0", "node0")
	if !cloudClient.CompareMSI("node0", []string{"ID0"}) {
		panic("MSI mismatch")
	}

	cloudClient.AssignUserMSI("ID1", "node0")
	if !cloudClient.CompareMSI("node0", []string{"ID0", "ID1"}) {
		panic("MSI mismatch")
	}

	cloudClient.RemoveUserMSI("ID0", "node0")
	cloudClient.RemoveUserMSI("ID1", "node0")
	// ID0 should not be deleted after deleting pod identity
	if !cloudClient.CompareMSI("node0", []string{"ID0"}) {
		panic("MSI mismatch")
	}
}
