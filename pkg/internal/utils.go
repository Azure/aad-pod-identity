package internal

// import {
//     "strings"
// }

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

// func IsV1Object(string input) bool {
// 	return strings.contains(input, "apis/aadpodidentity.k8s.io/v1")
// }

// func IsV2Object(string input) bool {
// 	return strings.contains(input, "apis/aadpodidentity.k8s.io/v2")
// }