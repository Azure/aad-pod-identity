// +build windows

package server

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	v1 "k8s.io/api/core/v1"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
)

type WindowsRedirector struct {
	Server *Server
}

func makeRedirectorInt(server *Server) RedirectorInt {
	return &WindowsRedirector{Server: server}
}

var podMap = make(map[types.UID]string)

func (s *WindowsRedirector) RedirectMetadataEndpoint() {
	exit := make(chan struct{})
	s.Server.PodClient.Start(exit)
	klog.V(6).Infof("Pod client started")

	s.ApplyRoutePolicyForExistingPods()
	go s.Sync()
}

func (s *WindowsRedirector) Sync() {
	klog.Info("Sync thread started.")

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM, syscall.SIGINT)
	s.Server.Initialized = true

	var pod *v1.Pod

	for {
		select {
		case <-signalChan:
			s.DeleteRoutePolicyForExistingPods()
			break
		case pod = <-s.Server.PodObjChannel:
			klog.V(6).Infof("Received event: %s", pod)

			fmt.Printf("Node IP and Node Name:%s %s \n", s.Server.HostIP, s.Server.NodeName)

			// fmt.Printf("Windows Server Host IP, Pod Node Name and Pod IP:%s %s %s \n", pod.Status.HostIP, pod.Spec.NodeName, pod.Status.PodIP)
			fmt.Printf("Windows Server Pod UID and Pod Name:%s %s \n", pod.UID, pod.Name)
			if s.Server.NodeName == pod.Spec.NodeName {
				if podIP, podExist := podMap[pod.UID]; podExist {
					fmt.Printf("Delete: Windows Server Pod UID and Pod Name:%s %s \n", pod.UID, pod.Name)
					DeleteEndpointRoutePolicy(podIP)
					delete(podMap, pod.UID)
				} else {
					fmt.Printf("Add: Windows Server Pod UID and Pod Name:%s %s \n", pod.UID, pod.Name)
					podMap[pod.UID] = pod.Status.PodIP
					ApplyEndpointRoutePolicy(pod.Status.PodIP)
				}
			}
		}
	}
}

func (s *WindowsRedirector) ApplyRoutePolicyForExistingPods() {
	klog.Info("Apply route pllicy for existing pods started.")

	listPods, err := s.Server.PodClient.ListPods()
	if err != nil {
		klog.Error(err)
	}

	for _, podItem := range listPods {
		fmt.Printf("Host IP, Node Name and Pod IP: \n %s %s %s \n", podItem.Status.HostIP, podItem.Spec.NodeName, podItem.Status.PodIP)
		if podItem.Spec.NodeName == s.Server.NodeName {
			ApplyEndpointRoutePolicy(podItem.Status.PodIP, s.Server.MetadataIP, s.Server.MetadataPort, s.HostIP, s.NMIPort)
		}
	}
}

func (s *WindowsRedirector) DeleteRoutePolicyForExistingPods() {
	klog.Info("Received SIGTERM, shutting down")
	klog.Info("Delete route policy for existing pods started.")

	exitCode := 0

	listPods, err := s.Server.PodClient.ListPods()
	if err != nil {
		klog.Error(err)
		exitCode = 1
	}

	for _, podItem := range listPods {
		fmt.Printf("Host IP, Node Name and Pod IP: \n %s %s %s \n", podItem.Status.HostIP, podItem.Spec.NodeName, podItem.Status.PodIP)
		if podItem.Spec.NodeName == s.Server.NodeName {
			DeleteEndpointRoutePolicy(podItem.Status.PodIP)
		}
	}

	// Wait for pod to delete
	klog.Info("Handled termination, awaiting pod deletion")
	time.Sleep(10 * time.Second)

	klog.Infof("Exiting with %v", exitCode)
	os.Exit(exitCode)
}
