package crd

import (
	"fmt"
	"time"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	"github.com/Azure/aad-pod-identity/pkg/stats"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type Client struct {
	rest                        *rest.RESTClient
	AzureIdentityBindingWatcher cache.SharedInformer
	AzureIdentityWatcher        cache.SharedInformer
}

func CreateRestClient(config *rest.Config) (r *rest.RESTClient, err error) {
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

func NewCRDClient(config *rest.Config) (crdClient *Client, err error) {
	restClient, err := CreateRestClient(config)
	if err != nil {
		glog.Error(err)
		return nil, err
	}
	crdClient = &Client{
		rest: restClient,
	}
	return crdClient, nil
}

func (c *Client) Start(exit <-chan struct{}) {
	go c.AzureIdentityBindingWatcher.Run(exit)
	go c.AzureIdentityWatcher.Run(exit)
}

func (c *Client) SyncCache(exit <-chan struct{}) {
	if !cache.WaitForCacheSync(exit) {
		panic("Cache could not be synchronized")
	}
}

func (c *Client) CreateCrdWatchers(eventCh chan aadpodid.EventType) (err error) {
	azBindingWatcher := cache.NewSharedInformer(
		cache.NewListWatchFromClient(c.rest, aadpodid.AzureIDBindingResource, "default", fields.Everything()),
		&aadpodid.AzureIdentityBinding{},
		time.Minute*10)
	if azBindingWatcher == nil {
		return fmt.Errorf("Could not create watcher for %s", aadpodid.AzureIDBindingResource)
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

	azIdWatcher := cache.NewSharedInformer(
		cache.NewListWatchFromClient(c.rest, aadpodid.AzureIDResource, "default", fields.Everything()),
		&aadpodid.AzureIdentity{},
		time.Minute*10)
	if azIdWatcher == nil {
		return fmt.Errorf("Could not create Identity watcher for %s", aadpodid.AzureIDResource)
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

	c.AzureIdentityWatcher = azIdWatcher
	c.AzureIdentityBindingWatcher = azBindingWatcher
	return nil
}

func (c *Client) RemoveAssignedIdentity(name string) error {
	glog.V(6).Infof("Deletion of id named: %s", name)
	begin := time.Now()
	err := c.rest.Delete().Namespace("default").Resource("azureassignedidentities").Name(name).Do().Error()
	stats.Update(stats.AssignedIDDel, time.Since(begin))
	return err
}

func (c *Client) CreateAssignIdentity(name string, binding *aadpodid.AzureIdentityBinding, id *aadpodid.AzureIdentity, podName string, podNameSpace string, nodeName string) error {
	glog.Infof("Got id %s to assign", id.Name)
	begin := time.Now()
	// Create a new AzureAssignedIdentity which maps the relationship between
	// id and pod
	assignedID := &aadpodid.AzureAssignedIdentity{
		ObjectMeta: v1.ObjectMeta{
			Name: name,
		},
		Spec: aadpodid.AzureAssignedIdentitySpec{
			AzureIdentityRef: id,
			AzureBindingRef:  binding,
			Pod:              podName,
			PodNamespace:     podNameSpace,
			NodeName:         nodeName,
		},
		Status: aadpodid.AzureAssignedIdentityStatus{
			AvailableReplicas: 1,
		},
	}

	glog.Infof("Creating assigned Id: %s", assignedID.Name)
	var res aadpodid.AzureAssignedIdentity
	// TODO: Ensure that the status reflects the corresponding
	err := c.rest.Post().Namespace("default").Resource("azureassignedidentities").Body(assignedID).Do().Into(&res)
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
	var ret aadpodid.AzureIdentityBindingList
	err = c.rest.Get().Namespace("default").Resource("azureidentitybindings").Do().Into(&ret)
	if err != nil {
		glog.Error(err)
		return nil, err
	}
	stats.Update(stats.BindingList, time.Since(begin))
	return &ret.Items, nil
}

func (c *Client) ListAssignedIDs() (res *[]aadpodid.AzureAssignedIdentity, err error) {
	begin := time.Now()
	var ret aadpodid.AzureAssignedIdentityList
	err = c.rest.Get().Namespace("default").Resource("azureassignedidentities").Do().Into(&ret)
	if err != nil {
		glog.Error(err)
		return nil, err
	}
	stats.Update(stats.AssignedIDList, time.Since(begin))
	return &ret.Items, nil
}

func (c *Client) ListIds() (res *[]aadpodid.AzureIdentity, err error) {
	begin := time.Now()
	var ret aadpodid.AzureIdentityList
	err = c.rest.Get().Namespace("default").Resource("azureidentities").Do().Into(&ret)
	if err != nil {
		glog.Error(err)
		return nil, err
	}
	stats.Update(stats.IDList, time.Since(begin))
	return &ret.Items, nil
}

//ListPodIds - given a pod with pod name space
func (c *Client) ListPodIds(podns, podname string) (*[]aadpodid.AzureIdentity, error) {
	var azAssignedIDList aadpodid.AzureAssignedIdentityList
	var matchedIds []aadpodid.AzureIdentity
	err := c.rest.Get().Namespace("default").Resource("azureassignedidentities").Do().Into(&azAssignedIDList)
	if err != nil {
		glog.Error(err)
		return nil, err
	}
	for _, v := range azAssignedIDList.Items {
		if v.Spec.Pod == podname && v.Spec.PodNamespace == podns {
			matchedIds = append(matchedIds, *v.Spec.AzureIdentityRef)
		}
	}

	return &matchedIds, nil
}
