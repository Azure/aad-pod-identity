# Using AAD Pod Identity with a custom REST API

## Intro?

myapi application

client-principal

## Prerequisites

* Install AAD Pod Identity

## Identity

```bash
rg=<name of the resource group>
cluster=<name of the AKS cluster>
```

```bash
az identity create -g $rg -n client-principal \
    --query "{ClientId: clientId, ManagedIdentityId: id, TenantId:  tenantId}" -o jsonc
```

```bash
aksPrincipalId=$(az aks show -g $rg -n $cluster --query "servicePrincipalProfile.clientId" -o tsv)
az role assignment create --role "Managed Identity Operator" --assignee $aksPrincipalId --scope <ManagedIdentityId>
```

## Identity & Binding in Kubernetes

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
az ad app create --display-name myapi \
    --identifier-uris http://myapi.restapi.aad-pod-identity \
    --query "appId" -o tsv
```

In client-pod.yaml, appId

In service.yaml, APPLICATION_ID & TENANT_ID

```bash
kubectl apply -f service.yaml --namespace pir
kubectl apply -f client-pod.yaml --namespace pir
```

```bash
kubectl get AzureAssignedIdentity --namespace pir
```

```bash
kubectl logs aad-id-client-pod --namespace pir  -f
```

## Clean up

```bash
kubectl delete namespace pir
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
