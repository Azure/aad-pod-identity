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
	GetNodeIP(hostname string) (nodeip string, err error)
	GetPodCidr(hostname string) (podcidr string, err error)
	GetPodName(podip string) (podns, podname string, err error)
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

// GetNodeIP get node ip from apiserver
func (c *KubeClient) GetNodeIP(hostname string) (nodeip string, err error) {
	return "127.0.0.1", nil
}

// GetPodName get pod ns,name from apiserver
func (c *KubeClient) GetPodName(podip string) (podns, poddname string, err error) {
	if c == nil {
		return "", "", fmt.Errorf("kubeclinet is nil")
	}

	if podip == "" {
		return "", "", fmt.Errorf("podip is nil")
	}

	podipFieldSel := fmt.Sprintf("status.podIp=%s", podip)
	podList, err := c.ClientSet.CoreV1().Pods("default").List(metav1.ListOptions{FieldSelector: podipFieldSel})
	if err != nil {
		return "", "", err
	}

	if len(podList.Items) != 1 {
		return "", "", fmt.Errorf("Expected 1 item in podList, got %d", len(podList.Items))
	}

	return podList.Items[0].Namespace, podList.Items[0].Name, nil
}

// GetPodCidr get node pod cidr from apiserver
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
