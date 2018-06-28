package cloudprovidertest

import (
	"flag"
	"testing"
)

func TestSimple(t *testing.T) {
	flag.Set("logtostderr", "true")
	//flag.Set("log_dir", ".")
	flag.Set("v", "3")
	flag.Parse()

	//	flag.Set("alsologtostderr", fmt.Sprintf("%t", true))
	//	flag.Parse()
	cloudClient := NewTestCloudClient()
	//glog.Infof("Test starting")
	//glog.Infof("Empty MSI")
	//cloudClient.PrintMSI()

	cloudClient.AssignUserMSI("ID0", "node0")
	cloudClient.AssignUserMSI("ID0", "node0")
	cloudClient.AssignUserMSI("ID0again", "node0")
	cloudClient.AssignUserMSI("ID1", "node1")
	cloudClient.AssignUserMSI("ID2", "node2")
	//glog.Infof("Add more MSI")
	//cloudClient.PrintMSI()
	testMSI := []string{"ID0", "ID0again"}
	if !cloudClient.CompareMSI("node0", testMSI) {
		panic("MSI mismatch")
	}

	cloudClient.RemoveUserMSI("ID0", "node0")
	cloudClient.RemoveUserMSI("ID2", "node2")
	testMSI = []string{"ID0again"}
	if !cloudClient.CompareMSI("node0", testMSI) {
		panic("MSI mismatch")
	}
	testMSI = []string{}
	if !cloudClient.CompareMSI("node2", testMSI) {
		panic("MSI mismatch")
	}
	//cloudClient.PrintMSI()
}
