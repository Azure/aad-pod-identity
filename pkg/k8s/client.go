package k8s

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	crd "github.com/Azure/aad-pod-identity/pkg/crd"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	getPodListRetries               = 4
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

	optionsModifier := func(options *metav1.ListOptions) {}
	podListWatch := cache.NewFilteredListWatchFromClient(
		clientset.CoreV1().RESTClient(),
		"pods",
		v1.NamespaceAll,
		optionsModifier,
	)

	kubeClient := &KubeClient{CrdClient: crdclient, ClientSet: clientset, PodListWatch: podListWatch}

	return kubeClient, nil
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
	numMatching := len(podList.Items)
	if numMatching == 1 {
		return podList.Items[0].Namespace, podList.Items[0].Name, c.getReplicasetName(podList.Items[0]), &metav1.LabelSelector{
			MatchLabels: podList.Items[0].Labels}, nil
	}

	return "", "", "", nil, fmt.Errorf("match failed, ip:%s matching pods:%v", podip, podList)
}

func (c *KubeClient) getPodList(podip string) (*v1.PodList, error) {
	listObject, err := c.PodListWatch.List(metav1.ListOptions{
		FieldSelector: "status.podIP==" + podip + phaseStatusFilter,
	})

	if err != nil {
		return nil, err
	}

	// Confirm that we are able to cast properly.
	podList, ok := listObject.(*v1.PodList)
	if !ok {
		return nil, fmt.Errorf("list object could not be converted to podlist")
	}

	if podList == nil {
		return nil, fmt.Errorf("pod list nil")
	}

	if len(podList.Items) == 0 {
		return nil, fmt.Errorf("pod list empty")
	}

	return podList, nil
}

func (c *KubeClient) getPodListRetry(podip string, retries int, sleeptime time.Duration) (*v1.PodList, error) {
	var podList *v1.PodList
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
