package main

import (
	"os"

	"github.com/Azure/aad-pod-identity/pkg/iptable"
	"github.com/Azure/aad-pod-identity/pkg/k8s"
	server "github.com/Azure/aad-pod-identity/pkg/nmi/server"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

var (
	nmiPort      = pflag.String("nmi-port", "2579", "NMI application port")
	metadataIP   = pflag.String("metadata-ip", "169.254.169.254", "instance metadata host ip")
	metadataPort = pflag.String("metadata-port", "80", "instance metadata host ip")
	test         = pflag.Bool("test", false, "set to true to use fake client")
)

func main() {
	log.Info("starting nmi process")
	s := server.NewServer()
	pflag.Parse()
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

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	s.Host = hostname
	log.Infof("hostname: %s", hostname)

	podcidr, err := s.KubeClient.GetPodCidr(hostname)
	if err != nil {
		log.Fatalf("%+v", err)
	}
	nodeip, err := s.KubeClient.GetNodeIP(hostname)
	if err != nil {
		log.Fatalf("%+v", err)
	}
	if err := iptable.AddRule(podcidr, s.MetadataIP, s.MetadataPort, nodeip, s.NMIPort); err != nil {
		log.Fatalf("%s", err)
	}

	if err := s.Run(); err != nil {
		log.Fatalf("%s", err)
	}
}
