package v1

import (
	b64 "encoding/base64"
	"testing"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity"

	"github.com/google/go-cmp/cmp"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes/fake"
)

var objectMetaName string = "objectMetaName"
var identityName string = "identityName"
var selectorName string = "selectorName"
var idTypeInternal aadpodid.IdentityType = aadpodid.UserAssignedMSI
var idTypeV1 IdentityType = UserAssignedMSI
var rID string = "resourceId"
var assignedIDPod string = "assignedIDPod"
var replicas int32 = 3
var weight int = 1
var podLabels = map[string]string{"testkey1": "testval1", "testkey2": "testval2"}

func CreateV1Binding() (retV1Binding AzureIdentityBinding) {
	return AzureIdentityBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: objectMetaName,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "AzureIdentityBinding",
			APIVersion: "aadpodidentity.k8s.io/v1",
		},
		Spec: AzureIdentityBindingSpec{
			AzureIdentity: identityName,
			Selector:      selectorName,
			Weight:        weight,
		},
		Status: AzureIdentityBindingStatus{
			AvailableReplicas: replicas,
		},
	}
}

func CreateV1Identity() (retV1Identity AzureIdentity) {
	return AzureIdentity{
		ObjectMeta: metav1.ObjectMeta{
			Name: objectMetaName,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "AzureIdentity",
			APIVersion: "aadpodidentity.k8s.io/v1",
		},
		Spec: AzureIdentitySpec{
			Type:       idTypeV1,
			ResourceID: rID,
			Replicas:   &replicas,
		},
		Status: AzureIdentityStatus{
			AvailableReplicas: replicas,
		},
	}
}

func CreateV1IdentityWithClientIdSecretRef() (retV1Identity AzureIdentity) {
	return AzureIdentity{
		ObjectMeta: metav1.ObjectMeta{
			Name: objectMetaName,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "AzureIdentity",
			APIVersion: "aadpodidentity.k8s.io/v1",
		},
		Spec: AzureIdentitySpec{
			Type:       idTypeV1,
			ResourceID: rID,
			Replicas:   &replicas,
			ClientIDSecretRef: &v1.SecretReference{
				Name:      "test",
				Namespace: "test",
			},
		},
		Status: AzureIdentityStatus{
			AvailableReplicas: replicas,
		},
	}
}

func CreateV1IdentityWithResourceIdSecretRef() (retV1Identity AzureIdentity) {
	return AzureIdentity{
		ObjectMeta: metav1.ObjectMeta{
			Name: objectMetaName,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "AzureIdentity",
			APIVersion: "aadpodidentity.k8s.io/v1",
		},
		Spec: AzureIdentitySpec{
			Type:       idTypeV1,
			ResourceID: rID,
			Replicas:   &replicas,
			ResourceIDSecretRef: &v1.SecretReference{
				Name:      "test",
				Namespace: "test",
			},
		},
		Status: AzureIdentityStatus{
			AvailableReplicas: replicas,
		},
	}
}

func CreateV1AssignedIdentity() (retV1AssignedIdentity AzureAssignedIdentity) {
	v1Identity := CreateV1Identity()
	v1Binding := CreateV1Binding()

	return AzureAssignedIdentity{
		ObjectMeta: metav1.ObjectMeta{
			Name: objectMetaName,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "AzureAssignedIdentity",
			APIVersion: "aadpodidentity.k8s.io/v1",
		},
		Spec: AzureAssignedIdentitySpec{
			AzureIdentityRef: &v1Identity,
			AzureBindingRef:  &v1Binding,
			Pod:              assignedIDPod,
			Replicas:         &replicas,
		},
		Status: AzureAssignedIdentityStatus{
			AvailableReplicas: replicas,
		},
	}
}

