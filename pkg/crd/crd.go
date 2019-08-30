package crd

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/golang/glog"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	"github.com/Azure/aad-pod-identity/pkg/stats"

	log "github.com/Azure/aad-pod-identity/pkg/logger"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// Client represents all the watchers
type Client struct {
	rest                         *rest.RESTClient
	BindingInformer              cache.SharedInformer
	IDInformer                   cache.SharedInformer
	AssignedIDInformer           cache.SharedInformer
	PodIdentityExceptionInformer cache.SharedInformer
}

// ClientInt ...
type ClientInt interface {
	Start(exit <-chan struct{})
	SyncCache(exit <-chan struct{})
	RemoveAssignedIdentity(assignedIdentity *aadpodid.AzureAssignedIdentity) error
	CreateAssignedIdentity(assignedIdentity *aadpodid.AzureAssignedIdentity) error
	UpdateAzureAssignedIdentityStatus(assignedIdentity *aadpodid.AzureAssignedIdentity, status string) error
	ListBindings() (res *[]aadpodid.AzureIdentityBinding, err error)
	ListAssignedIDs() (res *[]aadpodid.AzureAssignedIdentity, err error)
	ListIds() (res *[]aadpodid.AzureIdentity, err error)
	ListPodIds(podns, podname string) (map[string][]aadpodid.AzureIdentity, error)
	ListPodIdentityExceptions(ns string) (res *[]aadpodid.AzurePodIdentityException, err error)
}

// NewCRDClientLite ...
func NewCRDClientLite(config *rest.Config) (crdClient *Client, err error) {
	restClient, err := newRestClient(config)
	if err != nil {
		glog.Error(err)
		return nil, err
	}

	assignedIDListWatch := newAssignedIDListWatch(restClient)
	assignedIDListInformer, err := newAssignedIDInformer(assignedIDListWatch)
	if err != nil {
		glog.Error(err)
		return nil, err
	}
	podIdentityExceptionListWatch := newPodIdentityExceptionListWatch(restClient)
	podIdentityExceptionInformer, err := newPodIdentityExceptionInformer(podIdentityExceptionListWatch)
	if err != nil {
		glog.Error(err)
		return nil, err
	}

	return &Client{
		AssignedIDInformer:           assignedIDListInformer,
		PodIdentityExceptionInformer: podIdentityExceptionInformer,
		rest:                         restClient,
	}, nil
}

