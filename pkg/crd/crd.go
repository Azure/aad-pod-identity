package crd

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity"
	aadpodv1 "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	"github.com/Azure/aad-pod-identity/pkg/metrics"
	"github.com/Azure/aad-pod-identity/pkg/stats"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers/internalinterfaces"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
)

// Client represents all the watchers
type Client struct {
	rest                         *rest.RESTClient
	BindingInformer              cache.SharedInformer
	IDInformer                   cache.SharedInformer
	AssignedIDInformer           cache.SharedInformer
	PodIdentityExceptionInformer cache.SharedInformer
	reporter                     *metrics.Reporter
}

// ClientInt ...
type ClientInt interface {
	Start(exit <-chan struct{})
	SyncCache(exit <-chan struct{}, initial bool, cacheSyncs ...cache.InformerSynced)
	RemoveAssignedIdentity(assignedIdentity *aadpodid.AzureAssignedIdentity) error
	CreateAssignedIdentity(assignedIdentity *aadpodid.AzureAssignedIdentity) error
	UpdateAzureAssignedIdentityStatus(assignedIdentity *aadpodid.AzureAssignedIdentity, status string) error
	ListBindings() (res *[]aadpodid.AzureIdentityBinding, err error)
	ListAssignedIDs() (res *[]aadpodid.AzureAssignedIdentity, err error)
	ListAssignedIDsInMap() (res map[string]aadpodid.AzureAssignedIdentity, err error)
	ListIds() (res *[]aadpodid.AzureIdentity, err error)
	ListPodIds(podns, podname string) (map[string][]aadpodid.AzureIdentity, error)
	ListPodIdentityExceptions(ns string) (res *[]aadpodid.AzurePodIdentityException, err error)
}

// NewCRDClientLite ...
func NewCRDClientLite(config *rest.Config, nodeName string, scale, isStandardMode bool) (crdClient *Client, err error) {
	restClient, err := newRestClient(config)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	var assignedIDListInformer, bindingListInformer, idListInformer cache.SharedInformer

	// assigned identity informer is required only for standard mode
	if isStandardMode {
		var assignedIDListWatch *cache.ListWatch
		if scale {
			assignedIDListWatch = newAssignedIDNodeListWatch(restClient, nodeName)
		} else {
			assignedIDListWatch = newAssignedIDListWatch(restClient)
		}

		assignedIDListInformer, err = newAssignedIDInformer(assignedIDListWatch)
		if err != nil {
			klog.Error(err)
			return nil, err
		}
	} else {
		// creating binding and identity list informers for non standard mode
		if bindingListInformer, err = newBindingInformerLite(newBindingListWatch(restClient)); err != nil {
			return nil, err
		}
		if idListInformer, err = newIDInformerLite(newIDListWatch(restClient)); err != nil {
			return nil, err
		}
	}
	podIdentityExceptionListWatch := newPodIdentityExceptionListWatch(restClient)
	podIdentityExceptionInformer, err := newPodIdentityExceptionInformer(podIdentityExceptionListWatch)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	reporter, err := metrics.NewReporter()
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	return &Client{
		AssignedIDInformer:           assignedIDListInformer,
		PodIdentityExceptionInformer: podIdentityExceptionInformer,
		BindingInformer:              bindingListInformer,
		IDInformer:                   idListInformer,
		rest:                         restClient,
		reporter:                     reporter,
	}, nil
}

// NewCRDClient returns a new crd client and error if any
func NewCRDClient(config *rest.Config, eventCh chan aadpodid.EventType) (crdClient *Client, err error) {
	restClient, err := newRestClient(config)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	bindingListWatch := newBindingListWatch(restClient)
	bindingInformer, err := newBindingInformer(restClient, eventCh, bindingListWatch)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	idListWatch := newIDListWatch(restClient)
	idInformer, err := newIDInformer(restClient, eventCh, idListWatch)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	assignedIDListWatch := newAssignedIDListWatch(restClient)
	assignedIDListInformer, err := newAssignedIDInformer(assignedIDListWatch)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	reporter, err := metrics.NewReporter()
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	return &Client{
		rest:               restClient,
		BindingInformer:    bindingInformer,
		IDInformer:         idInformer,
		AssignedIDInformer: assignedIDListInformer,
		reporter:           reporter,
	}, nil
}

