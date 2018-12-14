pipeline {
    agent {
        docker {
            image 'microsoft/azure-cli'
            args '-u root:root -v /var/run/docker.sock:/var/run/docker.sock'
        }
    }
    stages {
        stage('Setup building environment') {
            steps {
                // musl-dev is a dependency of go
                sh 'apk upgrade && apk update && apk add --no-cache go make musl-dev docker'
                withCredentials([azureServicePrincipal("${SP_CREDENTIAL}")]) {
                    sh 'az login --service-principal -u "$AZURE_CLIENT_ID" -p "$AZURE_CLIENT_SECRET" -t "$AZURE_TENANT_ID"'
                }
                sh 'az acr login -n "$REGISTRY_NAME"'
            }
        }

        stage('Build images and push to registry') {
            steps {
                sh 'go get -u github.com/golang/dep/cmd/dep'
                sh 'go get github.com/Azure/aad-pod-identity || true'
                sh 'cd /root/go/src/github.com/Azure/aad-pod-identity && /root/go/bin/dep ensure && make build && make image && make push'
            }
        }
    }
}
