# Setup AAD-Pod-Identity for Ks8 Docker-Desktop

For Docker Desktop Kubernetes you need special light version "micdocker:v1.7.1-docker" and paired "nmi:v1.7.1-docker".
The original current version from https://github.com/Azure/aad-pod-identity doesn't work,
because it has expected to run inside of Azure AKS cluster nodes or Azure VMs. 
The "micdocker" light version of "mic was tested and used with Azure "Service Principal".
It should work with other type of identities, but it was not tested. 

The setup instruction assumes that you already have the recent version of Docker with WSL2 and build-in [Docker Kubernetes](https://www.docker.com/products/kubernetes).
and enable the build in Kubernetes.

## Setup steps

Get Kubernetes Docker deployment files  from "\aad-pod-identity\deploy\docker" or https://github.com/Wallsmedia/aad-pod-identity/tree/docker-desktop/deploy/docker .

###  Create the fake entities

All add-pod-identity components expected to run in the Linux environment and expected to have access
to the kubernetes configuration files.
So, for Windows Docker desktop you have to create a fake directory and story them into one location.

In the example files the default location is **"d:\kubefake\"**;  
It corresponds the docker-kubernates mount path **"/run/desktop/mnt/host/d/kubefake/..."**. So you can easy amend it to your chosen location.

Copy fake config files from this directory "from "\aad-pod-identity\deploy\docker\kubefake" into the directory **"d:\kubefake\"**.
Copy config file from **%HOME%/.kube/config** into the **"d:\kubefake\"** and rename it as  **kubeconfig**.

### Deployment Files

#### aad-docker-crd.yaml
Just deploy/aaply this file with kubectl, it contains AAD-Pod-Identity CRD resources.
It doesn't require any customization.

#### aad-docker-secrets.yaml
The context of the file has been described in details in the example.
https://github.com/Azure/aad-pod-identity/blob/master/website/content/en/docs/Concepts/azureidentity.md
Expand example for a details.
It contains the Azure identity that will be used for obtaining the valid access token from the Azure Cloud
and provided to your application. By other words, 
In the Azure identity request provide none of credential information, but you will have a valid acess token.
You have to configer this file with your valid identities or nothing would work.

Here an example for configuring secrets for the "service principal":

```yaml
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

#### aad-docker-nmi.yaml

In this file you might need to change the location of the "kubefake" directory.

```yaml
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


#### aad-docker-mic.yaml

In this file you might need to change the location of the "kubefake" directory.
```yaml
  volumes:
      - name: kubeconfig
        hostPath:
          path: /run/desktop/mnt/host/d/kubefake/kubeconfig
```
Apply this file with kubectl.

## That all 
There is an example for an application deployment below.

Don't use in your application Azure CLI, it should not work and supported by AAD-Pod-Identity.

```yaml
---
# Source: deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: azureblazor-kv-access
  labels:
    app: azureblazor-kv-access
    release: "example"
    heritage: "Helm"
    namespace: default
    aadpodidbinding:  azure-service-principal-pod
spec:
  replicas: 1
  selector:
    matchLabels:
      app: azureblazor-kv-access
  template:
    metadata:
      labels:
        app: azureblazor-kv-access
        appVersion: v1
        aadpodidbinding:  azure-service-principal-pod
    spec:
      containers:
      - name: azureblazor-kv-access
        image: "azureblazorkvaccess.server:1.0"
        imagePullPolicy: Never
        env:
          - name: ASPNETCORE_ENVIRONMENT
            value: Development
        resources:
          requests:
            memory: "250Mi"
          limits:
            memory: "300Mi"
---
# Source: service.yaml
apiVersion: v1
kind: Service
metadata:
  name: azureblazor-kv-access
  labels:
    app: azureblazor-kv-access
    chart: "aazureblazor-kv-accessv0.1.0"
    release: "alex"
  annotations:
    service.beta.kubernetes.io/azure-load-balancer-internal: "true"
    service.beta.kubernetes.io/azure-load-balancer-internal-subnet: "api-ingress-subnet"
spec:
  ports:
    - protocol: TCP
      port: 80
      targetPort: 80
  selector:
    app: azureblazor-kv-access
  type: LoadBalancer
```
