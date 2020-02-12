// +build linux

package server

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	iptables "github.com/Azure/aad-pod-identity/pkg/nmi/iptables"
	"k8s.io/klog"
)

type LinuxRedirector struct {
	Server *Server
}

func makeRedirectorInt(server *Server) RedirectorInt {
	return &LinuxRedirector{Server: server}
}

func (s *LinuxRedirector) RedirectMetadataEndpoint() {
	go s.updateIPTableRules()
}

func (s *LinuxRedirector) updateIPTableRulesInternal() {
	klog.V(5).Infof("node(%s) hostip(%s) metadataaddress(%s:%s) nmiport(%s)", s.Server.NodeName, s.Server.HostIP, s.Server.MetadataIP, s.Server.MetadataPort, s.Server.NMIPort)

	if err := iptables.AddCustomChain(s.Server.MetadataIP, s.Server.MetadataPort, s.Server.HostIP, s.Server.NMIPort); err != nil {
		klog.Fatalf("%s", err)
	}
	if err := iptables.LogCustomChain(); err != nil {
		klog.Fatalf("%s", err)
	}
}

// updateIPTableRules ensures the correct iptable rules are set
// such that metadata requests are received by nmi assigned port
// NOT originating from HostIP destined to metadata endpoint are
// routed to NMI endpoint
func (s *LinuxRedirector) updateIPTableRules() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM, syscall.SIGINT)

	ticker := time.NewTicker(time.Second * time.Duration(s.Server.IPTableUpdateTimeIntervalInSeconds))
	defer ticker.Stop()

	// Run once before the waiting on ticker for the rules to take effect
	// immediately.
	s.updateIPTableRulesInternal()
	s.Server.Initialized = true

loop:
	for {
		select {
		case <-signalChan:
			handleTermination()
			break loop

		case <-ticker.C:
			s.updateIPTableRulesInternal()
		}
	}
}

func handleTermination() {
	klog.Info("Received SIGTERM, shutting down")

	exitCode := 0
	// clean up iptables
	if err := iptables.DeleteCustomChain(); err != nil {
		klog.Errorf("Error cleaning up during shutdown: %v", err)
		exitCode = 1
	}

	// wait for pod to delete
	klog.Info("Handled termination, awaiting pod deletion")
	time.Sleep(10 * time.Second)

	klog.Infof("Exiting with %v", exitCode)
	os.Exit(exitCode)
}
