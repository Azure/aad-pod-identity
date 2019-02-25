# Using AAD Pod Identity with a custom REST API

We will look at the scenario where a client pod is calling a service (REST API) with an AAD Pod Identity.

The client is a bash script which will request a token, pass it as authorization HTTP header to query a REST API.  The client has a corresponding user managed identity which will be exposed as an AAD Pod Identity.

The REST API is implemented in C# / .NET Core.  It simply validates it receives a bearer token in the authorization header of each request.  The REST API has a corresponding Azure AD Application.  The client requests a token with that AAD application as the *resource*.

**<span style="background:yellow">TODO:  simple diagram showing client / service pods + identities involved ; everything is name so no surprised in the script</span>**

## Prerequisites

* An AKS cluster with [AAD Pod Identity installed on it](https://github.com/Azure/aad-pod-identity/blob/master/README.md)

## Identity

In this section, we'll create the user managed identity used for the client.

First, let's define those variable:

```bash
rg=<name of the resource group where AKS is>
cluster=<name of the AKS cluster>
```

Then, let's create the user managed identity:

```bash
az identity create -g $rg -n client-principal \
    --query "{ClientId: clientId, ManagedIdentityId: id, TenantId:  tenantId}" -o jsonc
```

This returns three values in a JSON document.  We will use those values later on.

We need to give the Service Principal running the cluster the *Managed Identity Operator* role on the user managed identity:

```bash
aksPrincipalId=$(az aks show -g $rg -n $cluster --query "servicePrincipalProfile.clientId" -o tsv)
managedId=$(az identity show -g $rg -n client-principal \
    --query "id" -o tsv)
az role assignment create --role "Managed Identity Operator" --assignee $aksPrincipalId --scope $managedId
```

The first line acquires the AKS service principal client ID.  The second line acquires the client ID of the user managed identity (the *ManagedIdentityId* returned in the JSON above).  The third line performs the role assignment.

## Identity & Binding in Kubernetes

In this section, we'll configure AAD pod identity with the user managed identity.

We'll create a Kubernetes namespace to put all our resources.  It makes it easier to clean up afterwards.

```bash
kubectl create namespace pir
kubectl label namespace/pir description=PodIdentityRestApi
```

In identity.yaml, ResourceID with principal's Id & ClientID with principal's ClientId...

```bash
kubectl apply -f identity.yaml --namespace pir
kubectl apply -f binding.yaml --namespace pir
```

```bash
kubectl get AzureIdentity --namespace pir
kubectl get AzureIdentityBinding --namespace pir
```

## Application

```bash
appId=$(az ad app create --display-name myapi \
    --identifier-uris http://myapi.restapi.aad-pod-identity \
    --query "appId" -o tsv)
echo $appId
```

In client-pod.yaml, appId

In service.yaml, APPLICATION_ID & TENANT_ID

```bash
kubectl apply -f service.yaml --namespace pir
kubectl apply -f client-pod.yaml --namespace pir
kubectl get AzureAssignedIdentity --all-namespaces
```

```bash
kubectl logs aad-id-client-pod --namespace pir  -f
```

```bash
kubectl delete AzureIdentityBinding client-principal-binding --namespace pir
```

```bash
kubectl apply -f binding.yaml --namespace pir
```

## Clean up

```bash
kubectl delete namespace pir
az identity delete -g $rg -n client-principal
az ad app delete --id $appId
```

#	Build client docker container
sudo docker build -t vplauzon/aad-pod-id-client .

#	Publish client image
sudo docker push vplauzon/aad-pod-id-client

#kubectl run aad-id-rest-client --image=vplauzon/aad-pod-id-svc --generator=run-pod/v1
kubectl apply -f pod.yaml

kubectl logs test-svc-pod

kubectl delete pod test-svc-pod

#	Build service docker container
sudo docker build -t vplauzon/aad-pod-id-svc .

#	Publish service image
sudo docker push vplauzon/aad-pod-id-svc
