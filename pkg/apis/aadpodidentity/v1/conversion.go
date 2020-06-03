package v1

import (
	"fmt"
	"reflect"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

func ConvertV1BindingToInternalBinding(identityBinding AzureIdentityBinding) (resIdentityBinding aadpodid.AzureIdentityBinding) {
	return aadpodid.AzureIdentityBinding{
		TypeMeta:   identityBinding.TypeMeta,
		ObjectMeta: identityBinding.ObjectMeta,
		Spec: aadpodid.AzureIdentityBindingSpec{
			ObjectMeta:    identityBinding.Spec.ObjectMeta,
			AzureIdentity: identityBinding.Spec.AzureIdentity,
			Selector:      identityBinding.Spec.Selector,
			Weight:        identityBinding.Spec.Weight,
		},
		Status: aadpodid.AzureIdentityBindingStatus(identityBinding.Status),
	}
}

func ConvertV1IdentityToInternalIdentity(identity AzureIdentity, c kubernetes.Interface) (resIdentity *aadpodid.AzureIdentity, err error) {
	clientID := identity.Spec.ClientID

	if identity.Spec.ClientIDSecretRef != nil {
		clientIDSecret, err := getSecret(c, identity.Spec.ClientIDSecretRef.Namespace, identity.Spec.ClientIDSecretRef.Name)

		if err != nil {
			return nil, fmt.Errorf("Unable to retrieve a secret named %s in namespace %s. %s", identity.Spec.ClientIDSecretRef.Name, identity.Spec.ClientIDSecretRef.Namespace, err.Error())
		}

		for _, v := range clientIDSecret.Data {
			clientID = string(v)
			break
		}
	}

	resourceID := identity.Spec.ResourceID

	if identity.Spec.ResourceIDSecretRef != nil {
		resourceIDSecret, err := getSecret(c, identity.Spec.ResourceIDSecretRef.Namespace, identity.Spec.ResourceIDSecretRef.Name)

		if err != nil {
			return nil, fmt.Errorf("Unable to retrieve a secret named %s in namespace %s. %s", identity.Spec.ResourceIDSecretRef.Name, identity.Spec.ResourceIDSecretRef.Namespace, err.Error())
		}

		for _, v := range resourceIDSecret.Data {
			resourceID = string(v)
			break
		}
	}

	return &aadpodid.AzureIdentity{
		TypeMeta:   identity.TypeMeta,
		ObjectMeta: identity.ObjectMeta,
		Spec: aadpodid.AzureIdentitySpec{
			ObjectMeta:     identity.Spec.ObjectMeta,
			Type:           aadpodid.IdentityType(identity.Spec.Type),
			ResourceID:     resourceID,
			ClientID:       clientID,
			ClientPassword: identity.Spec.ClientPassword,
			TenantID:       identity.Spec.TenantID,
			ADResourceID:   identity.Spec.ADResourceID,
			ADEndpoint:     identity.Spec.ADEndpoint,
			Replicas:       identity.Spec.Replicas,
		},
		Status: aadpodid.AzureIdentityStatus(identity.Status),
	}, nil
}

func ConvertV1AssignedIdentityToInternalAssignedIdentity(assignedIdentity AzureAssignedIdentity, c kubernetes.Interface) (resAssignedIdentity *aadpodid.AzureAssignedIdentity, err error) {
	var retIdentity aadpodid.AzureIdentity
	var retBinding aadpodid.AzureIdentityBinding
	if assignedIdentity.Spec.AzureIdentityRef != nil {
		tempIdentity, err := ConvertV1IdentityToInternalIdentity(*assignedIdentity.Spec.AzureIdentityRef, c)

		if err != nil {
			return nil, err
		}

		retIdentity = *tempIdentity
	}
	if assignedIdentity.Spec.AzureBindingRef != nil {
		retBinding = ConvertV1BindingToInternalBinding(*assignedIdentity.Spec.AzureBindingRef)
	}

	return &aadpodid.AzureAssignedIdentity{
		TypeMeta:   assignedIdentity.TypeMeta,
		ObjectMeta: assignedIdentity.ObjectMeta,
		Spec: aadpodid.AzureAssignedIdentitySpec{
			ObjectMeta:       assignedIdentity.Spec.ObjectMeta,
			AzureIdentityRef: &retIdentity,
			AzureBindingRef:  &retBinding,
			Pod:              assignedIdentity.Spec.Pod,
			PodNamespace:     assignedIdentity.Spec.PodNamespace,
			NodeName:         assignedIdentity.Spec.NodeName,
			Replicas:         assignedIdentity.Spec.Replicas,
		},
		Status: aadpodid.AzureAssignedIdentityStatus(assignedIdentity.Status),
	}, nil
}

func ConvertV1PodIdentityExceptionToInternalPodIdentityException(idException AzurePodIdentityException) (residException aadpodid.AzurePodIdentityException) {
	return aadpodid.AzurePodIdentityException{
		TypeMeta:   idException.TypeMeta,
		ObjectMeta: idException.ObjectMeta,
		Spec: aadpodid.AzurePodIdentityExceptionSpec{
			ObjectMeta: idException.Spec.ObjectMeta,
			PodLabels:  idException.Spec.PodLabels,
		},
		Status: aadpodid.AzurePodIdentityExceptionStatus(idException.Status),
	}
}

func ConvertInternalBindingToV1Binding(identityBinding aadpodid.AzureIdentityBinding) (resIdentityBinding AzureIdentityBinding) {
	out := AzureIdentityBinding{
		TypeMeta:   identityBinding.TypeMeta,
		ObjectMeta: identityBinding.ObjectMeta,
		Spec: AzureIdentityBindingSpec{
			ObjectMeta:    identityBinding.Spec.ObjectMeta,
			AzureIdentity: identityBinding.Spec.AzureIdentity,
			Selector:      identityBinding.Spec.Selector,
			Weight:        identityBinding.Spec.Weight,
		},
		Status: AzureIdentityBindingStatus(identityBinding.Status),
	}

	out.TypeMeta.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   CRDGroup,
		Version: CRDVersion,
		Kind:    reflect.TypeOf(out).Name()})

	return out
}

