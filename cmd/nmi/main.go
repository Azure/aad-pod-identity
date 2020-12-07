package main

import (
	goflag "flag"
	"net/http"
	_ "net/http/pprof" // #nosec
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/Azure/aad-pod-identity/pkg/log"
	"github.com/Azure/aad-pod-identity/pkg/metrics"
	"github.com/Azure/aad-pod-identity/pkg/nmi"
	"github.com/Azure/aad-pod-identity/pkg/nmi/server"
	"github.com/Azure/aad-pod-identity/pkg/probes"
	"github.com/Azure/aad-pod-identity/version"

	"github.com/spf13/pflag"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

const (
	defaultMetadataIP                         = "169.254.169.254"
	defaultMetadataPort                       = "80"
	defaultNmiPort                            = "2579"
	defaultIPTableUpdateTimeIntervalInSeconds = 60
	defaultlistPodIDsRetryAttemptsForCreated  = 16
	defaultlistPodIDsRetryAttemptsForAssigned = 4
	defaultlistPodIDsRetryIntervalInSeconds   = 5
)

var (
	versionInfo                        = pflag.Bool("version", false, "prints the version information")
	nmiPort                            = pflag.String("nmi-port", defaultNmiPort, "NMI application port")
	metadataIP                         = pflag.String("metadata-ip", defaultMetadataIP, "instance metadata host ip")
	metadataPort                       = pflag.String("metadata-port", defaultMetadataPort, "instance metadata host ip")
	hostIP                             = pflag.String("host-ip", "", "host IP address")
	nodename                           = pflag.String("node", "", "node name")
	ipTableUpdateTimeIntervalInSeconds = pflag.Int("ipt-update-interval-sec", defaultIPTableUpdateTimeIntervalInSeconds, "update interval of iptables")
	forceNamespaced                    = pflag.Bool("forceNamespaced", false, "Forces mic to namespace identities, binding, and assignment")
	micNamespace                       = pflag.String("MICNamespace", "default", "MIC namespace to short circuit MIC token requests")
	httpProbePort                      = pflag.String("http-probe-port", "8080", "Http health and liveness probe port")
	retryAttemptsForCreated            = pflag.Int("retry-attempts-for-created", defaultlistPodIDsRetryAttemptsForCreated, "Number of retries in NMI to find assigned identity in CREATED state")
	retryAttemptsForAssigned           = pflag.Int("retry-attempts-for-assigned", defaultlistPodIDsRetryAttemptsForAssigned, "Number of retries in NMI to find assigned identity in ASSIGNED state")
	findIdentityRetryIntervalInSeconds = pflag.Int("find-identity-retry-interval", defaultlistPodIDsRetryIntervalInSeconds, "Retry interval to find assigned identities in seconds")
	enableProfile                      = pflag.Bool("enableProfile", false, "Enable/Disable pprof profiling")
	enableScaleFeatures                = pflag.Bool("enableScaleFeatures", false, "Enable/Disable features for scale clusters")
	blockInstanceMetadata              = pflag.Bool("block-instance-metadata", false, "Block instance metadata endpoints")
	metadataHeaderRequired             = pflag.Bool("metadata-header-required", false, "Metadata header required for querying Azure Instance Metadata service")
	prometheusPort                     = pflag.String("prometheus-port", "9090", "Prometheus port for metrics")
	operationMode                      = pflag.String("operation-mode", "standard", "NMI operation mode")
	kubeconfig                         = pflag.String("kubeconfig", "", "Path to the kube config")
	allowNetworkPluginKubenet          = pflag.Bool("allow-network-plugin-kubenet", false, "Allow running aad-pod-identity in cluster with kubenet")
	kubeletConfig                      = pflag.String("kubelet-config", "/etc/default/kubelet", "Path to kubelet default config")
)

// Delay nmi startup due to DNS not being available during first seconds of nmi process execution.
// Bug tracks removal of this delay: https://o365exchange.visualstudio.com/O365%20Core/_workitems/edit/1739605
const nmiStatupDelay = time.Second * 10

func main() {
	klog.InitFlags(nil)
	defer klog.Flush()

	logOptions := log.NewOptions()
	logOptions.AddFlags()

	// this is done for glog used by client-go underneath
	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)

	pflag.Parse()

	if err := logOptions.Apply(); err != nil {
		klog.Fatalf("unable to apply logging options, error: %+v", err)
	}

	if *versionInfo {
		version.PrintVersionAndExit()
	}

	klog.Infof("starting nmi process. Version: %v. Build date: %v.", version.NMIVersion, version.BuildDate)
	// Bug tracks removal of this delay: https://o365exchange.visualstudio.com/O365%20Core/_workitems/edit/1739605
	time.Sleep(nmiStatupDelay)

	if *enableProfile {
		profilePort := "6060"
		klog.Infof("starting profiling on port %s", profilePort)
		go func() {
			addr := "localhost:" + profilePort
			if err := http.ListenAndServe(addr, nil); err != nil {
				klog.Errorf("failed to listen and serve %s, error: %+v", addr, err)
			}
		}()
	}
	if *enableScaleFeatures {
		klog.Infof("features for scale clusters enabled")
	}

	// Register and expose metrics views
	if err := metrics.RegisterAndExport(*prometheusPort); err != nil {
		klog.Fatalf("failed to register and export metrics on port %s, error: %+v", *prometheusPort, err)
	}

	// normalize operation mode
	*operationMode = strings.ToLower(*operationMode)

	client, err := nmi.GetKubeClient(*nodename, *operationMode, *enableScaleFeatures)
	if err != nil {
		klog.Fatalf("failed to get kube client, error: %+v", err)
	}

	klog.Infof("Build kubeconfig (%s)", kubeconfig)
	config, err := buildConfig(*kubeconfig)
	if err != nil {
		klog.Fatalf("Could not read config properly. Check the k8s config file, %+v", err)
	}

	exit := make(<-chan struct{})
	client.Start(exit)
	*forceNamespaced = *forceNamespaced || "true" == os.Getenv("FORCENAMESPACED")
	klog.Infof("running NMI in namespaced mode: %v", *forceNamespaced)

	s := server.NewServer(*micNamespace, *blockInstanceMetadata, *metadataHeaderRequired, config)
	s.KubeClient = client
	s.MetadataIP = *metadataIP
	s.MetadataPort = *metadataPort
	s.NMIPort = *nmiPort
	s.HostIP = *hostIP
	s.NodeName = *nodename
	s.IPTableUpdateTimeIntervalInSeconds = *ipTableUpdateTimeIntervalInSeconds

	nmiConfig := nmi.Config{
		Mode:                               strings.ToLower(*operationMode),
		RetryAttemptsForCreated:            *retryAttemptsForCreated,
		RetryAttemptsForAssigned:           *retryAttemptsForAssigned,
		FindIdentityRetryIntervalInSeconds: *findIdentityRetryIntervalInSeconds,
		Namespaced:                         *forceNamespaced,
	}

	// Create new token client based on the nmi mode
	tokenClient, err := nmi.GetTokenClient(client, nmiConfig)
	if err != nil {
		klog.Fatalf("failed to initialize token client, error: %+v", err)
	}
	s.TokenClient = tokenClient

	mainRoutineDone := make(chan struct{})
	subRoutineDone := make(chan struct{})

	var redirector server.RedirectorFunc
	if runtime.GOOS == "windows" {
		// NMI Windows health probe will report successful if it can call hcn agent sucessfully,
		// or it will return 500 error and the nmi windows pod will get reboot.
		probes.InitAndStartNMIWindowsProbe(*httpProbePort, &s.Initialized, *nodename)
		redirector = server.WindowsRedirector(s, subRoutineDone)
	} else {
		// NMI Linux Health probe will always report success once its started. The contents
		// will report "Active" once the iptables rules are set
		probes.InitAndStart(*httpProbePort, &s.Initialized)
		redirector = server.LinuxRedirector(s, subRoutineDone)
	}

	go redirector(s, subRoutineDone, mainRoutineDone)

	if err := s.Run(); err != nil {
		klog.Errorf("%s", err)
	}

	close(mainRoutineDone)
	<-subRoutineDone
}

// Create the client config. Use kubeconfig if given, otherwise assume in-cluster.
func buildConfig(kubeconfigPath string) (*rest.Config, error) {
	if kubeconfigPath != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}
	return rest.InClusterConfig()
}
