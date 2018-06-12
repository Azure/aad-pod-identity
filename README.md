# Project Status: Beta

----

Applications running on the POD on Azure Container Service (AKS/ACS Engine) require access to identities in Azure Active Directory (AAD)to access resources that use an identity provider. AAD provides a construct called a Service Principal that allows applications to assume identities with limited permissions, and Managed Service Identity (MSI) - automatically generated and rotated credentials that easily retrieved by an application at run-time to authenticate as a service principal. 
An cluster admin configures the Azure Identity Binding to the Pod. Without any change of auth code the application running on the pod works on the cluster.

----

## Design

The detailed design of the project can be found in the following docs:

- [Concept](https://github.com/Azure/aad-pod-identity/blob/master/docs/design/concept.md)
- [Block Diagram](https://github.com/Azure/aad-pod-identity/blob/master/docs/design/concept.png)

# Managed Identity Controller (MIC)

This controller watches for pod changes through the api server and caches pod to admin configured azure identity map.

# Node Managed Identity (NMI)

The authorization request of fetching Service Principal Token from MSI endpoint is sent to a standard Instance Metadata endpoint which is redirected to the NMI pod by adding ruled to redirect POD CIDR traffic with metadata endpoint IP on port 80 to be sent to the NMI endpoint. The NMI server identifies the pod based on the remote address of the request and then queries the k8s (through MIC) for a matching azure identity. It then make a adal request to get the token for the client id and returns as a reponse to the request. If the request had client id as part of the query it validates it againsts the admin configured client id.

# Demo Pod 

## Pod fetching Service Principal Token from MSI endpoint 

```
spt, err := adal.NewServicePrincipalTokenFromMSI(msiEndpoint, resource)
```

## Pod using identity to Azure Resource Manager (ARM) operation by doing seamless authorization 

```
import "github.com/Azure/go-autorest/autorest/azure/auth"

authorizer, err := auth.NewAuthorizerFromEnvironment()
if err != nil {
	logger.Errorf("failed NewAuthorizerFromEnvironment  %+v", authorizer)
	return
}
vmClient := compute.NewVirtualMachinesClient(subscriptionID)
vmClient.Authorizer = authorizer
vmlist, err := vmClient.List(context.Background(), resourceGroup)
```

## Deploy Specs

### Requirement 

A running k8s cluster on Azure using AKS or ACS Engine 

### Deploy the azure-aad-identity infra 

```
kubectl create -f deploy/infra/deployment.yaml
```


### Demo app

To deploy the demo app, update the deploy/demo/deployment.yaml arguments with your subscription, clientID and resource group.
Make your your identity with the client ID has reader permission to the resource group provided in the input. 


```
kubectl create -f deploy/demo/deployment.yaml
```

### Configure Identity Binding 

#### Install MIC Custom Resource Definition (CRD) for Azure Identity 

```
kubectl create -f crd/azureAssignedIdentityCrd.yaml
kubectl create -f crd/azureIdentityBindingCrd.yaml
kubectl create -f crd/azureIdentityCrd.yaml
```

#### Create User Azure Identity 

Get the client id and resource id for the identity 
```
az identity create -g <resourcegroup> -n <idname>
```

#### Install User Azure Identity on k8s cluster 

Edit and save this as aadpodidentity.yaml
```
apiVersion: "aadpodidentity.k8s.io/v1"
kind: AzureIdentity
metadata:
 name: <any-idname>
spec:
 type: 1
 ResourceID: /subscriptions/<subid>/resourcegroups/<resourcegroup>/providers/Microsoft.ManagedIdentity/userAssignedIdentities/<idname>
 ClientID: <clientid>
```

```
kubectl create -f aadpodidentity.yaml
```

#### Install Pod to Identity Binding on k8s cluster

Edit and save this as aadpodidentitybinding.yaml
```
apiVersion: "aadpodidentity.k8s.io/v1"
kind: AzureIdentityBinding
metadata:
 name: demo1-azure-identity-binding
spec:
 AzureIdentity: <idname>
 Selector: <label value to match>
``` 

```
kubectl create -f aadpodidentitybinding.yaml
```
The name of the identity which we created earlier needs to be filled in AzureIdentity.
For a pod to match a binding, it should have a label with the key '**aadpodidbinding**' 
whose value matches the  **Selector** field in the binding above.

Here an example pod with the label specified:
```
$ kubectl get po busybox0 --show-labels
NAME       READY     STATUS    RESTARTS   AGE       LABELS
busybox0   1/1       Running   10         10h       aadpodidbinding=select_it,app=busybox0
```

This pod will match the binding below:
```
apiVersion: "aadpodidentity.k8s.io/v1"
kind: AzureIdentityBinding
metadata:
  name: test-azure-id-binding
spec: 
  AzureIdentity: "test-azure-identity"
  Selector: "select_it"
```

# Contributing

This project welcomes contributions and suggestions.  Most contributions require you to agree to a
Contributor License Agreement (CLA) declaring that you have the right to, and actually do, grant us
the rights to use your contribution. For details, visit https://cla.microsoft.com.

When you submit a pull request, a CLA-bot will automatically determine whether you need to provide
a CLA and decorate the PR appropriately (e.g., label, comment). Simply follow the instructions
provided by the bot. You will only need to do this once across all repos using our CLA.

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/).
For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or
contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.
