package k8s

import (
	"fmt"
	"os"
	"strings"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	crd "github.com/Azure/aad-pod-identity/pkg/crd"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Client api client
type Client interface {
	// GetPodCidr return the pod cidr for the node
	GetPodCidr(nodename string) (podcidr string, err error)
	// GetPodName return the matching azure identity or nil
	GetPodName(podip string) (podns, podname string, err error)
	// GetAzureAssignedIdentity return the matching azure identity or nil
	GetUserAssignedIdentities(podns, podname string) (*[]aadpodid.AzureAssignedIdentity, error)
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

	crdclient, err := crd.NewCRDClient(config)
	if err != nil {
		return nil, err
	}

	kubeClient := &KubeClient{ClientSet: clientset, CrdClient: crdclient}

	return kubeClient, nil
}

// GetPodName get pod ns,name from apiserver
func (c *KubeClient) GetPodName(podip string) (podns, poddname string, err error) {
	if c == nil {
		return "", "", fmt.Errorf("kubeclinet is nil")
	}

	if podip == "" {
		return "", "", fmt.Errorf("podip is empty")
	}

	//podipFieldSel := fmt.Sprintf("status.podIp=%s", podip)
	podList, err := c.ClientSet.CoreV1().Pods("").List(metav1.ListOptions{})
	if err != nil {
		return "", "", err
	}
	for _, pod := range podList.Items {
		if !strings.EqualFold(pod.Status.PodIP, podip) {
			continue
		}

		return pod.Namespace, pod.Name, nil
	}

	return "", "", fmt.Errorf("not found")
}

// GetPodCidr get node pod cidr from apiserver
func (c *KubeClient) GetPodCidr(nodename string) (podcidr string, err error) {
	if c == nil {
		return "", fmt.Errorf("kubeclinet is nil")
	}

	if nodename == "" {
		return "", fmt.Errorf("nodename is nil")
	}

	n, err := c.ClientSet.CoreV1().Nodes().Get(nodename, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	if n.Spec.PodCIDR == "" {
		return "", fmt.Errorf("podcidr is nil or empty, nodename: %s", nodename)
	}

	return n.Spec.PodCIDR, nil
}

// GetUserAssignedIdentities return the matching azure identities or error
func (c *KubeClient) GetUserAssignedIdentities(podns, podname string) (*[]aadpodid.AzureAssignedIdentity, error) {
	return c.CrdClient.GetUserAssignedIdentities(podns, podname)
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
