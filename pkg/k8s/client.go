package k8s

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/fields"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	crd "github.com/Azure/aad-pod-identity/pkg/crd"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	getPodListTries                 = 5
	getPodListSleepTimeMilliseconds = 300
)

var (
	// We only want to allow pod-identity with Pending or Running phase status
	ignorePodPhaseStatuses = []string{"Succeeded", "Failed", "Unknown", "Completed", "CrashLoopBackOff"}
	phaseStatusFilter      = getPodPhaseFilter()
)

func getPodPhaseFilter() string {
	return ",status.phase!=" + strings.Join(ignorePodPhaseStatuses, ",status.phase!=")
}

// Client api client
type Client interface {
	// GetPodName return the matching azure identity or nil
	GetPodName(podip string) (podns, podname string, err error)
	// ListPodIds pod matching azure identity or nil
	ListPodIds(podns, podname string) (*[]aadpodid.AzureIdentity, error)
	// GetSecret returns secret the secretRef represents
	GetSecret(secretRef *v1.SecretReference) (*v1.Secret, error)
}

// KubeClient k8s client
type KubeClient struct {
	// Main Kubernetes client
	ClientSet kubernetes.Interface
	// Crd client used to access our CRD resources.
	CrdClient *crd.Client
	//PodListWatch is used to list the pods from cache
	PodListWatch *cache.ListWatch
}

// NewKubeClient new kubernetes api client
func NewKubeClient() (Client, error) {
	config, err := buildConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := getkubeclient(config)
	if err != nil {
		return nil, err
	}
	crdclient, err := crd.NewCRDClientLite(config)
	if err != nil {
		return nil, err
	}

	podListWatch := cache.NewListWatchFromClient(
		clientset.CoreV1().RESTClient(),
		"pods",
		v1.NamespaceAll,
		fields.Everything())

	kubeClient := &KubeClient{CrdClient: crdclient, ClientSet: clientset, PodListWatch: podListWatch}

	return kubeClient, nil
}

// GetPodName get pod ns,name from apiserver
func (c *KubeClient) GetPodName(podip string) (podns, poddname string, err error) {
	if podip == "" {
		return "", "", fmt.Errorf("podip is empty")
	}

	podList, err := c.getPodListWithTries(podip, getPodListTries, getPodListSleepTimeMilliseconds)

	if err != nil {
		return "", "", err
	}
	numMatching := len(podList.Items)
	if numMatching == 1 {
		return podList.Items[0].Namespace, podList.Items[0].Name, nil
	}

	return "", "", fmt.Errorf("match failed, ip:%s matching pods:%v", podip, podList)
}

func (c *KubeClient) getPodListWithTries(podip string, tries int, sleeptime time.Duration) (*v1.PodList, error) {
	podList, err := c.PodListWatch.List(metav1.ListOptions{
		FieldSelector: "status.podIP==" + podip + phaseStatusFilter,
	})

	if err != nil || len(podList.(*v1.PodList).Items) == 0 {
		if tries > 1 {
			time.Sleep(sleeptime * time.Millisecond)
			return c.getPodListWithTries(podip, tries-1, sleeptime)
		}
	}

	return podList.(*v1.PodList), err
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
func (c *KubeClient) ListPodIds(podns, podname string) (*[]aadpodid.AzureIdentity, error) {
	return c.CrdClient.ListPodIds(podns, podname)
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
