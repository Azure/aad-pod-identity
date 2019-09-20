# v1.5.3

### Bug Fixes

- Fix concurrent map read and map write while updating stats ([#344](https://github.com/Azure/aad-pod-identity/pull/344))
- Fix list calls to use local cache inorder to reduce api server load ([#358](https://github.com/Azure/aad-pod-identity/pull/358))
- Clean up assigned identities if node not found ([#367](https://github.com/Azure/aad-pod-identity/pull/367))
- Fixes to identity operations on VMSS ([#379](https://github.com/Azure/aad-pod-identity/pull/379))
- Fix namespaced multiple binding/identity handling and verbose logs ([#388](https://github.com/Azure/aad-pod-identity/pull/388))

### Other Improvements

- Set Content-Type on token response ([#341](https://github.com/Azure/aad-pod-identity/pull/341))
- Redact client id in NMI logs ([#343](https://github.com/Azure/aad-pod-identity/pull/343))
- Add user agent to kube-api calls ([#353](https://github.com/Azure/aad-pod-identity/pull/353))
- Add resource and request limits ([#372](https://github.com/Azure/aad-pod-identity/pull/372))
- Add user agent to ARM calls ([#387](https://github.com/Azure/aad-pod-identity/pull/387))

# v1.5.2

### Bug Fixes

- Fix the token backward compat in host based token fetching ([#337](https://github.com/Azure/aad-pod-identity/pull/337))

# v1.5.1

### Bug Fixes

- Append NMI version to the `User-Agent` for adal only once ([#333](https://github.com/Azure/aad-pod-identity/pull/333))

### Other Improvements

- Change 'updateStrategy' for nmi DaemonSet to `RollingUpdate` ([#334](https://github.com/Azure/aad-pod-identity/pull/334))

# v1.5

### Features

- Support aad-pod-identity in init containers ([#191](https://github.com/Azure/aad-pod-identity/pull/191))
- Cleanup iptable chain and rule on uninstall ([#211](https://github.com/Azure/aad-pod-identity/pull/211))
- Remove dependency on azure.json ([#221](https://github.com/Azure/aad-pod-identity/pull/221))
- Add states for AzureAssignedIdentity and improve performance ([#219](https://github.com/Azure/aad-pod-identity/pull/219))
- System MSI cluster support ([#265](https://github.com/Azure/aad-pod-identity/pull/265))
- Leader election in MIC ([#277](https://github.com/Azure/aad-pod-identity/pull/277))
- Liveness probe for MIC and NMI ([#309](https://github.com/Azure/aad-pod-identity/pull/309))
- Application Exception ([#310](https://github.com/Azure/aad-pod-identity/pull/310))

### Bug Fixes

- Fix AzureIdentity with sevice principal ([#197](https://github.com/Azure/aad-pod-identity/pull/197))
- Determine resource manager endpoint based on cloud name ([#226](https://github.com/Azure/aad-pod-identity/pull/226))
- Fix incorrect resource endpoint with sp ([#251](https://github.com/Azure/aad-pod-identity/pull/251))
- Fix vmss identity deletion for ID in use ([#203](https://github.com/Azure/aad-pod-identity/pull/203))
- Fix removal of user assigned identity from nodes with system assigned ([#259](https://github.com/Azure/aad-pod-identity/pull/259))
- Handle case sensitive id check ([#271](https://github.com/Azure/aad-pod-identity/pull/271))
- Fix assigned id deletion when no identity exists ([#320](https://github.com/Azure/aad-pod-identity/pull/320))

### Other Improvements

- Use go modules ([#179](https://github.com/Azure/aad-pod-identity/pull/179))
- Log binary versions of MIC and NMI in logs ([#216](https://github.com/Azure/aad-pod-identity/pull/216))
- List CRDs via cache and avoid extra work on pod update ([#232](https://github.com/Azure/aad-pod-identity/pull/232))
- Reduce identity assignment times ([#199](https://github.com/Azure/aad-pod-identity/pull/199))
- NMI retries and ticker for periodic sync reconcile ([#272](https://github.com/Azure/aad-pod-identity/pull/272))
- Update error status code based on state ([#292](https://github.com/Azure/aad-pod-identity/pull/292))
- Process identity assignment/removal for nodes in parallel ([#305](https://github.com/Azure/aad-pod-identity/pull/305))
- Update base alpine image to 3.10.1 ([#324](https://github.com/Azure/aad-pod-identity/pull/324))