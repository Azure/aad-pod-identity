package main

import (
	"flag"

	"github.com/Azure/aad-pod-identity/pkg/mic"
	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeconfig    string
	azurecredfile string
)

func main() {
	flag.StringVar(&kubeconfig, "kubeconfig", "config", "Path to the kube config")
	flag.StringVar(&azurecredfile, "azurecred", "config", "Path to the azure cred file (azure.json)")
	flag.Parse()
	if kubeconfig == "" {
		glog.Fatalf("Could not get the kubernetes cluster config to connect")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		glog.Fatalf("Could not read config properly. Check the k8s config file")
	}

	micClient, err := mic.NewMICClient(config, azurecredfile)
	if err != nil {
		glog.Fatalf("Could not get the crd client: %+v", err)
	}

	micClient.K8sInformers.Core().V1().Pods().Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				pod := obj.(*v1.Pod)
				glog.Infof("NodeName:%s<===", pod.Spec.NodeName)
				if pod.Spec.NodeName != "" {
					micClient.AssignIdentities(pod.Name, pod.Namespace, pod.Spec.NodeName)
				}
				//fmt.Printf("Adding pod: %+v \n", obj)
			},
			DeleteFunc: func(obj interface{}) {
				pod := obj.(*v1.Pod)
				micClient.RemoveAssignedIdentities(pod.Name, pod.Namespace)
				//fmt.Printf("Delete pod : %+v \n", obj)
			},
			UpdateFunc: func(OldObj, newObj interface{}) {
				pastPod := OldObj.(*v1.Pod)
				newPod := newObj.(*v1.Pod)
				if pastPod.Spec.NodeName == "" && newPod.Spec.NodeName != "" {
					glog.Infof("First NodeName update:%s<===", newPod.Spec.NodeName)
					micClient.AssignIdentities(newPod.Name, newPod.Namespace, newPod.Spec.NodeName)
				}
				//fmt.Printf("Update: %+v \n, New: %+v\n", OldObj, newObj)
			},
		})
	exit := make(chan struct{})
	micClient.K8sInformers.Start(exit)
	glog.Info("AAD Pod identity controller initialized!!")
	//Infinite loop :-)
	select {}
}
