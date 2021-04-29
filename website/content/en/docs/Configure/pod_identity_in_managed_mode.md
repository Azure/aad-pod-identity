---
title: "Pod Identity in Managed Mode"
linkTitle: "Pod Identity in Managed Mode"
weight: 6
description: >
  In this mode, there is only the NMI component deployed in the cluster. The identity assignment needs to be manually performed.
---

> Available from 1.6.0 release

> NOTE: The AKS pod-managed identities add-on installs AAD Pod Identity in Managed mode.

## Introduction

Starting from 1.6.0 release, 2 modes of operation are supported for pod-identity
- Standard Mode
- Managed Mode

### Standard Mode

This is the default mode in which pod-identity will be deployed. In this mode, there are 2 components, MIC (Managed Identity Controller) and NMI (Node Managed Identity). MIC handles the identity assignment/removal from the underlying vm/vmss when new pods using the identity are created/deleted.

### Managed Mode

In this mode, there is only the NMI component deployed in the cluster. The identity assignment needs to be manually performed.

Deploy `aad-pod-identity` components to an RBAC-enabled cluster in managed mode:

- This installs NMI in managed mode in the kube-system namespace

```bash
kubectl apply -f https://raw.githubusercontent.com/Azure/aad-pod-identity/master/deploy/infra/managed-mode-deployment.yaml
```

**NOTE** Managed mode is only supported in namespaced mode. This ensures pods in namespace are only matched with `AzureIdentity` and `AzureIdentityBinding` in the same namespace.

#### Helm

AAD Pod Identity allows users to customize their installation via Helm.

```
helm repo add aad-pod-identity https://raw.githubusercontent.com/Azure/aad-pod-identity/master/charts
helm install aad-pod-identity aad-pod-identity/aad-pod-identity --set operationMode=managed
```

##### Values

For a list of customizable values that can be injected when invoking `helm install`, please see the [Helm chart configurations](https://github.com/Azure/aad-pod-identity/tree/master/charts/aad-pod-identity#configuration).


To assign the identity to the VM, run the following command -

```shell
az vm identity assign -g <VM resource group name> -n <VM name> --identities <resource ID of managed identity>
```

To assign the identity to VMSS, run the following command -

```shell
az vmss identity assign -g <VM resource group name> -n <VMSS name> --identities <resource ID of managed identity>
```

## Why use Managed mode

- Identity assignment on VM takes 10-20s and 40-60s in case of VMSS. In case of cronjobs or applications that require access to the identity and can't tolerate the assignment delay, it's best to use managed mode as the identity is manually pre-assigned to the VM/VMSS.
- In standard mode, MIC requires write permissions on VM/VMSS and Managed Identity Operator permission on all user assigned MSIs. While running in managed mode, since there is no MIC, the role assignments are not required.