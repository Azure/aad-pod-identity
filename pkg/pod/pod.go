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
func NewPodClient(i informers.SharedInformerFactory, eventCh chan aadpodid.EventType) (c ClientInt) {
	podInformer := i.Core().V1().Pods()
	addPodHandler(podInformer, eventCh, nil)

	return &Client{
		PodWatcher: podInformer,
	}
}

func NewPodClientWithPodInfoCh(i informers.SharedInformerFactory, podInfoCh chan *v1.Pod) (c ClientInt) {
	podInformer := i.Core().V1().Pods()
	addPodHandler(podInformer, nil, podInfoCh)

	return &Client{
		PodWatcher: podInformer,
	}
}

func addPodHandler(i informersv1.PodInformer, eventCh chan aadpodid.EventType, podInfoCh chan *v1.Pod) {
	i.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				klog.V(6).Infof("Pod Created")
				if eventCh != nil {
					eventCh <- aadpodid.PodCreated
				}
			},
			DeleteFunc: func(obj interface{}) {
				klog.V(6).Infof("Pod Deleted")
				if eventCh != nil {
					eventCh <- aadpodid.PodDeleted
				}
			},
			UpdateFunc: func(OldObj, newObj interface{}) {
				oldPod := OldObj.(*v1.Pod)
				newPod := newObj.(*v1.Pod)

				// This is to handle windows nmi by observing ip change
				if podInfoCh != nil {
					if oldPod.Status.PodIP != newPod.Status.PodIP {
						klog.V(6).Infof("Pod IP Updated")
						klog.Infof("Old Pod IP: %s, Current Pod IP: %s", oldPod.Status.PodIP, newPod.Status.PodIP)
						if newPod.Status.PodIP == "" {
							podInfoCh <- oldPod
						} else {
							podInfoCh <- newPod
						}
					}
				}

				// We are interested in updates to pod if the node changes.
				// Having this check will ensure that mic sync loop does not do extra work
				// for every pod update.
				if oldPod.Spec.NodeName != newPod.Spec.NodeName {
					klog.V(6).Infof("Pod Updated")
					if eventCh != nil {
						eventCh <- aadpodid.PodUpdated
					}
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
	klog.Info("Pod watcher started!!")
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
