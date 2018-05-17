package main

import (
	"github.com/Azure/aad-pod-identity/pkg/iptable"
	"github.com/Azure/aad-pod-identity/pkg/k8s"
	server "github.com/Azure/aad-pod-identity/pkg/nmi/server"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

var (
	nmiPort      = pflag.String("nmi-port", "2579", "NMI application port")
	hostIP       = pflag.String("host-ip", "", "NMI application port")
	nodename     = pflag.String("node", "", "NMI application port")
	metadataIP   = pflag.String("metadata-ip", "169.254.169.254", "instance metadata host ip")
	metadataPort = pflag.String("metadata-port", "80", "instance metadata host ip")
	test         = pflag.Bool("test", false, "set to true to use fake client")
)

func main() {
	pflag.Parse()
	log.Info("starting nmi process")
	s := server.NewServer()
	if !*test {
		client, err := k8s.NewKubeClient()
		if err != nil {
			log.Fatalf("%+v", err)
		}
		s.KubeClient = client
	} else {
		client, _ := k8s.NewFakeClient()
		s.KubeClient = client
	}
	s.MetadataIP = *metadataIP
	s.MetadataPort = *metadataPort

	log.Infof("node: %s", *nodename)
	podcidr, err := s.KubeClient.GetPodCidr(*nodename)
	if err != nil {
		log.Fatalf("%+v", err)
	}
	log.Infof("node(%s) hostip(%s) podcidr(%s) metadataaddress(%s:%s) nmiport(%s)", *nodename, *hostIP, podcidr, s.MetadataIP, s.MetadataPort, s.NMIPort)
	if err := iptable.AddCustomChain(podcidr, s.MetadataIP, s.MetadataPort, *hostIP, s.NMIPort); err != nil {
		log.Fatalf("%s", err)
	}
	if err := iptable.LogCustomChain(); err != nil {
		log.Fatalf("%s", err)
	}
	if err := s.Run(); err != nil {
		log.Fatalf("%s", err)
	}
}
