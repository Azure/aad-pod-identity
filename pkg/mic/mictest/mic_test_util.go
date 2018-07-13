package mictest

import (
	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	cp "github.com/Azure/aad-pod-identity/pkg/cloudprovider/cloudprovidertest"
	crd "github.com/Azure/aad-pod-identity/pkg/crd/crdtest"
	pod "github.com/Azure/aad-pod-identity/pkg/pod/podtest"

	"github.com/Azure/aad-pod-identity/pkg/mic"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type TestEventRecorder struct {
}

func (c TestEventRecorder) Event(object runtime.Object, t string, r string, message string) {

}

func (c TestEventRecorder) Eventf(object runtime.Object, t string, r string, messageFmt string, args ...interface{}) {

}

func (c TestEventRecorder) PastEventf(object runtime.Object, timestamp v1.Time, t string, m1 string, messageFmt string, args ...interface{}) {

}

func (c TestEventRecorder) AnnotatedEventf(object runtime.Object, annotations map[string]string, eventtype, reason, messageFmt string, args ...interface{}) {

}

func NewMICClient(eventCh chan aadpodid.EventType, cpClient *cp.TestCloudClient, crdClient *crd.TestCrdClient, podClient *pod.TestPodClient) *TestMICClient {
	var eventRecorder TestEventRecorder

	realMICClient := &mic.Client{
		CloudClient:   cpClient,
		CRDClient:     crdClient,
		EventRecorder: eventRecorder,
		PodClient:     podClient,
		EventChannel:  eventCh,
	}

	return &TestMICClient{
		realMICClient,
	}
}

type TestMICClient struct {
	*mic.Client
}