// NewCRDClient returns a new crd client and error if any
func NewCRDClient(config *rest.Config, eventCh chan aadpodid.EventType) (crdClient *Client, err error) {
	restClient, err := newRestClient(config)
	if err != nil {
		glog.Error(err)
		return nil, err
	}

	bindingListWatch := newBindingListWatch(restClient)
	bindingInformer, err := newBindingInformer(restClient, eventCh, bindingListWatch)
	if err != nil {
		glog.Error(err)
		return nil, err
	}

	idListWatch := newIDListWatch(restClient)
	idInformer, err := newIDInformer(restClient, eventCh, idListWatch)
	if err != nil {
		glog.Error(err)
		return nil, err
	}

	assignedIDListWatch := newAssignedIDListWatch(restClient)
	assignedIDListInformer, err := newAssignedIDInformer(assignedIDListWatch)
	if err != nil {
		glog.Error(err)
		return nil, err
	}

	return &Client{
		rest:               restClient,
		BindingInformer:    bindingInformer,
		IDInformer:         idInformer,
		AssignedIDInformer: assignedIDListInformer,
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
		&aadpodid.AzureAssignedIdentityList{},
		&aadpodid.AzurePodIdentityException{},
		&aadpodid.AzurePodIdentityExceptionList{},
	)
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

func newBindingInformer(r *rest.RESTClient, eventCh chan aadpodid.EventType, lw *cache.ListWatch) (cache.SharedInformer, error) {
	azBindingInformer := cache.NewSharedInformer(
		lw,
		&aadpodid.AzureIdentityBinding{},
		time.Minute*10)
	if azBindingInformer == nil {
		return nil, fmt.Errorf("Could not create watcher for %s", aadpodid.AzureIDBindingResource)
	}
	azBindingInformer.AddEventHandler(
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
	return azBindingInformer, nil
}

func newIDListWatch(r *rest.RESTClient) *cache.ListWatch {
	return cache.NewListWatchFromClient(r, aadpodid.AzureIDResource, v1.NamespaceAll, fields.Everything())
}

func newIDInformer(r *rest.RESTClient, eventCh chan aadpodid.EventType, lw *cache.ListWatch) (cache.SharedInformer, error) {
	azIDInformer := cache.NewSharedInformer(
		lw,
		&aadpodid.AzureIdentity{},
		time.Minute*10)
	if azIDInformer == nil {
		return nil, fmt.Errorf("Could not create Identity watcher for %s", aadpodid.AzureIDResource)
	}
	azIDInformer.AddEventHandler(
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
	return azIDInformer, nil
}

func newAssignedIDListWatch(r *rest.RESTClient) *cache.ListWatch {
	return cache.NewListWatchFromClient(r, aadpodid.AzureAssignedIDResource, v1.NamespaceAll, fields.Everything())
}

func newAssignedIDInformer(lw *cache.ListWatch) (cache.SharedInformer, error) {
	azAssignedIDInformer := cache.NewSharedInformer(lw, &aadpodid.AzureAssignedIdentity{}, time.Minute*10)
	if azAssignedIDInformer == nil {
		return nil, fmt.Errorf("could not create %s informer", aadpodid.AzureAssignedIDResource)
	}

	return azAssignedIDInformer, nil
}

func newPodIdentityExceptionListWatch(r *rest.RESTClient) *cache.ListWatch {
	optionsModifier := func(options *v1.ListOptions) {}
	return cache.NewFilteredListWatchFromClient(
		r,
		aadpodid.AzureIdentityExceptionResource,
		v1.NamespaceAll,
		optionsModifier,
	)
}

func newPodIdentityExceptionInformer(lw *cache.ListWatch) (cache.SharedInformer, error) {
	azPodIDExceptionInformer := cache.NewSharedInformer(lw, &aadpodid.AzurePodIdentityException{}, time.Minute*10)
	if azPodIDExceptionInformer == nil {
		return nil, fmt.Errorf("could not create %s informer", aadpodid.AzureIdentityExceptionResource)
	}
	return azPodIDExceptionInformer, nil
}

// StartLite to be used only case of lite client
func (c *Client) StartLite(exit <-chan struct{}, log log.Logger) {
	go c.AssignedIDInformer.Run(exit)
	go c.PodIdentityExceptionInformer.Run(exit)
	c.SyncCache(exit)
	log.Info("CRD lite informers started ")
}

// Start ...
func (c *Client) Start(exit <-chan struct{}) {
	go c.BindingInformer.Run(exit)
	go c.IDInformer.Run(exit)
	go c.AssignedIDInformer.Run(exit)
	c.SyncCache(exit)
	glog.Info("CRD informers started")
}

// SyncCache synchronizes cache
func (c *Client) SyncCache(exit <-chan struct{}) {
	if !cache.WaitForCacheSync(exit) {
		panic("Cache could not be synchronized")
	}
}

// RemoveAssignedIdentity removes the assigned identity
func (c *Client) RemoveAssignedIdentity(assignedIdentity *aadpodid.AzureAssignedIdentity) error {
	glog.V(6).Infof("Deletion of assigned id named: %s", assignedIdentity.Name)
	begin := time.Now()
	err := c.rest.Delete().Namespace(assignedIdentity.Namespace).Resource("azureassignedidentities").Name(assignedIdentity.Name).Do().Error()
	stats.Update(stats.AssignedIDDel, time.Since(begin))
	return err
}

// CreateAssignedIdentity creates new assigned identity
func (c *Client) CreateAssignedIdentity(assignedIdentity *aadpodid.AzureAssignedIdentity) error {
	glog.Infof("Got assigned id %s to assign", assignedIdentity.Name)
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

// ListBindings returns a list of azureidentitybindings
func (c *Client) ListBindings() (res *[]aadpodid.AzureIdentityBinding, err error) {
	begin := time.Now()

	var resList []aadpodid.AzureIdentityBinding

	list := c.BindingInformer.GetStore().List()
	for _, binding := range list {
		o, ok := binding.(*aadpodid.AzureIdentityBinding)
		if !ok {
			err := fmt.Errorf("could not cast %T to %s", binding, aadpodid.AzureIDBindingResource)
			glog.Error(err)
			return nil, err
		}
		// Note: List items returned from cache have empty Kind and API version..
		// Work around this issue since we need that for event recording to work.
		o.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   aadpodid.CRDGroup,
			Version: aadpodid.CRDVersion,
			Kind:    reflect.TypeOf(*o).String()})
		resList = append(resList, *o)
		glog.V(6).Infof("Appending binding: %s/%s to list.", o.Namespace, o.Name)
	}

	stats.Update(stats.BindingList, time.Since(begin))
	return &resList, nil
}

// ListAssignedIDs returns a list of azureassignedidentities
func (c *Client) ListAssignedIDs() (res *[]aadpodid.AzureAssignedIdentity, err error) {
	begin := time.Now()

	var resList []aadpodid.AzureAssignedIdentity

	list := c.AssignedIDInformer.GetStore().List()
	for _, assignedID := range list {
		o, ok := assignedID.(*aadpodid.AzureAssignedIdentity)
		if !ok {
			err := fmt.Errorf("could not cast %T to %s", assignedID, aadpodid.AzureAssignedIDResource)
			glog.Error(err)
			return nil, err
		}
		// Note: List items returned from cache have empty Kind and API version..
		// Work around this issue since we need that for event recording to work.
		o.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   aadpodid.CRDGroup,
			Version: aadpodid.CRDVersion,
			Kind:    reflect.TypeOf(*o).String()})
		resList = append(resList, *o)
		glog.V(6).Infof("Appending Assigned ID: %s/%s to list.", o.Namespace, o.Name)
	}

	stats.Update(stats.AssignedIDList, time.Since(begin))
	return &resList, nil
}

