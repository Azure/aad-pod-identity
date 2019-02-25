# Using AAD Pod Identity with a custom REST API

## Intro?

myapi application

principal1, principal2

## Create Identity & applications

```bash
az identity create -g aks2 -n client-principal
```

```bash
az ad app create --display-name myapi --identifier-uris http://myapi.restapi.aad-pod-identity
```

Using groups:  issue https://github.com/Azure/azure-cli/issues/7283

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
