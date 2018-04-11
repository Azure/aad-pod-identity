
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AzureIdentity is the specification of the identity datastructure.
type AzureIdentity struct {
	metav1.TypeMeta             `json:",inline"`
	metav1.ObjectMeta           `json:"metadata,omitempty"`

	Spec	AzureIdentitySpec   `json:"spec"`
	Status  AzureIdentityStatus `json:"status"`
}

type IdentityType int

const (
	UserAssignedMSI IdentityType = 0
	ServicePrincipal IdentityType = 1
)

type AzureIdentitySpec struct {
	// EMSI or Service Principle
	Type  IdentityType `json:"type"`
	Id  string      `json:"id"`
	Password SecretReference
}

type AzureIdentityStatus {
	AvailableReplicas int32 `json:"availableRepliacs"`
}


type AzureIdentityBinding struct {
	metav1.TypeMeta                    `json:",inline"`
	metav1.ObjectMeta                  `json:"metadata,omitempty"`

	Spec	AzureIdentityBindingSpec   `json:"spec"`
	Status  AzureIdentityBindingStatus `json:"status"`
}

type MatchType int

const (
	Explicit  MatchType = 0
	Selector  MatchType = 1
)

type AzureIdentityBindingSpec struct {
	AzureIdRef *AzureIdentity
	MatchType  MatchType
	MatchName string
	Weight int
}

type AzureAssignedIdentity struct {
	AzureIdRef *AzureIdentity
	PodRef PodReference
}




