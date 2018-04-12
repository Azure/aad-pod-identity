
package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

/*** Global datastructures ***/

// AzureIdentity is the specification of the identity datastructure.
type AzureIdentity struct {
	metav1.TypeMeta             `json:",inline"`
	metav1.ObjectMeta           `json:"metadata,omitempty"`

	Spec	AzureIdentitySpec   `json:"spec"`
	Status  AzureIdentityStatus `json:"status"`
}

// AzureIdentityBinding brings together the spec of matching pods and the identity which they can use.
type AzureIdentityBinding struct {
	metav1.TypeMeta                    `json:",inline"`
	metav1.ObjectMeta                  `json:"metadata,omitempty"`

	Spec	AzureIdentityBindingSpec   `json:"spec"`
	Status  AzureIdentityBindingStatus `json:"status"`
}

//AzureAssignedIdentity contains the identity <-> pod mapping which is matched.
type AzureAssignedIdentity struct {
	metav1.TypeMeta                    `json:",inline"`
	metav1.ObjectMeta                  `json:"metadata,omitempty"`

	Spec AzureAssignedIdentitySpec     `json:"spec"`
	Status AzureAssignedIdentityStatus `json:"spec"`
}

/**** Lists ****/
type AzureIdentityList struct {
	metav1.TypeMeta       `json:",inline"`
	metav1.ListMeta       `json:"metadata"`

	Items []AzureIdentity `json:"items"`
}

type AzureIdentityBindingList struct {
	metav1.TypeMeta              `json:",inline"`
	metav1.ListMeta              `json:"metadata"`

	Items []AzureIdentityBinding `json:"items"`

}

type AzureAssignedIdentityList struct {
	metav1.TypeMeta                   `json:",inline"`
	metav1.ListMeta                   `json:"metadata"`

	Items []AzureAssignedIdentityList `json:"items"`
}

/*** AzureIdentity ***/
type IdentityType int

const (
	UserAssignedMSI IdentityType = 0
	ServicePrincipal IdentityType = 1
)

// AzureIdentitySpec specifies the identity. It can either be User assigned MSI or can be service principle based.
type AzureIdentitySpec struct {
	// EMSI or Service Principle
	Type  IdentityType       `json:"type"`
	Id  string               `json:"id"`
	Password SecretReference `json:"password"`
	Replicas  *int32         `json:"replicas"`
}

type AzureIdentityStatus struct {
	AvailableReplicas int32 `json:"availableRepliacs"`
}

/*** AzureIdentityBinding ***/
type MatchType int

const (
	Explicit  MatchType = 0
	Selector  MatchType = 1
)

// AzureIdentittyBindingSpec matches the pod with the Identity.
// Used to indicate the potential matches to look for between the pod/deployment
// and the identities present..
type AzureIdentityBindingSpec struct {
	AzureIdRef *AzureIdentity `json:"azureidref"`
	MatchType  MatchType      `json:"matchtype"`
	MatchName string          `json:"matchname"`
	Weight int                `json:"weight"`
}

type AzureIdentityBindingStatus struct {
	AvailableReplicas int32 `json:"availableRepliacs"`
}

/*** AzureAssignedIdentitySpec ***/
type AzureAsssignedIdentitySpec struct {
	AzureIdRef *AzureIdentity `json:"azureidref"`
	PodRef PodReference       `json:"podref"`
	Replicas  *int32          `json:"replicas"`
}

type AzureAssignedIdentityStatus struct {
	AvailableReplicas int32 `json:"availableRepliacs"`
}
