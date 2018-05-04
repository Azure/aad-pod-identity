package main

import (
	"flag"

	aadpodidentity "../pkg/apis/aadpodidentity/v1"
	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeconfig string
)

func main() {
	flag.StringVar(&kubeconfig, "kubeconfig", "config", "Path to the kube config")
	flag.Parse()
	if kubeconfig == "" {
		glog.Fatalf("Could not get the kubernetes cluster config to connect")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		glog.Fatalf("Could not read config properly. Check the k8s config file")
	}

	crdClient, err := aadpodidentity.NewAadPodIdentityCrdClient(config)
	if err != nil {
		glog.Fatalf("Could not get the crd client: %+v", err)
	}

	crdClient.K8sInformers.Core().V1().Pods().Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				pod := obj.(*v1.Pod)
				crdClient.Bind(pod.Name)
				//fmt.Printf("Adding pod: %+v \n", obj)
			},
			DeleteFunc: func(obj interface{}) {
				//fmt.Printf("Delete pod : %+v \n", obj)
			},
			UpdateFunc: func(OldObj, newObj interface{}) {
				//fmt.Printf("Update: %+v \n, New: %+v\n", OldObj, newObj)
			},
		})
	exit := make(chan struct{})
	crdClient.K8sInformers.Start(exit)
	//Infinite loop :-)
	select {}
}
