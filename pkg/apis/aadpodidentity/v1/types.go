package aadpodidentity

import (
	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

/*** Global data structures ***/

// AzureIdentity is the specification of the identity data structure.
//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AzureIdentity struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AzureIdentitySpec   `json:"spec"`
	Status AzureIdentityStatus `json:"status"`
}

// AzureIdentityBinding brings together the spec of matching pods and the identity which they can use.

//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AzureIdentityBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AzureIdentityBindingSpec   `json:"spec"`
	Status AzureIdentityBindingStatus `json:"status"`
}

//AzureAssignedIdentity contains the identity <-> pod mapping which is matched.

//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AzureAssignedIdentity struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AzureAssignedIdentitySpec   `json:"spec"`
	Status AzureAssignedIdentityStatus `json:"spec"`
}

/*** Lists ***/
//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AzureIdentityList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AzureIdentity `json:"items"`
}

//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AzureIdentityBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AzureIdentityBinding `json:"items"`
}

//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AzureAssignedIdentityList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AzureAssignedIdentityList `json:"items"`
}

/*** AzureIdentity ***/
//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type IdentityType int

const (
	UserAssignedMSI  IdentityType = 0
	ServicePrincipal IdentityType = 1
)

type AzureIdentitySpec struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// EMSI or Service Principle
	Type     IdentityType        `json:"type"`
	ID       string              `json:"id"`
	Password api.SecretReference `json:"password"`
	Replicas *int32              `json:"replicas"`
}

type AzureIdentityStatus struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
	AvailableReplicas int32 `json:"availableReplicas"`
}

/*** AzureIdentityBinding ***/
type MatchType int

const (
	Explicit MatchType = 0
	Selector MatchType = 1
)

// AzureIdentityBindingSpec matches the pod with the Identity.
// Used to indicate the potential matches to look for between the pod/deployment
// and the identities present..
type AzureIdentityBindingSpec struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
	AzureIdRef        *AzureIdentity `json:"azureidref"`
	MatchType         MatchType      `json:"matchtype"`
	MatchName         string         `json:"matchname"`
	// Weight is used to figure out which of the matching identities would be selected.
	Weight int `json:"weight"`
}

type AzureIdentityBindingStatus struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
	AvailableReplicas int32 `json:"availableReplicas"`
}

/*** AzureAssignedIdentitySpec ***/

// AzureAssignedIdentitySpec has the contents of Azure identity<->POD

type AzureAssignedIdentitySpec struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
	AzureIDRef        *AzureIdentity `json:"azureidref"`
	Pod               string         `json:"podref"`
	Replicas          *int32         `json:"replicas"`
}

// AzureAssignedIdentityStatus has the replica status of the resouce.

type AzureAssignedIdentityStatus struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
	AvailableReplicas int32 `json:"availableReplicas"`
}