func newRestClient(config *rest.Config) (r *rest.RESTClient, err error) {
	crdconfig := *config
	crdconfig.GroupVersion = &schema.GroupVersion{Group: aadpodv1.CRDGroup, Version: aadpodv1.CRDVersion}
	crdconfig.APIPath = "/apis"
	crdconfig.ContentType = runtime.ContentTypeJSON
	s := runtime.NewScheme()
	s.AddKnownTypes(*crdconfig.GroupVersion,
		&aadpodv1.AzureIdentity{},
		&aadpodv1.AzureIdentityList{},
		&aadpodv1.AzureIdentityBinding{},
		&aadpodv1.AzureIdentityBindingList{},
		&aadpodv1.AzureAssignedIdentity{},
		&aadpodv1.AzureAssignedIdentityList{},
		&aadpodv1.AzurePodIdentityException{},
		&aadpodv1.AzurePodIdentityExceptionList{},
	)
	crdconfig.NegotiatedSerializer = serializer.DirectCodecFactory{
		CodecFactory: serializer.NewCodecFactory(s)}

	//Client interacting with our CRDs
	restClient, err := rest.RESTClientFor(&crdconfig)
	if err != nil {
		return nil, err
	}
	return restClient, nil
}

func newBindingListWatch(r *rest.RESTClient) *cache.ListWatch {
	return cache.NewListWatchFromClient(r, aadpodv1.AzureIDBindingResource, v1.NamespaceAll, fields.Everything())
}

func newBindingInformer(r *rest.RESTClient, eventCh chan aadpodid.EventType, lw *cache.ListWatch) (cache.SharedInformer, error) {
	azBindingInformer := cache.NewSharedInformer(
		lw,
		&aadpodv1.AzureIdentityBinding{},
		time.Minute*10)
	if azBindingInformer == nil {
		return nil, fmt.Errorf("Could not create watcher for %s", aadpodv1.AzureIDBindingResource)
	}
	azBindingInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				klog.V(6).Infof("Binding created")
				eventCh <- aadpodid.BindingCreated
			},
			DeleteFunc: func(obj interface{}) {
				klog.V(6).Infof("Binding deleted")
				eventCh <- aadpodid.BindingDeleted
			},
			UpdateFunc: func(OldObj, newObj interface{}) {
				klog.V(6).Infof("Binding updated")
				eventCh <- aadpodid.BindingUpdated
			},
		},
	)
	return azBindingInformer, nil
}

func newIDListWatch(r *rest.RESTClient) *cache.ListWatch {
	return cache.NewListWatchFromClient(r, aadpodv1.AzureIDResource, v1.NamespaceAll, fields.Everything())
}

func newIDInformer(r *rest.RESTClient, eventCh chan aadpodid.EventType, lw *cache.ListWatch) (cache.SharedInformer, error) {
	azIDInformer := cache.NewSharedInformer(
		lw,
		&aadpodv1.AzureIdentity{},
		time.Minute*10)
	if azIDInformer == nil {
		return nil, fmt.Errorf("Could not create Identity watcher for %s", aadpodv1.AzureIDResource)
	}
	azIDInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				klog.V(6).Infof("Identity created")
				eventCh <- aadpodid.IdentityCreated
			},
			DeleteFunc: func(obj interface{}) {
				klog.V(6).Infof("Identity deleted")
				eventCh <- aadpodid.IdentityDeleted
			},
			UpdateFunc: func(OldObj, newObj interface{}) {
				klog.V(6).Infof("Identity updated")
				eventCh <- aadpodid.IdentityUpdated
			},
		},
	)
	return azIDInformer, nil
}

// NodeNameFilter - CRDs do not yet support field selectors. Instead of that we
// apply labels with node name and then later use the NodeNameFilter to tweak
// options to filter using nodename label.
func NodeNameFilter(nodeName string) internalinterfaces.TweakListOptionsFunc {
	return func(l *v1.ListOptions) {
		if l == nil {
			l = &v1.ListOptions{}
		}
		l.LabelSelector = l.LabelSelector + "nodename=" + nodeName
		return
	}
}

func newAssignedIDNodeListWatch(r *rest.RESTClient, nodeName string) *cache.ListWatch {
	return cache.NewFilteredListWatchFromClient(r, aadpodv1.AzureAssignedIDResource, v1.NamespaceAll, NodeNameFilter(nodeName))
}

func newAssignedIDListWatch(r *rest.RESTClient) *cache.ListWatch {
	return cache.NewListWatchFromClient(r, aadpodv1.AzureAssignedIDResource, v1.NamespaceAll, fields.Everything())
}

func newAssignedIDInformer(lw *cache.ListWatch) (cache.SharedInformer, error) {
	azAssignedIDInformer := cache.NewSharedInformer(lw, &aadpodv1.AzureAssignedIdentity{}, time.Minute*10)
	if azAssignedIDInformer == nil {
		return nil, fmt.Errorf("could not create %s informer", aadpodv1.AzureAssignedIDResource)
	}
	return azAssignedIDInformer, nil
}

