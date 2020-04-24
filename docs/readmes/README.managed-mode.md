# Pod Identity in Managed mode
> Available from 1.6.0 release

## Introduction

Starting from 1.6.0 release,2 modes of operation are supported for pod-identity
- Standard Mode
- Managed Mode

### Standard Mode

This is the default mode in which pod-identity will be deployed. In this mode, there are 2 components, MIC (Managed Identity Controller) and NMI (Node Managed Identity). MIC handles the identity assignment/removal from the underlying vm/vmss when new pods using the identity are created/deleted.

### Managed Mode

In this mode, there is only the NMI component deployed in the cluster. The identity assignment needs to be manually performed.

Deploy `aad-pod-identity` components to an RBAC-enabled cluster in managed mode:

```bash
kubectl apply -f https://raw.githubusercontent.com/Azure/aad-pod-identity/master/deploy/infra/managed-mode-deployment.yaml
```

To assign the identity to the VM, run the following command -

```shell
az vm identity assign -g <VM resource group name> -n <VM name> --identities <resource ID of managed identity>
```

To assign the identity to VMSS, run the following command -

```shell
az vmss identity assign -g <VM resource group name> -n <VM name> --identities <resource ID of managed identity>
```

## Why use Managed mode

- Identity assignment on VM takes 10-20s and 40-60s in case of VMSS. In case of cronjobs or applications that require access to the identity and can't tolerate the assignment delay, it's best to use managed mode as the identity is manually pre-assigned to the VM/VMSS.
- In standard mode, MIC requires write permissions on VM/VMSS and Managed Identity Operator permission on all user assigned MSIs. While running in managed mode, since there is no MIC, the role assignments are not required.