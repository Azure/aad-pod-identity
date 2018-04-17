package main

import (
	"flag"
	"fmt"
	"time"

	"../pkg/apis/aadpodidentity/v1"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/fields"
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

	_, controller := cache.NewInformer(
		cache.NewListWatchFromClient(crdClient, "azureidentities",
			"default", fields.Everything()),
		&aadpodidentity.AzureIdentity{},
		time.Minute*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				fmt.Printf("Adding: %s \n", obj)
			},
			DeleteFunc: func(obj interface{}) {
				fmt.Printf("Delete: %s \n", obj)
			},
			UpdateFunc: func(OldObj, newObj interface{}) {
				fmt.Printf("Update: %s \n, New: %s\n", OldObj, newObj)
			},
		},
	)
	exit := make(chan struct{})
	go controller.Run(exit)
	//Infinite loop :-)
	select {}
}
