package v1

import (
	api "k8s.io/api/core/v1"
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

const (
	CRDGroup    = "aadpodidentity.k8s.io"
	CRDVersion  = "v1"
	CRDLabelKey = "aadpodidbinding"

	BehaviorKey = "aadpodidentity.k8s.io/Behavior"
	// Namespaced Behavior
	BehaviorNamespaced = "namespaced"
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
	Status AzureAssignedIdentityStatus `json:"Status"`
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

	Items []AzureAssignedIdentity `json:"items"`
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
	// UserAssignedMSI or Service Principal
	Type IdentityType `json:"type"`

	// User assigned MSI resource id.
	ResourceID string `json:"resourceid"`
	//Both User Assigned MSI and SP can use this field.
	ClientID string `json:"clientid"`

	//Used for service principal
	ClientPassword api.SecretReference `json:"clientpassword"`
	// Service principal tenant id.
	TenantID string `json:"tenantid"`
	// For service principal. Option param for specifying the  AD details.
	ADResourceID string `json:"adresourceid"`
	ADEndpoint   string `json:"adendpoint"`

	Replicas *int32 `json:"replicas"`
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

//AssignedIDState -  State indicator for the AssignedIdentity
type AssignedIDState int

const (
	//Created - Default state of the assigned identity
	Created AssignedIDState = 0
	//Assigned - When the underlying platform assignment of EMSI is complete
	//the state moves to assigned
	Assigned AssignedIDState = 1
)

const (
	AzureIDResource         = "azureidentities"
	AzureIDBindingResource  = "azureidentitybindings"
	AzureAssignedIDResource = "azureassignedidentities"
)

// AzureIdentityBindingSpec matches the pod with the Identity.
// Used to indicate the potential matches to look for between the pod/deployment
// and the identities present..
type AzureIdentityBindingSpec struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
	AzureIdentity     string `json:"azureidentity"`
	Selector          string `json:"selector"`
	// Weight is used to figure out which of the matching identities would be selected.
	Weight int `json:"weight"`
}

type AzureIdentityBindingStatus struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
	AvailableReplicas int32 `json:"availableReplicas"`
}

/*** AzureAssignedIdentitySpec ***/

//AzureAssignedIdentitySpec has the contents of Azure identity<->POD
type AzureAssignedIdentitySpec struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
	AzureIdentityRef  *AzureIdentity        `json:"azureidentityref"`
	AzureBindingRef   *AzureIdentityBinding `json:"azurebindingref"`
	Pod               string                `json:"pod"`
	PodNamespace      string                `json:"podnamespace"`
	NodeName          string                `json:"nodename"`

	Replicas *int32 `json:"replicas"`
}

// AzureAssignedIdentityStatus has the replica status of the resouce.
type AzureAssignedIdentityStatus struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Status            string `json:"status"`
	AvailableReplicas int32  `json:"availableReplicas"`
}
