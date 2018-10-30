# E2E Testing

End-to-end (e2e) testing is used to test whether the modules of AAD pod identity is behaving as designed as a group.

## Get Started

To run the e2e tests in a given Azure subscription, a running Kubernetes cluster created through acs-engine or Azure Kubernetes Service (AKS) is required. To collect the cluster's service principal credential, for AKS, you can refer to [here](https://docs.microsoft.com/en-us/azure/aks/kubernetes-service-principal). For acs-engine, if you have an existing cluster, search for the `servicePrincipalProfile` field in `apimodel.json` under the deployment folder. Otherwise, refer to [here](https://github.com/Azure/acs-engine/blob/master/docs/serviceprincipal.md).

Execute the following commands to run the e2e tests:

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
kubectl exec <pod name> -- identityvalidator --subscriptionid $SUBSCRIPTION_ID --resourcegroup $RESOURCE_GROUP --clientid $AZURE_CLIENT_ID

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

## Development

The test utilized [ginkgo](http://onsi.github.io/ginkgo/) as the base test framework. The tests are written in [aadpodidentity_test.go](aadpodidentity_test.go) and more tests can be appended at the end of the file.