# Quickstart Demo - AAD Pod Identity

The quickstart demo for AAD Pod Identity is a set of scripts to steamline and expedite the installation and setup of AAD Pod Identity for a demo and/or POC with the minimum amount of infrastructure and file editing needed to showcase it's core functionality. It is recommended to run the demo from the Azure Cloud Shell, as all the tooling needed is available in the Azure Cloud Shell terminal environment. If you are unfamiliar with the concept of AAD Pod Identity, it may be best to visit the main [Getting Started](https://github.com/Azure/aad-pod-identity#getting-started) page, and review all the concepts.

## Getting Started
> **NOTE: This quickstart makes the following assumptions:**
> * You currently have an AKS cluster deployed and it is set to your current context
> * You will be deploying the azureidentity and azureidentitybinding in the K8 default namespace
> * Your AKS cluster is RBAC enabled
> * You have admin config of the AKS cluster
> * You will be using a Managed System Identity and not a Managed User Identity

> If any of the assumptions are not true, it may best best to opt for the full lenght [AAD Pod Tutorial](https://github.com/Azure/aad-pod-identity/tree/master/docs/tutorial#aad-pod-identity-tutorial) or reference the [Getting Started](https://github.com/Azure/aad-pod-identity#getting-started) page.

### 1. Deploy the aad-pod-identity-quickstart.sh
If you need to make changes to the default variables of the script, please due prior to running the script. The top of the script has and area that has been identified to make changes to the default script variable settings.

```
/aad-pod-identity/deploy/demo/quickstart$ ./aad-pod-identity-quickstart.sh
```

Once the script has completed. Verify that you have both the azureidentity and azureidentitybinding setup in the cluster.
```
kubectl get azureidentity
kubectl get azureidentitybinding
```




