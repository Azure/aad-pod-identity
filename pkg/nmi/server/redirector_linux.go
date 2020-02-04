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

// LinuxRedirector returns sync function for linux redirector
func LinuxRedirector(server *Server, subRoutineDone <-chan struct{}) func(*Server, chan<- struct{}, <-chan struct{}) {
	return func(server *Server, subRoutineDone chan<- struct{}, mainRoutineDone <-chan struct{}) {
		updateIPTableRules(server, subRoutineDone, mainRoutineDone)
	}
}

// WindowsRedirector returns sync function for windows redirector
func WindowsRedirector(server *Server, subRoutineDone <-chan struct{}) func(*Server, chan<- struct{}, <-chan struct{}) {
	panic("Windows Redirector is not applicable")
}

func updateIPTableRulesInternal(server *Server) {
	klog.V(5).Infof("node(%s) hostip(%s) metadataaddress(%s:%s) nmiport(%s)", server.NodeName, server.HostIP, server.MetadataIP, server.MetadataPort, server.NMIPort)

	if err := iptables.AddCustomChain(server.MetadataIP, server.MetadataPort, server.HostIP, server.NMIPort); err != nil {
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
func updateIPTableRules(server *Server, subRoutineDone chan<- struct{}, mainRoutineDone <-chan struct{}) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM, syscall.SIGINT)

	ticker := time.NewTicker(time.Second * time.Duration(server.IPTableUpdateTimeIntervalInSeconds))
	defer ticker.Stop()

	// Run once before the waiting on ticker for the rules to take effect
	// immediately.
	updateIPTableRulesInternal(server)
	server.Initialized = true

	for {
		select {
		case <-signalChan:
			handleTermination()
			close(subRoutineDone)
		case <-mainRoutineDone:
			handleTermination()
			close(subRoutineDone)
		case <-ticker.C:
			updateIPTableRulesInternal(server)
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
