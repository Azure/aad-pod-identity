package k8s

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	crd "github.com/Azure/aad-pod-identity/pkg/crd"
	inlog "github.com/Azure/aad-pod-identity/pkg/logger"
	"github.com/Azure/aad-pod-identity/version"
	"github.com/golang/glog"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	informersv1 "k8s.io/client-go/informers/core/v1"
)

const (
	getPodListRetries               = 4
	getPodListSleepTimeMilliseconds = 300
)

// Client api client
type Client interface {
	// Start just starts any informers required.
	Start(<-chan struct{})
	// GetPodInfo returns the pod name, namespace & replica set name for a given pod ip
	GetPodInfo(podip string) (podns, podname, rsName string, selectors *metav1.LabelSelector, err error)
	// ListPodIds pod matching azure identity or nil
	ListPodIds(podns, podname string) (map[string][]aadpodid.AzureIdentity, error)
	// GetSecret returns secret the secretRef represents
	GetSecret(secretRef *v1.SecretReference) (*v1.Secret, error)
	// ListPodIdentityExceptions returns list of azurepodidentityexceptions
	ListPodIdentityExceptions(namespace string) (*[]aadpodid.AzurePodIdentityException, error)
}

// KubeClient k8s client
type KubeClient struct {
	// Main Kubernetes client
	ClientSet kubernetes.Interface
	// Crd client used to access our CRD resources.
	CrdClient   *crd.Client
	PodInformer informersv1.PodInformer
	log         inlog.Logger
}

// NewKubeClient new kubernetes api client
func NewKubeClient(log inlog.Logger) (Client, error) {
	config, err := buildConfig()
	if err != nil {
		return nil, err
	}
	config.UserAgent = version.GetUserAgent("NMI", version.NMIVersion)
	clientset, err := getkubeclient(config)
	if err != nil {
		return nil, err
	}
	crdclient, err := crd.NewCRDClientLite(config, log)
	if err != nil {
		return nil, err
	}

	informer := informers.NewSharedInformerFactory(clientset, 30*time.Second)
	podInformer := informer.Core().V1().Pods()

	kubeClient := &KubeClient{
		CrdClient:   crdclient,
		ClientSet:   clientset,
		PodInformer: podInformer,
		log:         log,
	}

	return kubeClient, nil
}

// Start the corresponding starts
func (c *KubeClient) Start(exit <-chan struct{}) {
	go c.PodInformer.Informer().Run(exit)
	c.CrdClient.StartLite(exit)
	c.CrdClient.SyncCache(exit, true)
}

func (c *KubeClient) getReplicasetName(pod v1.Pod) string {
	for _, owner := range pod.OwnerReferences {
		if strings.EqualFold(owner.Kind, "ReplicaSet") {
			return owner.Name
		}
	}
	return ""
}

// GetPodInfo get pod ns,name from apiserver
func (c *KubeClient) GetPodInfo(podip string) (podns, poddname, rsName string, labels *metav1.LabelSelector, err error) {
	if podip == "" {
		return "", "", "", nil, fmt.Errorf("podip is empty")
	}

	podList, err := c.getPodListRetry(podip, getPodListRetries, getPodListSleepTimeMilliseconds)

	if err != nil {
		return "", "", "", nil, err
	}
	numMatching := len(podList)
	if numMatching == 1 {
		return podList[0].Namespace, podList[0].Name, c.getReplicasetName(*podList[0]), &metav1.LabelSelector{
			MatchLabels: podList[0].Labels}, nil
	}

	return "", "", "", nil, fmt.Errorf("match failed, ip:%s matching pods:%v", podip, podList)
}

func isPhaseValid(p v1.PodPhase) bool {
	return p == v1.PodPending || p == v1.PodRunning
}

func (c *KubeClient) getPodList(podip string) ([]*v1.Pod, error) {
	list, err := c.PodInformer.Lister().List(labels.Everything())
	if err != nil {
		glog.Error(err)
		return nil, err
	}

	var podList []*v1.Pod
	for _, pod := range list {
		if pod.Status.PodIP == podip && isPhaseValid(pod.Status.Phase) {
			podList = append(podList, pod)
		}
	}
	if len(podList) == 0 {
		err := fmt.Errorf("pod list empty")
		glog.Error(err)
		return nil, err
	}
	return podList, nil
}

func (c *KubeClient) getPodListRetry(podip string, retries int, sleeptime time.Duration) ([]*v1.Pod, error) {
	var podList []*v1.Pod
	var err error
	i := 0

	for {
		// Atleast run the getpodlist once.
		podList, err = c.getPodList(podip)
		if err == nil {
			return podList, nil
		}
		if i >= retries {
			break
		}
		i++
		log.Warningf("List pod error: %+v. Retrying, attempt number: %d", err, i)
		time.Sleep(sleeptime * time.Millisecond)
	}
	// We reach here only if there is an error and we have exhausted all retries.
	// Return the last error
	return nil, err
}

// GetLocalIP returns the non loopback local IP of the host
func GetLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}
	return "", fmt.Errorf("non loopback ip address not found")
}

// ListPodIds lists matching ids for pod or error
func (c *KubeClient) ListPodIds(podns, podname string) (map[string][]aadpodid.AzureIdentity, error) {
	return c.CrdClient.ListPodIds(podns, podname)
}

// ListPodIdentityExceptions lists azurepodidentityexceptions
func (c *KubeClient) ListPodIdentityExceptions(ns string) (*[]aadpodid.AzurePodIdentityException, error) {
	return c.CrdClient.ListPodIdentityExceptions(ns)
}

// GetSecret returns secret the secretRef represents
func (c *KubeClient) GetSecret(secretRef *v1.SecretReference) (*v1.Secret, error) {
	secret, err := c.ClientSet.CoreV1().Secrets(secretRef.Namespace).Get(secretRef.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return secret, nil
}

func getkubeclient(config *rest.Config) (*kubernetes.Clientset, error) {
	// creates the clientset
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return kubeClient, err
}

// Create the client config. Use kubeconfig if given, otherwise assume in-cluster.
func buildConfig() (*rest.Config, error) {
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}

	return rest.InClusterConfig()
}