func ConvertInternalIdentityToV1Identity(identity aadpodid.AzureIdentity) (resIdentity AzureIdentity) {
	out := AzureIdentity{
		TypeMeta:   identity.TypeMeta,
		ObjectMeta: identity.ObjectMeta,
		Spec: AzureIdentitySpec{
			ObjectMeta:     identity.Spec.ObjectMeta,
			Type:           IdentityType(identity.Spec.Type),
			ResourceID:     identity.Spec.ResourceID,
			ClientID:       identity.Spec.ClientID,
			ClientPassword: identity.Spec.ClientPassword,
			TenantID:       identity.Spec.TenantID,
			ADResourceID:   identity.Spec.ADResourceID,
			ADEndpoint:     identity.Spec.ADEndpoint,
			Replicas:       identity.Spec.Replicas,
		},
		Status: AzureIdentityStatus(identity.Status),
	}

	out.TypeMeta.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   CRDGroup,
		Version: CRDVersion,
		Kind:    reflect.TypeOf(out).Name()})

	return out
}

func ConvertInternalAssignedIdentityToV1AssignedIdentity(assignedIdentity aadpodid.AzureAssignedIdentity) (resAssignedIdentity AzureAssignedIdentity) {
	retIdentity := ConvertInternalIdentityToV1Identity(*assignedIdentity.Spec.AzureIdentityRef)
	retBinding := ConvertInternalBindingToV1Binding(*assignedIdentity.Spec.AzureBindingRef)

	out := AzureAssignedIdentity{
		TypeMeta:   assignedIdentity.TypeMeta,
		ObjectMeta: assignedIdentity.ObjectMeta,
		Spec: AzureAssignedIdentitySpec{
			ObjectMeta:       assignedIdentity.Spec.ObjectMeta,
			AzureIdentityRef: &retIdentity,
			AzureBindingRef:  &retBinding,
			Pod:              assignedIdentity.Spec.Pod,
			PodNamespace:     assignedIdentity.Spec.PodNamespace,
			NodeName:         assignedIdentity.Spec.NodeName,
			Replicas:         assignedIdentity.Spec.Replicas,
		},
		Status: AzureAssignedIdentityStatus(assignedIdentity.Status),
	}

	out.TypeMeta.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   CRDGroup,
		Version: CRDVersion,
		Kind:    reflect.TypeOf(out).Name()})

	return out
}

func getSecret(c kubernetes.Interface, namespace string, secretName string) (*corev1.Secret, error) {

	secret, err := c.CoreV1().Secrets(namespace).Get(secretName, v1.GetOptions{})

	if err != nil {
		return nil, fmt.Errorf("get failed for Secret with error: %v", err)
	}

	return secret, err
}

// ConvertInternalPodIdentityExceptionToV1PodIdentityException is currently not needed, as AzurePodIdentityException are only listed and not created within the project
