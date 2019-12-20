package main

import (
	"flag"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"
	"time"

	"github.com/Azure/aad-pod-identity/pkg/metrics"
	"github.com/Azure/aad-pod-identity/pkg/mic"
	"github.com/Azure/aad-pod-identity/pkg/probes"
	"github.com/Azure/aad-pod-identity/version"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
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
	clientQPS           float64
	prometheusPort      string
	immutableUserMSIs   string
)

func main() {
	defer klog.Flush()
	hostName, err := os.Hostname()
	if err != nil {
		klog.Fatalf("Get hostname failure. Error: %+v", err)
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

	// Prometheus port
	flag.StringVar(&prometheusPort, "prometheus-port", "8888", "Prometheus port for metrics")

	// Profile
	flag.BoolVar(&enableProfile, "enableProfile", false, "Enable/Disable pprof profiling")

	// Enable scale features handles the label based azureassignedidentity.
	flag.BoolVar(&enableScaleFeatures, "enableScaleFeatures", false, "Enable/Disable new features used for clusters at scale")

	// createDeleteBatch can be used for tuning the number of outstanding api server operations we do per node/VMSS.
	flag.Int64Var(&createDeleteBatch, "createDeleteBatch", 20, "Per node/VMSS create/delete batches")

	// Client QPS is used to configure the client-go QPS throttling and bursting.
	flag.Float64Var(&clientQPS, "clientQps", 5, "Client QPS used for throttling of calls to kube-api server")

	//Identities that should be never removed from Azure AD (used defined managed identities)
	flag.StringVar(&immutableUserMSIs, "immutable-user-msis", "", "prevent deletion of these IDs from the underlying VM/VMSS")

	flag.Parse()
	if versionInfo {
		version.PrintVersionAndExit()
	}
	klog.Infof("Starting mic process. Version: %v. Build date: %v", version.MICVersion, version.BuildDate)
	if cloudconfig == "" {
		klog.Warningf("--cloudconfig not passed will use aadpodidentity-admin-secret")
	}
	if kubeconfig == "" {
		klog.Warningf("--kubeconfig not passed will use InClusterConfig")
	}
	if enableProfile {
		profilePort := "6060"
		klog.Infof("Starting profiling on port %s", profilePort)
		go func() {
			klog.Error(http.ListenAndServe("localhost:"+profilePort, nil))
		}()
	}

	if enableScaleFeatures {
		klog.Infof("Enabling features for scale clusters")
	}

	klog.Infof("kubeconfig (%s) cloudconfig (%s)", kubeconfig, cloudconfig)
	config, err := buildConfig(kubeconfig)
	if err != nil {
		klog.Fatalf("Could not read config properly. Check the k8s config file, %+v", err)
	}
	config.UserAgent = version.GetUserAgent("MIC", version.MICVersion)

	forceNamespaced = forceNamespaced || "true" == os.Getenv("FORCENAMESPACED")

	config.QPS = float32(clientQPS)
	config.Burst = int(clientQPS)
	klog.Infof("Client QPS set to: %v. Burst to: %v", config.QPS, config.Burst)

	var immutableUserMSIsList []string
	if immutableUserMSIs != "" {
		immutableUserMSIsList = strings.Split(immutableUserMSIs, ",")
	}

	micClient, err := mic.NewMICClient(cloudconfig, config, forceNamespaced, syncRetryDuration, &leaderElectionCfg, enableScaleFeatures, createDeleteBatch, immutableUserMSIsList)
	if err != nil {
		klog.Fatalf("Could not get the MIC client: %+v", err)
	}

	// Health probe will always report success once its started.
	// MIC instance will report the contents as "Active" only once its elected the leader
	// and starts the sync loop.
	probes.InitAndStart(httpProbePort, &micClient.SyncLoopStarted)

	// Register and expose metrics views
	if err = metrics.RegisterAndExport(prometheusPort); err != nil {
		klog.Fatalf("Could not register and export metrics: %+v", err)
	}

	// Starts the leader election loop
	micClient.Run()
	klog.Info("AAD Pod identity controller initialized!!")
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
