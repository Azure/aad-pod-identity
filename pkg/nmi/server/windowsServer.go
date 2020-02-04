// +build windows

package server

import (
	"fmt"
	"sync"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

type WindowsServer struct {
	Server *Server
}

func RunServer(s *Server) {
	ws := WindowsServer{Server: s}
	if err := ws.Run(); err != nil {
		klog.Fatalf("%s", err)
	}
}

// Run runs the specified Server.
func (s *WindowsServer) Run() error {

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		exit := make(chan struct{})
		s.Server.PodClient.Start(exit)
		klog.V(6).Infof("Pod client started")
		wg.Done()
	}()

	wg.Wait()

	s.ApplyRoutePolicyForExistingPods()
	go s.Sync()

	return s.Server.Run()
}

func (s *WindowsServer) Sync() {
	klog.Info("Sync thread started.")
	var pod *v1.Pod

	for {
		select {
		case pod = <-s.Server.PodObjChannel:
			klog.V(6).Infof("Received event: %s", pod)

			fmt.Printf("Host IP, Pod Node Name and Pod IP:%s %s %s \n", pod.Status.HostIP, pod.Spec.NodeName, pod.Status.PodIP)
			if s.Server.NodeName == pod.Spec.NodeName {
				applyRoutePolicy(pod.Status.PodIP)
			}
		}
	}
}

func (s *WindowsServer) ApplyRoutePolicyForExistingPods() {
	klog.Info("Apply existing pods started.")

	listPods, err := s.Server.PodClient.ListPods()
	if err != nil {
		klog.Error(err)
	}

	for _, podItem := range listPods {
		fmt.Printf("Host IP, Node Name and Pod IP: \n %s %s %s \n", podItem.Status.HostIP, podItem.Spec.NodeName, podItem.Status.PodIP)
		if podItem.Spec.NodeName == s.Server.NodeName {
			applyRoutePolicy(podItem.Status.PodIP)
		}
	}
}

func applyRoutePolicy(podIP string) {

	// Retrieve all the endpoints
	// var endpoints = HCNProxy.EnumerateEndpoints()

	// Foreach ennpoint, find the one for the podinfo and apply route policy
	//for _, val := range endpoints {
	//	if podIP == val.podIP {
	//		HCNProxy.ApplyRoutePolicy(val)
	//		break
	//	}
	//}
}
