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
var rID string = "resourceId"
var assignedIDPod string = "assignedIDPod"
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
			ResourceID: rID,
			Replicas:   &replicas,
		},
		Status: aadpodidv1.AzureIdentityStatus{
			AvailableReplicas: replicas,
		},
	}
}

func CreateV1AssignedIdentity() (retV1AssignedIdentity aadpodidv1.AzureAssignedIdentity) {
	v1Identity := CreateV1Identity()
	v1Binding := CreateV1Binding()

	return aadpodidv1.AzureAssignedIdentity{
		ObjectMeta: metav1.ObjectMeta{
			Name: objectMetaName,
		},
		Spec: aadpodidv1.AzureAssignedIdentitySpec{
			AzureIdentityRef: &v1Identity,
			AzureBindingRef:  &v1Binding,
			Pod:              assignedIDPod,
			Replicas:         &replicas,
		},
		Status: aadpodidv1.AzureAssignedIdentityStatus{
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
			ResourceID: rID,
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
			Pod:              assignedIDPod,
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
	convertedIDInternal := ConvertV1IdentityToInternalIdentity(idV1)
	idInternal := CreateInternalIdentity()

	if !cmp.Equal(idInternal, convertedIDInternal) {
		t.Errorf("Failed to convert from v1 to internal AzureIdentity")
	}
}

func TestConvertV1AssignedIdentityToInternalAssignedIdentity(t *testing.T) {
	assignedIDV1 := CreateV1AssignedIdentity()

	convertedAssignedIDInternal := ConvertV1AssignedIdentityToInternalAssignedIdentity(assignedIDV1)
	assignedIDInternal := CreateInternalAssignedIdentity()

	if !cmp.Equal(assignedIDInternal, convertedAssignedIDInternal) {
		t.Errorf("Failed to convert from v1 to internal AzureAssignedIdentity")
	}
}

func TestConvertInternalBindingToV1Binding(t *testing.T) {
	bindingInternal := CreateInternalBinding()
	convertedBindingV1 := ConvertInternalBindingToV1Binding(bindingInternal)
	bindingV1 := CreateV1Binding()

	if !cmp.Equal(bindingV1, convertedBindingV1) {
		t.Errorf("Failed to convert from internal to v1 AzureIdentityBinding")
	}
}

func TestConvertInternalIdentityToV1Identity(t *testing.T) {
	idInternal := CreateInternalIdentity()
	convertedIDV1 := ConvertInternalIdentityToV1Identity(idInternal)
	idV1 := CreateV1Identity()

	if !cmp.Equal(idV1, convertedIDV1) {
		t.Errorf("Failed to convert from internal to v1 AzureIdentity")
	}
}

func TestConvertInternalAssignedIdentityToV1AssignedIdentity(t *testing.T) {
	assignedIDInternal := CreateInternalAssignedIdentity()

	convertedAssignedIDV1 := ConvertInternalAssignedIdentityToV1AssignedIdentity(assignedIDInternal)
	assignedIDV1 := CreateV1AssignedIdentity()

	if !cmp.Equal(assignedIDV1, convertedAssignedIDV1) {
		t.Errorf("Failed to convert from v1 to internal AzureAssignedIdentity")
	}
}
