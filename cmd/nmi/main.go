package main

import (
	server "github.com/Azure/aad-pod-identity/nmi/server"

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
	s := server.NewServer()
	s.HostInterface = *hostInterface

	//	if err := iptable.AddRule(s.NMIPort, s.MetadataAddress, s.HostInterface, "127.0.0.1"); err != nil {
	//		glog.Fatalf("%s", err)
	//	}

	if err := s.Run(); err != nil {
		glog.Fatalf("%s", err)
	}
}
