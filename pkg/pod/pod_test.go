package pod

import (
	"time"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity"
	"github.com/Azure/aad-pod-identity/pkg/stats"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TestPodClient struct {
	pods []*corev1.Pod
}

func NewTestPodClient() *TestPodClient {
	var pods []*corev1.Pod
	var podInterface *corev1.PodInterface
	return &TestPodClient{
		pods: pods,
	}
}

func (c TestPodClient) Start(exit <-chan struct{}) {
	glog.Info("Start called from the test interface")
}

func (c TestPodClient) GetPods(labelSelector *metav1.LabelSelector) (pods []*corev1.Pod, err error) {
	//TODO: Add label matching. For now we add only pods which we want to add.
	selector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		return nil, err
	}

	listPods, err := c.PodWatcher.Lister().List(selector)
	if err != nil {
		return nil, err
	}
	stats.Put(stats.PodList, time.Since(begin))
	return listPods, nil
	return c.pods, nil
}

func (c *TestPodClient) AddPod(podName string, podNs string, nodeName string, binding string) {
	labels := make(map[string]string, 0)
	labels[aadpodid.CRDLabelKey] = binding
	pod := &corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:      podName,
			Namespace: podNs,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			NodeName: nodeName,
		},
	}
	c.pods = append(c.pods, pod)
}

func (c *TestPodClient) DeletePod(podName string, podNs string) {
	var newPods []*corev1.Pod
	changed := false
	for _, pod := range c.pods {
		if pod.Name == podName && pod.Namespace == podNs {
			changed = true
			continue
		} else {
			newPods = append(newPods, pod)
		}
	}
	if changed {
		c.pods = newPods
	}
}
