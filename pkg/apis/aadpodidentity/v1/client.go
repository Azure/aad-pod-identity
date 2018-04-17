package aadpodidentity

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

const (
	CRDGroup   = "aadpodidentity.k8s.io"
	CRDVersion = "v1"
)

func NewAadPodIdentityCrdClient(client *rest.Config) (*rest.RESTClient, error) {

	crdconfig := *client
	crdconfig.GroupVersion = &schema.GroupVersion{Group: CRDGroup, Version: CRDVersion}
	crdconfig.APIPath = "/apis"
	crdconfig.ContentType = runtime.ContentTypeJSON
	crdconfig.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}

	crdclient, err := rest.RESTClientFor(&crdconfig)
	if err != nil {
		return nil, err
	}
	return crdclient, nil
}
