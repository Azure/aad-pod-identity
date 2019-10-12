## Feature flags

## Enable Scale Features flag
> Available from 1.5.3 release

Aad-pod-identity adds labels to AzureAssignedIdentities which denote the nodename, podname and podnamespace.
When the optional parameter `enabledScaleFeatures` is set to 'true', the NMI watches for AzureAssignedIdentities will do a label based filtering on
the nodename label. This approach is taken because currently K8s does not support field selectors in CRD watches. This reduces the load which
NMIs add on API server. When this flag is enabled, NMI will no longer work for AzureAssignedIdentities which were created before 1.5.3-rc5, since
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