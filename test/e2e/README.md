# E2E Testing

End-to-end (e2e) testing is used to test whether the modules of AAD pod identity is behaving as designed as a group.

## Get Started

To run the e2e tests in a given Azure subscription, a running Kubernetes cluster created through acs-engine or Azure Kubernetes Service (AKS) is required. To collect the cluster's service principal credential, for AKS, you can refer to [here](https://docs.microsoft.com/en-us/azure/aks/kubernetes-service-principal). For acs-engine, if you have an existing cluster, search for the `servicePrincipalProfile` field in `apimodel.json` under the deployment folder. Otherwise, refer to [here](https://github.com/Azure/acs-engine/blob/master/docs/serviceprincipal.md). Also, an Azure keyvault is created to simulate the action of accessing Azure resources.

Currently, e2e tests does not automate the creation of Azure resources such as identities and keyvault (but plan to support it in the future). In order to run the tests without errors, you have to create a keyvault within the same resource group and insert a test secret into the keyvault.

Note that the tests only utilized two identities, `keyvault-identity` (have read access to the keyvault that you create) and `cluster-identity` (have read access to the resource group level). To programmatically create a `keyvault-identity` on Azure, you can use `CreateIdentity` from `test/common/azure/azure.go` before running the tests.

The e2e tests utilizes environment variables to extract sensitive information. Execute the following commands to run the e2e tests:

```bash
cd $GOPATH/src/github.com/Azure/aad-pod-identity

# Ensure that the local project and the dependencies are in sync
$ dep ensure

# Set environment variables before testing
# The Azure subscription ID
$ export SUBSCRIPTION_ID=$(az account list --query "[?name=='<Azure subscription name>'].id" -otsv)

# The Azure resource group name of the Kubernetes cluster
$ export RESOURCE_GROUP='...'

# The client ID of the service principal that the Azure Kubernetes cluster is using
$ export AZURE_CLIENT_ID='...'

# The name of the keyvault
$ export KEYVAULT_NAME='...'

# Tee name of the secret inserted into the keyvault
$ export KEYVAULT_SECRET_NAME='...'

# Tee version of the secret inserted into the keyvault
$ export KEYVAULT_SECRET_VERSION='...'

$ make e2e
```

## Identity Validator

During the e2e test run , the image [`identityvalidator`](../../images/identityvalidator/Dockerfile) is deployed as a Kubernetes deployment to the cluster to validate the pod identity. The binary `identityvalidator` within the pod is essentially the compiled version of [`identityvalidator.go`](identityvalidator/identityvalidator.go). If the binary execution returns an exit status of 0, it means that the pod identity and its binding are working properly. Otherwise, it means that the pod identity is not established. To execute the binary within the pod, execute the following command:
```bash
# Create an identityvalidator (make sure the go template parameters is replaced by the desired values)
$ kubectl apply -f test/e2e/template/deployment.yaml

# Get the pod name of identityvalidator deployment
$ kubectl get pods

# Execute the binary within the pod
kubectl exec <pod name> -- identityvalidator --subscription-id $SUBSCRIPTION_ID \
                                             --resource-group $RESOURCE_GROUP \
                                             --client-id $AZURE_CLIENT_ID \
                                             --keyvault-name $KEYVAULT_NAME \
                                             --keyvault-secret-name $KEYVAULT_SECRET_NAME \
                                             --keyvault-secret-version $KEYVAULT_SECRET_VERSION

# Check the exit status
echo "$?"
```

## Test Flow

To ensure consistency across all tests, they generally follow the format below:

1. Alter the Azure resource group (create user assigned identities, assign reader role, etc)
2. Alter the Kubernetes cluster (creating pods, deployments, services, etc)
3. Inject Kubernetes pod identities and pod identity bindings
4. Deploy the identity validator to the cluster
5. Assertions
6. Clean up the testing environment

## Supported Tests

| Test Description | Expected Result | Category |
| - | - | - |
| Add an AzureIdentity and AzureBinding, deploy identity validator with the label marked in binding | New AzureAssignedIdentity is created and the underlying node assigned identity, and identity validator should be able to access Azure resources | Basic |
| With AzureIdentity, AzureBinding and identity validator deployed, remove the AzureIdentity | AzureAssignedIdentity should get removed and identity validator should not be able to access Azure resources | Basic |
| With AzureIdentity, AzureBinding and identity validator deployed, remove the AzureIdentityBinding | AzureAssignedIdentity should get removed and identity validator should not be able to access Azure resources | Basic |
| With AzureIdentity, AzureBinding and identity validator deployed, remove the identity validator deployment | AzureAssignedIdentity should get removed | Basic |
| Add an AzureIdentity and AzureBinding, deploy identity validator with the label marked in binding, then drain the node containing the identity validator deployment | A new AzureAssignedIdentity should be established with the new pod and the old one should be removed | Advanced |
| Add a number of AzureIdentities and AzureIdentityBindings in order and remove them in random order | The correct identities and identity binding should be removed and the rest should remain untouched | Random |
| Enable a user assigned identity on VMs, then assign a diferent user assigned identity to pod | Pod identity should work as expected and the user assigned identity on VMs should not be altered after deleting the pod identity | Advanced |
| Enable a user assigned identity on VMs, then assign the same user assigned identity to pod | Pod identity should work as expected and the user assigned identity on VMs should not be altered after deleting the pod identity | Advanced |
| Enable system assigned identity on VMs, then assign a user assigned identity to pod | Pod identity should work as expected and the system assigned identity on VMs should not be altered after deleting the pod identity | Advanced |

## Development

The test utilized [ginkgo](http://onsi.github.io/ginkgo/) as the base test framework. The tests are written in [aadpodidentity_test.go](aadpodidentity_test.go) and more tests can be appended at the end of the file.