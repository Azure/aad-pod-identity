---
title: "Role Assignment"
linkTitle: "Role Assignment"
weight: 1
description: >
  Your cluster will need the correct role assignment configuration to perform Azure-related operations.
---

Your cluster will need the correct role assignment configuration to perform Azure-related operations such as assigning and un-assigning the identity on the underlying VM/VMSS. You can run the following commands to help you set up the appropriate role assignments for your cluster identity before deploying aad-pod-identity.

AKS and aks-engine clusters require an identity to communicate with Azure. This identity can be either a **managed identity** (in the form of system-assigned identity or user-assigned identity) or a **service principal**. This section explains various role assignments that need to be performed before using AAD Pod Identity. Without the proper role assignments, your Azure cluster will not have the correct permission to assign and un-assign identities from the underlying virtual machines (VM) or virtual machine scale sets (VMSS). 

In the case of self-managed clusters (manual installation of Kubernetes on Azure VMs), you'll need to assign a **user-assigned managed identity** to the VM or VMSS or use a **service principal**. This is required for MIC to perform Azure-related operations for assigning/un-assigning the identity required for applications.

```bash
export SUBSCRIPTION_ID="<SubscriptionID>"
export RESOURCE_GROUP="<AKSResourceGroup>"
export CLUSTER_NAME="<AKSClusterName>"

# Optional: if you are planning to deploy your user-assigned identities
# in a separate resource group instead of your node resource group
export IDENTITY_RESOURCE_GROUP="<IdentityResourceGroup>"

curl -s https://raw.githubusercontent.com/Azure/aad-pod-identity/master/hack/role-assignment.sh | bash
```

> Note: `<AKSResourceGroup>` is where your AKS cluster is deployed to.

## Introduction

Currently, [MIC](../../concepts/mic) uses one of the following two ways to authenticate with Azure:

1. [Managed Identity](https://docs.microsoft.com/en-us/azure/aks/use-managed-identity) (system-assigned identity or user-assigned identity)
2. [Service Principal](https://docs.microsoft.com/en-us/azure/aks/kubernetes-service-principal) through `/etc/kubernetes/azure.json`, which is available in every node, or credentials defined by environment variables;

> Clusters with managed identity are only compatible with AAD Pod Identity 1.5+.

## More on authentication methods

[`/etc/kubernetes/azure.json`](https://kubernetes-sigs.github.io/cloud-provider-azure/install/configs/) is a well-known JSON file in each node that provides the details about which method MIC uses for authentication:

| Authentication method            | `/etc/kubernetes/azure.json` fields used                                                    |
|----------------------------------|---------------------------------------------------------------------------------------------|
| System-assigned managed identity | `useManagedIdentityExtension: true` and `userAssignedIdentityID:""`                         |
| User-assigned managed identity   | `useManagedIdentityExtension: true` and `userAssignedIdentityID:"<UserAssignedIdentityID>"` |
| Service principal (default)      | `aadClientID: "<AADClientID>"` and `aadClientSecret: "<AADClientSecret>"`                   |

## Obtaining the ID of the managed identity / service principal

After your cluster is provisioned, depending on your cluster identity configuration, run one of the following commands to retrieve the **ID** of your managed identity or service principal, which will be used for role assignment in the next section:

| Cluster configuration                            | Command                                                                                                                                                                     |
|--------------------------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| AKS cluster with service principal               | `az aks show -g <AKSResourceGroup> -n <AKSClusterName> --query servicePrincipalProfile.clientId -otsv`                                                                      |
| AKS cluster with managed identity                | `az aks show -g <AKSResourceGroup> -n <AKSClusterName> --query identityProfile.kubeletidentity.clientId -otsv`                                                              |
| aks-engine cluster with service principal        | Use the client ID of the service principal defined in the API model                                                                                                         |
| aks-engine cluster with system-assigned identity | `az <vm|vmss> identity show -g <NodeResourceGroup> -n <VM|VMSS Name> --query principalId -otsv`                                                                             |
| aks-engine cluster with user-assigned identity   | `az <vm|vmss> identity show -g <NodeResourceGroup> -n <VM|VMSS Name> --query userAssignedIdentities -otsv`, then copy the `clientID` of the selected user-assigned identity |

## Performing role assignments

The roles [**Managed Identity Operator**](https://docs.microsoft.com/en-us/azure/role-based-access-control/built-in-roles#managed-identity-operator) and [**Virtual Machine Contributor**](https://docs.microsoft.com/en-us/azure/role-based-access-control/built-in-roles#virtual-machine-contributor) must be assigned to the cluster managed identity or service principal, identified by the **ID** obtained above, before deploying AAD Pod Identity so that it can assign and un-assign identities from the underlying VM/VMSS.

> For AKS cluster, the node resource group refers to the resource group with a `MC_` prefix, which contains all of the infrastructure resources associated with the cluster like VM/VMSS.

```bash
az role assignment create --role "Managed Identity Operator" --assignee <ID> --scope /subscriptions/<SubscriptionID>/resourcegroups/<NodeResourceGroup>
az role assignment create --role "Virtual Machine Contributor" --assignee <ID> --scope /subscriptions/<SubscriptionID>/resourcegroups/<NodeResourceGroup>
```

> RBAC and non-RBAC clusters require the same role assignments.

## User-assigned identities that are not within the node resource group

There are additional role assignments required if you wish to assign user-assigned identities that are not within the node resource group. You can run the following command to assign the **Managed Identity Operator** role with the identity resource group scope:

```bash
az role assignment create --role "Managed Identity Operator" --assignee <ID> --scope /subscriptions/<SubscriptionID>/resourcegroups/<IdentityResourceGroup>
```

To enable fine-grained control on which user-assigned identity the cluster has access to, run the following command:

```bash
az role assignment create --role "Managed Identity Operator" --assignee <ID>  --scope /subscriptions/<SubscriptionID>/resourcegroups/<IdentityResourceGroup>/providers/Microsoft.ManagedIdentity/userAssignedIdentities/<IdentityName>
```

## User-assigned managed identities for self-managed clusters

If you deploy the VMs and install Kubernetes instead of using tools like [aks-engine](https://github.com/Azure/aks-engine) or [capz](https://github.com/kubernetes-sigs/cluster-api-provider-azure), you'll need to assign the user-assigned managed identity to the underlying VMs.

### For VMSS

```bash 
az vmss identity assign -n <VMSS name> -g <rg> --identities <IdentityResourceID>
```

### For VMs

```bash
az vm identity assign -n <VM name> -g <rg> --identities <IdentityResourceID>
```
Repeat for all your worker node VMs.

## Reducing number of role assignments

Currently there's a limit of [2000 role assignments](https://docs.microsoft.com/en-us/azure/role-based-access-control/troubleshooting#azure-role-assignments-limit) allowed within an Azure subscription. Once you've hit this limit, you will not be able to assign new roles.

To reduce the number of role assignments, one thing you could do is instead of assigning the `Managed Identity Operator` role to managed identities individually, you could assign the `Managed Identity Operator` role to the resource group the managed identities belong to. Resources will inherit roles from the resource group, meaning you can create as many managed identities as you need and not affect the subscription's overall role assignment count.

## Useful links

- [Use managed identities in AKS](https://docs.microsoft.com/en-us/azure/aks/use-managed-identity)
- [Service principals with AKS](https://docs.microsoft.com/en-us/azure/aks/kubernetes-service-principal)
- [What are managed identities for Azure resources?](https://docs.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/overview)
