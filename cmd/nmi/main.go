package main

import (
	goflag "flag"
	"os"

	"net/http"
	_ "net/http/pprof"

	"github.com/Azure/aad-pod-identity/pkg/k8s"
	"github.com/Azure/aad-pod-identity/pkg/metrics"
	server "github.com/Azure/aad-pod-identity/pkg/nmi/server"
	"github.com/Azure/aad-pod-identity/pkg/probes"
	"github.com/Azure/aad-pod-identity/version"
	"github.com/spf13/pflag"
	"k8s.io/klog"
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
	debug                              = pflag.Bool("debug", false, "sets log to debug level")
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
	prometheusPort                     = pflag.String("prometheus-port", "9090", "Prometheus port for metrics")
)

func main() {
	klog.InitFlags(nil)
	// this is done for glog used by client-go underneath
	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)

	pflag.Parse()
	if *versionInfo {
		version.PrintVersionAndExit()
	}

	klog.Infof("Starting nmi process. Version: %v. Build date: %v.", version.NMIVersion, version.BuildDate)

	if *enableProfile {
		profilePort := "6060"
		klog.Infof("Starting profiling on port %s", profilePort)
		go func() {
			klog.Error(http.ListenAndServe("localhost:"+profilePort, nil))
		}()
	}
	if *enableScaleFeatures {
		klog.Infof("Features for scale clusters enabled")
	}

	client, err := k8s.NewKubeClient(*nodename, *enableScaleFeatures)
	if err != nil {
		klog.Fatalf("%+v", err)
	}
	exit := make(<-chan struct{})
	client.Start(exit)
	*forceNamespaced = *forceNamespaced || "true" == os.Getenv("FORCENAMESPACED")
	s := server.NewServer(*forceNamespaced, *micNamespace, *blockInstanceMetadata)
	s.KubeClient = client
	s.MetadataIP = *metadataIP
	s.MetadataPort = *metadataPort
	s.NMIPort = *nmiPort
	s.HostIP = *hostIP
	s.NodeName = *nodename
	s.IPTableUpdateTimeIntervalInSeconds = *ipTableUpdateTimeIntervalInSeconds
	s.ListPodIDsRetryAttemptsForCreated = *retryAttemptsForCreated
	s.ListPodIDsRetryAttemptsForAssigned = *retryAttemptsForAssigned
	s.ListPodIDsRetryIntervalInSeconds = *findIdentityRetryIntervalInSeconds

	// Health probe will always report success once its started. The contents
	// will report "Active" once the iptables rules are set
	probes.InitAndStart(*httpProbePort, &s.Initialized)

	// Register and expose metrics views
	if err = metrics.RegisterAndExport(*prometheusPort); err != nil {
		klog.Fatalf("Could not register and export metrics: %+v", err)
	}

	if err := s.Run(); err != nil {
		klog.Fatalf("%s", err)
	}
}
