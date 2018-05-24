Applications running on the POD on Azure Container Service (AKS/ACS Engine) require access to identities in Azure Active Directory (AAD) to access resources that use an identity provider. AAD provides a construct called a Service Principal that allows applications to assume identities with limited permissions, and Managed Service Identity (MSI) - automatically generated and rotated credentials that easily retrieved by an application at run-time to authenticate as a service principal. 

An cluster admin configures the Azure Identity Binding to the Pod. Without any change of auth code the application running on the pod works on the cluster.

# Node Managed Identity (NMI)
The authorization request of fetching Service Principal Token from MSI endpoint is sent to a standard Instance Metadata endpoint which is redirected to the NMI pod by adding ruled to redirect POD CIDR traffic with metadata endpoint IP on port 80 to be sent to the NMI endpoint. 

The NMI server identifies the pod based on the remote address of the request and then queries the k8s (through MIC) for a matching azure identity. It then make a adal request to get the token for the client id and returns as a reponse to the request. If the request had client id as part of the query it validates it againsts the admin configured client id.

# Managed Identity Controller (MIC)
TBD

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

## Deploy Specs

### Requirement 
A running k8s cluster on Azure using AKS or ACS Engine 

### Deploy the azure-aad-identity infra 
```
kubectl create -f deploy/mic/deployment.yaml
kubectl create -f deploy/nmi/deployment.yaml
kubectl create -f deploy/demo/deployment.yaml
```
### Configure Identity Binding 
TBD

# Demo
## Requirement 
A running k8s cluster on Azure using AKS or ACS Engine 

## Scenario 
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

## Deploy Specs
```
kubectl create -f deploy/mic/deployment.yaml
kubectl create -f deploy/nmi/deployment.yaml
kubectl create -f deploy/demo/deployment.yaml
```
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
