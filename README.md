# Important Note

This is a fork repository of the AAD Pod Identity https://github.com/Azure/aad-pod-identity repository. 
The default branch "docker-desktop" has been created in an attempt to make contribution or enhancement request https://azure.github.io/aad-pod-identity. 
See the status of the effort/issue/enhancement https://github.com/Azure/aad-pod-identity/issues/921.

Regardless of all above, the branch "docker-desktop" contains No-Azure (running outside Azure cloud) kubernetes pods, or docker containers and support AAD Pod Identity.
"MicDocker" module is the light version of the "mic" module,
and allows todo almost the same, but not calling/depend on Azure cloud provider, i.e. it is running outside Azure.
You can use the local docker kubernetes containers for development Azure\AKS containers.
It is much easy then do the similar things in AKS. This was the main goal of development. The version works fine with "service-principal" type of identity,
others should work too, but you probably not need them.  

The License file has terms of use. 

The prebuilt images are located on the public hub.docker.com container repository. However, you can build, in minutes,
the own ones. Edit/run build-ks8-docker-images.ps1 script.
All you need is to set up the recent version of Docker Desktop with  WSL2 and build-in Docker Kubernetes  https://www.docker.com/products/kubernetes.

### Step-by-step setup

Please refer to [Docker-Setup.md](https://github.com/Wallsmedia/aad-pod-identity/blob/docker-desktop/Docker-Setup.md)

# AAD Pod Identity

[![Build Status](https://dev.azure.com/azure/aad-pod-identity/_apis/build/status/aad-pod-identity-nightly?branchName=master)](https://dev.azure.com/azure/aad-pod-identity/_build/latest?definitionId=77&branchName=master)
[![codecov](https://codecov.io/gh/Azure/aad-pod-identity/branch/master/graph/badge.svg)](https://codecov.io/gh/Azure/aad-pod-identity)
[![GoDoc](https://godoc.org/github.com/Azure/aad-pod-identity?status.svg)](https://godoc.org/github.com/Azure/aad-pod-identity)
[![Go Report Card](https://goreportcard.com/badge/github.com/Azure/aad-pod-identity)](https://goreportcard.com/report/github.com/Azure/aad-pod-identity)

AAD Pod Identity enables Kubernetes applications to access cloud resources securely with [Azure Active Directory](https://azure.microsoft.com/en-us/services/active-directory/).

Using Kubernetes primitives, administrators configure identities and bindings to match pods. Then without any code modifications, your containerized applications can leverage any resource in the cloud that depends on AAD as an identity provider.

## Getting Started

Setup the correct [role assignments](https://azure.github.io/aad-pod-identity/docs/getting-started/role-assignment/) on Azure and install AAD Pod Identity through [Helm](https://azure.github.io/aad-pod-identity/docs/getting-started/installation/#helm) or [YAML deployment files](https://azure.github.io/aad-pod-identity/docs/getting-started/installation/#quick-install). Get familiar with our [CRDs](https://azure.github.io/aad-pod-identity/docs/concepts/azureidentity/) and [core components](https://azure.github.io/aad-pod-identity/docs/concepts/mic/).

Try our [walkthrough](https://azure.github.io/aad-pod-identity/docs/demo/standard_walkthrough/) to get a better understanding of the application workflow.

## Code of Conduct

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/). For more information, see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq) or contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.

## Support

aad-pod-identity is an open source project that is [**not** covered by the Microsoft Azure support policy](https://support.microsoft.com/en-us/help/2941892/support-for-linux-and-open-source-technology-in-azure). [Please search open issues here](https://github.com/Azure/aad-pod-identity/issues), and if your issue isn't already represented please [open a new one](https://github.com/Azure/aad-pod-identity/issues/new/choose). The project maintainers will respond to the best of their abilities.
