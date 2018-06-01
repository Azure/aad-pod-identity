package main

import (
	"flag"

	"github.com/Azure/aad-pod-identity/pkg/mic"
	"github.com/golang/glog"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeconfig  string
	cloudconfig string
)

func main() {
	defer glog.Flush()
	flag.StringVar(&kubeconfig, "kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig", "Path to the kube config")
	flag.StringVar(&cloudconfig, "cloudconfig", "/etc/kubernetes/azure.json", "Path to cloud config e.g. Azure.json file")
	flag.Parse()
	if kubeconfig == "" {
		glog.Fatalf("Could not get the kubernetes cluster config to connect")
	}
	if cloudconfig == "" {
		glog.Fatalf("Could not get the cloud config")
	}

	glog.Infof("kubeconfig (%s) cloudconfig (%s)", kubeconfig, cloudconfig)

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		glog.Fatalf("Could not read config properly. Check the k8s config file, %+v", err)
	}

	micClient, err := mic.NewMICClient(cloudconfig, config)
	if err != nil {
		glog.Fatalf("Could not get the crd client: %+v", err)
	}

	exit := make(chan struct{})
	micClient.Start(exit)
	glog.Info("AAD Pod identity controller initialized!!")
	//Infinite loop :-)
	select {}
}
