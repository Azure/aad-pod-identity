---
title: "Setup AAD Pod Identity on Azure RedHat OpenShift (ARO)"
linkTitle: "Setup AAD Pod Identity on Azure RedHat OpenShift (ARO)"
weight: 2
description: >
  How to setup AAD Pod Identity on Azure RedHat OpenShift (ARO)
---

### Installation

#### Standard mode

The MIC component by default relies on `/etc/kubernetes/azure.json` to get cluster configuration and credentials. Since the `/etc/kubernetes/azure.json` doesn't exist in ARO clusters, the AAD Pod Identity components will need to be deployed with a dedicated SP/managed identity to provide access to Azure.

##### Helm

```shell
helm repo add aad-pod-identity https://raw.githubusercontent.com/Azure/aad-pod-identity/master/charts

# Helm 3
# If using managed identity to provide MIC access to Azure, then set adminsecret.clientID=msi and adminsecret.clientSecret=msi
# Set adminsecret.useMSI=false if using service principal to provide MIC access to Azure
helm install aad-pod-identity aad-pod-identity/aad-pod-identity \
    --set adminsecret.cloud=<azure cloud name> \
    --set adminsecret.subscriptionID=<subscription id> \
    --set adminsecret.resourceGroup=<node resource group> \
    --set adminsecret.vmType=vmss \
    --set adminsecret.tenantID=<tenant id> \
    --set adminsecret.clientID=<service principal clientID> \
    --set adminsecret.clientSecret=<service principal clientSecret> \
    --set-string adminsecret.useMSI=false \
    --set adminsecret.userAssignedMSIClientID=<ClientID from identity>
```

##### Using deployment yamls

If deploying using deployment yamls, refer to the [doc here](../deploy_aad_pod_dedicated_sp).

#### Managed mode

Follow the [docs here](../pod_identity_in_managed_mode) on how to install AAD Pod Identity in managed mode.

### Validate pod identity components are running

1. If deploying in standard mode, check the MIC pods are up and running.
2. Check if NMI is running on all nodes.
3. Follow the [doc here](../../troubleshooting/#ensure-that-iptables-rule-exists) to ensure the iptables rules exist.
