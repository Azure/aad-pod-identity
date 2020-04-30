# Role Assignment

Without the proper role assignment, your Azure cluster will not have the correct permission to perform Azure-related operations such as creating and removing identity assignments.

## Introduction

The [MIC](../../README.md#managed-identity-controller) component in AAD Pod Identity needs to authenticate with Azure to create and remove identity assignment from the underlying virtual machines (VM) or virtual machine scale sets (VMSS). Currently, MIC will use one of the following two ways to autheticate with Azure:

1. [Service principal](https://docs.microsoft.com/en-us/azure/aks/kubernetes-service-principal) through `/etc/kubernetes/azure.json` available in every node, or credential defined by environment variables;
2. [Managed identity](https://docs.microsoft.com/en-us/azure/aks/use-managed-identity) (it can be using a system-assigned identity or user-assigned identity)

> Clusters with managed identity are only compatible with AAD Pod Identity 1.5+.

## Performing Role Assignment

After your cluster is provisioned, depending on your clsuter configuration, run one of the following commands to retrieve the **Principal ID** of your identity:

| Cluster configurations                           | Commands                                                                                                 |
|--------------------------------------------------|----------------------------------------------------------------------------------------------------------|
| AKS cluster with service principal               | `az aks show -g <ResourceGroup> -n <ClusterName> --query servicePrincipalProfile.clientId -otsv`          |
| AKS cluster with managed identity                | `az aks show -g <ResourceGroup> -n <ClusterName> --query identityProfile.kubeletidentity.clientId -otsv` |
| aks-engine cluster with service principal        | Use the client ID of the service principal defined in the API model                                      |
| aks-engine cluster with system-assigned identity | `az <vm\|vmss> identity show -g <ResourceGroup> -n <VM\|VMSS Name> --query principalId -otsv`            |
| aks-engine cluster with user-assigned identity   | `az <vm\|vmss> identity show -g <ResourceGroup> -n <VM\|VMSS Name> --query userAssignedIdentities -otsv` |

The roles [Managed Identity Operator](https://docs.microsoft.com/en-us/azure/role-based-access-control/built-in-roles#managed-identity-operator) and [Virtual Machine Contributor](https://docs.microsoft.com/en-us/azure/role-based-access-control/built-in-roles#virtual-machine-contributor) must be assigned to the service principal / identiy before deploying AAD Pod Identity so that it can perform identity-related operations on Azure. You can run the following command to assign the role with the resource group scope:

```bash
az role assignment create --role "Virtual Machine Contributor" --assignee <PrincipalID> --scope /subscriptions/<SubscriptionID>/resourcegroups/<ResourceGroup>
az role assignment create --role "Managed Identity Operator" --assignee <PrincipalID> --scope /subscriptions/<SubscriptionID>/resourcegroups/<ResourceGroup>
```

To enable fine-grained control on which user-assigned identity the cluster has access to, run the following command:

```bash
az role assignment create --role "Managed Identity Operator" --assignee <PrincipalID>  --scope /subscriptions/<SubscriptionID>/resourcegroups/<ResourceGroup>/providers/Microsoft.ManagedIdentity/userAssignedIdentities/<IdentityName>
```

## VNet outside of the cluster resource group

If the VNET of your cluster is not located within the same resource group as the AKS cluster resource group, the [Virtual Machine Contributor](https://docs.microsoft.com/en-us/azure/role-based-access-control/built-in-roles#virtual-machine-contributor) role is needed against the VNet resource group:

```bash
az role assignment create --role "Virtual Machine Contributor" --assignee <PrincipalID>  --scope /subscriptions/<SubscriptionID>/resourcegroups/<VNetResourceGroup>
```

Optionally, it is possible to grant the managed identity additional role permissions using a custom role as documented [here](https://docs.microsoft.com/en-us/azure/aks/kubernetes-service-principal#networking).
