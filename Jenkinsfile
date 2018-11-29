pipeline {
    agent {
        docker {
            image 'microsoft/azure-cli'
            args '-u root:root'
        }
    }
    environment {
        KUBECONFIG = credentials("${KUBECONFIG}")
    }
    stages {
        stage('Setup testing environment') {
            steps {
                withCredentials([azureServicePrincipal("${SP_CREDENTIAL}")]) {
                    sh 'az login --service-principal -u $AZURE_CLIENT_ID -p $AZURE_CLIENT_SECRET -t $AZURE_TENANT_ID'
                }
                sh 'az aks install-cli'
                sh 'kubectl cluster-info'
                sh 'apk upgrade && apk update && apk add --no-cache go make musl-dev'
                sh 'go get -u github.com/golang/dep/cmd/dep'
                sh 'go get github.com/$GITHUB_USERNAME/aad-pod-identity || true'
                sh 'mv /root/go/src/github.com/$GITHUB_USERNAME /root/go/src/github.com/Azure'
                sh 'cd /root/go/src/github.com/Azure/aad-pod-identity && git checkout "$BRANCH" && /root/go/bin/dep ensure'
            }
        }

        stage('Unit Test') {
            steps {
                sh 'cd /root/go/src/github.com/Azure/aad-pod-identity && make unit-test || true'
            }
        }

        stage('End-to-end test') {
            steps {
                ansiColor('xterm') {
                    withCredentials([azureServicePrincipal("${SP_CREDENTIAL}")]) {
                        sh 'cd /root/go/src/github.com/Azure/aad-pod-identity && make e2e'
                    }
                }
            }
        }
    }
}
