package cloudprovidertest

import (
	"flag"
	"testing"

	"github.com/Azure/aad-pod-identity/pkg/config"
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

			cloudClient.AssignUserMSI("ID0", "node0")
			cloudClient.AssignUserMSI("ID0", "node0")
			cloudClient.AssignUserMSI("ID0again", "node0")
			cloudClient.AssignUserMSI("ID1", "node1")
			cloudClient.AssignUserMSI("ID2", "node2")

			testMSI := []string{"ID0", "ID0again"}
			if !cloudClient.CompareMSI("node0", testMSI) {
				t.Fatal("MSI mismatch")
			}

			cloudClient.RemoveUserMSI("ID0", "node0")
			cloudClient.RemoveUserMSI("ID2", "node2")
			testMSI = []string{"ID0again"}
			if !cloudClient.CompareMSI("node0", testMSI) {
				t.Fatal("MSI mismatch")
			}
			testMSI = []string{}
			if !cloudClient.CompareMSI("node2", testMSI) {
				t.Fatal("MSI mismatch")
			}
		})
	}
}
