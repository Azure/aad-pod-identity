# Helm chart for Azure Active Directory Pod Identity
A simple [helm](https://helm.sh/) chart for setting up the components needed to use [Azure Active Directory Pod Identity](https://github.com/Azure/aad-pod-identity) in Kubernetes.

## Chart resources
This helm chart will deploy the following resources:
* AzureIdentity `CustomResourceDefinition`
* AzureIdentityBinding `CustomResourceDefinition`
* AzureAssignedIdentity `CustomResourceDefinition`
* AzureIdentity instance (optional)
* AzureIdentityBinding instance (optional)
* Managed Identity Controller (MIC) `Deployment`
* Node Managed Identity (NMI) `DaemonSet`

## Getting Started
The following steps will help you create a new Azure identity ([Managed Service Identity](https://docs.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/overview) or [Service Principal](https://docs.microsoft.com/en-us/azure/active-directory/develop/app-objects-and-service-principals)) and assign it to pods running in your Kubernetes cluster.

### Prerequisites
* [Azure Subscription](https://azure.microsoft.com/)
* [Azure Kubernetes Service (AKS)](https://azure.microsoft.com/services/kubernetes-service/) or [ACS-Engine](https://github.com/Azure/acs-engine) deployment
* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) (authenticated to your Kubernetes cluster)
* [Helm v1.10+](https://github.com/helm/helm)
* [Azure CLI 2.0](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli?view=azure-cli-latest)
* [git](https://git-scm.com/downloads)

### Steps

1. Create a new [Azure User Identity](https://docs.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/overview) using the Azure CLI:
> __NOTE:__ It's simpler to use the same resource group as your Kubernetes nodes are deployed in. For AKS this is the MC_* resource group. If you can't use the same resource group, you'll need to grant the Kubernetes cluster's service principal the "Managed Identity Operator" role.
```shell
az identity create -g <resource-group> -n <id-name>
```

2. Assign your newly created identity the role of _Reader_ for the resource group:
```shell
az role assignment create --role Reader --assignee <principal-id> --scope /subscriptions/<subscription-id>/resourcegroups/<resource-group>
```

3. Clone this repository and navigate to the helm chart's directory.
```shell
git clone git@github.com:Azure/aad-pod-identity.git && cd aad-pod-identity/charts/aad-pod-identity
```

4. Open the `values.yaml` file in a text editor.

5. Update the `azureIdentity` values with your Azure identity's resource ID and client ID (retrievable from the CLI or portal).

6. Update the `azureIdentityBinding` selector value with a value that will match the label applied to the pods you wish to assign the identity to i.e. `selector: demo`.

7. Ensure you have helm initialized correctly to work with your cluster. If not, follow this [guide](https://docs.helm.sh/using_helm/#initialize-helm-and-install-tiller). If your cluster has rbac-enabled, you'll need to initialize tiller with a suitable `service-account`.

8. Install the helm chart into your Kubernetes cluster using your updated `values.yaml`.
```shell
helm install --values values.yaml .
```

9. Deploy your application to Kubernetes. The application should use [ADAL](https://docs.microsoft.com/en-us/azure/active-directory/develop/active-directory-authentication-libraries) to request a token from the MSI endpoint as usual. If you do not currently have such an application, a demo application is available [here](https://github.com/Azure/aad-pod-identity#demo-app). If you do use the demo application, please update the `deployment.yaml` with the appropriate subscription ID, client ID and resource group name. Also make sure the selector you defined in your `AzureIdentityBinding` matches the `aadpodidbinding` label on the deployment i.e. `aadpodidbinding: demo`.

10. Once you have successfully deployed your application, validate the MIC has detected your `AzureIdentityBinding` by viewing its logs.
```shell
kubectl logs mic-768489d94-pjxqf
...
I0919 13:12:34.222107       1 event.go:218] Event(v1.ObjectReference{Kind:"AzureIdentityBinding", Namespace:"default", Name:"msi-binding", UID:"UID", APIVersion:"aadpodidentity.k8s.io/v1", ResourceVersion:"231329", FieldPath:""}): type: 'Normal' reason: 'binding applied' Binding msi-binding applied on node aks-agentpool-12603492-1 for pod demo-77d858d9f9-62d4r-default-msi
```

11. Check the MIC has created a new `AzureAssignedIdentity` for your deployment.
```shell
kubectl get AzureAssignedIdentity
...
NAME                                CREATED AT
demo-77d858d9f9-62d4r-default-msi   1m
```

12. Check the NMI successfully retrieved a token for your app by viewing its logs
```shell
kubectl logs nmi-cn4cc
...
time="2018-09-19T13:56:20Z" level=info msg="Status (200) took 37041685 ns" req.method=GET req.path=/metadata/identity/oauth2/token req.remote=10.244.0.12
```

13. Finally, validate your application is behaving and logging as expected. The demo application will log details about the MSI acquisition
```
kubectl logs demo-77d858d9f9-62d4r
...
time="2018-09-19T13:14:28Z" level=info msg="succesfully acquired a token using the MSI, msiEndpoint(http://169.254.169.254/metadata/identity/oauth2/token)" podip=10.244.0.12 podname=demo-77d858d9f9-62d4r podnamespace=demo-77d858d9f9-62d4r
```

## Known Issues

__Error Redeploying Chart__

If you have previously installed the helm chart, you may come across the following error message:
```shell
Error: object is being deleted: customresourcedefinitions.apiextensions.k8s.io ? "azureassignedidentities.aadpodidentity.k8s.io" already exists
```
This is because helm doesn't actively manage the `CustomResourceDefinition` resources that the chart created. The full discussion concerning this issue is available [here](https://github.com/helm/helm/issues/2994). We are using helm's [crd-install hooks](https://docs.helm.sh/developing_charts#defining-a-crd-with-the-crd-install-hook) to provision the `CustomResourceDefintion` resources before the rest of the chart is verified and deployed. We also use `hook-delete-policy` to try and clean down the resources before the next helm release is applied. Unfortunately, as CRD deletion is slow it doesn't appear to resolve the issue. This issue is tracked [here](https://github.com/helm/helm/issues/4440). The easiest solution is to manually delete the `CustomerResourceDefintion` resources or setup a job to do so.


