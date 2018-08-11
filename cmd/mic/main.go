package main

import (
	"flag"

	"github.com/Azure/aad-pod-identity/pkg/mic"
	"github.com/golang/glog"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeconfig  string
	cloudconfig string
)

func main() {
	defer glog.Flush()
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to the kube config")
	flag.StringVar(&cloudconfig, "cloudconfig", "", "Path to cloud config e.g. Azure.json file")
	flag.Parse()
	if cloudconfig == "" {
		glog.Fatalf("Could not get the cloud config")
	}
	if kubeconfig == "" {
		glog.Warningf("--kubeconfig not passed will use InClusterConfig")
	}

	glog.Infof("kubeconfig (%s) cloudconfig (%s)", kubeconfig, cloudconfig)
	config, err := buildConfig(kubeconfig)
	if err != nil {
		glog.Fatalf("Could not read config properly. Check the k8s config file, %+v", err)
	}

	micClient, err := mic.NewMICClient(cloudconfig, config)
	if err != nil {
		glog.Fatalf("Could not get the MIC client: %+v", err)
	}

	exit := make(chan struct{})
	micClient.Start(exit)
	glog.Info("AAD Pod identity controller initialized!!")
	//Infinite loop :-)
	select {}
}

// Create the client config. Use kubeconfig if given, otherwise assume in-cluster.
func buildConfig(kubeconfigPath string) (*rest.Config, error) {
	if kubeconfigPath != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}
	return rest.InClusterConfig()
}
