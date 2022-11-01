---
title: "Documentation"
linkTitle: "Documentation"
menu:
  main:
    weight: 20
---

## 📣 Announcement

**❗ IMPORTANT**: As of Monday 10/24/2022, AAD Pod Identity is **deprecated**. As mentioned in the [announcement](https://cloudblogs.microsoft.com/opensource/2022/01/18/announcing-azure-active-directory-azure-ad-workload-identity-for-kubernetes/), AAD Pod Identity has been replaced with [Azure Workload Identity](https://azure.github.io/azure-workload-identity). Going forward, we will no longer add new features to this project in favor of Azure Workload Identity. We will continue to provide critical bug fixes until Azure Workload Identity reaches general availability. Following that, we will provide CVE patches until September 2023, at which time the project will be archived.

AAD Pod Identity enables Kubernetes applications to access cloud resources securely with [Azure Active Directory](https://azure.microsoft.com/en-us/services/active-directory/) using User-assigned managed identity and Service Principal.

> Note: Configuring system-assigned managed identity with AAD Pod Identity to access cloud resources is not supported.

Using Kubernetes primitives, administrators configure identities and bindings to match pods. Then without any code modifications, your containerized applications can leverage any resource in the cloud that depends on AAD as an identity provider.

## Breaking Changes

### v1.8.4

The metadata header required flag is enabled by default to prevent SSRF attacks. Check [Metadata Header Required](./configure/feature_flags/#metadata-header-required-flag) for more information. To disable the metadata header check, set `--metadata-header-required=false` in NMI [container args](https://github.com/Azure/aad-pod-identity/blob/v1.8.6/deploy/infra/deployment-rbac.yaml#L483).

### v1.8.0

- The API version of Pod Identity's CRDs (`AzureIdentity`, `AzureIdentityBinding`, `AzureAssignedIdentity`, `AzurePodIdentityException`) have been upgraded from `apiextensions.k8s.io/v1beta1` to `apiextensions.k8s.io/v1`. For Kubernetes clusters with < 1.16, `apiextensions.k8s.io/v1` CRDs would not work. You can either:
  1. Continue using AAD Pod Identity v1.7.5 or
  2. Upgrade your cluster to 1.16+, then upgrade AAD Pod Identity.

  If AAD Pod Identity was previously installed using Helm, subsequent `helm install` or `helm upgrade` would not upgrade the CRD API version from `apiextensions.k8s.io/v1beta1` to `apiextensions.k8s.io/v1` (although `kubectl get crd -oyaml` would display `apiextensions.k8s.io/v1` since the API server internally converts v1beta1 CRDs to v1, it lacks a [structural schema](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#specifying-a-structural-schema), which is what AAD Pod Identity introduced in v1.8.0). If you wish to upgrade to the official v1 CRDs for AAD Pod Identity:

  ```bash
  kubectl apply -f https://raw.githubusercontent.com/Azure/aad-pod-identity/master/charts/aad-pod-identity/crds/crd.yaml
  ```

  With [managed mode](./configure/pod_identity_in_managed_mode) enabled, you can remove the unused AzureAssignedIdentity CRD if you wish.

  ```bash
  # MANAGED MODE ONLY!
  kubectl delete crd azureassignedidentities.aadpodidentity.k8s.io
  ```

### v1.7.5

- AAD Pod Identity has dropped Helm 2 starting from chart version 4.0.0/app version 1.7.5. To install or upgrade to the latest version of AAD Pod Identity, please use Helm 3 instead. Refer to this [guide](https://helm.sh/blog/migrate-from-helm-v2-to-helm-v3/) on how to migrate from Helm 2 to Helm 3.

### v1.7.2

- The `forceNameSpaced` helm configuration variable is removed. Use `forceNamespaced` instead to configure pod identity to run in namespaced mode.

### v1.7.1

- `azureIdentities` in `values.yaml` is converted to a map instead of a list of identities.

  The following is an example of the required change in `values.yaml` from helm chart 2.x.x to 3.x.x:

  ```diff
  -azureIdentities:
  -  - name: "azure-identity"
  -    # if not defined, then the azure identity will be deployed in the same namespace as the chart
  -    namespace: ""
  -    # type 0: MSI, type 1: Service Principal
  -    type: 0
  -    # /subscriptions/subscription-id/resourcegroups/resource-group/providers/Microsoft.ManagedIdentity/userAssignedIdentities/identity-name
  -    resourceID: "resource-id"
  -    clientID: "client-id"
  -    binding:
  -      name: "azure-identity-binding"
  -      # The selector will also need to be included in labels for app deployment
  -      selector: "demo"
  +azureIdentities:
  +  "azure-identity":
  +    # if not defined, then the azure identity will be deployed in the same namespace as the chart
  +    namespace: ""
  +    # type 0: MSI, type 1: Service Principal
  +    type: 0
  +    # /subscriptions/subscription-id/resourcegroups/resource-group/providers/Microsoft.ManagedIdentity/userAssignedIdentities/identity-name
  +    resourceID: "resource-id"
  +    clientID: "client-id"
  +    binding:
  +      name: "azure-identity-binding"
  +      # The selector will also need to be included in labels for app deployment
  +      selector: "demo"
  ```

### v1.7.0

- With [Azure/aad-pod-identity#842](https://github.com/Azure/aad-pod-identity/pull/842), aad-pod-identity no longer works on clusters with kubenet as the network plugin. For more details, please see [Deploy AAD Pod Identity in a Cluster with Kubenet](configure/aad_pod_identity_on_kubenet/).

  If you still wish to install aad-pod-identity on a kubenet-enabled cluster, set the helm chart value `nmi.allowNetworkPluginKubenet` to `true` in the helm command:

  ```bash
  helm (install|upgrade) ... --set nmi.allowNetworkPluginKubenet=true ...
  ```

### v1.6.0

With [Azure/aad-pod-identity#398](https://github.com/Azure/aad-pod-identity/pull/398), the [client-go](https://github.com/kubernetes/client-go) library is upgraded to v0.17.2, where CRD [fields are now case sensitive](https://github.com/kubernetes/kubernetes/issues/64612). If you are upgrading MIC and NMI from v1.x.x to v1.6.0, MIC v1.6.0+ will upgrade the fields of existing `AzureIdentity` and `AzureIdentityBinding` on startup to the new format to ensure backward compatibility. A configmap called `aad-pod-identity-config` is created to record and confirm the successful type upgrade.

However, for future `AzureIdentity` and `AzureIdentityBinding` created using v1.6.0+, the following fields need to be changed:

### `AzureIdentity`

| < 1.6.0          | >= 1.6.0         |
| ---------------- | ---------------- |
| `ClientID`       | `clientID`       |
| `ClientPassword` | `clientPassword` |
| `ResourceID`     | `resourceID`     |
| `TenantID`       | `tenantID`       |

### `AzureIdentityBinding`

| < 1.6.0         | >= 1.6.0        |
| --------------- | --------------- |
| `AzureIdentity` | `azureIdentity` |
| `Selector`      | `selector`      |

### `AzurePodIdentityException`

| < 1.6.0     | >= 1.6.0    |
| ----------- | ----------- |
| `PodLabels` | `podLabels` |


## Ready to get started?

To get started, see the [Getting Started](./getting-started/) page, or you can visit the [GitHub repo](https://github.com/Azure/aad-pod-identity).
