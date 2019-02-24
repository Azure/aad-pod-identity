# Using AAD Pod Identity with a custom REST API

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
