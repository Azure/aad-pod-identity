package internal

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type EventType int

const (
	PodCreated      EventType = 0
	PodDeleted      EventType = 1
	PodUpdated      EventType = 2
	IdentityCreated EventType = 3
	IdentityDeleted EventType = 4
	IdentityUpdated EventType = 5
	BindingCreated  EventType = 6
	BindingDeleted  EventType = 7
	BindingUpdated  EventType = 8
	Exit            EventType = 9
)

func IsNamespacedIdentity(azureId *AzureIdentity) bool {
	if val, ok := azureId.Annotations[BehaviorKey]; ok {
		if val == BehaviorNamespaced {
			return true
		}
	}
	return false
}

//TODO: is this the right place for the conversion functions?
//TODO: V2 and internal are currently the same API. 

func ConvertV1ToInternal(identityBinding *[]AzureIdentityBinding, assignedIdentity *[]AzureAssignedIdentity, identity *[]AzureIdentity) (resIdentityBinding *[]AzureIdentityBinding, resAssignedIdentity *[]AzureAssignedIdentity, resIdentity *[]AzureIdentity) {
	var outIdentityBinding []AzureIdentityBinding
	for _, allBinding := range *identityBinding {
		var labelValue = allBinding.Spec.Selector

 		binding := &AzureIdentityBinding{
			TypeMeta: allBinding.TypeMeta,
			ObjectMeta: allBinding.ObjectMeta,
			Spec: AzureIdentityBindingSpec{
				ObjectMeta: allBinding.Spec.ObjectMeta,
				AzureIdentity: allBinding.Spec.AzureIdentity,
				LabelSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{CRDLabelKey: labelValue},
				},
				Weight: allBinding.Spec.Weight,
			},
			Status: AzureIdentityBindingStatus {
				ObjectMeta: allBinding.Status.ObjectMeta,
				AvailableReplicas: allBinding.Status.AvailableReplicas,
			},
		}

 		outIdentityBinding = append(outIdentityBinding, *binding)
	}

	return &outIdentityBinding, assignedIdentity, identity
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