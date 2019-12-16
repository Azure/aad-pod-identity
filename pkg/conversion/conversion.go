package conversion

import (
	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity"
	aadpodidv1 "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
)

func ConvertV1BindingToInternalBinding(identityBinding aadpodidv1.AzureIdentityBinding) (resIdentityBinding aadpodid.AzureIdentityBinding) {
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

func ConvertV1IdentityToInternalIdentity(identity aadpodidv1.AzureIdentity) (resIdentity aadpodid.AzureIdentity) {
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

func ConvertV1AssignedIdentityToInternalAssignedIdentity(assignedIdentity aadpodidv1.AzureAssignedIdentity) (resAssignedIdentity aadpodid.AzureAssignedIdentity) {
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

func ConvertInternalBindingToV1Binding(identityBinding aadpodid.AzureIdentityBinding) (resIdentityBinding aadpodidv1.AzureIdentityBinding) {
	return aadpodidv1.AzureIdentityBinding{
		TypeMeta:   identityBinding.TypeMeta,
		ObjectMeta: identityBinding.ObjectMeta,
		Spec: aadpodidv1.AzureIdentityBindingSpec{
			ObjectMeta:    identityBinding.Spec.ObjectMeta,
			AzureIdentity: identityBinding.Spec.AzureIdentity,
			Selector:      identityBinding.Spec.Selector,
			Weight:        identityBinding.Spec.Weight,
		},
		Status: aadpodidv1.AzureIdentityBindingStatus(identityBinding.Status),
	}
}

func ConvertInternalIdentityToV1Identity(identity aadpodid.AzureIdentity) (resIdentity aadpodidv1.AzureIdentity) {
	return aadpodidv1.AzureIdentity{
		TypeMeta:   identity.TypeMeta,
		ObjectMeta: identity.ObjectMeta,
		Spec: aadpodidv1.AzureIdentitySpec{
			ObjectMeta:     identity.Spec.ObjectMeta,
			Type:           aadpodidv1.IdentityType(identity.Spec.Type),
			ResourceID:     identity.Spec.ResourceID,
			ClientID:       identity.Spec.ClientID,
			ClientPassword: identity.Spec.ClientPassword,
			TenantID:       identity.Spec.TenantID,
			ADResourceID:   identity.Spec.ADResourceID,
			ADEndpoint:     identity.Spec.ADEndpoint,
			Replicas:       identity.Spec.Replicas,
		},
		Status: aadpodidv1.AzureIdentityStatus(identity.Status),
	}
}

func ConvertInternalAssignedIdentityToV1AssignedIdentity(assignedIdentity aadpodid.AzureAssignedIdentity) (resAssignedIdentity aadpodidv1.AzureAssignedIdentity) {
	retIdentity := ConvertInternalIdentityToV1Identity(*assignedIdentity.Spec.AzureIdentityRef)
	retBinding := ConvertInternalBindingToV1Binding(*assignedIdentity.Spec.AzureBindingRef)

	return aadpodidv1.AzureAssignedIdentity{
		TypeMeta:   assignedIdentity.TypeMeta,
		ObjectMeta: assignedIdentity.ObjectMeta,
		Spec: aadpodidv1.AzureAssignedIdentitySpec{
			ObjectMeta:       assignedIdentity.Spec.ObjectMeta,
			AzureIdentityRef: &retIdentity,
			AzureBindingRef:  &retBinding,
			Pod:              assignedIdentity.Spec.Pod,
			PodNamespace:     assignedIdentity.Spec.PodNamespace,
			NodeName:         assignedIdentity.Spec.NodeName,
			Replicas:         assignedIdentity.Spec.Replicas,
		},
		Status: aadpodidv1.AzureAssignedIdentityStatus(assignedIdentity.Status),
	}
}
