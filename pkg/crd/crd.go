package crd

import (
	"fmt"
	"time"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"

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
	rest                 *rest.RESTClient
	CrdWatcher           cache.Controller
	AzureIdentityWatcher cache.Controller
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
	go c.CrdWatcher.Run(exit)
	go c.AzureIdentityWatcher.Run(exit)
}

func (c *Client) CreateCRDWatcher(eventCh chan aadpodid.EventType) (err error) {
	_, crdWatcher := cache.NewInformer(
		cache.NewListWatchFromClient(c.rest, aadpodid.AzureIDBindingResource, "default", fields.Everything()),
		&aadpodid.AzureIdentityBinding{},
		time.Minute*10,
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
	if crdWatcher == nil {
		return fmt.Errorf("Could not create watcher for %s", aadpodid.AzureIDBindingResource)
	}
	_, azIdWatcher := cache.NewInformer(
		cache.NewListWatchFromClient(c.rest, aadpodid.AzureIDResource, "default", fields.Everything()),
		&aadpodid.AzureIdentity{},
		time.Minute*10,
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
	if azIdWatcher == nil {
		return fmt.Errorf("Could not create Identity watcher for %s", aadpodid.AzureIDResource)
	}
	c.AzureIdentityWatcher = azIdWatcher
	c.CrdWatcher = crdWatcher
	return nil
}

func (c *Client) RemoveAssignedIdentity(name string) error {
	glog.V(6).Infof("Deletion of id named: %s", name)
	return c.rest.Delete().Namespace("default").Resource("azureassignedidentities").Name(name).Do().Error()
}

func (c *Client) CreateAssignIdentity(name string, binding *aadpodid.AzureIdentityBinding, id *aadpodid.AzureIdentity, podName string, podNameSpace string, nodeName string) error {
	glog.Infof("Got id %s to assign", id.Name)
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

	//TODO: Update the status of the assign identity to indicate that the node assignment got done.
	return nil
}

func (c *Client) ListBindings() (res *[]aadpodid.AzureIdentityBinding, err error) {
	//Update the cache of the
	var ret aadpodid.AzureIdentityBindingList
	err = c.rest.Get().Namespace("default").Resource("azureidentitybindings").Do().Into(&ret)
	if err != nil {
		glog.Error(err)
		return nil, err
	}
	//glog.Infof("%+v", ret)
	return &ret.Items, nil
}

func (c *Client) ListAssignedIDs() (res *[]aadpodid.AzureAssignedIdentity, err error) {
	var ret aadpodid.AzureAssignedIdentityList
	err = c.rest.Get().Namespace("default").Resource("azureassignedidentities").Do().Into(&ret)
	if err != nil {
		glog.Error(err)
		return nil, err
	}
	return &ret.Items, nil
}

func (c *Client) ListIds() (res *[]aadpodid.AzureIdentity, err error) {
	var ret aadpodid.AzureIdentityList
	err = c.rest.Get().Namespace("default").Resource("azureidentities").Do().Into(&ret)
	if err != nil {
		glog.Error(err)
		return nil, err
	}
	return &ret.Items, nil
}

//GetUserAssignedIdentities - given a pod with pod name space
func (c *Client) GetUserAssignedIdentities(podns, podname string) (*[]aadpodid.AzureAssignedIdentity, error) {
	var azAssignedIDList aadpodid.AzureAssignedIdentityList
	err := c.rest.Get().Namespace("default").Resource("azureassignedidentities").Do().Into(&azAssignedIDList)
	if err != nil {
		glog.Error(err)
		return nil, err
	}

	return &azAssignedIDList.Items, nil
}