func CreateInternalBinding() (retV1Binding aadpodid.AzureIdentityBinding) {
	return aadpodid.AzureIdentityBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: objectMetaName,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "AzureIdentityBinding",
			APIVersion: "aadpodidentity.k8s.io/v1",
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
		TypeMeta: metav1.TypeMeta{
			Kind:       "AzureIdentity",
			APIVersion: "aadpodidentity.k8s.io/v1",
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
		TypeMeta: metav1.TypeMeta{
			Kind:       "AzureAssignedIdentity",
			APIVersion: "aadpodidentity.k8s.io/v1",
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

func CreateInternalPodIdentityException() (retPodIdentityException aadpodid.AzurePodIdentityException) {
	return aadpodid.AzurePodIdentityException{
		ObjectMeta: metav1.ObjectMeta{
			Name: objectMetaName,
		},
		Spec: aadpodid.AzurePodIdentityExceptionSpec{
			PodLabels: podLabels,
		},
		Status: aadpodid.AzurePodIdentityExceptionStatus{},
	}
}

func CreateV1PodIdentityException() (retPodIdentityException AzurePodIdentityException) {
	return AzurePodIdentityException{
		ObjectMeta: metav1.ObjectMeta{
			Name: objectMetaName,
		},
		Spec: AzurePodIdentityExceptionSpec{
			PodLabels: podLabels,
		},
		Status: AzurePodIdentityExceptionStatus{},
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
	convertedIDInternal, err := ConvertV1IdentityToInternalIdentity(idV1, nil)

	if err != nil {
		t.Error(err)
		t.Errorf("Failed to convert from v1 to internal AzureAssignedIdentity. %s", err)
	}

	idInternal := CreateInternalIdentity()

	if !cmp.Equal(idInternal, convertedIDInternal) {
		t.Errorf("Failed to convert from v1 to internal AzureIdentity %s %s", idInternal.Name, convertedIDInternal.Name)
	}
}

func TestConvertV1IdentityWithSecretClientIdToInternalIdentity(t *testing.T) {
	idV1 := CreateV1IdentityWithClientIdSecretRef()

	secretClientId := "mysecretclientid"
	encodedSecretClientId := []byte(b64.StdEncoding.EncodeToString([]byte(secretClientId)))

	client := fake.NewSimpleClientset(&v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Data: map[string][]byte{
			"data": encodedSecretClientId,
		},
	})
	convertedIDInternal, err := ConvertV1IdentityToInternalIdentity(idV1, client)

	if err != nil {
		t.Error(err)
		t.Errorf("Failed to convert from v1 to internal AzureAssignedIdentity")
	}

	if !cmp.Equal(string(encodedSecretClientId), convertedIDInternal.Spec.ClientID) {
		t.Errorf("Failed to convert from v1 to internal AzureIdentity %s %s", encodedSecretClientId, convertedIDInternal.Spec.ClientID)
	}
}

func TestConvertV1IdentityWithSecretClientIdToInternalIdentityIfNoSecretFound(t *testing.T) {
	idV1 := CreateV1IdentityWithClientIdSecretRef()

	client := fake.NewSimpleClientset()
	_, err := ConvertV1IdentityToInternalIdentity(idV1, client)

	if err == nil {
		t.Errorf("Error was not returned.")
	}

	errorMessage := "Unable to retrieve a secret named test in namespace test. secrets \"test\" not found"
	if !cmp.Equal(errorMessage, err.Error()) {
		t.Errorf("Error message did not match expected. Expected: %s. Actual: %s", errorMessage, err.Error())
	}
}

func TestConvertV1IdentityWithSecretResourceIdToInternalIdentityIfNoSecretFound(t *testing.T) {
	idV1 := CreateV1IdentityWithResourceIdSecretRef()

	client := fake.NewSimpleClientset()
	_, err := ConvertV1IdentityToInternalIdentity(idV1, client)

	if err == nil {
		t.Errorf("Error was not returned.")
	}

	errorMessage := "Unable to retrieve a secret named test in namespace test. secrets \"test\" not found"
	if !cmp.Equal(errorMessage, err.Error()) {
		t.Errorf("Error message did not match expected. Expected: %s. Actual: %s", errorMessage, err.Error())
	}
}

func TestConvertV1IdentityWithSecretResourceIdToInternalIdentity(t *testing.T) {
	idV1 := CreateV1IdentityWithResourceIdSecretRef()

	secretClientId := "mysecretclientid"
	encodedSecretClientId := []byte(b64.StdEncoding.EncodeToString([]byte(secretClientId)))

	client := fake.NewSimpleClientset(&v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Data: map[string][]byte{
			"data": encodedSecretClientId,
		},
	})
	convertedIDInternal, err := ConvertV1IdentityToInternalIdentity(idV1, client)

	if err != nil {
		t.Error(err)
		t.Errorf("Failed to convert from v1 to internal AzureAssignedIdentity")
	}

	if !cmp.Equal(string(encodedSecretClientId), convertedIDInternal.Spec.ResourceID) {
		t.Errorf("Failed to convert from v1 to internal AzureIdentity %s %s", encodedSecretClientId, convertedIDInternal.Spec.ResourceID)
	}
}

func TestConvertV1AssignedIdentityToInternalAssignedIdentity(t *testing.T) {
	assignedIDV1 := CreateV1AssignedIdentity()

	convertedAssignedIDInternal, err := ConvertV1AssignedIdentityToInternalAssignedIdentity(assignedIDV1, nil)

	if err != nil {
		t.Errorf("Failed to convert from v1 to internal AzureAssignedIdentity")
	}

	assignedIDInternal := CreateInternalAssignedIdentity()

	if !cmp.Equal(assignedIDInternal, convertedAssignedIDInternal) {
		t.Errorf("Failed to convert from v1 to internal AzureAssignedIdentity")
	}

	// test no panics when azure identity or binding ref is nil
	assignedIDV1.Spec.AzureIdentityRef = nil
	assignedIDV1.Spec.AzureBindingRef = nil

	convertedAssignedIDInternal, err = ConvertV1AssignedIdentityToInternalAssignedIdentity(assignedIDV1, nil)

	if err != nil {
		t.Errorf("Failed to convert from v1 to internal AzureAssignedIdentity")
	}

	assignedIDInternal = CreateInternalAssignedIdentity()
	assignedIDInternal.Spec.AzureIdentityRef = &aadpodid.AzureIdentity{}
	assignedIDInternal.Spec.AzureBindingRef = &aadpodid.AzureIdentityBinding{}

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

func TestConvertV1PodIdentityExceptionToInternalPodIdentityException(t *testing.T) {
	podExceptionV1 := CreateV1PodIdentityException()

	convertedPodExceptionInternal := ConvertV1PodIdentityExceptionToInternalPodIdentityException(podExceptionV1)
	podExceptionInternal := CreateInternalPodIdentityException()

	if !cmp.Equal(convertedPodExceptionInternal, podExceptionInternal) {
		t.Errorf("Failed to convert from v1 to internal AzureAssignedIdentity")
	}
}
