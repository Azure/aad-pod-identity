package crd

import (
	"fmt"
	"time"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	"github.com/Azure/aad-pod-identity/pkg/stats"

	"github.com/golang/glog"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type Client struct {
	rest                *rest.RESTClient
	BindingListWatch    *cache.ListWatch
	BindingWatcher      cache.SharedInformer
	IDListWatch         *cache.ListWatch
	IdWatcher           cache.SharedInformer
	AssignedIDListWatch *cache.ListWatch
}

type ClientInt interface {
	Start(exit <-chan struct{})
	SyncCache(exit <-chan struct{})
	RemoveAssignedIdentity(assignedIdentity *aadpodid.AzureAssignedIdentity) error
	CreateAssignedIdentity(assignedIdentity *aadpodid.AzureAssignedIdentity) error
	ListBindings() (res *[]aadpodid.AzureIdentityBinding, err error)
	ListAssignedIDs() (res *[]aadpodid.AzureAssignedIdentity, err error)
	ListIds() (res *[]aadpodid.AzureIdentity, err error)
	ListPodIds(podns, podname string) (*[]aadpodid.AzureIdentity, error)
}

func NewCRDClientLite(config *rest.Config) (crdClient *Client, err error) {
	restClient, err := newRestClient(config)
	if err != nil {
		glog.Error(err)
		return nil, err
	}
	return &Client{
		rest: restClient,
	}, nil
}

func NewCRDClient(config *rest.Config, eventCh chan aadpodid.EventType) (crdClient *Client, err error) {
	restClient, err := newRestClient(config)
	if err != nil {
		glog.Error(err)
		return nil, err
	}

	bindingListWatch := newBindingListWatch(restClient)

	bindingWatcher, err := newBindingWatcher(restClient, eventCh, bindingListWatch)
	if err != nil {
		glog.Error(err)
		return nil, err
	}

	idListWatch := newIDListWatch(restClient)

	idWatcher, err := newIdWatcher(restClient, eventCh, idListWatch)
	if err != nil {
		glog.Error(err)
		return nil, err
	}

	assignedIDListWatch := newAssignedIDListWatch(restClient)

	return &Client{
		rest:                restClient,
		BindingListWatch:    bindingListWatch,
		BindingWatcher:      bindingWatcher,
		IdWatcher:           idWatcher,
		IDListWatch:         idListWatch,
		AssignedIDListWatch: assignedIDListWatch,
	}, nil
}

func newRestClient(config *rest.Config) (r *rest.RESTClient, err error) {
	crdconfig := *config
	crdconfig.GroupVersion = &schema.GroupVersion{Group: aadpodid.CRDGroup, Version: aadpodid.CRDVersion}
	crdconfig.APIPath = "/apis"
	crdconfig.ContentType = runtime.ContentTypeJSON
	s := runtime.NewScheme()
	s.AddKnownTypes(*crdconfig.GroupVersion,
		&aadpodid.AzureIdentity{},
		&aadpodid.AzureIdentityList{},
		&aadpodid.AzureIdentityBinding{},
		&aadpodid.AzureIdentityBindingList{},
		&aadpodid.AzureAssignedIdentity{},
		&aadpodid.AzureAssignedIdentityList{})
	crdconfig.NegotiatedSerializer = serializer.DirectCodecFactory{
		CodecFactory: serializer.NewCodecFactory(s)}

	//Client interacting with our CRDs
	restClient, err := rest.RESTClientFor(&crdconfig)
	if err != nil {
		glog.Error(err)
		return nil, err
	}
	return restClient, nil
}

func newBindingListWatch(r *rest.RESTClient) *cache.ListWatch {
	return cache.NewListWatchFromClient(r, aadpodid.AzureIDBindingResource, v1.NamespaceAll, fields.Everything())
}

func newBindingWatcher(r *rest.RESTClient, eventCh chan aadpodid.EventType, lw *cache.ListWatch) (cache.SharedInformer, error) {
	azBindingWatcher := cache.NewSharedInformer(
		lw,
		&aadpodid.AzureIdentityBinding{},
		time.Minute*10)
	if azBindingWatcher == nil {
		return nil, fmt.Errorf("Could not create watcher for %s", aadpodid.AzureIDBindingResource)
	}
	azBindingWatcher.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				glog.V(6).Infof("Binding created")
				eventCh <- aadpodid.BindingCreated
			},
			DeleteFunc: func(obj interface{}) {
				glog.V(6).Infof("Binding deleted")
				eventCh <- aadpodid.BindingDeleted
			},
			UpdateFunc: func(OldObj, newObj interface{}) {
				glog.V(6).Infof("Binding updated")
				eventCh <- aadpodid.BindingUpdated
			},
		},
	)
	return azBindingWatcher, nil
}

func newIDListWatch(r *rest.RESTClient) *cache.ListWatch {
	return cache.NewListWatchFromClient(r, aadpodid.AzureIDResource, v1.NamespaceAll, fields.Everything())
}