func newBindingInformerLite(lw *cache.ListWatch) (cache.SharedInformer, error) {
	azBindingInformer := cache.NewSharedInformer(lw, &aadpodv1.AzureIdentityBinding{}, time.Minute*10)
	if azBindingInformer == nil {
		return nil, fmt.Errorf("could not create %s informer", aadpodv1.AzureIDBindingResource)
	}
	return azBindingInformer, nil
}

func newIDInformerLite(lw *cache.ListWatch) (cache.SharedInformer, error) {
	azIDInformer := cache.NewSharedInformer(lw, &aadpodv1.AzureIdentity{}, time.Minute*10)
	if azIDInformer == nil {
		return nil, fmt.Errorf("could not create %s informer", aadpodv1.AzureIDResource)
	}
	return azIDInformer, nil
}

func newPodIdentityExceptionListWatch(r *rest.RESTClient) *cache.ListWatch {
	optionsModifier := func(options *v1.ListOptions) {}
	return cache.NewFilteredListWatchFromClient(
		r,
		aadpodv1.AzureIdentityExceptionResource,
		v1.NamespaceAll,
		optionsModifier,
	)
}

func newPodIdentityExceptionInformer(lw *cache.ListWatch) (cache.SharedInformer, error) {
	azPodIDExceptionInformer := cache.NewSharedInformer(lw, &aadpodv1.AzurePodIdentityException{}, time.Minute*10)
	if azPodIDExceptionInformer == nil {
		return nil, fmt.Errorf("could not create %s informer", aadpodv1.AzureIdentityExceptionResource)
	}
	return azPodIDExceptionInformer, nil
}

// StartLite to be used only case of lite client
func (c *Client) StartLite(exit <-chan struct{}) {
	var cacheHasSynced []cache.InformerSynced

	if c.AssignedIDInformer != nil {
		go c.AssignedIDInformer.Run(exit)
		cacheHasSynced = append(cacheHasSynced, c.AssignedIDInformer.HasSynced)
	}
	if c.BindingInformer != nil {
		go c.BindingInformer.Run(exit)
		cacheHasSynced = append(cacheHasSynced, c.BindingInformer.HasSynced)
	}
	if c.IDInformer != nil {
		go c.IDInformer.Run(exit)
		cacheHasSynced = append(cacheHasSynced, c.IDInformer.HasSynced)
	}
	if c.PodIdentityExceptionInformer != nil {
		go c.PodIdentityExceptionInformer.Run(exit)
		cacheHasSynced = append(cacheHasSynced, c.PodIdentityExceptionInformer.HasSynced)
	}
	c.SyncCache(exit, true, cacheHasSynced...)
	klog.Info("CRD lite informers started ")
}

// Start ...
func (c *Client) Start(exit <-chan struct{}) {
	go c.BindingInformer.Run(exit)
	go c.IDInformer.Run(exit)
	go c.AssignedIDInformer.Run(exit)
	c.SyncCache(exit, true, c.BindingInformer.HasSynced, c.IDInformer.HasSynced, c.AssignedIDInformer.HasSynced)
	klog.Info("CRD informers started")
}

// SyncCache synchronizes cache
func (c *Client) SyncCache(exit <-chan struct{}, initial bool, cacheSyncs ...cache.InformerSynced) {
	if !cache.WaitForCacheSync(exit, cacheSyncs...) {
		if !initial {
			klog.Errorf("Cache could not be synchronized")
			return
		}
		panic("Cache could not be synchronized")
	}
}

// RemoveAssignedIdentity removes the assigned identity
func (c *Client) RemoveAssignedIdentity(assignedIdentity *aadpodid.AzureAssignedIdentity) (err error) {
	klog.V(6).Infof("Deletion of assigned id named: %s", assignedIdentity.Name)
	begin := time.Now()

	defer func() {
		if err != nil {
			c.reporter.ReportKubernetesAPIOperationError(metrics.AssignedIdentityDeletionOperationName)
			return
		}
		c.reporter.Report(
			metrics.AssignedIdentityDeletionCountM.M(1),
			metrics.AssignedIdentityDeletionDurationM.M(metrics.SinceInSeconds(begin)))

	}()

	err = c.rest.Delete().Namespace(assignedIdentity.Namespace).Resource("azureassignedidentities").Name(assignedIdentity.Name).Do().Error()
	klog.V(5).Infof("Deletion %s took: %v", assignedIdentity.Name, time.Since(begin))
	stats.Update(stats.AssignedIDDel, time.Since(begin))
	return err
}

