# Quickstart Demo - AAD Pod Identity

The quickstart demo for AAD Pod Identity is a single script designed to steamline and expedite the installation and setup of AAD Pod Identity for a demo and/or POC with a minimum amount of file editing and Azure infrastructure needed to showcase it's core functionality. If you are unfamiliar with the concept of AAD Pod Identity, it may be best to visit the main [Getting Started](https://github.com/Azure/aad-pod-identity#getting-started) page, and review all the concepts.

## Getting Started
> **NOTE: This quickstart makes the following assumptions:**
> * You currently have an AKS cluster deployed and it is set to your current context
> * You will be deploying the azureidentity and azureidentitybinding in the K8 default namespace
> * Your AKS cluster is RBAC enabled
> * You have admin config of the AKS cluster
> * You will be using a Managed System Identity and not a Managed User Identity
> * The latest Azure CLI installed
> * jq and sed installed | This script was designed to be run in a Linux BASH environment

> If any of the assumptions are not true, it may best best to opt for the full lenght [AAD Pod Tutorial](https://github.com/Azure/aad-pod-identity/tree/master/docs/tutorial#aad-pod-identity-tutorial) or reference the [Getting Started](https://github.com/Azure/aad-pod-identity#getting-started) page.

### 1. Edit the aad-pod-identity-demo-config.json file
The aad-pod-identity-demo-config.json file is the only file you will have to edit to get AAD Pod Identity quickstart demo deployed. Please open the json file and edit the following properties:

* ASSETSRESOURCEGROUPNAME
* ACCESSRESOURCEGROUPNAME
* NOACCESSRESOURCEGROUPNAME
* LOCATION
* MSINAME
* PODIDENTITYLABEL
* PODIDENTITYJSONFILENAME
* AZUREIDENTITYBINDINGJSONFILENAME

Once you have finished editing the aad-pod-identity-demo-config.json to your specific configuration, it should look similar to the example below.

```
{
    "ASSETSRESOURCEGROUPNAME": "aad-pod-identity-assets",
    "ACCESSRESOURCEGROUPNAME": "aad-pod-identity-access",
    "NOACCESSRESOURCEGROUPNAME": "aad-pod-identity-noaccess",
    "LOCATION": "eastus2",
    "MSINAME": "pod-identity-acct",
    "PODIDENTITYLABEL": "use-pod-identity",
    "PODIDENTITYJSONFILENAME": "aadpodidentity.json",
    "AZUREIDENTITYBINDINGJSONFILENAME": "azureidentitybindings.json"
}
```

Ensure you have saved the configuration file before proceeding to the next step.

### 2. Deploy the demo using the aad-pod-identity-quickstart.sh script
The aad-pod-identity-quickstart.sh script takes two parameters. The **first** parameter is the action of the script, and the two available valures are **deploy** and **clean**. The **second** parameter of the script is the file path of the aad-pod-identity-demo-config.json configuration file. The script will read in the values of the configuration file to deploy out the demo environment. 

#### Example command for deploying the AAD Pod Identity Demo environment

```
/aad-pod-identity/deploy/demo/quickstart$ ./aad-pod-identity-quickstart.sh deploy ./aad-pod-identity-demo-config.json
```

### 3. Test AAD Pod Identity Access
The example below is using the default AAD Pod Identity label created from the property PODIDENTITYLABEL in the aad-pod-identity-demo-config.json file. In this test we will create a pod using the azure-cli image, attaching the necessary label for the pod to use the MSI for Azure access. Once you exec into the pod, you will log into Azure as the MSI identity, then issue a command to create a VNet in the access resource group. Since the MSI has been granted contributor access to the access resource group, the creation of the VNet will happen with no issue.

```
kubectl run myaadpodaccess -it --image=mcr.microsoft.com/azure-cli --labels="aadpodidbinding=use-pod-identity" --restart=Never

az login --identity

az network vnet create --name myVirtualNetwork1 --resource-group <Access Resource Group> --subnet-name default
```

### 4. Test AAD Pod Identity Denied Access
Similar to the test in step 3. We will create a pod using the azure-cli image, attaching the necessary label for the pod to use the MSI for Azure access. In this case, once you exec into the pod, and log into Azure with the MSI assigned to the pod, you will attempt to create another VNet in a resource group where the MSI only has read access. You will receive an error for not having the necessary permissions to create the VNet.

```
kubectl run myaadpodnoaccess -it --image=mcr.microsoft.com/azure-cli --labels="aadpodidbinding=use-pod-identity" --restart=Never

az login --identity

az network vnet create --name myVirtualNetwork2 --resource-group <No Access Resource Group> --subnet-name default
```

### 5. Remove the demo using the aad-pod-identity-quickstart.sh script

#### Example command for removing (clean) the AAD Pod Identity Demo environment
Using the same aad-pod-identity-quickstart.sh script, we can simply use the **clean** parameter to remove all the Azure assets created to demo AAD Pod Identity. Similar to the same command from step 2 to deploy teh AAD Pod Identity demo, just change **deploy** to **clean** as shown in the example below.

```
/aad-pod-identity/deploy/demo/quickstart$ ./aad-pod-identity-quickstart.sh clean ./aad-pod-identity-demo-config.json
```

### Summary
I hope you found this script helpful. This quickstart demo aim is to allow you to quickly showcase and experience how to utlize Azure Managed System Identities for authentication for your pods being deployed in your Azure AKS environment. The Azure Compute Upsteam Program Team is hard at work on simplifying all of the security feature experience on the Azure platform.  
