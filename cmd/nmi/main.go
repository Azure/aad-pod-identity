package main

import (
	"os"

	"github.com/Azure/aad-pod-identity/pkg/iptable"
	"github.com/Azure/aad-pod-identity/pkg/k8s"
	server "github.com/Azure/aad-pod-identity/pkg/nmi/server"

	"github.com/golang/glog"

	"github.com/spf13/pflag"
)

var (
	nmiPort       = pflag.String("nmi-port", "2579", "NMI application port")
	hostInterface = pflag.String("host-interface", "eth0", "Host interface for instance metadata traffic")
)

func main() {
	defer glog.Flush()
	glog.Info("starting nmi process")
	kubeClient, err := k8s.NewKubeClient()
	if err != nil {
		glog.Fatalf("%+v", err)
	}

	s := server.NewServer()
	s.KubeClient = kubeClient

	hostname, err := os.Hostname()
	if err != nil {
		glog.Fatalf("%+v", err)
	}
	s.Host = hostname
	glog.Infof("hostname: %s", hostname)

	podcidr, err := kubeClient.GetPodCidr(hostname)
	if err != nil {
		glog.Fatalf("%+v", err)
	}

	nodeip, err := kubeClient.GetNodeIP(hostname)
	if err != nil {
		glog.Fatalf("%+v", err)
	}

	if err := iptable.AddRule(podcidr, s.MetadataAddress, nodeip, s.NMIPort); err != nil {
		glog.Fatalf("%s", err)
	}

	if err := s.Run(); err != nil {
		glog.Fatalf("%s", err)
	}
}
