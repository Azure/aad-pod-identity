### Setup AAD-Pod-Identity for Ks8 Docker-Desktop
For Docker Desktop Kubernetes you need special light version "micdocker:v1.7.1-docker" and paired "nmi:v1.7.1-docker".
The original version from https://github.com/Azure/aad-pod-identity doesn't work as it hard coded to run inside of Azure cloud.
The implementation was tested and used with "Azure Service Principal". It shold work with other types identies, but not it was not tested.  

This is the fork of the original repository https://github.com/Wallsmedia/aad-pod-identity and has light implementation version of the mic docker version.

The setup instruction is based/applicable on the recent version of Docker with WSL2 and build-in [Docker Kubernetes](https://www.docker.com/products/kubernetes).

### Setup steps

1. Get Kubernetes Docker deployment files  from "\aad-pod-identity\deploy\docker" or https://github.com/Wallsmedia/aad-pod-identity/tree/docker-desktop/deploy/docker .

### aad-docker-crd.yaml
Just deploy/aaply this file with kubectl, it has CRD resources. It doesn't require any customization.

### aad-docker-secrets.yaml
The context of the file has been described in details in the example.
https://github.com/Azure/aad-pod-identity/blob/master/website/content/en/docs/Concepts/azureidentity.md

Expand example for a details. It contains the Azure identity that will be used to obtain the valid access token from the Azure Cloud.
Here an example for "service principal":

``` yaml
apiVersion: "aadpodidentity.k8s.io/v1"
kind: AzureIdentity
metadata:
  name: azure-service-principal
spec:
  type: 1
##resourceID: RESOURCE_ID
  clientID: "X23b7685-XXXX-XXXX-XXXX-9c0b63bc28fX"
  tenantID: "X367c01e-XXXX-XXXX-XXXX-cfc2beb7264X"
  clientPassword:
    name: "azdevsecret"
    namespace: "default"
---
apiVersion: v1
kind: Secret
metadata:
  name: azdevsecret
type: Opaque
stringData:
    clientSecret: "TO...................K4kq......1LR"
---
apiVersion: "aadpodidentity.k8s.io/v1"
kind: AzureIdentityBinding
metadata:
  name: azure-service-principal-binding
spec:
  azureIdentity: azure-service-principal
  selector: azure-service-principal-pod
```
###  Create the fake entities
All add-pod-identity components expected to run in the Linux environment and expected to have access
to the kubenetes configuration files. So Windows Docker desktop you have to create and collect them in one location.

In the example files the location is **"d:\kubefake\"**
It corresponds the docker-kubernates mount path **"/run/desktop/mnt/host/d/kubefake/..."**

Copy files from this directory "from "\aad-pod-identity\deploy\docker\kubefake" into your directory **"d:\kubefake\"**.

Copy config file from **%HOME%/.kube/config** into the **"d:\kubefake\"** and rename it as  **kubeconfig**.

### aad-docker-nmi.yaml

In this file you might need to change the location of the "kubefake" directory.

``` yaml
   volumes:
      - name: iptableslock
        hostPath:
          path: /run/desktop/mnt/host/d/kubefake/xtables.lock
          type: FileOrCreate
      - name: kubelet-config
        hostPath:
          path: /run/desktop/mnt/host/d/kubefake/kubelet
          
```
Apply this file with kubectl.


### aad-docker-mic.yaml

In this file you might need to change the location of the "kubefake" directory.
```yaml
  volumes:
      - name: kubeconfig
        hostPath:
          path: /run/desktop/mnt/host/d/kubefake/kubeconfig
```
Apply this file with kubectl.