// ListIds returns a list of azureidentities
func (c *Client) ListIds() (res *[]aadpodid.AzureIdentity, err error) {
	begin := time.Now()

	var resList []aadpodid.AzureIdentity

	list := c.IDInformer.GetStore().List()
	for _, id := range list {
		o, ok := id.(*aadpodid.AzureIdentity)
		if !ok {
			err := fmt.Errorf("could not cast %T to %s", id, aadpodid.AzureIDResource)
			glog.Error(err)
			return nil, err
		}
		// Note: List items returned from cache have empty Kind and API version..
		// Work around this issue since we need that for event recording to work.
		o.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   aadpodid.CRDGroup,
			Version: aadpodid.CRDVersion,
			Kind:    reflect.TypeOf(*o).String()})
		resList = append(resList, *o)
		glog.V(6).Infof("Appending Identity: %s/%s to list.", o.Namespace, o.Name)
	}

	stats.Update(stats.IDList, time.Since(begin))
	return &resList, nil
}

// ListPodIdentityExceptions returns list of azurepodidentityexceptions
func (c *Client) ListPodIdentityExceptions(ns string) (res *[]aadpodid.AzurePodIdentityException, err error) {
	begin := time.Now()

	var resList []aadpodid.AzurePodIdentityException

	list := c.PodIdentityExceptionInformer.GetStore().List()
	for _, binding := range list {
		o, ok := binding.(*aadpodid.AzurePodIdentityException)
		if !ok {
			err := fmt.Errorf("could not cast %T to %s", binding, aadpodid.AzureIdentityExceptionResource)
			glog.Error(err)
			return nil, err
		}
		if o.Namespace == ns {
			// Note: List items returned from cache have empty Kind and API version..
			// Work around this issue since we need that for event recording to work.
			o.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   aadpodid.CRDGroup,
				Version: aadpodid.CRDVersion,
				Kind:    reflect.TypeOf(*o).String()})
			resList = append(resList, *o)
			glog.V(6).Infof("Appending exception: %s/%s to list.", o.Namespace, o.Name)
		}
	}

	stats.Update(stats.ExceptionList, time.Since(begin))
	return &resList, nil
}

// ListPodIds - given a pod with pod name space
// returns a map with list of azure identities in each state
func (c *Client) ListPodIds(podns, podname string) (map[string][]aadpodid.AzureIdentity, error) {
	list, err := c.ListAssignedIDs()
	if err != nil {
		return nil, err
	}

	idStateMap := make(map[string][]aadpodid.AzureIdentity)
	for _, v := range *list {
		if v.Spec.Pod == podname && v.Spec.PodNamespace == podns {
			idStateMap[v.Status.Status] = append(idStateMap[v.Status.Status], *v.Spec.AzureIdentityRef)
		}
	}
	return idStateMap, nil
}

type patchStatusOps struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}

// UpdateAzureAssignedIdentityStatus updates the status field in AzureAssignedIdentity to indicate current status
func (c *Client) UpdateAzureAssignedIdentityStatus(assignedIdentity *aadpodid.AzureAssignedIdentity, status string) error {
	glog.Infof("Updating assigned identity %s/%s status to %s", assignedIdentity.Namespace, assignedIdentity.Name, status)

	ops := make([]patchStatusOps, 1)
	ops[0].Op = "replace"
	ops[0].Path = "/Status/status"
	ops[0].Value = status

	patchBytes, err := json.Marshal(ops)
	if err != nil {
		return err
	}

	err = c.rest.
		Patch(types.JSONPatchType).
		Namespace(assignedIdentity.Namespace).
		Resource("azureassignedidentities").
		Name(assignedIdentity.Name).
		Body(patchBytes).
		Do().
		Error()

	return err
}
