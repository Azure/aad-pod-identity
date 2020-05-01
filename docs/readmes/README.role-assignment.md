# Role Assignment

> This section only applies if the user-assigned identities you wish to assign to your workload are not within the cluster resource group. For AKS, the cluster resource group refers to the resource group with a `MC_` prefix.

Without the proper role assignments, your Azure cluster will not have the correct permission to assign and un-assign identities that are not within the cluster resource group.

## Introduction

The [MIC](../../README.md#managed-identity-controller) component in AAD Pod Identity needs to authenticate with Azure to create and remove identity assignment from the underlying virtual machines (VM) or virtual machine scale sets (VMSS). Currently, MIC will use one of the following two ways to authenticate with Azure:

1. [Service principal](https://docs.microsoft.com/en-us/azure/aks/kubernetes-service-principal) through `/etc/kubernetes/azure.json` available in every node, or credential defined by environment variables;
2. [Managed identity](https://docs.microsoft.com/en-us/azure/aks/use-managed-identity) (it can be using a system-assigned identity or user-assigned identity)

> Clusters with managed identity are compatible with AAD Pod Identity 1.5+.

## More on authentication method

[`/etc/kubernetes/azure.json`](https://github.com/kubernetes-sigs/cloud-provider-azure/blob/master/docs/cloud-provider-config.md#auth-configs) is a well-known JSON file in each node that provides the details about which method MIC uses for authentication:

| Authentication method                  | `/etc/kubernetes/azure.json` fields used                                                     |
|----------------------------------------|---------------------------------------------------------------------------------------------|
| System-assigned managed identity       | `useManagedIdentityExtension: true` and `userAssignedIdentityID:""`                         |
| User-assigned managed identity cluster | `useManagedIdentityExtension: true` and `userAssignedIdentityID:"<UserAssignedIdentityID>"` |
| Service principal (default)            | `aadClientID: "<AADClientID>"` and `aadClientSecret: "<AADClientSecret>"`                   |

## Performing Role Assignment

After your cluster is provisioned, depending on your cluster configuration, run one of the following commands to retrieve the **Principal ID** of your service principal / managed identity:

| Cluster configuration                            | Command                                                                                                  |
|--------------------------------------------------|----------------------------------------------------------------------------------------------------------|
| AKS cluster with service principal               | `az aks show -g <ClusterResourceGroup> -n <ClusterName> --query servicePrincipalProfile.clientId -otsv`          |
| AKS cluster with managed identity                | `az aks show -g <ClusterResourceGroup> -n <ClusterName> --query identityProfile.kubeletidentity.clientId -otsv` |
| aks-engine cluster with service principal        | Use the client ID of the service principal defined in the API model                                      |
| aks-engine cluster with system-assigned identity | `az <vm\|vmss> identity show -g <ClusterResourceGroup> -n <VM\|VMSS Name> --query principalId -otsv`            |
| aks-engine cluster with user-assigned identity   | `az <vm\|vmss> identity show -g <ClusterResourceGroup> -n <VM\|VMSS Name> --query userAssignedIdentities -otsv` |

The roles [Managed Identity Operator](https://docs.microsoft.com/en-us/azure/role-based-access-control/built-in-roles#managed-identity-operator) must be assigned to the cluster service principal / managed identity before deploying AAD Pod Identity so that it can assign and un-assign identities that are not within the cluster resource group. You can run the following command to assign the role with the identity resource group scope:

```bash
az role assignment create --role "Managed Identity Operator" --assignee <PrincipalID> --scope /subscriptions/<SubscriptionID>/resourcegroups/<IdentityResourceGroup>
```

To enable fine-grained control on which user-assigned identity the cluster has access to, run the following command:

```bash
az role assignment create --role "Managed Identity Operator" --assignee <PrincipalID>  --scope /subscriptions/<SubscriptionID>/resourcegroups/<IdentityResourceGroup>/providers/Microsoft.ManagedIdentity/userAssignedIdentities/<IdentityName>
```
