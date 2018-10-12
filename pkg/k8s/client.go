package k8s

import (
	"fmt"
	"net"
	"os"
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	crd "github.com/Azure/aad-pod-identity/pkg/crd"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	getPodListTries                 = 5
	getPodListSleepTimeMilliseconds = 300
)

// Client api client
type Client interface {
	// GetPodName return the matching azure identity or nil
	GetPodName(podip string) (podns, podname string, err error)
	// ListPodIds pod matching azure identity or nil
	ListPodIds(podns, podname string) (*[]aadpodid.AzureIdentity, error)
}

// KubeClient k8s client
type KubeClient struct {
	// Main Kubernetes client
	ClientSet *kubernetes.Clientset
	// Crd client used to access our CRD resources.
	CrdClient *crd.Client
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
	kubeClient := &KubeClient{ClientSet: clientset, CrdClient: crdclient}

	return kubeClient, nil
}

// GetPodName get pod ns,name from apiserver
func (c *KubeClient) GetPodName(podip string) (podns, poddname string, err error) {
	if podip == "" {
		return "", "", fmt.Errorf("podip is empty")
	}

	podList, err := c.getPodListWithTries(podip, getPodListTries)

	if err != nil {
		return "", "", err
	}
	numMatching := len(podList.Items)
	if numMatching == 1 {
		return podList.Items[0].Namespace, podList.Items[0].Name, nil
	}

	return "", "", fmt.Errorf("match failed, ip:%s matching pods:%v", podip, podList)
}

func (c *KubeClient) getPodListWithTries(podip string, tries int) (*v1.PodList, error) {
	podList, err := c.ClientSet.CoreV1().Pods("").List(metav1.ListOptions{
		FieldSelector: "status.podIP==" + podip + ",status.phase==Running",
	})

	if err != nil || len(podList.Items) == 0 {
		if tries > 1 {
			time.Sleep(getPodListSleepTimeMilliseconds * time.Millisecond)
			return c.getPodListWithTries(podip, tries-1)
		}
	}

	return podList, err
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
