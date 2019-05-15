----
Aad Pod Identity enables applications running in pods on Kubernetes clusters deployed on Azure(managed or unmanaged) to securely access cloud resources by leveraging Azure Active Directory(AAD). Administrators can configure identities and bindings, to match pods with identities. Following this, without any code modifications, the applications running within the pod can access any cloud resources that depend on AAD as Identity Provider. The administrator interactions with the aad-pod-identity are via Kubernetes primitives.

----

[![Build Status](https://dev.azure.com/azure/aad-pod-identity/_apis/build/status/aad-pod-identity-CI?branchName=master)](https://dev.azure.com/azure/aad-pod-identity/_build/latest?definitionId=22&branchName=master)

## Table of Contents
* [Design](#design)
* [Components](#components)
* [Getting Started](#getting-started)
* [Demonstration](#demonstration)
* [Tutorial](#tutorial)
* [Debugging](#debugging)
* [Contributing](#contributing)

## Design

The detailed design of the project can be found in the following docs:
- [Concept](https://github.com/Azure/aad-pod-identity/blob/master/docs/design/concept.md)
- [Block Diagram](https://github.com/Azure/aad-pod-identity/blob/master/docs/design/concept.png)

## Components

### Managed Identity Controller(MIC)

This controller watches for relevant changes to pods, identities, and bindings through the API server. When a change is detected it runs a loop to check if there are actions to be performed. Based on the change detected MIC would either add or delete assigned identities. In case of user assigned identity usage, MIC is responsible for assigning it to the underlying VM in which the pod gets scheduled during pod creation. During pod deletion it would remove those identities from the VM. Similar steps are taken by MIC when identities or bindings are created or deleted.

### Node Managed Identity(NMI)

The authorization request for fetching Service Principal Token from MSI endpoint is sent to a standard Instance Metadata endpoint which is redirected to the NMI pod by adding ruled to redirect POD CIDR traffic with metadata endpoint IP on port 80 to be sent to the NMI endpoint. The NMI server identifies the pod based on the remote address of the request and then queries the k8s (through MIC) for a matching Azure identity. It then makes an Azure Active Directory Authentication Library ([ADAL](https://docs.microsoft.com/en-us/azure/active-directory/develop/active-directory-authentication-libraries)) request to get the token for the client id and returns as a response to the request. If the request had client id as part of the query it validates it against the admin configured client id.

Similarly a host can make an authorization request to fetch Service Principal Token for a resource directly from the NMI host endpoint (http://127.0.0.1:2579/host/token/). The request must include the pod namespace `podns` and the pod name `podname` in the request header and the resource endpoint of the resource requesting the token. The NMI server identifies the pod based on the `podns` and `podname` in the request header and then queries k8s (through MIC) for a matching azure identity.  Then NMI makes an ADAL request to get a token for the resource in the request, returns the `token` and the `clientid` as a response to the request.

An example cURL command:

```bash
curl http://127.0.0.1:2579/host/token/?resource=https://vault.azure.net -H "podname: nginx-flex-kv-int" -H "podns: default"
```

## Getting Started

### Prerequisites 

A running k8s cluster on Azure using AKS or AKS Engine 

### Deploy the aad-pod-identity infra 

Deploy the infrastructure with the following command to deploy MIC, NMI, and the MIC CRDs on an RBAC enabled cluster.

```
kubectl create -f https://raw.githubusercontent.com/Azure/aad-pod-identity/master/deploy/infra/deployment-rbac.yaml
```
and for non-RBAC clusters:
```
kubectl create -f https://raw.githubusercontent.com/Azure/aad-pod-identity/master/deploy/infra/deployment.yaml
```
### Create User Azure Identity 

Get the client id and resource id for the identity 
```
az identity create -g <resourcegroup> -n <managedidentity-resourcename>
```

### Install User Azure Identity on k8s cluster 

Edit and save this as aadpodidentity.yaml

Set `type: 0` for User Assigned MSI; `type: 1` for Service Principal

```
apiVersion: "aadpodidentity.k8s.io/v1"
kind: AzureIdentity
metadata:
 name: <a-idname>
spec:
 type: 0
 ResourceID: /subscriptions/<subid>/resourcegroups/<resourcegroup>/providers/Microsoft.ManagedIdentity/userAssignedIdentities/<managedidentity-resourcename>
 ClientID: <clientid>
```

```
kubectl create -f aadpodidentity.yaml
```

### Understanding Namespaced identities
The system will match `pod` to `identity` across namespaces by default. This behavior can be modified to match to pods within the namespace that holds `AzureIdentity` by

1. On Azure Identity basis, by adding `aadpodidentity.k8s.io/Behavior: namespaced` annotation (You have to add the annotation on each `AzureIdentity` you want to apply this behavior on).
2. Default namespaced behavior on all identities by adding `--forceNamespaced=true` argument on the command line  or declare `FORCENAMESPACED=true` environment variable (for both `nmi` and `mic`).


### Install Pod to Identity Binding on k8s cluster

Edit and save this as aadpodidentitybinding.yaml.  Note the AzureIdentity name must match the one chosen in aadpodidentity.yaml.
```
apiVersion: "aadpodidentity.k8s.io/v1"
kind: AzureIdentityBinding
metadata:
 name: demo1-azure-identity-binding
spec:
 AzureIdentity: <a-idname>
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

### Providing required permissions for MIC

This step is only required if you are using User assigned MSI and if you are using AKS or aks-engine.

MIC uses the service principal credentials [stored within the Kubernetes cluster on Azure](https://docs.microsoft.com/en-us/azure/aks/kubernetes-service-principal) cluster to access azure resources. This service principal needs to have Microsoft.ManagedIdentity/userAssignedIdentities/\*/assign/action permission on the identity for usage with User assigned MSI. If your identity is created in the same resource group as that of the AKS nodes (typically this resource group is prefixed with 'MC_' string and is automatically generated when you create the cluster), you can skip the next steps.

1. Find the service principal used by your cluster - please refer the [AKS docs](https://docs.microsoft.com/en-us/azure/aks/kubernetes-service-principal) to figure out the service principle app Id. For example, if your cluster uses automatically generated service principal, the ~/.azure/aksServicePrincipal.json file in the machine from which you created the AKS cluster has the required information.

2. Assign the required permissions - the following command can be used to assign the required permission:
```
az role assignment create --role "Managed Identity Operator" --assignee <sp id> --scope <full id of the managed identity>
```

## Demonstration

The demo program can be found here: [cmd/demo](cmd/demo). It demonstrates how after setting the identity and binding the sample app can list the vms in a resurce group. To deploy the demo app, ensure you have completed the above prerequisite steps!

Here are some excerpts from the demo.

### Pod fetching Service Principal Token from MSI endpoint 

```
spt, err := adal.NewServicePrincipalTokenFromMSI(msiEndpoint, resource)
```

### Pod using identity to Azure Resource Manager (ARM) operation by doing seamless authorization 

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

### Assign Reader Role to new Identity

The user assigned identity should be assigned 'Reader' role on the resource group for performing the vm listing. Use the principalid of your user assigned identity to do it:

```
az role assignment create --role Reader --assignee <principalid> --scope /subscriptions/<subscriptionid>/resourcegroups/<resourcegroup>
```
### Starting the demo pod

Update the `deploy/demo/deployment.yaml` arguments with your subscription, clientID and resource group.

```
kubectl create -f deploy/demo/deployment.yaml
```

## Uninstall notes

The NMI pods modify the nodes iptables to intercept calls to Azure Instance Metadata endpoint. This allows NMI to assert identities assigned to pod before executing the request on behalf of the caller. These iptable entries will be cleaned up when the pod-identity pods are uninstalled. However if the pods are terminated because of other reasons (non actionable signals), they can be manually removed with

```
#remove the custom chain reference
iptables -t nat -D PREROUTING -j aad-metadata

#flush the custom chain
iptables -t nat -F aad-metadata

#remove the custom chain
iptables -t nat -X aad-metadata
```

## Tutorial

A detailed tutorial can be found here: [docs/tutorial/README.md](docs/tutorial/README.md).

## Debugging

Information about how to debug this project can be found at [Debugging](https://github.com/Azure/aad-pod-identity/wiki/Debugging) 

## Contributing

This project welcomes contributions and suggestions.  Most contributions require you to agree to a
Contributor License Agreement (CLA) declaring that you have the right to, and actually do, grant us
the rights to use your contribution. For details, visit https://cla.microsoft.com.

When you submit a pull request, a CLA-bot will automatically determine whether you need to provide
a CLA and decorate the PR appropriately (e.g., label, comment). Simply follow the instructions
provided by the bot. You will only need to do this once across all repos using our CLA.

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/).
For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or
contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.
