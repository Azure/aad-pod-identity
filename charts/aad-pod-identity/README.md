# aad-pod-identity

[aad-pod-identity](https://github.com/Azure/aad-pod-identity) enables Kubernetes applications to access cloud resources securely with [Azure Active Directory](https://azure.microsoft.com/en-us/services/active-directory/) (AAD).

## TL;DR:

```console
$ helm repo add aad-pod-identity https://raw.githubusercontent.com/Azure/aad-pod-identity/master/charts
$ helm install aad-pod-identity/aad-pod-identity
```

Expected output:

```console
NAME:   pod-identity
LAST DEPLOYED: Mon Sep 16 11:47:45 2019
NAMESPACE: default
STATUS: DEPLOYED

RESOURCES:
==> v1/ClusterRole
NAME                  AGE
aad-pod-identity-mic  1s
aad-pod-identity-nmi  1s

==> v1/Pod(related)
NAME                                  READY  STATUS             RESTARTS  AGE
aad-pod-identity-mic-9658685f4-vnmwx  0/1    ContainerCreating  0         1s
aad-pod-identity-mic-9658685f4-xrzmv  0/1    ContainerCreating  0         1s
aad-pod-identity-nmi-d5hvt            0/1    ContainerCreating  0         1s
aad-pod-identity-nmi-rq27p            0/1    ContainerCreating  0         1s
aad-pod-identity-nmi-wdgdf            0/1    ContainerCreating  0         1s

==> v1/ServiceAccount
NAME                  SECRETS  AGE
aad-pod-identity-mic  1        1s
aad-pod-identity-nmi  1        1s

==> v1beta1/ClusterRoleBinding
NAME                  AGE
aad-pod-identity-mic  1s
aad-pod-identity-nmi  1s

==> v1beta1/DaemonSet
NAME                  DESIRED  CURRENT  READY  UP-TO-DATE  AVAILABLE  NODE SELECTOR                AGE
aad-pod-identity-nmi  3        3        0      3           0          beta.kubernetes.io/os=linux  1s

==> v1beta1/Deployment
NAME                  READY  UP-TO-DATE  AVAILABLE  AGE
aad-pod-identity-mic  0/2    2           0          1s
```

## Introduction

A simple [helm](https://helm.sh/) chart for setting up the components needed to use [Azure Active Directory Pod Identity](https://github.com/Azure/aad-pod-identity) in Kubernetes.

This helm chart will deploy the following resources:
* AzureIdentity `CustomResourceDefinition`
* AzureIdentityBinding `CustomResourceDefinition`
* AzureAssignedIdentity `CustomResourceDefinition`
* AzurePodIdentityException `CustomResourceDefinition`
* AzureIdentity instance (optional)
* AzureIdentityBinding instance (optional)
* Managed Identity Controller (MIC) `Deployment`
* Node Managed Identity (NMI) `DaemonSet`

## Getting Started
The following steps will help you create a new Azure identity ([Managed Service Identity](https://docs.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/overview) or [Service Principal](https://docs.microsoft.com/en-us/azure/active-directory/develop/app-objects-and-service-principals)) and assign it to pods running in your Kubernetes cluster.

### Prerequisites
* [Azure Subscription](https://azure.microsoft.com/)
* [Azure Kubernetes Service (AKS)](https://azure.microsoft.com/services/kubernetes-service/) or [AKS Engine](https://github.com/Azure/aks-engine) deployment
* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) (authenticated to your Kubernetes cluster)
* [Helm v2.14+](https://github.com/helm/helm)
* [Azure CLI 2.0](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli?view=azure-cli-latest)
* [git](https://git-scm.com/downloads)

> Recommended Helm version > `2.14.2`. Issue with CRD during upgrade has been resolved after that release.

<details>
<summary><strong>[Optional] Creating user identity</strong></summary>

1. Create a new [Azure User Identity](https://docs.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/overview) using the Azure CLI:
> __NOTE:__ It's simpler to use the same resource group as your Kubernetes nodes are deployed in. For AKS this is the MC_* resource group. If you can't use the same resource group, you'll need to grant the Kubernetes cluster's service principal the "Managed Identity Operator" role.
```shell
az identity create -g <resource-group> -n <id-name>
```

2. Assign your newly created identity the appropriate role to the resource you want to access.
</details>


#### Installing charts

* If you need the `AzureIdentity` and `AzureIdentityBinding` resources to be created as part of the chart installation, update the values.yml to enable the azureIdentity and replace the resourceID, clientID using the values for the user identity.
* If you need the aad-pod-identity deployment to use it's own service principal credentials instead of the cluster service prinicipal '/etc/kubernetes/azure.json`, then uncomment this section and add the appropriate values for each required field.

```
adminsecret:
  cloud: <cloud environment name>
  subscriptionID: <subscription id>
  resourceGroup: <cluster resource group>
  vmType: <`standard` for normal virtual machine nodes, and `vmss` for cluster deployed with a virtual machine scale set>
  tenantID: <service principal tenant id>
  clientID: <service principal client id>
  clientSecret: <service principal client secret>
```

To install the chart with the release name `my-release`:

```console
$ helm install --name my-release aad-pod-identity/aad-pod-identity
```

Deploy your application to Kubernetes. The application can use [ADAL](https://docs.microsoft.com/en-us/azure/active-directory/develop/active-directory-authentication-libraries) to request a token from the MSI endpoint as usual. If you do not currently have such an application, a demo application is available [here](https://github.com/Azure/aad-pod-identity#demo-app). If you do use the demo application, please update the `deployment.yaml` with the appropriate subscription ID, client ID and resource group name. Also make sure the selector you defined in your `AzureIdentityBinding` matches the `aadpodidbinding` label on the deployment.

## Uninstalling the Chart

To uninstall/delete the last deployment:

```console
$ helm ls
$ helm delete [last deployment] --purge
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

> The CRD created by the chart are not removed by default and should be manually cleaned up (if required)

```bash
kubectl delete crd azureassignedidentities.aadpodidentity.k8s.io
kubectl delete crd azureidentities.aadpodidentity.k8s.io
kubectl delete crd azureidentitybindings.aadpodidentity.k8s.io
kubectl delete crd azurepodidentityexceptions.aadpodidentity.k8s.io
```

## Configuration

The following tables list the configurable parameters of the aad-pod-identity chart and their default values.

| Parameter                                | Description                                                                                                                                                                                                                                                                                                                   | Default                                         |
|------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------------------------------------------------|
| `nameOverride`                           | String to partially override aad-pod-identity.fullname template with a string (will prepend the release name)                                                                                                                                                                                                                 | `""`                                            |
| `fullnameOverride`                       | String to fully override aad-pod-identity.fullname template with a string                                                                                                                                                                                                                                                     | `""`                                            |
| `image.repository`                       | Image repository                                                                                                                                                                                                                                                                                                              | `mcr.microsoft.com/k8s/aad-pod-identity`        |
| `image.pullPolicy`                       | Image pull policy                                                                                                                                                                                                                                                                                                             | `Always`                                        |
| `forceNameSpaced`                        | By default, AAD Pod Identity matches pods to identities across namespaces. To match only pods in the namespace containing AzureIdentity set this to true.                                                                                                                                                                     | `false`                                         |
| `installMICException`                    | When NMI runs on a node where MIC is running, then MIC token request call is also intercepted by NMI. MIC can't get a valid token to initialize and then assign the identity. Installing an exception for MIC would ensure all token requests for MIC pods directly go to IMDS and not go through the pod-identity validation | `true`                                          |
| `adminsecret.cloud`                      | Azure cloud environment name                                                                                                                                                                                                                                                                                                  | ` `                                             |
| `adminsecret.subscriptionID`             | Azure subscription ID                                                                                                                                                                                                                                                                                                         | ` `                                             |
| `adminsecret.resourceGroup`              | Azure resource group                                                                                                                                                                                                                                                                                                          | ` `                                             |
| `adminsecret.vmType`                     | `standard` for normal virtual machine nodes, and `vmss` for cluster deployed with a virtual machine scale set                                                                                                                                                                                                                 | ` `                                             |
| `adminsecret.tenantID`                   | Azure service principal tenantID                                                                                                                                                                                                                                                                                              | ` `                                             |
| `adminsecret.clientID`                   | Azure service principal clientID                                                                                                                                                                                                                                                                                              | ` `                                             |
| `adminsecret.clientSecret`               | Azure service principal clientSecret                                                                                                                                                                                                                                                                                          | ` `                                             |
| `mic.image`                              | MIC image name                                                                                                                                                                                                                                                                                                                | `mic`                                           |
| `mic.tag`                                | MIC image tag                                                                                                                                                                                                                                                                                                                 | `1.6.0`                                         |
| `mic.PriorityClassName`                  | MIC priority class (can only be set when deploying to kube-system namespace)                                                                                                                                                                                                                                                  |
| `mic.logVerbosity`                       | Log level. Uses V logs (glog)                                                                                                                                                                                                                                                                                                 | `0`                                             |
| `mic.resources`                          | Resource limit for MIC                                                                                                                                                                                                                                                                                                        | `{}`                                            |
| `mic.podAnnotations`                     | Pod annotations for MIC                                                                                                                                                                                                                                                                                                       | `{}`                                            |
| `mic.tolerations`                        | Affinity settings                                                                                                                                                                                                                                                                                                             | `{}`                                            |
| `mic.affinity`                           | List of node taints to tolerate                                                                                                                                                                                                                                                                                               | `[]`                                            |
| `mic.leaderElection.instance`            | Override leader election instance name                                                                                                                                                                                                                                                                                        | If not provided, default value is `hostname`    |
| `mic.leaderElection.namespace`           | Override the namespace to create leader election objects                                                                                                                                                                                                                                                                      | `default`                                       |
| `mic.leaderElection.name`                | Override leader election name                                                                                                                                                                                                                                                                                                 | If not provided, default value is `aad-pod-identity-mic` |
| `mic.leaderElection.duration`            | Override leader election duration                                                                                                                                                                                                                                                                                             | If not provided, default value is `15s`         |
| `mic.probePort`                          | Override http liveliness probe port                                                                                                                                                                                                                                                                                           | If not provided, default port is `8080`         |
| `mic.syncRetryDuration`                  | Override interval in seconds at which sync loop should periodically check for errors and reconcile                                                                                                                                                                                                                            | If not provided, default value is `3600s`       |
| `mic.immutableUserMSIs`                  | List of  user-defined identities that shouldn't be deleted from VM/VMSS.                                                                                                                                                                                                                                                      | If not provided, default value is empty         |
| `mic.cloudConfig`                        | The cloud configuration used to authenticate with Azure                                                                                                                                                                                                                                                                       | If not provided, default value is `/etc/kubernetes/azure.json` |
| `nmi.image`                              | NMI image name                                                                                                                                                                                                                                                                                                                | `nmi`                                           |
| `nmi.tag`                                | NMI image tag                                                                                                                                                                                                                                                                                                                 | `1.6.0`                                         |
| `nmi.PriorityClassName`                  | NMI priority class (can only be set when deploying to kube-system namespace)                                                                                                                                                                                                                                                  |
| `nmi.resources`                          | Resource limit for NMI                                                                                                                                                                                                                                                                                                        | `{}`                                            |
| `nmi.podAnnotations`                     | Pod annotations for NMI                                                                                                                                                                                                                                                                                                       | `{}`                                            |
| `nmi.tolerations`                        | Affinity settings                                                                                                                                                                                                                                                                                                             | `{}`                                            |
| `nmi.affinity`                           | List of node taints to tolerate                                                                                                                                                                                                                                                                                               | `[]`                                            |
| `nmi.ipTableUpdateTimeIntervalInSeconds` | Override iptables update interval in seconds                                                                                                                                                                                                                                                                                  | `60`                                            |
| `nmi.micNamespace`                       | Override mic namespace to short circuit MIC token requests                                                                                                                                                                                                                                                                    | If not provided, default is `default` namespace |
| `nmi.probePort`                          | Override http liveliness probe port                                                                                                                                                                                                                                                                                           | If not provided, default is `8080`              |
| `nmi.retryAttemptsForCreated`            | Override number of retries in NMI to find assigned identity in CREATED state                                                                                                                                                                                                                                                  | If not provided, default is  `16`               |
| `nmi.retryAttemptsForAssigned`           | Override number of retries in NMI to find assigned identity in ASSIGNED state                                                                                                                                                                                                                                                 | If not provided, default is  `4`                |
| `nmi.findIdentityRetryIntervalInSeconds` | Override retry interval to find assigned identities in seconds                                                                                                                                                                                                                                                                | If not provided, default is  `5`                |
| `rbac.enabled`                           | Create and use RBAC for all aad-pod-identity resources                                                                                                                                                                                                                                                                        | `true`                                          |
| `rbac.allowAccessToSecrets`              | NMI requires permissions to get secrets when service principal (type: 1) is used in AzureIdentity. If using only MSI (type: 0) in AzureIdentity, secret get permission can be disabled by setting this to false.                                                                                                              | `true`                                          |
| `azureIdentity.enabled`                  | Create azure identity and azure identity binding resource                                                                                                                                                                                                                                                                     | `false`                                         |
| `azureIdentity.name`                     | Azure identity resource name                                                                                                                                                                                                                                                                                                  | `azure-identity`                                |
| `azureIdentity.namespace`                | Azure identity resource namespace. Default value is release namespace                                                                                                                                                                                                                                                         | ` `                                             |
| `azureIdentity.type`                     | Azure identity type - type 0: MSI, type 1: Service Principal                                                                                                                                                                                                                                                                  | `0`                                             |
| `azureIdentity.resourceID`               | Azure identity resource ID                                                                                                                                                                                                                                                                                                    | ` `                                             |
| `azureIdentity.clientID`                 | Azure identity client ID                                                                                                                                                                                                                                                                                                      | ` `                                             |
| `azureIdentityBinding.name`              | Azure identity binding name                                                                                                                                                                                                                                                                                                   | `azure-identity-binding`                        |
| `azureIdentityBinding.selector`          | Azure identity binding selector. The selector defined here will also need to be included in labels for app deployment.                                                                                                                                                                                                        | `demo`                                          |

## Troubleshooting

If the helm chart is deleted and then reinstalled without manually deleting the crds, then you can get an error like -

```console
➜ helm install aad-pod-identity/aad-pod-identity --name pod-identity
Error: customresourcedefinitions.apiextensions.k8s.io "azureassignedidentities.aadpodidentity.k8s.io" already exists
```

In this case, since there is no update to the crd definition since it was last installed, you can use a parameter to say not to use hook to install the CRD:

```console
helm install aad-pod-identity/aad-pod-identity --name pod-identity --no-hooks
```