// CreateAssignedIdentity creates new assigned identity
func (c *Client) CreateAssignedIdentity(assignedIdentity *aadpodid.AzureAssignedIdentity) (err error) {
	klog.Infof("Got assigned id %s to assign", assignedIdentity.Name)
	begin := time.Now()

	defer func() {
		if err != nil {
			c.reporter.ReportKubernetesAPIOperationError(metrics.AssignedIdentityAdditionOperationName)
			return
		}
		c.reporter.Report(
			metrics.AssignedIdentityAdditionCountM.M(1),
			metrics.AssignedIdentityAdditionDurationM.M(metrics.SinceInSeconds(begin)))

	}()

	// Create a new AzureAssignedIdentity which maps the relationship between id and pod
	var res aadpodv1.AzureAssignedIdentity
	v1AssignedID := aadpodv1.ConvertInternalAssignedIdentityToV1AssignedIdentity(*assignedIdentity)
	// TODO: Ensure that the status reflects the corresponding
	err = c.rest.Post().Namespace(assignedIdentity.Namespace).Resource("azureassignedidentities").Body(&v1AssignedID).Do().Into(&res)
	if err != nil {
		klog.Error(err)
		return err
	}

	klog.V(5).Infof("Time take to create %s: %v", assignedIdentity.Name, time.Since(begin))
	stats.Update(stats.AssignedIDAdd, time.Since(begin))
	return nil
}

// ListBindings returns a list of azureidentitybindings
func (c *Client) ListBindings() (res *[]aadpodid.AzureIdentityBinding, err error) {
	begin := time.Now()

	var resList []aadpodid.AzureIdentityBinding

	list := c.BindingInformer.GetStore().List()
	for _, binding := range list {
		o, ok := binding.(*aadpodv1.AzureIdentityBinding)
		if !ok {
			err := fmt.Errorf("could not cast %T to %s", binding, aadpodv1.AzureIDBindingResource)
			klog.Error(err)
			return nil, err
		}
		// Note: List items returned from cache have empty Kind and API version..
		// Work around this issue since we need that for event recording to work.
		o.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   aadpodv1.CRDGroup,
			Version: aadpodv1.CRDVersion,
			Kind:    reflect.TypeOf(*o).String()})

		internalBinding := aadpodv1.ConvertV1BindingToInternalBinding(*o)

		resList = append(resList, internalBinding)
		klog.V(6).Infof("Appending binding: %s/%s to list.", o.Namespace, o.Name)
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
		o, ok := assignedID.(*aadpodv1.AzureAssignedIdentity)
		if !ok {
			err := fmt.Errorf("could not cast %T to %s", assignedID, aadpodv1.AzureAssignedIDResource)
			klog.Error(err)
			return nil, err
		}
		// Note: List items returned from cache have empty Kind and API version..
		// Work around this issue since we need that for event recording to work.
		o.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   aadpodv1.CRDGroup,
			Version: aadpodv1.CRDVersion,
			Kind:    reflect.TypeOf(*o).String()})
		out := aadpodv1.ConvertV1AssignedIdentityToInternalAssignedIdentity(*o)
		resList = append(resList, out)
		klog.V(6).Infof("Appending Assigned ID: %s/%s to list.", o.Namespace, o.Name)
	}

	stats.Update(stats.AssignedIDList, time.Since(begin))
	return &resList, nil
}

// ListAssignedIDsInMap gets the list of current assigned ids, adds it to a map
// with assigned identity name as key and assigned identity as value.
func (c *Client) ListAssignedIDsInMap() (map[string]aadpodid.AzureAssignedIdentity, error) {
	begin := time.Now()

	result := make(map[string]aadpodid.AzureAssignedIdentity)
	list := c.AssignedIDInformer.GetStore().List()

	for _, assignedID := range list {

		o, ok := assignedID.(*aadpodv1.AzureAssignedIdentity)
		if !ok {
			err := fmt.Errorf("could not cast %T to %s", assignedID, aadpodv1.AzureAssignedIDResource)
			klog.Error(err)
			return nil, err
		}
		// Note: List items returned from cache have empty Kind and API version..
		// Work around this issue since we need that for event recording to work.
		o.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   aadpodv1.CRDGroup,
			Version: aadpodv1.CRDVersion,
			Kind:    reflect.TypeOf(*o).String()})

		out := aadpodv1.ConvertV1AssignedIdentityToInternalAssignedIdentity(*o)

		// assigned identities names are unique across namespaces as we use pod name-<id ns>-<id name>
		result[o.Name] = out
	}

	stats.Update(stats.AssignedIDList, time.Since(begin))
	return result, nil
}

