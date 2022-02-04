---
title: "Feature Flags"
linkTitle: "Feature Flags"
weight: 5
description: >
  Optional configuration feature flags.
---

## Enable Scale Features flag

> Available from 1.5.3 release
> This flag is enabled by default starting from v1.8.1 release

AAD Pod Identity adds labels to `AzureAssignedIdentities` which denote the nodename, podname and podnamespace.
When the optional parameter `enableScaleFeatures` is set to `true`, the NMI watches for `AzureAssignedIdentities` will do a label based filtering on
the nodename label. This approach is taken because currently Kubernetes does not support field selectors in CRD watches. This reduces the load which
NMIs add on API server. When this flag is enabled, NMI will no longer work for `AzureAssignedIdentities` which were created before 1.5.3-rc5, since
they don't have the labels. Hence please note that this flag renders your setup incompatible with releases before 1.5.3-rc5.

## Batch Create Delete flag

> Available from 1.5.3 release

MIC groups operations based on nodes/VMSS during the given cycle. With `createDeleteBatch` parameter we can
tune the number of operations (CREATE/DELETE/UPDATE) to the API server which are performed in parallel in the context of a
node/VMSS.

## Client QPS flag

> Available from 1.5.3 release

Aad-pod-identity has a new flag clientQps which can be used to control the total number of client operations performed per second
to the API server by MIC.

## Block Instance Metadata flag

The Azure Metadata API includes endpoints under `/metadata/instance` which
provide information about the virtual machine. You can see examples of this
endpoint in [the Azure documentation](https://docs.microsoft.com/en-us/azure/virtual-machines/linux/instance-metadata-service#retrieving-all-metadata-for-an-instance).

Some of the information returned by this endpoint may be considered sensitive
or secret. The response includes information on the operating system and image,
tags, resource IDs, network, and VM custom data.

This information is legitimately useful for many use cases, but also presents a
risk. If an attacker can exploit a vulnerability that allows them to read from
this endpoint, they may be able to access sensitive information even if the
vulnerable Pod does not use Managed Identity.

The `blockInstanceMetadata` flag for NMI will intercept any  requests to this
endpoint from Pods which are not using host networking and return an HTTP 403
Forbidden response. This flag is disabled by default to maximize compatibility.
Users are encouraged to determine if this option is relevant and beneficial for
their use cases.

## ImmutableUserMSIs flag

> Available from 1.5.4 release

Aad-pod-identity has a new flag `immutable-user-msis` which can be used to prevent deletion of specified identities from VM/VMSS.
The list is comma separated. Example: 00000000-0000-0000-0000-000000000000,11111111-1111-1111-1111-111111111111

## Metadata header required flag

> Available from 1.6.0 release

> This flag is enabled by default starting from v1.8.4 release

When you query the Instance Metadata Service, you must provide the header `Metadata: true` to ensure the request was not unintentionally redirected. You can see examples of this header in [the Azure documentation](https://docs.microsoft.com/en-us/azure/virtual-machines/linux/instance-metadata-service#using-headers).

This is critical especially when you [acquire an access token](https://docs.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/how-to-use-vm-token#get-a-token-using-http) as a mitigation against Server Side Request Forgery (SSRF) attack.

The `metadataHeaderRequired` flag for NMI will block all requests without Metadata header and return an HTTP 400 response. This flag is disabled by default for compatibility, but recommended for users to enable this feature.

## Set Retry-After header in NMI response

> Available from v1.8.2 release

NMI currently has internal retries to handle delays in the identity assignment when the pod requests for a token. In case of clients that have shorter timeouts, the retries can be terminated and the client will not receive a token in the first attempt. This feature flag when enabled will set the `Retry-After` header to 20s in the NMI response to the client and return a HTTP 503 response. The SDK used by the client will retry the request after 20s.

### How to enable this feature

While enabling this feature, you must also disable the internal retries in NMI.

- If using the [yaml](../../getting-started/installation/#quick-install) to deploy aad-pod-identity, you can enable this feature by setting the `--set-retry-after-header=true` flag in the NMI container.
  - Set `--retry-attempts-for-created=1`, `--retry-attempts-for-assigned=1` and `--find-identity-retry-interval=1` flags in the NMI container to disable the internal retries in NMI.
- If using [helm](../../getting-started/installation/#helm) to deploy aad-pod-identity, you can enable this feature by setting `nmi.setRetryAfterHeader=true` as part of helm install/upgrade.
  - Set `nmi.retryAttemptsForCreated=1`, `nmi.retryAttemptsForAssigned=1` and `nmi.findIdentityRetryIntervalInSeconds=1` flags in the helm install/upgrade command to disable the internal retries in NMI.

## Enable deletion of conntrack entries

> Available from v1.8.7 release

NMI redirects Instance Metadata Service (IMDS) requests to itself by setting up iptables rules after it starts running on the node.
However, these rules are not applicable to pre-existing connections. In such a scenario, the token request will be directly sent to IMDS instead of being intercepted by NMI. What this means is that the workload pod that runs before the NMI pod on the node can access identities that it doesn't have access to.
The `enable-conntrack-deletion` flag enables deletion of entries for pre-existing connections to IMDS endpoint, this causes applications which had pre-existing connections to be intercepted by NMI.

### How to enable this feature

- If using the [yaml](../../getting-started/installation/#quick-install) to deploy aad-pod-identity, you can enable this feature by setting the `--enable-conntrack-deletion=true` flag in the NMI container.
- If using [helm](../../getting-started/installation/#helm) to deploy aad-pod-identity, you can enable this feature by setting `nmi.enableConntrackDeletion=true` as part of helm install/upgrade.
