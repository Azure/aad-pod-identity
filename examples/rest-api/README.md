# Using AAD Pod Identity with a custom REST API

#	Build docker container
sudo docker build -t vplauzon/aad-pod-id-svc .

#	Publish image
sudo docker push vplauzon/aad-pod-id-svc

#kubectl run aad-id-rest-client --image=vplauzon/aad-pod-id-svc --generator=run-pod/v1
kubectl apply -f pod.yaml

kubectl logs test-svc-pod

kubectl delete pod test-svc-pod