package pod

import (
	"fmt"
	"reflect"
	"time"

	"github.com/Azure/aad-pod-identity/pkg/stats"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	"github.com/golang/glog"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type Client struct {
	PodWatcher informers.SharedInformerFactory
}

type ClientInt interface {
	GetPods() (pods []*corev1.Pod, err error)
	Start(exit <-chan struct{})
}

func NewPodClient(k8sClient *kubernetes.Clientset, eventCh chan aadpodid.EventType) (c ClientInt, e error) {
	podWatcher, err := newPodWatcher(k8sClient, eventCh)
	if err != nil {
		glog.Error(err)
		return nil, err
	}

	return &Client{
		PodWatcher: podWatcher,
	}, nil
}

func newPodWatcher(k8sClient *kubernetes.Clientset, eventCh chan aadpodid.EventType) (i informers.SharedInformerFactory, err error) {
	k8sInformers := informers.NewSharedInformerFactory(k8sClient, time.Second*30)
	if k8sInformers == nil {
		return nil, fmt.Errorf("k8s informers could not be created")
	}
	k8sInformers.Core().V1().Pods().Informer().AddEventHandler(
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
	return k8sInformers, nil
}

func (c *Client) syncCache(exit <-chan struct{}) {
	cacheSyncStarted := time.Now()
	cacheSynced := false
	glog.V(6).Infof("Wait for cache to sync")
	for !cacheSynced {
		mapSync := c.PodWatcher.WaitForCacheSync(exit)
		if len(mapSync) > 0 && mapSync[reflect.TypeOf(&corev1.Pod{})] == true {
			cacheSynced = true
		}
	}
	glog.Infof("Pod cache synchronized. Took %s", time.Since(cacheSyncStarted).String())
}

func (c *Client) Start(exit <-chan struct{}) {
	go c.PodWatcher.Start(exit)
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
	listPods, err := c.PodWatcher.Core().V1().Pods().Lister().List(crdSelector)
	//ClientSet.CoreV1().Pods("").List(v1.ListOptions{})
	if err != nil {
		return nil, err
	}
	stats.Put(stats.PodList, time.Since(begin))
	return listPods, nil
}
