package k8s

import (
	"fmt"
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Client api client
type Client interface {
	GetPodCidr(hostname string) (podcidr string, err error)
}

// KubeClient k8s client
type KubeClient struct {
	// Main Kubernetes client
	ClientSet *kubernetes.Clientset
}

// NewKubeClient new kubernetes api client
func NewKubeClient() (Client, error) {
	clientset, err := getkubeclient()
	if err != nil {
		return nil, nil
	}

	kubeClient := &KubeClient{ClientSet: clientset}

	return kubeClient, nil
}

// GetPodCidr get node cidr for from apiserver
func (c *KubeClient) GetPodCidr(hostname string) (podcidr string, err error) {
	if c == nil {
		return "", fmt.Errorf("kubeclinet is nil")
	}

	if hostname == "" {
		return "", fmt.Errorf("hostname is nil")
	}

	n, err := c.ClientSet.CoreV1().Nodes().Get(hostname, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	if n.Spec.PodCIDR == "" {
		return "", fmt.Errorf("podcidr is nil or empty, hostname: %s", hostname)
	}

	return n.Spec.PodCIDR, nil
}

func getkubeclient() (*kubernetes.Clientset, error) {
	config, err := buildConfig()
	if err != nil {
		return nil, err
	}
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
