---
title: "Standard Walkthrough"
linkTitle: "Standard Walkthrough"
weight: 1
description: >
  You will need Azure CLI installed and a Kubernetes cluster running on Azure, either managed by AKS or provisioned with AKS Engine.
---

Run the following commands to set Azure-related environment variables and login to Azure via `az login`:

```bash
export SUBSCRIPTION_ID="<SubscriptionID>"

# login as a user and set the appropriate subscription ID
az login
az account set -s "${SUBSCRIPTION_ID}"

export RESOURCE_GROUP="<AKSResourceGroup>"
export CLUSTER_NAME="<AKSClusterName>"

# for this demo, we will be deploying a user-assigned identity to the AKS node resource group
export IDENTITY_RESOURCE_GROUP="$(az aks show -g ${RESOURCE_GROUP} -n ${CLUSTER_NAME} --query nodeResourceGroup -otsv)"
export IDENTITY_NAME="demo"
```

> For AKS clusters, there are two resource groups that you need to be aware of - the resource group where you deploy your AKS cluster to (denoted by the environment variable `RESOURCE_GROUP`), and the node resource group (`MC_<AKSResourceGroup>_<AKSClusterName>_<AKSClusterLocation>`). The latter contains all of the infrastructure resources associated with the cluster like VM/VMSS and VNet. Depending on where you deploy your user-assigned identities, you might need additional role assignments. Please refer to [Role Assignment](../../getting-started/role-assignment/) for more information. For this demo, it is recommended to deploy the demo identity to your node resource group (the one with `MC_` prefix).

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

> Important: For AKS clusters with [limited egress traffic](https://docs.microsoft.com/en-us/azure/aks/limit-egress-traffic), Please install aad-pod-identity in `kube-system` namespace using the helm charts.

```bash
helm install aad-pod-identity aad-pod-identity/aad-pod-identity --namespace=kube-system
```

### 2. Create an identity on Azure

Create an identity on Azure and store the client ID and resource ID of the identity as environment variables:

```bash
az identity create -g ${IDENTITY_RESOURCE_GROUP} -n ${IDENTITY_NAME}
export IDENTITY_CLIENT_ID="$(az identity show -g ${IDENTITY_RESOURCE_GROUP} -n ${IDENTITY_NAME} --query clientId -otsv)"
export IDENTITY_RESOURCE_ID="$(az identity show -g ${IDENTITY_RESOURCE_GROUP} -n ${IDENTITY_NAME} --query id -otsv)"
```

Assign the role "Reader" to the identity so it has read access to the resource group. At the same time, store the identity assignment ID as an environment variable.

```bash
export IDENTITY_ASSIGNMENT_ID="$(az role assignment create --role Reader --assignee ${IDENTITY_CLIENT_ID} --scope /subscriptions/${SUBSCRIPTION_ID}/resourceGroups/${IDENTITY_RESOURCE_GROUP} --query id -otsv)"
```

### 3. Deploy `AzureIdentity`

Create an `AzureIdentity` in your cluster that references the identity you created above:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: "aadpodidentity.k8s.io/v1"
kind: AzureIdentity
metadata:
  name: ${IDENTITY_NAME}
spec:
  type: 0
  resourceID: ${IDENTITY_RESOURCE_ID}
  clientID: ${IDENTITY_CLIENT_ID}
EOF
```

> Set `type: 0` for user-assigned MSI, `type: 1` for Service Principal with client secret, or `type: 2` for Service Principal with certificate. For more information, see [here](https://github.com/Azure/aad-pod-identity/tree/master/deploy/demo).

### 4. (Optional) Match pods in the namespace

For matching pods in the namespace, please refer to the [namespaced documentation](../../configure/match_pods_in_namespace/).

### 5. Deploy `AzureIdentityBinding`

Create an `AzureIdentityBinding` that reference the `AzureIdentity` you created above:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: "aadpodidentity.k8s.io/v1"
kind: AzureIdentityBinding
metadata:
  name: ${IDENTITY_NAME}-binding
spec:
  azureIdentity: ${IDENTITY_NAME}
  selector: ${IDENTITY_NAME}
EOF
```

### 6. Deployment and Validation

For a pod to match an identity binding, it needs a label with the key `aadpodidbinding` whose value is that of the `selector:` field in the `AzureIdentityBinding`. Deploy a pod that validates the functionality:

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
    image: mcr.microsoft.com/oss/azure/aad-pod-identity/demo:v1.7.5
    args:
      - --subscription-id=${SUBSCRIPTION_ID}
      - --resource-group=${IDENTITY_RESOURCE_GROUP}
      - --identity-client-id=${IDENTITY_CLIENT_ID}
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

> `mcr.microsoft.com/oss/azure/aad-pod-identity/demo` is an image that demonstrates the use of AAD pod identity. The source code can be found [here](https://github.com/Azure/aad-pod-identity/blob/master/cmd/demo/main.go).

To verify that the pod is indeed using the identity correctly:

```bash
kubectl logs demo
```

If successful, the log output would be similar to the following output:

```log
I0107 23:54:23.215900       1 main.go:42] starting aad-pod-identity demo pod (namespace: default, name: demo, IP: 10.244.0.73)
I0107 23:56:03.223398       1 main.go:167] curl -H Metadata:true "http://169.254.169.254/metadata/instance?api-version=2017-08-01": {"compute":{"location":"westus2","name":"aks-nodepool1-55282659-0","offer":"aks","osType":"Linux","placementGroupId":"","platformFaultDomain":"0","platformUpdateDomain":"0","publisher":"microsoft-aks","resourceGroupName":"MC_chuwon-aks_chuwon_westus2","sku":"aks-ubuntu-1604-202004","subscriptionId":"2d31b5ab-0ddc-4991-bf8d-61b6ad196f5a","tags":"aksEngineVersion:aks-release-v0.47.0-1-aks;creationSource:aks-aks-nodepool1-55282659-0;orchestrator:Kubernetes:1.15.10;poolName:nodepool1;resourceNameSuffix:55282659","version":"2020.04.16","vmId":"33b6a978-a2ad-4bbe-8331-f7e08c48e175","vmSize":"Standard_DS2_v2"},"network":{"interface":[{"ipv4":{"ipAddress":[{"privateIpAddress":"10.240.0.4","publicIpAddress":""}],"subnet":[{"address":"10.240.0.0","prefix":"16"}]},"ipv6":{"ipAddress":[]},"macAddress":"000D3AC47E7D"}]}}
I0107 23:56:03.816403       1 main.go:90] successfully listed VMSS instances from ARM via AAD Pod Identity: []
I0107 23:56:03.825676       1 main.go:114] successfully acquired a service principal token from http://169.254.169.254/metadata/identity/oauth2/token
I0107 23:56:03.834516       1 main.go:139] successfully acquired a service principal token from http://169.254.169.254/metadata/identity/oauth2/token using a user-assigned identity (91ff0ee7-35fc-46d0-9cd0-51a7ea9ec7f2)
```

Once you are done with the demo, clean up your resources:

```bash
kubectl delete pod demo
kubectl delete azureidentity ${IDENTITY_NAME}
kubectl delete azureidentitybinding ${IDENTITY_NAME}-binding
az role assignment delete --id ${IDENTITY_ASSIGNMENT_ID}
az identity delete -g ${IDENTITY_RESOURCE_GROUP} -n ${IDENTITY_NAME}
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