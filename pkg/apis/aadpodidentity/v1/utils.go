package v1

func IsNamespacedIdentity(azureId *AzureIdentity) bool {
	if val, ok := azureId.Annotations[BehaviorKey]; ok {
		if val == BehaviorNamespaced {
			return true
		}
	}
	return false
}
