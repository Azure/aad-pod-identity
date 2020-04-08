# Quickstart Demo - AAD Pod Identity

The quickstart demo for AAD Pod Identity is a set of scripts to steamline and expedite the installation and setup of AAD Pod Identity for a demo and/or POC with the minimum amount of infrastructure and file editing needed to showcase it's core functionality. It is recommended to run the demo from the Azure Cloud Shell, as all the tooling needed is available in the Azure Cloud Shell terminal environment. If you are unfamiliar with the concept of AAD Pod Identity, it may be best to visit the main [Getting Started](https://github.com/Azure/aad-pod-identity#getting-started) page, and review all the concepts.

## Getting Started
> **NOTE: This quickstart makes the following assumptions:**
> * You currently have an AKS cluster deployed
> * Your AKS cluster is RBAC enabled
> * You have admin config of the AKS cluster
> * You will be using a Managed System Identity and not a Managed User Identity

> If any of the assumptions are not true, it may best best to opt for the full lenght [AAD Pod Tutorial](https://github.com/Azure/aad-pod-identity/tree/master/docs/tutorial#aad-pod-identity-tutorial) or reference the [Getting Started](https://github.com/Azure/aad-pod-identity#getting-started) page.

### 1. Deploy the aad-pod-identity-quickstart.sh

```
/aad-pod-identity/deploy/demo/quickstart$ ./aad-pod-identity-quickstart.sh
```




