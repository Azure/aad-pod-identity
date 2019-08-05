# v1.5

### Features

- Support aad-pod-identity in init containers (https://github.com/Azure/aad-pod-identity/pull/191)
- Cleanup iptable chain and rule on uninstall (https://github.com/Azure/aad-pod-identity/pull/211)
- Remove dependency on azure.json (https://github.com/Azure/aad-pod-identity/pull/221)
- Add states for AzureAssignedIdentity and improve performance (https://github.com/Azure/aad-pod-identity/pull/219)
- System MSI cluster support (https://github.com/Azure/aad-pod-identity/pull/265)
- Leader election in MIC (https://github.com/Azure/aad-pod-identity/pull/277)
- Liveness probe for MIC and NMI (https://github.com/Azure/aad-pod-identity/pull/309)
- Application Exception (https://github.com/Azure/aad-pod-identity/pull/310)

### Bug Fixes

- Fix AzureIdentity with sevice principal (https://github.com/Azure/aad-pod-identity/pull/197)
- Determine resource manager endpoint based on cloud name (https://github.com/Azure/aad-pod-identity/pull/226)
- Fix incorrect resource endpoint with sp (https://github.com/Azure/aad-pod-identity/pull/251)
- Fix vmss identity deletion for ID in use (https://github.com/Azure/aad-pod-identity/pull/203)
- Fix removal of user assigned identity from nodes with system assigned (https://github.com/Azure/aad-pod-identity/pull/259)
- Handle case sensitive id check (https://github.com/Azure/aad-pod-identity/pull/271)
- Fix assigned id deletion when no identity exists (https://github.com/Azure/aad-pod-identity/pull/320)

### Other Improvements

- Use go modules (https://github.com/Azure/aad-pod-identity/pull/179)
- Log binary versions of MIC and NMI in logs (https://github.com/Azure/aad-pod-identity/pull/216)
- List CRDs via cache and avoid extra work on pod update (https://github.com/Azure/aad-pod-identity/pull/232)
- Reduce identity assignment times (https://github.com/Azure/aad-pod-identity/pull/199)
- NMI retries and ticker for periodic sync reconcile (https://github.com/Azure/aad-pod-identity/pull/272)
- Update error status code based on state (https://github.com/Azure/aad-pod-identity/pull/292)
- Process identity assignment/removal for nodes in parallel (https://github.com/Azure/aad-pod-identity/pull/305)
- Update base alpine image to 3.10.1 (https://github.com/Azure/aad-pod-identity/pull/324)