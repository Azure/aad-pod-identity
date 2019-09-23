package main

import (
	"os"

	"net/http"
	_ "net/http/pprof"

	"github.com/Azure/aad-pod-identity/pkg/k8s"
	server "github.com/Azure/aad-pod-identity/pkg/nmi/server"
	"github.com/Azure/aad-pod-identity/pkg/probes"
	"github.com/Azure/aad-pod-identity/version"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
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
)

func main() {
	pflag.Parse()
	if *versionInfo {
		version.PrintVersionAndExit()
	}

	log.SetLevel(log.InfoLevel)
	if *debug {
		log.SetLevel(log.DebugLevel)
	}
	log.Infof("Starting nmi process. Version: %v. Build date: %v. Log level: %s.", version.NMIVersion, version.BuildDate, log.GetLevel())

	if *enableProfile {
		profilePort := "6060"
		log.Infof("Starting profiling on port %s", profilePort)
		go func() {
			log.Error(http.ListenAndServe("localhost:"+profilePort, nil))
		}()
	}
	if *enableScaleFeatures {
		log.Infof("Features for scale clusters enabled")
	}

	logger := &server.Log{}

	client, err := k8s.NewKubeClient(logger, *nodename, *enableScaleFeatures)
	if err != nil {
		log.Fatalf("%+v", err)
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
	probes.InitAndStart(*httpProbePort, &s.Initialized, logger)

	if err := s.Run(); err != nil {
		log.Fatalf("%s", err)
	}
}
