 kubectl apply -f .\aad-docker-crd.yaml
 kubectl apply -f .\aad-docker-secrets.yaml
 kubectl apply -f .\aad-docker-nmi.yaml
 kubectl apply -f .\aad-docker-mic.yaml
 kubectl apply -f .\app-deployment.yaml
 kubectl apply -f .\app-service.yaml


# kubectl delete -f .\aad-docker-crd.yaml
# kubectl delete -f .\aad-docker-secrets.yaml
# kubectl delete -f .\aad-docker-nmi.yaml
# kubectl delete -f .\aad-docker-mic.yaml
# kubectl delete -f .\app-deployment.yaml
# kubectl delete -f .\app-service.yaml
