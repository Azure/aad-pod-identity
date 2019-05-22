package v1

import (
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var SchemeGroupVersion = schema.GroupVersion{Group: CRDGroup, Version: CRDVersion}

// Kind takes an unqualified kind and returns back a Group qualified GroupKind
func Kind(kindName string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kindName).GroupKind()
}

func Resource(resourceName string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resourceName).GroupResource()
}

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

func addKnownTypes(s *runtime.Scheme) error {
	s.AddKnownTypes(SchemeGroupVersion,
		&AzureIdentity{},
		&AzureIdentityList{},
		&AzureIdentityBinding{},
		&AzureIdentityBindingList{},
		&AzureAssignedIdentity{},
		&AzureAssignedIdentityList{})

	meta.AddToGroupVersion(s, SchemeGroupVersion)
	return nil
}
