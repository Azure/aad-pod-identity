package main

import (
	"flag"
	"os"

	"github.com/Azure/aad-pod-identity/pkg/mic"
	"github.com/Azure/aad-pod-identity/version"
	"github.com/golang/glog"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeconfig      string
	cloudconfig     string
	forceNamespaced bool
	versionInfo     bool
)

func main() {
	defer glog.Flush()
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to the kube config")
	flag.StringVar(&cloudconfig, "cloudconfig", "", "Path to cloud config e.g. Azure.json file")
	flag.BoolVar(&forceNamespaced, "forceNamespaced", false, "Forces namespaced identities, binding, and assignment")
	flag.BoolVar(&versionInfo, "version", false, "Prints the version information")
	flag.Parse()
	if versionInfo {
		version.PrintVersionAndExit()
	}
	glog.Infof("Starting mic process. Version: %v. Build date: %v", version.MICVersion, version.BuildDate)
	if cloudconfig == "" {
		glog.Warningf("--cloudconfig not passed will use aadpodidentity-admin-secret")
	}
	if kubeconfig == "" {
		glog.Warningf("--kubeconfig not passed will use InClusterConfig")
	}

	glog.Infof("kubeconfig (%s) cloudconfig (%s)", kubeconfig, cloudconfig)
	config, err := buildConfig(kubeconfig)
	if err != nil {
		glog.Fatalf("Could not read config properly. Check the k8s config file, %+v", err)
	}

	forceNamespaced = forceNamespaced || "true" == os.Getenv("FORCENAMESPACED")
	micClient, err := mic.NewMICClient(cloudconfig, config, forceNamespaced)
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
