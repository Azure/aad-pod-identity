# System MSI cluster support
> Available from 1.5 release

## Introduction

The MIC component in aad-pod-identity needs to authenticate with the cloud to assign and remove user assigned identities onto
virtual machines (VMS) or virtual machine scale sets(VMSS). This authentication is performed using either the cluster credentials
obtained from azure.json in AKS/aks-engine clusters or credentials given via environment variables.

MIC can authenticate using the following options:
1. Service principal
2. System assigned MSI
3. User assigned MSI

The rest of the README describes the prerequisite role assignments to be performed for using MSI and how to configure MIC to use system assigned/user assigned MSI.

## Pre-requisites - role assignments
MIC is responsible for performing operations such as assigning user assigned identity to the underlying vm or vmss which makes up the
nodes in the Kubernetes cluster. The system/user assigned MSI needs to have role assignments authorizing such operations on the vms/vmss
and also operations on the user assigned identities.

After the cluster is created, run these commands to retrieve the cluster's user assigned principal id:

**For managed AKS clusters**
```bash
az aks show -g <resource group> -n <aks cluster name> --query identityProfile.kubeletidentity.clientId -o tsv
```

**For aks-engine clusters with VMS nodes**
```bash
az vm identity show -g <resource group> -n <vm name> -o tsv
```

**For aks-engine clusters with VMSS nodes**
```bash
az vmss identity show -g <resource group>  -n <vmss scaleset name> -o tsv
```

The type in the output of the above command will identify the user assigned MSI. Please record the principal id.

For creating a role assignment to authorize assignment/removal of user assigned identities on VMS/VMSS, run the following commands:
```bash
az role assignment create --role "Virtual Machine Contributor" --assignee <principal id from above command>  --scope /subscriptions/<sub id>/resourcegroups/<resource group name>
az role assignment create --role "Managed Identity Operator" --assignee <principal id from above command>  --scope /subscriptions/<sub id>/resourcegroups/<resource group name>
```

Now to ensure that the operations are allowed on individual identity, perform the following for every identity in use:
```bash
az role assignment create --role "Managed Identity Operator" --assignee <principal id from above command>  --scope /subscriptions/<subscription id>/resourcegroups/<resource group name>/providers/Microsoft.ManagedIdentity/userAssignedIdentities/<identity name>
```
Note: If the VNET is located outside of the AKS resource group, the following command will need to be run on the VNET resource group as well.
```bash
az role assignment create --role "Virtual Machine Contributor" --assignee <principal id from above command>  --scope /subscriptions/<sub id>/resourcegroups/<resource group name>
```
Optionally, it is possible to grant the managed identity additional role permissions using a custom role as documented below:

https://docs.microsoft.com/en-us/azure/aks/kubernetes-service-principal#networking

## Authentication method
In case the azure.json is used, the following keys indicates whether the cluster is configured with system assigned or user assigned identity:
```UseManagedIdentityExtension``` shows that MSI is to be used. If ```UserAssignedIdentityID``` is set, then the user assigned
identity is used, otherwise system assigned identity is used for authentication.

In case where the azure.json is not used and the environment variables are used, the following variables are used to setup the configuration:
```USE_MSI``` is to setup MSI. If the ```USER_ASSIGNED_MSI_CLIENT_ID``` is used then user assigned identity is used, otherwise system assigned identity is used.
