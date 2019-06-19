package v1

func IsNamespacedIdentity(azureID *AzureIdentity) bool {
	if val, ok := azureID.Annotations[BehaviorKey]; ok {
		if val == BehaviorNamespaced {
			return true
		}
	}
	return false
}
