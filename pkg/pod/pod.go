package pod

import (
	"time"

	"github.com/Azure/aad-pod-identity/pkg/stats"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity"
	"github.com/golang/glog"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	informersv1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/tools/cache"
)

type Client struct {
	PodWatcher informersv1.PodInformer
}

type ClientInt interface {
	GetPods(labelSelector *metav1.LabelSelector) (pods []*corev1.Pod, err error)
	Start(exit <-chan struct{})
}

func NewPodClient(i informers.SharedInformerFactory, eventCh chan aadpodid.EventType) (c ClientInt) {
	podInformer := i.Core().V1().Pods()
	addPodHandler(podInformer, eventCh)

	return &Client{
		PodWatcher: podInformer,
	}
}

func addPodHandler(i informersv1.PodInformer, eventCh chan aadpodid.EventType) {
	i.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				glog.V(6).Infof("Pod Created")
				eventCh <- aadpodid.PodCreated

			},
			DeleteFunc: func(obj interface{}) {
				glog.V(6).Infof("Pod Deleted")
				eventCh <- aadpodid.PodDeleted

			},
			UpdateFunc: func(OldObj, newObj interface{}) {
				// We are only interested in updates to pod if the node changes.
				// Having this check will ensure that mic sync loop does not do extra work
				// for every pod update.
				if (OldObj.(*v1.Pod)).Spec.NodeName != (newObj.(*v1.Pod)).Spec.NodeName {
					glog.V(6).Infof("Pod Updated")
					eventCh <- aadpodid.PodUpdated
				}
			},
		},
	)
}

func (c *Client) syncCache(exit <-chan struct{}) {
	cacheSyncStarted := time.Now()
	glog.V(6).Infof("Wait for cache to sync")
	if !cache.WaitForCacheSync(exit, c.PodWatcher.Informer().HasSynced) {
		glog.Error("Wait for pod cache sync failed")
		return
	}
	glog.Infof("Pod cache synchronized. Took %s", time.Since(cacheSyncStarted).String())
}

func (c *Client) Start(exit <-chan struct{}) {
	go c.PodWatcher.Informer().Run(exit)
	c.syncCache(exit)
	glog.Info("Pod watcher started !!")
}

func (c *Client) GetPods(labelSelector *metav1.LabelSelector) (pods []*corev1.Pod, err error) {
	begin := time.Now()

	// LabelSelector selects all pods if empty by design
	if (labelSelector.Size() == 0) {
		stats.Put(stats.PodList, time.Since(begin))
		return make([]*corev1.Pod, 0), nil
	}

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
}
