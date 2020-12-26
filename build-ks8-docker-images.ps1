docker build  --target nmi --build-arg IMAGE_VERSION=v1.7.1-docker -t wallsmedia/nmi:v1.7.1-docker .
# docker build  --target mic --build-arg IMAGE_VERSION=v1.7.1-docker -t wallsmedia/mic:v1.7.1-docker .
docker build  --target micdocker --build-arg IMAGE_VERSION=v1.7.1-docker -t wallsmedia/micdocker:v1.7.1-docker .

docker push   wallsmedia/nmi:v1.7.1-docker
# docker push   wallsmedia/mic:v1.7.1-docker
docker push   wallsmedia/micdocker:v1.7.1-docker