func newIdWatcher(r *rest.RESTClient, eventCh chan aadpodid.EventType, lw *cache.ListWatch) (cache.SharedInformer, error) {
	azIdWatcher := cache.NewSharedInformer(
		lw,
		&aadpodid.AzureIdentity{},
		time.Minute*10)
	if azIdWatcher == nil {
		return nil, fmt.Errorf("Could not create Identity watcher for %s", aadpodid.AzureIDResource)
	}
	azIdWatcher.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				glog.V(6).Infof("Identity created")
				eventCh <- aadpodid.IdentityCreated
			},
			DeleteFunc: func(obj interface{}) {
				glog.V(6).Infof("Identity deleted")
				eventCh <- aadpodid.IdentityDeleted
			},
			UpdateFunc: func(OldObj, newObj interface{}) {
				glog.V(6).Infof("Identity updated")
				eventCh <- aadpodid.IdentityUpdated
			},
		},
	)
	return azIdWatcher, nil
}

func newAssignedIDListWatch(r *rest.RESTClient) *cache.ListWatch {
	return cache.NewListWatchFromClient(r, aadpodid.AzureAssignedIDResource, v1.NamespaceAll, fields.Everything())
}

func (c *Client) Start(exit <-chan struct{}) {
	go c.BindingWatcher.Run(exit)
	go c.IdWatcher.Run(exit)
	glog.Info("CRD watchers started")
}

func (c *Client) SyncCache(exit <-chan struct{}) {
	if !cache.WaitForCacheSync(exit) {
		panic("Cache could not be synchronized")
	}
}

func (c *Client) RemoveAssignedIdentity(assignedIdentity *aadpodid.AzureAssignedIdentity) error {
	glog.V(6).Infof("Deletion of id named: %s", assignedIdentity.Name)
	begin := time.Now()
	err := c.rest.Delete().Namespace(assignedIdentity.Namespace).Resource("azureassignedidentities").Name(assignedIdentity.Name).Do().Error()
	stats.Update(stats.AssignedIDDel, time.Since(begin))
	return err
}

func (c *Client) CreateAssignedIdentity(assignedIdentity *aadpodid.AzureAssignedIdentity) error {
	glog.Infof("Got id %s to assign", assignedIdentity.Name)
	begin := time.Now()
	// Create a new AzureAssignedIdentity which maps the relationship between
	// id and pod
	glog.Infof("Creating assigned Id: %s", assignedIdentity.Name)
	var res aadpodid.AzureAssignedIdentity
	// TODO: Ensure that the status reflects the corresponding
	err := c.rest.Post().Namespace(assignedIdentity.Namespace).Resource("azureassignedidentities").Body(assignedIdentity).Do().Into(&res)
	if err != nil {
		glog.Error(err)
		return err
	}

	stats.Update(stats.AssignedIDAdd, time.Since(begin))
	//TODO: Update the status of the assign identity to indicate that the node assignment got done.
	return nil
}

func (c *Client) ListBindings() (res *[]aadpodid.AzureIdentityBinding, err error) {
	begin := time.Now()

	ret, err := c.BindingListWatch.List(v1.ListOptions{})
	if err != nil {
		glog.Error(err)
		return nil, err
	}
	stats.Update(stats.BindingList, time.Since(begin))
	return &ret.(*aadpodid.AzureIdentityBindingList).Items, nil
}

func (c *Client) ListAssignedIDs() (res *[]aadpodid.AzureAssignedIdentity, err error) {
	begin := time.Now()
	ret, err := c.AssignedIDListWatch.List(v1.ListOptions{})
	if err != nil {
		glog.Error(err)
		return nil, err
	}
	stats.Update(stats.AssignedIDList, time.Since(begin))
	return &ret.(*aadpodid.AzureAssignedIdentityList).Items, nil
}

func (c *Client) ListIds() (res *[]aadpodid.AzureIdentity, err error) {
	begin := time.Now()
	ret, err := c.IDListWatch.List(v1.ListOptions{})
	if err != nil {
		glog.Error(err)
		return nil, err
	}
	stats.Update(stats.IDList, time.Since(begin))
	return &ret.(*aadpodid.AzureIdentityList).Items, nil
}

//ListPodIds - given a pod with pod name space
func (c *Client) ListPodIds(podns, podname string) (*[]aadpodid.AzureIdentity, error) {
	azAssignedIDList, err := c.AssignedIDListWatch.List(v1.ListOptions{})
	if err != nil {
		glog.Error(err)
		return nil, err
	}

	var matchedIds []aadpodid.AzureIdentity
	for _, v := range azAssignedIDList.(*aadpodid.AzureAssignedIdentityList).Items {
		if v.Spec.Pod == podname && v.Spec.PodNamespace == podns {
			matchedIds = append(matchedIds, *v.Spec.AzureIdentityRef)
		}
	}

	return &matchedIds, nil
}
