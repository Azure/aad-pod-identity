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

type WindowsServer struct {
	Server *Server
}

var podMap = make(map[types.UID]string)

func RunServer(s *Server) {
	ws := WindowsServer{Server: s}
	if err := ws.Run(); err != nil {
		klog.Fatalf("%s", err)
	}
}

// Run the specified Server.
func (s *WindowsServer) Run() error {
	exit := make(chan struct{})
	s.Server.PodClient.Start(exit)
	klog.V(6).Infof("Pod client started")

	s.ApplyRoutePolicyForExistingPods()
	go s.Sync()

	return s.Server.Run()
}

func (s *WindowsServer) Sync() {
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

			// fmt.Printf("Windows Server Host IP, Pod Node Name and Pod IP:%s %s %s \n", pod.Status.HostIP, pod.Spec.NodeName, pod.Status.PodIP)
			fmt.Printf("Windows Server Pod UID and Pod Name:%s %s \n", pod.UID, pod.Name)
			if s.Server.NodeName == pod.Spec.NodeName {
				if podIP, podExist := podMap[pod.UID]; podExist {
					fmt.Printf("Delete: Windows Server Pod UID and Pod Name:%s %s \n", pod.UID, pod.Name)
					deleteRoutePolicy(podIP)
				} else {
					fmt.Printf("Add: Windows Server Pod UID and Pod Name:%s %s \n", pod.UID, pod.Name)
					podMap[pod.UID] = pod.Status.PodIP
					applyRoutePolicy(pod.Status.PodIP)
				}
			}
		}
	}
}

func (s *WindowsServer) ApplyRoutePolicyForExistingPods() {
	klog.Info("Apply route pllicy for existing pods started.")

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

func (s *WindowsServer) DeleteRoutePolicyForExistingPods() {
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
			deleteRoutePolicy(podItem.Status.PodIP)
		}
	}

	// Wait for pod to delete
	klog.Info("Handled termination, awaiting pod deletion")
	time.Sleep(10 * time.Second)

	klog.Infof("Exiting with %v", exitCode)
	os.Exit(exitCode)
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

func deleteRoutePolicy(podIP string) {

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
