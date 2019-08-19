# AAD Pod Identity

[![Build Status](https://dev.azure.com/azure/aad-pod-identity/_apis/build/status/aad-pod-identity-CI?branchName=master)](https://dev.azure.com/azure/aad-pod-identity/_build/latest?definitionId=22&branchName=master)
[![GoDoc](https://godoc.org/github.com/Azure/aad-pod-identity?status.svg)](https://godoc.org/github.com/Azure/aad-pod-identity)
[![Go Report Card](https://goreportcard.com/badge/github.com/Azure/aad-pod-identity)](https://goreportcard.com/report/github.com/Azure/aad-pod-identity)

AAD Pod Identity enables Kubernetes applications to access cloud resources securely with [Azure Active Directory] (AAD).

Using Kubernetes primitives, administrators configure identities and bindings to match pods. Then without any code modifications, your containerized applications can leverage any resource in the cloud that depends on AAD as an identity provider.

----

## Contents

* [Getting Started](#getting-started)
* [Demo](#demo)
* [Components](#components)
* [Features](docs/readmes/README.features.md)
* [What To Do Next?](#what-to-do-next)
* [Code of Conduct](#code-of-conduct)

## Getting Started

### Prerequisites

You will need a Kubernetes cluster running on Azure, either managed by [AKS] or provisioned with [AKS Engine].

### 1. Create the Deployment

AAD Pod Identity consists of the Managed Identity Controller (MIC) deployment, the Node Managed Identity (NMI) daemon set, and several standard and custom resources. For more information, see [Components].

Run this command to create the `aad-pod-identity` deployment on an RBAC-enabled cluster:

```shell
kubectl apply -f https://raw.githubusercontent.com/Azure/aad-pod-identity/master/deploy/infra/deployment-rbac.yaml
```

Or run this command to deploy to a non-RBAC cluster:

```shell
kubectl apply -f https://raw.githubusercontent.com/Azure/aad-pod-identity/master/deploy/infra/deployment.yaml
```

### 2. Create an Azure Identity

Run this [Azure CLI] command, and take note of the `clientId` and `id` values it returns:

```shell
az identity create -g <resourcegroup> -n <name> -o json
```

Here is an example of the output:

```json
$ az identity create -g myresourcegroup -n myidentity -o json
{
  "clientId": "00000000-0000-0000-0000-000000000000",
  "clientSecretUrl": "https://control-eastus.identity.azure.net/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/myresourcegroup/providers/Microsoft.ManagedIdentity/userAssignedIdentities/myidentity/credentials?tid=00000000-0000-0000-0000-000000000000&oid=00000000-0000-0000-0000-000000000000&aid=00000000-0000-0000-0000-000000000000",
  "id": "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/myresourcegroup/providers/Microsoft.ManagedIdentity/userAssignedIdentities/myidentity",
  "location": "eastus",
  "name": "myidentity",
  "principalId": "00000000-0000-0000-0000-000000000000",
  "resourceGroup": "myresourcegroup",
  "tags": {},
  "tenantId": "00000000-0000-0000-0000-000000000000",
  "type": "Microsoft.ManagedIdentity/userAssignedIdentities"
}
```

### 3. Install the Azure Identity

Save this Kubernetes manifest to a file named `aadpodidentity.yaml`:

```yaml
apiVersion: "aadpodidentity.k8s.io/v1"
kind: AzureIdentity
metadata:
  name: <a-idname>
spec:
  type: 0
  ResourceID: /subscriptions/<subid>/resourcegroups/<resourcegroup>/providers/Microsoft.ManagedIdentity/userAssignedIdentities/<name>
  ClientID: <clientId>
```

Replace the placeholders with your user identity values. Set `type: 0` for user-assigned MSI or `type: 1` for Service Principal.

Finally, save your changes to the file, then create the `AzureIdentity` resource in your cluster:

```shell
kubectl apply -f aadpodidentity.yaml
```

### 4. (Optional) Match pods in the namespace

For matching pods in the namespace, please refer to namespaced [README](docs/readmes/README.namespaced.md).

### 5. Install the Azure Identity Binding

Save this Kubernetes manifest to a file named `aadpodidentitybinding.yaml`:

```yaml
apiVersion: "aadpodidentity.k8s.io/v1"
kind: AzureIdentityBinding
metadata:
  name: demo1-azure-identity-binding
spec:
  AzureIdentity: <a-idname>
  Selector: <label value to match>
```

Replace the placeholders with your values. Ensure that the `AzureIdentity` name matches the one in `aadpodidentity.yaml`.

Finally, save your changes to the file, then create the `AzureIdentityBinding` resource in your cluster:

```shell
kubectl apply -f aadpodidentitybinding.yaml
```

For a pod to match an identity binding, it needs a [label] with the key `aadpodidbinding` whose value is that of the `Selector:` field in the binding. Here is an example pod with a label:

```shell
$ kubectl get po busybox0 --show-labels
NAME       READY     STATUS    RESTARTS   AGE       LABELS
busybox0   1/1       Running   10         10h       aadpodidbinding=select_it,app=busybox0
```

This pod will match the binding below:

```yaml
apiVersion: "aadpodidentity.k8s.io/v1"
kind: AzureIdentityBinding
metadata:
  name: test-azure-id-binding
spec:
  AzureIdentity: "test-azure-identity"
  Selector: "select_it"
```

### 6. Set Permissions for MIC

This step is only required for user-assigned MSI.

The MIC uses the service principal [credentials stored in the cluster] to access Azure resources. This service principal needs `Microsoft.ManagedIdentity/userAssignedIdentities/\*/assign/action` permission on the identity to work with user-assigned MSI.

If the Azure identity is in the same resource group as your AKS cluster nodes, you can skip this section. (For [AKS], a resource group was added with an `MC_` prefix when you created the cluster.) Otherwise, follow these steps to assign the required permissions:

1. Find the service principal used by your cluster. Please refer to the [AKS docs] for details. For example, if you didn't specify a service principal to the `az aks create` command, it will have generated one in the `~/.azure/aksServicePrincipal.json` file.

2. Assign the required permissions with the following command:

```shell
az role assignment create --role "Managed Identity Operator" --assignee <sp id> --scope <full id of the managed identity>
```

### Uninstall Notes

The NMI pods modify the nodes' [iptables] to intercept calls to Azure Instance Metadata endpoint. This allows NMI to insert identities assigned to a pod before executing the request on behalf of the caller.

These iptables entries will be cleaned up when the pod-identity pods are uninstalled. However, if the pods are terminated for unexpected reasons, the iptables entries can be removed with these commands on the node:

```shell
# remove the custom chain reference
iptables -t nat -D PREROUTING -j aad-metadata

# flush the custom chain
iptables -t nat -F aad-metadata

# remove the custom chain
iptables -t nat -X aad-metadata
```

## Demo

The demonstration program illustrates how, after setting the identity and binding, the sample app can list VMs in an Azure resource group. To deploy the demo, please ensure you have completed the [Prerequisites] and understood the previous sections in this document.

The demo program can be found here: [cmd/demo](cmd/demo).

Here are some excerpts from the demo.

### Get a Service Principal Token from an MSI Endpoint

```go
spt, err := adal.NewServicePrincipalTokenFromMSI(msiEndpoint, resource)
```

### List VMs with Seamless Authorization

```go
import "github.com/Azure/go-autorest/autorest/azure/auth"

authorizer, err := auth.NewAuthorizerFromEnvironment()
if err != nil {
    logger.Errorf("failed NewAuthorizerFromEnvironment: %+v", authorizer)
    return
}
vmClient := compute.NewVirtualMachinesClient(subscriptionID)
vmClient.Authorizer = authorizer
vmlist, err := vmClient.List(context.Background(), resourceGroup)
```

### Assign the Reader Role

The user-assigned identity needs the "Reader" role on the resource group to list its VMs. Provide the `principalId` from the user-assigned identity to this command:

```shell
az role assignment create --role Reader --assignee <principalid> --scope /subscriptions/<subscriptionid>/resourcegroups/<resourcegroup>
```

### Start the Demo Pod

Update the `deploy/demo/deployment.yaml` arguments with your subscription, clientID and resource group, then create the demo deployment:

```shell
kubectl apply -f deploy/demo/deployment.yaml
```

## Components

AAD Pod Identity has two components: the [Managed Identity Controller] (MIC) and the [Node Managed Identity] (NMI) pod.

### Managed Identity Controller

The Managed Identity Controller (MIC) is a Kubernetes [custom resource] that watches for changes to pods, identities, and bindings through the Kubernetes API server. When it detects a relevant change, the MIC adds or deletes assigned identities as needed.

Specifically, when a pod is scheduled, the MIC assigns an identity to the underlying VM during the creation phase. When the pod is deleted, it removes the assigned identity from the VM. The MIC takes similar actions when identities or bindings are created or deleted.

### Node Managed Identity

The authorization request to fetch a Service Principal Token from an MSI endpoint is sent to a standard Instance Metadata endpoint which is redirected to the NMI pod. The redirection is accomplished by adding rules to redirect POD CIDR traffic with metadata endpoint IP on port 80 to the NMI endpoint. The NMI server identifies the pod based on the remote address of the request and then queries Kubernetes (through MIC) for a matching Azure identity. NMI then makes an Azure Active Directory Authentication Library ([ADAL]) request to get the token for the client id and returns it as a response. If the request had client id as part of the query, it is validated against the admin-configured client id.

Similarly, a host can make an authorization request to fetch Service Principal Token for a resource directly from the NMI host endpoint (http://127.0.0.1:2579/host/token/). The request must include the pod namespace `podns` and the pod name `podname` in the request header and the resource endpoint of the resource requesting the token. The NMI server identifies the pod based on the `podns` and `podname` in the request header and then queries k8s (through MIC) for a matching azure identity. Then NMI makes an ADAL request to get a token for the resource in the request, returning the `token` and the `clientid` as a response.

Here is an example cURL command:

```bash
curl http://127.0.0.1:2579/host/token/?resource=https://vault.azure.net -H "podname: nginx-flex-kv-int" -H "podns: default"
```

## What To Do Next?

* Dive deeper into AAD Pod Identity by following the detailed [Tutorial].
* Learn more about the design of AAD Pod Identity:
  - [Concept]
  - [Block Diagram]
* Learn how to debug this project at the [Debugging] wiki page.
* Join us by [Contributing] to AAD Pod Identity.

## Code of Conduct

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/). For more information, see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq) or contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.


[ADAL]: https://docs.microsoft.com/azure/active-directory/develop/active-directory-authentication-libraries
[AKS]: https://azure.microsoft.com/services/kubernetes-service/
[AKS Docs]: https://docs.microsoft.com/azure/aks/kubernetes-service-principal
[AKS Engine]: https://github.com/Azure/aks-engine
[annotation]: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/
[Azure Active Directory]: https://azure.microsoft.com/services/active-directory/
[Azure CLI]: https://docs.microsoft.com/cli/azure/install-azure-cli?view=azure-cli-latest
[Block Diagram]: docs/design/concept.png
[Components]: #components
[Concept]: docs/design/concept.md
[Contributing]: CONTRIBUTING.md
[credentials stored in the cluster]: https://docs.microsoft.com/azure/aks/kubernetes-service-principal
[custom resource]: https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/
[Debugging]: https://github.com/Azure/aad-pod-identity/wiki/Debugging
[iptables]: https://en.wikipedia.org/wiki/Iptables
[label]: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/
[Managed Identity Controller]: #managed-identity-controller
[Node Managed Identity]: #node-managed-identity
[Prerequisites]: #prerequisites
[Tutorial]: docs/tutorial/README.md
