pipeline {
	parameters {
		string defaultValue: '', description: 'Git repo to build from.', name: 'GIT_REPO', trim: false
		credentials credentialType: 'com.cloudbees.plugins.credentials.impl.UsernamePasswordCredentialsImpl', defaultValue: '', description: 'Git repo credentials.', name: 'GIT_REPO_CREDENTIALS', required: true
		string defaultValue: '', description: 'Git commit to build from.', name: 'GIT_COMMIT', trim: false
		string defaultValue: '', description: 'Git branch to build from.', name: 'GIT_BRANCH', trim: false

		string defaultValue: '', description: 'Name of the ACR registry to push images to.', name: 'REGISTRY_NAME', trim: false
		string defaultValue: '', description: 'The repository namespace to push the images to.', name: 'REGISTRY_REPO', trim: false
		credentials credentialType: 'com.microsoft.azure.util.AzureCredentials', defaultValue: '', description: 'Which stored credentials to use to push image to.', name: 'REGISTRY_CREDENTIALS', required: true


		string defaultValue: '', description: '', name: 'MIC_VERSION', trim: false
		string defaultValue: '', description: '', name: 'NMI_VERSION', trim: false
		string defaultValue: '', description: '', name: 'DEMO_VERSION', trim: false
		string defaultValue: '', description: '', name: 'IDENTITY_VALIDATOR_VERSION', trim: false
	}

	agent {
		docker {
			image "microsoft/azure-cli"
			args "-u root:root --mount type=bind,source=/var/run/docker.sock,target=/var/run/docker.sock"
		}
	}

	stages {
		stage("setup env") {
			steps {
				sh "apk add --no-cache docker make"
			}
		}

		stage("checkout source") {
			steps {
				git changelog: false, credentialsId: env.GIT_REPO_CREDENTIALS, poll: false, url: env.GIT_REPO
				sh "git checkout -b '${GIT_BRANCH}'"
				sh "git checkout -f '${GIT_COMMIT}'"
			}
		}

		stage('Build images') {
			steps {
				sh "make REGISTRY_NAME='${REGISTRY_NAME}' REGISTRY='${REGISTRY_NAME}.azurecr.io' REPO_PREFIX='${REGISTRY_REPO}' image"
			}
		}

		stage("Push images") {
			steps {
				withCredentials([azureServicePrincipal("${REGISTRY_CREDENTIALS}")]) {
						sh "az login --service-principal -u '${AZURE_CLIENT_ID}' -p '${AZURE_CLIENT_SECRET}' -t '${AZURE_TENANT_ID}'"
				}
				sh "az acr login -n '${REGISTRY_NAME}'"
				sh "make REGISTRY_NAME='${REGISTRY_NAME}' REGISTRY='${REGISTRY_NAME}.azurecr.io' REPO_PREFIX='${REGISTRY_REPO}' push"
			}
		}
	}
}
