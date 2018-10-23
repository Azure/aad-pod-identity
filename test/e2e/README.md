# E2E Testing

End-to-end (e2e) testing is used to automate the flow of

## Get Started

To run the e2e tests locally, a running k8s created through acs-engine or Azure Kubernetes Service (AKS) is required. If the client ID of the service principal is unknown, for AKS, you can refer to [here](https://github.com/Azure/aad-pod-identity#providing-required-permissions-for-mic). For acs-engine, search for the `servicePrincipalProfile` field in `apimodel.json` under the deployment folder.

Execute the following commands to run the e2e tests:

```bash
# Ensure that the local project and the dependencies are in sync
$ dep ensure

# Set environment variables before testing
# The Azure subscription ID
$ export SUBSCRIPTION_ID='...'

# The resource group name
$ export RESOURCE_GROUP='...'

# The client ID of the service principal that the Azure k8s cluster is using
$ export AZURE_CLIENT_ID='...'

$ make e2e
```

## Identity Validator

To validate the pod identity functionality, you can deploy the image [`identityvalidator`](../../images/identityvalidator/Dockerfile) as a Kubernetes deployment to the cluster. The binary `identityvalidator` within the pod is essentially the compiled version of [`identityvalidator.go`](identityvalidator/identityvalidator.go). If the binary execution returns an exit status of 0, it means that the pod identity and its binding are working properly. Otherwise, it means that the pod identity is not established. To execute the binary within the pod, execute the following command:
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


## Development

The test utilized [ginkgo](http://onsi.github.io/ginkgo/) as the base test framework. The tests are written in [aadpodidentity_test.go](aadpodidentity_test.go) and more tests can be appended at the end of the file.