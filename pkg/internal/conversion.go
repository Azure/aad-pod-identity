package internal

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	aadpodidv1 "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	// aadpodidv2 "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v2"
)

func Convert_V1_Binding_To_Internal_Binding(identityBinding aadpodidv1.AzureIdentityBinding) (resIdentityBinding AzureIdentityBinding) {
	var labelValue = identityBinding.Spec.Selector

	binding := &AzureIdentityBinding{
		TypeMeta: identityBinding.TypeMeta,
		ObjectMeta: identityBinding.ObjectMeta,
		Spec: AzureIdentityBindingSpec{
			ObjectMeta: identityBinding.Spec.ObjectMeta,
			AzureIdentity: identityBinding.Spec.AzureIdentity,
			LabelSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{CRDLabelKey: labelValue},
			},
			Weight: identityBinding.Spec.Weight,
		},
		Status: AzureIdentityBindingStatus {
			ObjectMeta: identityBinding.Status.ObjectMeta,
			AvailableReplicas: identityBinding.Status.AvailableReplicas,
		},
	}

	return *binding
}

func ConvertInternalToV1(identityBinding *[]AzureIdentityBinding, assignedIdentity *[]AzureAssignedIdentity, identity *[]AzureIdentity) (resIdentityBinding *[]AzureIdentityBinding, resAssignedIdentity *[]AzureAssignedIdentity, resIdentity *[]AzureIdentity) {
	return identityBinding, assignedIdentity, identity
}

func ConvertV2ToInternal(identityBinding *[]AzureIdentityBinding, assignedIdentity *[]AzureAssignedIdentity, identity *[]AzureIdentity) (resIdentityBinding *[]AzureIdentityBinding, resAssignedIdentity *[]AzureAssignedIdentity, resIdentity *[]AzureIdentity) {
	return identityBinding, assignedIdentity, identity
}

func ConvertInternaltoV2(identityBinding *[]AzureIdentityBinding, assignedIdentity *[]AzureAssignedIdentity, identity *[]AzureIdentity) (resIdentityBinding *[]AzureIdentityBinding, resAssignedIdentity *[]AzureAssignedIdentity, resIdentity *[]AzureIdentity) {
	return identityBinding, assignedIdentity, identity
}