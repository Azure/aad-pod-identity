package conversion

import (
	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity"
	aadpodidv1 "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	aadpodidv2 "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ConvertV1BindingToInternalBinding(identityBinding aadpodidv1.AzureIdentityBinding) (resIdentityBinding aadpodid.AzureIdentityBinding) {
	return aadpodid.AzureIdentityBinding{
		TypeMeta: identityBinding.TypeMeta,
		ObjectMeta: identityBinding.ObjectMeta,
		Spec: aadpodid.AzureIdentityBindingSpec{
			ObjectMeta: identityBinding.Spec.ObjectMeta,
			AzureIdentity: identityBinding.Spec.AzureIdentity,
			LabelSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{aadpodid.CRDLabelKey: identityBinding.Spec.Selector},
			},
			Weight: identityBinding.Spec.Weight,
		},
		Status: aadpodid.AzureIdentityBindingStatus(identityBinding.Status),
	}
}

func ConvertV1IdentityToInternalIdentity(identity aadpodidv1.AzureIdentity) (resIdentity aadpodid.AzureIdentity) {
	var identityType string = ""
	switch identity.Spec.Type {
	case 0:
		identityType = "UserAssignedMSI"
	case 1:
		identityType = "ServicePrincipal"
	}
	
	return aadpodid.AzureIdentity{
		TypeMeta: identity.TypeMeta,
		ObjectMeta: identity.ObjectMeta,
		Spec: aadpodid.AzureIdentitySpec{
			ObjectMeta: identity.Spec.ObjectMeta,
			Type:       aadpodid.IdentityType(identityType),
			ResourceID: identity.Spec.ResourceID,
			ClientID:   identity.Spec.ClientID,
			ClientPassword: identity.Spec.ClientPassword,
			TenantID:     identity.Spec.TenantID,
			ADResourceID: identity.Spec.ADResourceID,
			ADEndpoint:   identity.Spec.ADEndpoint,
			Replicas: 		identity.Spec.Replicas,
		},
		Status: aadpodid.AzureIdentityStatus(identity.Status),
	}
}

func ConvertV1AssignedIdentityToInternalAssignedIdentity(assignedIdentity aadpodidv1.AzureAssignedIdentity) (resAssignedIdentity aadpodid.AzureAssignedIdentity) {
	retIdentity := ConvertV1IdentityToInternalIdentity(*assignedIdentity.Spec.AzureIdentityRef)
	retBinding :=ConvertV1BindingToInternalBinding(*assignedIdentity.Spec.AzureBindingRef)
	
	return aadpodid.AzureAssignedIdentity{
		TypeMeta: assignedIdentity.TypeMeta,
		ObjectMeta: assignedIdentity.ObjectMeta,
		Spec: aadpodid.AzureAssignedIdentitySpec{
			ObjectMeta: assignedIdentity.Spec.ObjectMeta,
			AzureIdentityRef: &retIdentity,
			AzureBindingRef: &retBinding,
			Pod: assignedIdentity.Spec.Pod,
			PodNamespace: assignedIdentity.Spec.PodNamespace,
			NodeName: assignedIdentity.Spec.NodeName,
			Replicas: assignedIdentity.Spec.Replicas,
		},
		Status: aadpodid.AzureAssignedIdentityStatus(assignedIdentity.Status),
	}
}

func ConvertV2BindingToInternalBinding(identityBinding aadpodidv2.AzureIdentityBinding) (resIdentityBinding aadpodid.AzureIdentityBinding) {
	return aadpodid.AzureIdentityBinding{
		TypeMeta: identityBinding.TypeMeta,
		ObjectMeta: identityBinding.ObjectMeta,
		Spec: aadpodid.AzureIdentityBindingSpec{
			ObjectMeta: identityBinding.Spec.ObjectMeta,
			AzureIdentity: identityBinding.Spec.AzureIdentity,
			LabelSelector: identityBinding.Spec.LabelSelector,
			Weight: identityBinding.Spec.Weight,
		},
		Status: aadpodid.AzureIdentityBindingStatus(identityBinding.Status),
	}
}

func ConvertV2IdentityToInternalIdentity(identity aadpodidv2.AzureIdentity) (resIdentity aadpodid.AzureIdentity) {
	return aadpodid.AzureIdentity{
		TypeMeta: identity.TypeMeta,
		ObjectMeta: identity.ObjectMeta,
		Spec: aadpodid.AzureIdentitySpec{
			ObjectMeta: identity.Spec.ObjectMeta,
			Type:       aadpodid.IdentityType(identity.Spec.Type),
			ResourceID: identity.Spec.ResourceID,
			ClientID:   identity.Spec.ClientID,
			ClientPassword: identity.Spec.ClientPassword,
			TenantID:     identity.Spec.TenantID,
			ADResourceID: identity.Spec.ADResourceID,
			ADEndpoint:   identity.Spec.ADEndpoint,
			Replicas: 		identity.Spec.Replicas,
		},
		Status: aadpodid.AzureIdentityStatus(identity.Status),
	}
}

func ConvertV2AssignedIdentityToInternalAssignedIdentity(assignedIdentity aadpodidv2.AzureAssignedIdentity) (resAssignedIdentity aadpodid.AzureAssignedIdentity) {
	retIdentity := ConvertV2IdentityToInternalIdentity(*assignedIdentity.Spec.AzureIdentityRef)
	retBinding :=ConvertV2BindingToInternalBinding(*assignedIdentity.Spec.AzureBindingRef)
	
	return aadpodid.AzureAssignedIdentity{
		TypeMeta: assignedIdentity.TypeMeta,
		ObjectMeta: assignedIdentity.ObjectMeta,
		Spec: aadpodid.AzureAssignedIdentitySpec{
			ObjectMeta: assignedIdentity.Spec.ObjectMeta,
			AzureIdentityRef: &retIdentity,
			AzureBindingRef: &retBinding,
			Pod: assignedIdentity.Spec.Pod,
			PodNamespace: assignedIdentity.Spec.PodNamespace,
			NodeName: assignedIdentity.Spec.NodeName,
			Replicas: assignedIdentity.Spec.Replicas,
		},
		Status: aadpodid.AzureAssignedIdentityStatus(assignedIdentity.Status),
	}
}