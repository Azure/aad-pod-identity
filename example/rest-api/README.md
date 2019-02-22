# Using AAD Pod Identity with a custom REST API

#	Build docker container
sudo docker build -t vplauzon/aad-pod-id-svc .

#	Publish image
sudo docker push vplauzon/aad-pod-id-svc

kubectl run aad-id-rest-client --image=vplauzon/aad-pod-id-svc --generator=run-pod/v1

kubectl logs aad-id-rest-client

kubectl delete pod aad-id-rest-client