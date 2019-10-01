package main

import (
	"flag"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/Azure/aad-pod-identity/pkg/mic"
	"github.com/Azure/aad-pod-identity/pkg/probes"
	"github.com/Azure/aad-pod-identity/version"
	"github.com/golang/glog"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeconfig          string
	cloudconfig         string
	forceNamespaced     bool
	versionInfo         bool
	syncRetryDuration   time.Duration
	leaderElectionCfg   mic.LeaderElectionConfig
	httpProbePort       string
	enableProfile       bool
	enableScaleFeatures bool
	createDeleteBatch   int64
)

func main() {
	defer glog.Flush()
	hostName, err := os.Hostname()
	if err != nil {
		glog.Fatalf("Get hostname failure. Error: %+v", err)
	}
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to the kube config")
	flag.StringVar(&cloudconfig, "cloudconfig", "", "Path to cloud config e.g. Azure.json file")
	flag.BoolVar(&forceNamespaced, "forceNamespaced", false, "Forces namespaced identities, binding, and assignment")
	flag.BoolVar(&versionInfo, "version", false, "Prints the version information")
	flag.DurationVar(&syncRetryDuration, "syncRetryDuration", 3600*time.Second, "The interval in seconds at which sync loop should periodically check for errors and reconcile.")

	// Leader election parameters
	flag.StringVar(&leaderElectionCfg.Instance, "leader-election-instance", hostName, "leader election instance name. default is 'hostname'")
	flag.StringVar(&leaderElectionCfg.Namespace, "leader-election-namespace", "default", "namespace to create leader election objects")
	flag.StringVar(&leaderElectionCfg.Name, "leader-election-name", "aad-pod-identity-mic", "leader election name")
	flag.DurationVar(&leaderElectionCfg.Duration, "leader-election-duration", time.Second*15, "leader election duration")

	//Probe port
	flag.StringVar(&httpProbePort, "http-probe-port", "8080", "http liveliness probe port")

	// Profile
	flag.BoolVar(&enableProfile, "enableProfile", false, "Enable/Disable pprof profiling")

	// Profile
	flag.BoolVar(&enableScaleFeatures, "enableScaleFeatures", false, "Enable/Disable new features used for clusters at scale")

	// Profile
	flag.Int64Var(&createDeleteBatch, "createDeleteBatch", 200, "Per node/VMSS create/delete batches")

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
	if enableProfile {
		profilePort := "6060"
		glog.Infof("Starting profiling on port %s", profilePort)
		go func() {
			glog.Error(http.ListenAndServe("localhost:"+profilePort, nil))
		}()
	}

	if enableScaleFeatures {
		glog.Infof("Enabling features for scale clusters")
	}

	glog.Infof("kubeconfig (%s) cloudconfig (%s)", kubeconfig, cloudconfig)
	config, err := buildConfig(kubeconfig)
	if err != nil {
		glog.Fatalf("Could not read config properly. Check the k8s config file, %+v", err)
	}
	config.UserAgent = version.GetUserAgent("MIC", version.MICVersion)

	forceNamespaced = forceNamespaced || "true" == os.Getenv("FORCENAMESPACED")

	micClient, err := mic.NewMICClient(cloudconfig, config, forceNamespaced, syncRetryDuration, &leaderElectionCfg, enableScaleFeatures, createDeleteBatch)
	if err != nil {
		glog.Fatalf("Could not get the MIC client: %+v", err)
	}

	// Health probe will always report success once its started.
	// MIC instance will report the contents as "Active" only once its elected the leader
	// and starts the sync loop.
	probes.InitAndStart(httpProbePort, &micClient.SyncLoopStarted, &mic.Log{})

	// Starts the leader election loop
	micClient.Run()
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
