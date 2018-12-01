package pod

import (
	"time"

	"github.com/Azure/aad-pod-identity/pkg/stats"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	"github.com/golang/glog"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	informersv1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/tools/cache"
)

type Client struct {
	PodWatcher informersv1.PodInformer
}

type ClientInt interface {
	GetPods() (pods []*corev1.Pod, err error)
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
				glog.V(6).Infof("Pod Updated")
				eventCh <- aadpodid.PodUpdated
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

func (c *Client) GetPods() (pods []*corev1.Pod, err error) {
	begin := time.Now()
	crdReq, err := labels.NewRequirement(aadpodid.CRDLabelKey, selection.Exists, nil)
	if err != nil {
		glog.Error(err)
		return nil, err
	}
	crdSelector := labels.NewSelector().Add(*crdReq)
	listPods, err := c.PodWatcher.Lister().List(crdSelector)
	//ClientSet.CoreV1().Pods("").List(v1.ListOptions{})
	if err != nil {
		return nil, err
	}
	stats.Put(stats.PodList, time.Since(begin))
	return listPods, nil
}
