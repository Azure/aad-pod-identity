# AAD Pod Identity

[![Build Status](https://dev.azure.com/azure/aad-pod-identity/_apis/build/status/aad-pod-identity-nightly?branchName=master)](https://dev.azure.com/azure/aad-pod-identity/_build/latest?definitionId=77&branchName=master)
[![GoDoc](https://godoc.org/github.com/Azure/aad-pod-identity?status.svg)](https://godoc.org/github.com/Azure/aad-pod-identity)
[![Go Report Card](https://goreportcard.com/badge/github.com/Azure/aad-pod-identity)](https://goreportcard.com/report/github.com/Azure/aad-pod-identity)

AAD Pod Identity enables Kubernetes applications to access cloud resources securely with [Azure Active Directory] (AAD).

Using Kubernetes primitives, administrators configure identities and bindings to match pods. Then without any code modifications, your containerized applications can leverage any resource in the cloud that depends on AAD as an identity provider.

----

## Contents

* [v1.6.0 Breaking Change](#v160-breaking-change)
* [Getting Started](#getting-started)
* [Components](#components)
  + [Managed Identity Controller](#managed-identity-controller)
  + [Node Managed Identity](#node-managed-identity)
* [Role Assignment](#role-assignment)
* [Demo](#demo)
  + [1. Deploy aad-pod-identity](#1-deploy-aad-pod-identity)
  + [2. Create an identity on Azure](#2-create-an-identity-on-azure)
  + [3. Deploy AzureIdentity](#3-deploy-azureidentity)
  + [4. (Optional) Match pods in the namespace](#4--optional--match-pods-in-the-namespace)
  + [5. Deploy AzureIdentityBinding](#5-deploy-azureidentitybinding)
  + [6. Deployment and Validation](#6-deployment-and-validation)
* [Uninstall Notes](#uninstall-notes)
* [What To Do Next?](#what-to-do-next)
* [Code of Conduct](#code-of-conduct)
* [Support](#support)

## v1.6.0 Breaking Change

With https://github.com/Azure/aad-pod-identity/pull/398, the [client-go](https://github.com/kubernetes/client-go) library is upgraded to v0.17.2, where CRD [fields are now case sensitive](https://github.com/kubernetes/kubernetes/issues/64612). If you are upgrading MIC and NMI from v1.x.x to v1.6.0, MIC v1.6.0+ will upgrade the fields of existing `AzureIdentity` and `AzureIdentityBinding` on startup to the new format to ensure backward compatibility. A configmap called `aad-pod-identity-config` is created to record and confirm the successful type upgrade.

However, for future `AzureIdentity` and `AzureIdentityBinding` created using v1.6.0+, the following fields need to be changed:

### `AzureIdentity`

| < 1.6.0          | >= 1.6.0         |
|------------------|------------------|
| `ClientID`       | `clientID`       |
| `ClientPassword` | `clientPassword` |
| `ResourceID`     | `resourceID`     |
| `TenantID`       | `tenantID`       |

### `AzureIdentityBinding`

| < 1.6.0         | >= 1.6.0        |
|-----------------|-----------------|
| `AzureIdentity` | `azureIdentity` |
| `Selector`      | `selector`      |

### `AzurePodIdentityException`

| < 1.6.0         | >= 1.6.0        |
|-----------------|-----------------|
| `PodLabels`     | `podLabels`     |

## Getting Started

It is recommended to get familiar with the AAD Pod Identity ecosystem before diving into the demo. It consists of the Managed Identity Controller (MIC) deployment, the Node Managed Identity (NMI) DaemonSet, and several standard and custom resources.

## Components

AAD Pod Identity has two components: the [Managed Identity Controller] (MIC) and the [Node Managed Identity] (NMI).

### Managed Identity Controller

The Managed Identity Controller (MIC) is a Kubernetes [custom resource] that watches for changes to pods, `AzureIdentity` and `AzureIdentityBindings` through the Kubernetes API server. When it detects a relevant change, the MIC adds or deletes `AzureAssignedIdentity` as needed.

Specifically, when a pod is scheduled, the MIC assigns the identity on Azure to the underlying VM/VMSS during the creation phase. When the pod is deleted, it removes the identity from the underlying VM/VMSS on Azure. The MIC takes similar actions when `AzureIdentity` or `AzureIdentityBinding` are created or deleted.

### Node Managed Identity

The authorization request to fetch a Service Principal Token from an MSI endpoint is sent to Azure Instance Metadata Service (IMDS) endpoint (169.254.169.254), which is redirected to the NMI pod. The redirection is accomplished by adding rules to redirect POD CIDR traffic with IMDS endpoint on port 80 to the NMI endpoint. The NMI server identifies the pod based on the remote address of the request and then queries Kubernetes (through MIC) for a matching Azure identity. NMI then makes an Azure Active Directory Authentication Library ([ADAL]) request to get the token for the client ID and returns it as a response. If the request had client ID as part of the query, it is validated against the admin-configured client ID.

Here is an example cURL command that will fetch an access token to access ARM within a pod identified by an AAD-Pod-Identity selector:

```bash
curl 'http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=https%3A%2F%2Fmanagement.azure.com%2F' -H Metadata:true -s
```

For different ways to acquire an access token within a pod, please refer to this [documentation](https://docs.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/how-to-use-vm-token).

Similarly, a host can make an authorization request to fetch Service Principal Token for a resource directly from the NMI host endpoint (http://127.0.0.1:2579/host/token/). The request must include the pod namespace `podns` and the pod name `podname` in the request header and the resource endpoint of the resource requesting the token. The NMI server identifies the pod based on the `podns` and `podname` in the request header and then queries k8s (through MIC) for a matching azure identity. Then NMI makes an ADAL request to get a token for the resource in the request, returning the `token` and the `clientid` as a response.

Here is an example cURL command:

```bash
curl http://127.0.0.1:2579/host/token/?resource=https://vault.azure.net -H "podname: nginx-flex-kv-int" -H "podns: default"
```

For more information, please refer to the [design documentation](./docs/design/concept.md).

## Role Assignment

Your cluster will need the correct role assignment configuration to perform Azure-related operations such as assigning and un-assigning the identity on the underlying VM/VMSS. Please refer to the [role assignment](./docs/readmes/README.role-assignment.md) documentation to review and set required role assignments.

## Demo

You will need [Azure CLI] installed and a Kubernetes cluster running on Azure, either managed by [AKS] or provisioned with [AKS Engine].

Set the following Azure-related environment variables before getting started:

```bash
export SUBSCRIPTION_ID="<SubscriptionId>"
export RESOURCE_GROUP="<ResourceGroup>"
export IDENTITY_NAME="demo"
```

> For AKS cluster, there are two resource groups that you need to be aware of - the resource group that contains the AKS cluster itself, and the cluster resource group (`MC_<AKSClusterName>_<AKSResourceGroup>_<Location>`). The latter contains all of the infrastructure resources associated with the cluster like VM/VMSS and VNet. Depending on where you deploy your user-assigned identities, you might need additional role assignments. Please refer to [Role Assignment](#role-assignment) for more information. For this demo, it is recommended to use the cluster resource group (the one with `MC_` prefix) as the `RESOURCE_GROUP` environment variable.

### 1. Deploy aad-pod-identity

Deploy `aad-pod-identity` components to an RBAC-enabled cluster:

```bash
kubectl apply -f https://raw.githubusercontent.com/Azure/aad-pod-identity/master/deploy/infra/deployment-rbac.yaml

# For AKS clusters, deploy the MIC and AKS add-on exception by running -
kubectl apply -f https://raw.githubusercontent.com/Azure/aad-pod-identity/master/deploy/infra/mic-exception.yaml
```

Deploy `aad-pod-identity` components to a non-RBAC cluster:

```bash
kubectl apply -f https://raw.githubusercontent.com/Azure/aad-pod-identity/master/deploy/infra/deployment.yaml

# For AKS clusters, deploy the MIC and AKS add-on exception by running -
kubectl apply -f https://raw.githubusercontent.com/Azure/aad-pod-identity/master/deploy/infra/mic-exception.yaml
```

Deploy `aad-pod-identity` using [Helm 3](https://v3.helm.sh/):

```bash
helm repo add aad-pod-identity https://raw.githubusercontent.com/Azure/aad-pod-identity/master/charts
helm install aad-pod-identity aad-pod-identity/aad-pod-identity
```

For a list of overwritable values when installing with Helm, please refer to [this section](https://github.com/Azure/aad-pod-identity/tree/master/charts/aad-pod-identity#configuration).

> Important: For AKS clusters with limited [egress-traffic], Please install pod-identity in `kube-system` namespace using the [helm charts].

### 2. Create an identity on Azure

Create an identity on Azure and store the client ID and resource ID of the identity as environment variables:

```bash
az identity create -g $RESOURCE_GROUP -n $IDENTITY_NAME --subscription $SUBSCRIPTION_ID
export IDENTITY_CLIENT_ID="$(az identity show -g $RESOURCE_GROUP -n $IDENTITY_NAME --subscription $SUBSCRIPTION_ID --query clientId -otsv)"
export IDENTITY_RESOURCE_ID="$(az identity show -g $RESOURCE_GROUP -n $IDENTITY_NAME --subscription $SUBSCRIPTION_ID --query id -otsv)"
```

Assign the role "Reader" to the identity so it has read access to the resource group. At the same time, store the identity assignment ID as an environment variable.

```bash
export IDENTITY_ASSIGNMENT_ID="$(az role assignment create --role Reader --assignee $IDENTITY_CLIENT_ID --scope /subscriptions/$SUBSCRIPTION_ID/resourceGroups/$RESOURCE_GROUP --query id -otsv)"
```

### 3. Deploy `AzureIdentity`

Create an `AzureIdentity` in your cluster that references the identity you created above:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: "aadpodidentity.k8s.io/v1"
kind: AzureIdentity
metadata:
  name: $IDENTITY_NAME
spec:
  type: 0
  resourceID: $IDENTITY_RESOURCE_ID
  clientID: $IDENTITY_CLIENT_ID
EOF
```

> Set `type: 0` for user-assigned MSI, `type: 1` for Service Principal with client secret, or `type: 2` for Service Principal with certificate. For more information, see [here](https://github.com/Azure/aad-pod-identity/tree/master/deploy/demo).

### 4. (Optional) Match pods in the namespace

For matching pods in the namespace, please refer to namespaced [README](docs/readmes/README.namespaced.md).

### 5. Deploy `AzureIdentityBinding`

Create an `AzureIdentityBinding` that reference the `AzureIdentity` you created above:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: "aadpodidentity.k8s.io/v1"
kind: AzureIdentityBinding
metadata:
  name: $IDENTITY_NAME-binding
spec:
  azureIdentity: $IDENTITY_NAME
  selector: $IDENTITY_NAME
EOF
```

### 6. Deployment and Validation

For a pod to match an identity binding, it needs a [label] with the key `aadpodidbinding` whose value is that of the `selector:` field in the `AzureIdentityBinding`. Deploy a pod that validates the functionality:

```bash
cat << EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: demo
  labels:
    aadpodidbinding: $IDENTITY_NAME
spec:
  containers:
  - name: demo
    image: mcr.microsoft.com/k8s/aad-pod-identity/demo:1.2
    args:
      - --subscriptionid=$SUBSCRIPTION_ID
      - --clientid=$IDENTITY_CLIENT_ID
      - --resourcegroup=$RESOURCE_GROUP
    env:
      - name: MY_POD_NAME
        valueFrom:
          fieldRef:
            fieldPath: metadata.name
      - name: MY_POD_NAMESPACE
        valueFrom:
          fieldRef:
            fieldPath: metadata.namespace
      - name: MY_POD_IP
        valueFrom:
          fieldRef:
            fieldPath: status.podIP
  nodeSelector:
    kubernetes.io/os: linux
EOF
```

> `mcr.microsoft.com/k8s/aad-pod-identity/demo` is an image that demostrates the use of AAD pod identity. The source code can be found [here](./cmd/demo/main.go).

To verify that the pod is indeed using the identity correctly:

```bash
kubectl logs demo
```

If successful, the log output would be similar to the following output:
```
...
successfully doARMOperations vm count 1
successfully acquired a token using the MSI, msiEndpoint(http://169.254.169.254/metadata/identity/oauth2/token)
successfully acquired a token, userAssignedID MSI, msiEndpoint(http://169.254.169.254/metadata/identity/oauth2/token) clientID(xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)
successfully made GET on instance metadata
...
```

Once you are done with the demo, clean up your resources:

```bash
kubectl delete pod demo
kubectl delete azureidentity $IDENTITY_NAME
kubectl delete azureidentitybinding $IDENTITY_NAME-binding
az role assignment delete --id $IDENTITY_ASSIGNMENT_ID
az identity delete -g $RESOURCE_GROUP -n $IDENTITY_NAME
```

## Uninstall Notes

The NMI pods modify the nodes' [iptables] to intercept calls to IMDS endpoint within a node. This allows NMI to insert identities assigned to a pod before executing the request on behalf of the caller.

These iptables entries will be cleaned up when the pod-identity pods are uninstalled. However, if the pods are terminated for unexpected reasons, the iptables entries can be removed with these commands on the node:

```bash
# remove the custom chain reference
iptables -t nat -D PREROUTING -j aad-metadata

# flush the custom chain
iptables -t nat -F aad-metadata

# remove the custom chain
iptables -t nat -X aad-metadata
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

## Support

aad-pod-identity is an open source project that is [**not** covered by the Microsoft Azure support policy](https://support.microsoft.com/en-us/help/2941892/support-for-linux-and-open-source-technology-in-azure). [Please search open issues here](https://github.com/Azure/aad-pod-identity/issues), and if your issue isn't already represented please [open a new one](https://github.com/Azure/aad-pod-identity/issues/new/choose). The project maintainers will respond to the best of their abilities.


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
[helm charts]: https://github.com/Azure/aad-pod-identity/tree/master/charts/aad-pod-identity
[egress-traffic]: https://docs.microsoft.com/en-us/azure/aks/limit-egress-traffic
