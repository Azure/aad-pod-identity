[![CircleCI](https://circleci.com/gh/Azure/aad-pod-identity/tree/master.svg?style=shield)](https://circleci.com/gh/Azure/aad-pod-identity/tree/master)

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

Similarly a host can make an authorization request to fetch Service Principal Token for a resource directly from the NMI host endpoint (http://127.0.0.1:2579/host/token/). The request must include the pod namespace `podns` and the pod name `podname` in the request header and the resource endpoint of the resource requesting the token. The NMI server identifies the pod based on the `podns` and `podname` in the request header and then queries k8s (through MIC) for a matching azure identity.  Then nmi makes a adal request to get a token for the resource in the request, returns the `token` and the `clientid` as a reponse to the request.

An example curl command:

```bash
curl http://127.0.0.1:2579/host/token/?resource=https://vault.azure.net -H "podname: nginx-flex-kv-int" -H "podns: default"
```

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

## Get Started

### Prerequisites 

A running k8s cluster on Azure using AKS or ACS Engine 

### Deploy the azure-aad-identity infra 

Deploy the infrastructure with the following command to deploy MIC, NMI, and the MIC CRDs.
```
kubectl create -f https://raw.githubusercontent.com/Azure/aad-pod-identity/master/deploy/infra/deployment.yaml
```

Pod Identity requires two components:

 1. Managed Identity Controller (MIC). A pod that binds Azure Ids to other pods - creates azureAssignedIdentity CRD. 
 2. Node Managed Identity (NMI). Identifies the pod based on the remote address of the incoming request, and then queries k8s (through MIC) for a matching Azure Id. It then makes an adal request to get the token for the client id and returns as a reponse to the request. Implemented as a [DaemonSet](https://kubernetes.io/docs/concepts/workloads/controllers/daemonset/).

If you have RBAC enabled, use the following deployment instead:

```
kubectl create -f https://raw.githubusercontent.com/Azure/aad-pod-identity/master/deploy/infra/deployment-rbac.yaml
```
#### Create User Azure Identity 

Get the client id and resource id for the identity 
```
az identity create -g <resourcegroup> -n <idname>
```

#### Assign Reader Role to new Identity

Using the principalid from the last step, assign reader role to new identity for this resource group
```
az role assignment create --role Reader --assignee <principalid> --scope /subscriptions/<subscriptionid>/resourcegroups/<resourcegroup>
```

#### Providing required permissions for MIC

This step is only required if you are using User assigned MSI.

MIC uses the service principal credentials [stored within the the AKS](https://docs.microsoft.com/en-us/azure/aks/kubernetes-service-principal) cluster to access azure resources. This service principal needs to have Microsoft.ManagedIdentity/userAssignedIdentities/\*/assign/action permission on the identity for usage with User assigned MSI. If your identity is created in the same resource group as that of the AKS nodes (typically this resource group is prefixed with 'MC_' string and is automatically generated when you create the cluster), you can skip the next steps.

1. Find the service principal used by your cluster - please refer the [AKS docs](https://docs.microsoft.com/en-us/azure/aks/kubernetes-service-principal) to figure out the service principle app Id. For example, if your cluster uses automatically generated service principal, the ~/.azure/aksServicePrincipal.json file in the machine from which you created the AKS cluster has the required information.

2. Assign the required permissions - the following command can be used to assign the required permission:
```
az role assignment create --role "Managed Identity Operator" --assignee <sp id> --scope <full id of the identity>
```

#### Install User Azure Identity on k8s cluster 

Edit and save this as aadpodidentity.yaml

Set `type: 0` for Managed Service Identity; `type: 1` for Service Principal

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

### Demo app

To deploy the demo app, ensure you have completed the above prerequisite steps!

Update the `deploy/demo/deployment.yaml` arguments with your subscription, clientID and resource group.
Make sure your identity with the client ID has reader permission to the resource group provided in the input. 

```
kubectl create -f deploy/demo/deployment.yaml
```

> There's also a detailed tutorial [here](docs/tutorial/README.md).

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
