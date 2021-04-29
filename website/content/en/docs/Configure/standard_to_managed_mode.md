---
title: "Migrating from Standard to Managed Mode"
linkTitle: "Migrating from Standard to Managed Mode"
weight: 6
description: >
  Migrating from Standard to Managed mode for AAD Pod Identity
---

> Available from 1.6.0 release

## Introduction

AAD Pod Identity supports 2 modes of operation:

1. Standard Mode: In this mode, there is MIC and NMI components deployed to the cluster. MIC handles assigning/un-assigning the identity to the underlying VM/VMSS. NMI will intercept token request, validate if the pod has access to the identity it's requesting a token for and fetch the token on behalf of the application.
2. Managed Mode: In this mode, there is only NMI. The identity needs to be manually assigned and managed by the user. Refer to [this doc](../pod_identity_in_managed_mode) for more details on this mode.

## Steps to migrate AAD Pod Identity from Standard to Managed mode

If you already have AAD Pod Identity setup with Standard mode and would like to migrate to Managed mode:

> NOTE: AAD Pod Identity in Managed Mode only works in namespaced mode. This means the `AzureIdentity` and `AzureIdentityBinding` needs to be in the same namespace as the application pod referencing it. This it to ensure RBAC best practices. If you're running in non-namespace mode, move the `AzureIdentity` and `AzureIdentityBinding` to the correct namespaces before proceeding with the steps.

1. Assign the pod identities to the VM/VMSS:

    To assign the identity to the VM, run the following command:

    ```shell
    az vm identity assign -g <VM resource group name> -n <VM name> --identities <resource ID of managed identity>
    ```

    To assign the identity to VMSS, run the following command:

    ```shell
    az vmss identity assign -g <VM resource group name> -n <VMSS name> --identities <resource ID of managed identity>
    ```

1. Delete the MIC deployment and NMI daemonset

    ```shell
    kubectl delete deploy <mic deployment name> -n <namespace>
    kubectl delete daemonset <nmi daemonset name> -n <namespace>
    ```

    Delete the MIC service accounts and cluster-role

    ```shell
    kubectl delete sa aad-pod-id-mic-service-account -n <namespace>
    kubectl delete clusterrole aad-pod-id-mic-role
    kubectl delete clusterrolebinding aad-pod-id-mic-binding
    ```

1. Delete AzureAssignedIdentity custom resource definition

    The `AzureAssignedIdentity` is created and managed by MIC in standard mode. This is not required for managed mode.

    Refer to [this doc](../../troubleshooting/#unable-to-remove-azureassignedidentity-after-mic-pods-are-deleted) on how to delete the `AzureAssignedIdentities`.

1. Install AAD Pod Identity in managed mode

    Refer to [this doc](../pod_identity_in_managed_mode) on how install AAD Pod Identity in managed mode.
