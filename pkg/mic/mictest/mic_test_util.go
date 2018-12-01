package mictest

import (
	"errors"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	cp "github.com/Azure/aad-pod-identity/pkg/cloudprovider/cloudprovidertest"
	crd "github.com/Azure/aad-pod-identity/pkg/crd/crdtest"
	pod "github.com/Azure/aad-pod-identity/pkg/pod/podtest"
	"github.com/golang/glog"

	"github.com/Azure/aad-pod-identity/pkg/mic"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type LastEvent struct {
	Type    string
	Reason  string
	Message string
}

type TestEventRecorder struct {
	lastEvent *LastEvent
}

func (c TestEventRecorder) Event(object runtime.Object, t string, r string, message string) {
	c.lastEvent.Type = t
	c.lastEvent.Reason = r
	c.lastEvent.Message = message
}

func (c TestEventRecorder) Validate(e *LastEvent) bool {
	t := c.lastEvent.Type
	r := c.lastEvent.Reason
	m := c.lastEvent.Message

	if t != e.Type || r != e.Reason || m != e.Message {
		glog.Errorf("event mismatch. expected - (t:%s, r:%s, m:%s). got - (t:%s, r:%s, m:%s)", e.Type, e.Reason, e.Message, t, r, m)
		return false
	}
	return true
}

func (c TestEventRecorder) Eventf(object runtime.Object, t string, r string, messageFmt string, args ...interface{}) {

}

func (c TestEventRecorder) PastEventf(object runtime.Object, timestamp v1.Time, t string, m1 string, messageFmt string, args ...interface{}) {

}

func (c TestEventRecorder) AnnotatedEventf(object runtime.Object, annotations map[string]string, eventtype, reason, messageFmt string, args ...interface{}) {

}

func NewMICClient(eventCh chan aadpodid.EventType, cpClient *cp.TestCloudClient, crdClient *crd.TestCrdClient, podClient *pod.TestPodClient, nodeClient *TestNodeClient, eventRecorder *TestEventRecorder) *TestMICClient {

	realMICClient := &mic.Client{
		CloudClient:   cpClient,
		CRDClient:     crdClient,
		EventRecorder: eventRecorder,
		PodClient:     podClient,
		EventChannel:  eventCh,
		NodeClient:    nodeClient,
	}

	return &TestMICClient{
		realMICClient,
	}
}

type TestMICClient struct {
	*mic.Client
}

type TestNodeClient struct {
	nodes map[string]*corev1.Node
}

func NewTestNodeClient() *TestNodeClient {
	return &TestNodeClient{nodes: make(map[string]*corev1.Node)}
}

func (c *TestNodeClient) Get(name string) (*corev1.Node, error) {
	node, exists := c.nodes[name]
	if !exists {
		return nil, errors.New("node not found")
	}
	return node, nil
}

func (c *TestNodeClient) Start(<-chan struct{}) {}

func (c *TestNodeClient) AddNode(name string) {
	c.nodes[name] = &corev1.Node{ObjectMeta: v1.ObjectMeta{Name: name}}
}
