kubectl create -f ../crd/azureIdentityCrd.yaml
& go run ../controller.go -kubeconfig ~/kube/config
kubectl create -f simpletest-azureidentity-object.yaml