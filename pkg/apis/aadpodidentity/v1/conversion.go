package v1

import (
	"reflect"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

func ConvertV1IdentityToInternalIdentity(identity AzureIdentity) (resIdentity aadpodid.AzureIdentity) {
	return aadpodid.AzureIdentity{
		TypeMeta:   identity.TypeMeta,
		ObjectMeta: identity.ObjectMeta,
		Spec: aadpodid.AzureIdentitySpec{
			ObjectMeta:     identity.Spec.ObjectMeta,
			Type:           aadpodid.IdentityType(identity.Spec.Type),
			ResourceID:     identity.Spec.ResourceID,
			ClientID:       identity.Spec.ClientID,
			ClientPassword: identity.Spec.ClientPassword,
			TenantID:       identity.Spec.TenantID,
			ADResourceID:   identity.Spec.ADResourceID,
			ADEndpoint:     identity.Spec.ADEndpoint,
			Replicas:       identity.Spec.Replicas,
		},
		Status: aadpodid.AzureIdentityStatus(identity.Status),
	}
}

func ConvertV1AssignedIdentityToInternalAssignedIdentity(assignedIdentity AzureAssignedIdentity) (resAssignedIdentity aadpodid.AzureAssignedIdentity) {
	retIdentity := ConvertV1IdentityToInternalIdentity(*assignedIdentity.Spec.AzureIdentityRef)
	retBinding := ConvertV1BindingToInternalBinding(*assignedIdentity.Spec.AzureBindingRef)

	return aadpodid.AzureAssignedIdentity{
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
	}
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

// ConvertInternalPodIdentityExceptionToV1PodIdentityException is currently not needed, as AzurePodIdentityException are only listed and not created within the project
