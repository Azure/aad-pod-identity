//go:build e2e
// +build e2e

package framework

import (
	"reflect"

	aadpodv1 "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1beta "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	// NamespaceKubeSystem is the name of kube-system namespace.
	NamespaceKubeSystem = "kube-system"
)

// TryAddDefaultSchemes tries to add various schemes.
func TryAddDefaultSchemes(scheme *runtime.Scheme) {
	// Add the core schemes.
	_ = corev1.AddToScheme(scheme)

	// Add the apps schemes.
	_ = appsv1.AddToScheme(scheme)

	// Add the api extensions (CRD) to the scheme.
	_ = apiextensionsv1beta.AddToScheme(scheme)
	_ = apiextensionsv1.AddToScheme(scheme)

	// Add rbac to the scheme.
	_ = rbacv1.AddToScheme(scheme)

	// Add aadpodidentity v1 to the scheme.
	_ = aadpodv1.AddToScheme(scheme)
}

// TypeMeta returns the TypeMeta struct of a given runtime object.
func TypeMeta(i runtime.Object) metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: aadpodv1.SchemeGroupVersion.String(),
		Kind:       reflect.ValueOf(i).Elem().Type().Name(),
	}
}
