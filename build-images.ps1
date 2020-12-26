docker build  --target nmi --build-arg IMAGE_VERSION=v1.7.1-docker -t wallsmedia/k8s/aad-pod-identity/nmi:v1.7.1-docker .
docker build  --target mic --build-arg IMAGE_VERSION=v1.7.1-docker -t wallsmedia/k8s/aad-pod-identity/mic:v1.7.1-docker .
docker build  --target micdocker --build-arg IMAGE_VERSION=v1.7.1-docker -t wallsmedia/k8s/aad-pod-identity/micdocker:v1.7.1-docker .
