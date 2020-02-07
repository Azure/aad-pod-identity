package pod

import (
	"fmt"
	"strings"
	"time"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity"
	"github.com/Azure/aad-pod-identity/pkg/stats"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/informers"
	informersv1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
)

// Client represents new pod client
type Client struct {
	PodWatcher informersv1.PodInformer
}

// ClientInt represents pod client interface
type ClientInt interface {
	GetPods() (pods []*v1.Pod, err error)
	Start(exit <-chan struct{})
	ListPods() (pods []*v1.Pod, err error)
}

// NewPodClient returns new pod client
func NewPodClient(i informers.SharedInformerFactory, eventCh chan aadpodid.EventType, podInfoCh chan *v1.Pod) (c ClientInt) {
	podInformer := i.Core().V1().Pods()
	addPodHandler(podInformer, eventCh, podInfoCh)

	return &Client{
		PodWatcher: podInformer,
	}
}

func addPodHandler(i informersv1.PodInformer, eventCh chan aadpodid.EventType, podInfoCh chan *v1.Pod) {
	i.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				klog.V(6).Infof("Pod Created")
				eventCh <- aadpodid.PodCreated

				currentPod, _ := GetPod(obj, i)
				podInfoCh <- currentPod
			},
			DeleteFunc: func(obj interface{}) {
				klog.V(6).Infof("Pod Deleted")
				eventCh <- aadpodid.PodDeleted

				// The following code may not be necessary. Add for log purpose. May remove them finally.
				currentPod := obj.(*v1.Pod)

				fmt.Printf("Pod UID:%s \n", currentPod.UID)
				podInfoCh <- currentPod
			},
			UpdateFunc: func(OldObj, newObj interface{}) {
				// We are only interested in updates to pod if the node changes.
				// Having this check will ensure that mic sync loop does not do extra work
				// for every pod update.
				if (OldObj.(*v1.Pod)).Spec.NodeName != (newObj.(*v1.Pod)).Spec.NodeName {
					klog.V(6).Infof("Pod Updated")
					eventCh <- aadpodid.PodUpdated
				}
			},
		},
	)
}

func (c *Client) syncCache(exit <-chan struct{}) {
	cacheSyncStarted := time.Now()
	klog.V(6).Infof("Wait for cache to sync")
	if !cache.WaitForCacheSync(exit, c.PodWatcher.Informer().HasSynced) {
		klog.Error("Wait for pod cache sync failed")
		return
	}
	klog.Infof("Pod cache synchronized. Took %s", time.Since(cacheSyncStarted).String())
}

// Start ...
func (c *Client) Start(exit <-chan struct{}) {
	go c.PodWatcher.Informer().Run(exit)
	c.syncCache(exit)
	klog.Info("Pod watcher started !!")
}

// GetPods returns list of all pods
func (c *Client) GetPods() (pods []*v1.Pod, err error) {
	begin := time.Now()
	crdReq, err := labels.NewRequirement(aadpodid.CRDLabelKey, selection.Exists, nil)
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	crdSelector := labels.NewSelector().Add(*crdReq)
	listPods, err := c.PodWatcher.Lister().List(crdSelector)
	if err != nil {
		return nil, err
	}
	stats.Put(stats.PodList, time.Since(begin))
	return listPods, nil
}

// ListPods returns list of all pods
func (c *Client) ListPods() (pods []*v1.Pod, err error) {
	var resList []*v1.Pod
	listPods := c.PodWatcher.Informer().GetStore().List()

	for _, pod := range listPods {
		v1Pod, ok := pod.(*v1.Pod)
		if !ok {
			err := fmt.Errorf("could not cast %T to v1.Pod", pod)
			klog.Error(err)
			return nil, err
		}

		resList = append(resList, v1Pod)
		klog.V(6).Infof("Appending pod ID: %s/%s to list.", v1Pod.UID, v1Pod.Name)
	}

	return resList, nil
}

// GetPod returns the pod object
func GetPod(obj interface{}, i informersv1.PodInformer) (pod *v1.Pod, err error) {
	currentPod, exists, err := i.Informer().GetStore().Get(obj)
	if !exists || err != nil {
		err := fmt.Errorf("Could not get Pod %s", obj.(*v1.Pod))
		klog.Error(err)
		return nil, err
	}

	pod = currentPod.(*v1.Pod)
	for pod.Status.PodIP == "" || pod.Spec.NodeName == "" {
		fmt.Printf("Sleep 1 second to wait for pod ip and node name\n")
		time.Sleep(1 * time.Second)
		currentPod, exists, err := i.Informer().GetStore().Get(obj)
		if !exists || err != nil {
			err := fmt.Errorf("Could not get Pod %s", obj.(*v1.Pod))
			klog.Error(err)
			return nil, err
		}

		pod = currentPod.(*v1.Pod)
	}

	return pod, nil
}

// IsPodExcepted returns true if pod label is part of exception crd
func IsPodExcepted(podLabels map[string]string, exceptionList []aadpodid.AzurePodIdentityException) bool {
	return len(exceptionList) > 0 && labelInExceptionList(podLabels, exceptionList)
}

// labelInExceptionList checks if the labels defined in azurepodidentityexception match label defined in pods
func labelInExceptionList(podLabels map[string]string, exceptionList []aadpodid.AzurePodIdentityException) bool {
	for _, exception := range exceptionList {
		for exceptionLabelKey, exceptionLabelValue := range exception.Spec.PodLabels {
			if val, ok := podLabels[exceptionLabelKey]; ok {
				if strings.EqualFold(val, exceptionLabelValue) {
					return true
				}
			}
		}
	}
	return false
}