// ListIds returns a list of azureidentities
func (c *Client) ListIds() (res *[]aadpodid.AzureIdentity, err error) {
	begin := time.Now()

	var resList []aadpodid.AzureIdentity

	list := c.IDInformer.GetStore().List()
	for _, id := range list {
		o, ok := id.(*aadpodv1.AzureIdentity)
		if !ok {
			err := fmt.Errorf("could not cast %T to %s", id, aadpodv1.AzureIDResource)
			klog.Error(err)
			return nil, err
		}
		// Note: List items returned from cache have empty Kind and API version..
		// Work around this issue since we need that for event recording to work.
		o.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   aadpodv1.CRDGroup,
			Version: aadpodv1.CRDVersion,
			Kind:    reflect.TypeOf(*o).String()})

		out := aadpodv1.ConvertV1IdentityToInternalIdentity(*o)

		resList = append(resList, out)
		klog.V(6).Infof("Appending Identity: %s/%s to list.", o.Namespace, o.Name)
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
		o, ok := binding.(*aadpodv1.AzurePodIdentityException)
		if !ok {
			err := fmt.Errorf("could not cast %T to %s", binding, aadpodid.AzureIdentityExceptionResource)
			klog.Error(err)
			return nil, err
		}
		if o.Namespace == ns {
			// Note: List items returned from cache have empty Kind and API version..
			// Work around this issue since we need that for event recording to work.
			o.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   aadpodv1.CRDGroup,
				Version: aadpodv1.CRDVersion,
				Kind:    reflect.TypeOf(*o).String()})
			out := aadpodv1.ConvertV1PodIdentityExceptionToInternalPodIdentityException(*o)

			resList = append(resList, out)
			klog.V(6).Infof("Appending exception: %s/%s to list.", o.Namespace, o.Name)
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

// GetPodIDsWithBinding returns list of azure identity based on bindings
// that match pod label.
func (c *Client) GetPodIDsWithBinding(namespace string, labels map[string]string) ([]aadpodid.AzureIdentity, error) {
	// get all bindings
	bindings, err := c.ListBindings()
	if err != nil {
		return nil, err
	}
	if bindings == nil {
		return nil, fmt.Errorf("binding list is nil from cache")
	}
	matchingIds := make(map[string]bool)
	podLabel := labels[aadpodid.CRDLabelKey]

	for _, binding := range *bindings {
		// check if binding selector in pod labels
		if podLabel == binding.Spec.Selector && binding.Namespace == namespace {
			matchingIds[binding.Spec.AzureIdentity] = true
		}
	}
	// get the azure identity objects based on the list generated
	azIdentities, err := c.ListIds()
	if err != nil {
		return nil, err
	}
	if azIdentities == nil {
		return nil, fmt.Errorf("azure identities list is nil from cache")
	}
	var azIds []aadpodid.AzureIdentity
	for _, azIdentity := range *azIdentities {
		if _, exists := matchingIds[azIdentity.Name]; exists && azIdentity.Namespace == namespace {
			azIds = append(azIds, azIdentity)
		}
	}
	return azIds, nil
}

type patchStatusOps struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}

// UpdateAzureAssignedIdentityStatus updates the status field in AzureAssignedIdentity to indicate current status
func (c *Client) UpdateAzureAssignedIdentityStatus(assignedIdentity *aadpodid.AzureAssignedIdentity, status string) (err error) {
	klog.Infof("Updating assigned identity %s/%s status to %s", assignedIdentity.Namespace, assignedIdentity.Name, status)

	defer func() {
		if err != nil {
			c.reporter.ReportKubernetesAPIOperationError(metrics.UpdateAzureAssignedIdentityStatusOperationName)
		}
	}()

	ops := make([]patchStatusOps, 1)
	ops[0].Op = "replace"
	ops[0].Path = "/Status/status"
	ops[0].Value = status

	patchBytes, err := json.Marshal(ops)
	if err != nil {
		return err
	}

	begin := time.Now()
	err = c.rest.
		Patch(types.JSONPatchType).
		Namespace(assignedIdentity.Namespace).
		Resource("azureassignedidentities").
		Name(assignedIdentity.Name).
		Body(patchBytes).
		Do().
		Error()
	klog.V(5).Infof("Patch of %s took: %v", assignedIdentity.Name, time.Since(begin))
	return err
}
