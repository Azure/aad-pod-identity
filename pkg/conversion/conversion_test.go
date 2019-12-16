package conversion

import (
	"testing"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity"
	aadpodidv1 "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var objectMetaName string = "objectMetaName"
var identityName string = "identityName"
var selectorName string = "selectorName"
var idTypeInternal aadpodid.IdentityType = aadpodid.UserAssignedMSI
var idTypeV1 aadpodidv1.IdentityType = aadpodidv1.UserAssignedMSI
var rId string = "resourceId"
var assignedIdPod string = "assignedIdpod"
var replicas int32 = 3
var weight int = 1

func CreateV1Binding() (retV1Binding aadpodidv1.AzureIdentityBinding) {
	return aadpodidv1.AzureIdentityBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: objectMetaName,
		},
		Spec: aadpodidv1.AzureIdentityBindingSpec{
			AzureIdentity: identityName,
			Selector:      selectorName,
			Weight:        weight,
		},
		Status: aadpodidv1.AzureIdentityBindingStatus{
			AvailableReplicas: replicas,
		},
	}
}

func CreateV1Identity() (retV1Identity aadpodidv1.AzureIdentity) {
	return aadpodidv1.AzureIdentity{
		ObjectMeta: metav1.ObjectMeta{
			Name: objectMetaName,
		},
		Spec: aadpodidv1.AzureIdentitySpec{
			Type:       idTypeV1,
			ResourceID: rId,
			Replicas:   &replicas,
		},
		Status: aadpodidv1.AzureIdentityStatus{
			AvailableReplicas: replicas,
		},
	}
}

func CreateInternalBinding() (retV1Binding aadpodid.AzureIdentityBinding) {
	return aadpodid.AzureIdentityBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: objectMetaName,
		},
		Spec: aadpodid.AzureIdentityBindingSpec{
			AzureIdentity: identityName,
			Selector:      selectorName,
			Weight:        weight,
		},
		Status: aadpodid.AzureIdentityBindingStatus{
			AvailableReplicas: replicas,
		},
	}
}

func CreateInternalIdentity() (retInternalIdentity aadpodid.AzureIdentity) {
	return aadpodid.AzureIdentity{
		ObjectMeta: metav1.ObjectMeta{
			Name: objectMetaName,
		},
		Spec: aadpodid.AzureIdentitySpec{
			Type:       idTypeInternal,
			ResourceID: rId,
			Replicas:   &replicas,
		},
		Status: aadpodid.AzureIdentityStatus{
			AvailableReplicas: replicas,
		},
	}
}

func CreateInternalAssignedIdentity() (retInternalAssignedIdentity aadpodid.AzureAssignedIdentity) {
	internalIdentity := CreateInternalIdentity()
	internalBinding := CreateInternalBinding()

	return aadpodid.AzureAssignedIdentity{
		ObjectMeta: metav1.ObjectMeta{
			Name: objectMetaName,
		},
		Spec: aadpodid.AzureAssignedIdentitySpec{
			AzureIdentityRef: &internalIdentity,
			AzureBindingRef:  &internalBinding,
			Pod:              assignedIdPod,
			Replicas:         &replicas,
		},
		Status: aadpodid.AzureAssignedIdentityStatus{
			AvailableReplicas: replicas,
		},
	}
}

func TestConvertV1BindingToInternalBinding(t *testing.T) {
	bindingV1 := CreateV1Binding()
	convertedBindingInternal := ConvertV1BindingToInternalBinding(bindingV1)
	bindingInternal := CreateInternalBinding()

	if !cmp.Equal(bindingInternal, convertedBindingInternal) {
		t.Errorf("Failed to convert from v1 to internal AzureIdentityBinding")
	}
}

func TestConvertV1IdentityToInternalIdentity(t *testing.T) {
	idV1 := CreateV1Identity()
	convertedIdInternal := ConvertV1IdentityToInternalIdentity(idV1)
	idInternal := CreateInternalIdentity()

	if !cmp.Equal(idInternal, convertedIdInternal) {
		t.Errorf("Failed to convert from v1 to internal AzureIdentityBinding")
	}
}

func TestConvertV1AssignedIdentityToInternalAssignedIdentity(t *testing.T) {
	retV1Identity := CreateV1Identity()
	retV1IdentityBinding := CreateV1Binding()

	assignedIdV1 := aadpodidv1.AzureAssignedIdentity{
		ObjectMeta: metav1.ObjectMeta{
			Name: objectMetaName,
		},
		Spec: aadpodidv1.AzureAssignedIdentitySpec{
			AzureIdentityRef: &retV1Identity,
			AzureBindingRef:  &retV1IdentityBinding,
			Pod:              assignedIdPod,
			Replicas:         &replicas,
		},
		Status: aadpodidv1.AzureAssignedIdentityStatus{
			AvailableReplicas: replicas,
		},
	}

	convertedAssignedIdInternal := ConvertV1AssignedIdentityToInternalAssignedIdentity(assignedIdV1)
	assignedIdInternal := CreateInternalAssignedIdentity()

	if !cmp.Equal(assignedIdInternal, convertedAssignedIdInternal) {
		t.Errorf("Failed to convert from v1 to internal AzureAssignedIdentity")
	}
}
