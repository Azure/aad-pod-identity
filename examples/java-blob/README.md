# Spring Boot + Azure Storage + AAD Pod Identity

This tutorial is based on [this documentation](https://docs.microsoft.com/en-us/azure/developer/java/spring-framework/configure-spring-boot-starter-java-app-with-azure-storage).

This repository contains sample code and instructions demonstrating how you can access Azure Storage using a user-assigned identity with the aid of AAD Pod Identity from a spring boot application.

This sample takes a `POST` request and saves it as a text file in an Azure Blob Storage container.

## Prerequisites

* an AKS cluster with [managed identity enabled](https://docs.microsoft.com/en-us/azure/aks/use-managed-identity)
* AAD Pod Identity installed and configured
* Azure Container Registry
* Azure Storage
    * Make sure you have already created a container

## Setup

### Create a Managed Identity and Assign Roles

In this step, we'll create a new user-assigned identity which will be used to interact with the Azure Storage account.

**Note:** this step can be skipped if you already have a manage identity you'd like to reuse.

1. Setup:

    ```sh
    CLUSTER_NAME=<YOUR_AKS_CLUSTER_NAME>
    CLUSTER_RESOURCE_GROUP=$(az aks list --query "[?name == '$CLUSTER_NAME'].resourceGroup" -o tsv)

    IDENTITY_NAME=<YOUR_IDENTITY_NAME>
    IDENTITY_RESOURCE_GROUP=$(az aks show -g $CLUSTER_RESOURCE_GROUP -n $CLUSTER_NAME --query nodeResourceGroup -otsv)

    STORAGE_ACCOUNT_NAME=<STORAGE_ACCOUNT_NAME>
    STORAGE_ACCOUNT_RESOURCE_GROUP=$(az storage account list --query "[?name == '$STORAGE_ACCOUNT_NAME'].resourceGroup" -o tsv)

    CLUSTER_MSI_CLIENT_ID=$(az aks show \
        -n $CLUSTER_NAME \
        -g $CLUSTER_RESOURCE_GROUP \
        --query "identityProfile.kubeletidentity.clientId" \
        -o tsv)

    STORAGE_ACCOUNT_RESOURCE_ID=$(az storage account show \
        --name $STORAGE_ACCOUNT_NAME \
        --query 'id' -o tsv)
    ```

2. Create the manage identity:

    ```sh
    IDENTITY_RESOURCE_ID=$(az identity create \
        --name $IDENTITY_NAME \
        --resource-group $IDENTITY_RESOURCE_GROUP \
        --query 'id' -o tsv)

    IDENTITY_CLIENT_ID=$(az identity show \
        -n $IDENTITY_NAME \
        -g $IDENTITY_RESOURCE_GROUP \
        --query 'clientId' -o tsv)
    ```

3. Assign role to allow AAD Pod Identity to access our newly created managed identity:

    ```sh
    az role assignment create \
        --role "Managed Identity Operator" \
        --assignee $CLUSTER_MSI_CLIENTID \
        --scope $IDENTITY_RESOURCE_ID
    ```

4. Grant permission to specific container in the Azure Storage Account:

    ```sh
    CONTAINER=test

    az role assignment create \
        --role "Storage Blob Data Contributor" \
        --assignee $IDENTITY_CLIENT_ID \
        --scope "$STORAGE_ACCOUNT_RESOURCE_ID/blobServices/default/containers/$CONTAINER"
    ```

    **Note**: if you want the managed identity to access your entire Storage Account, you can ignore `/blobServices/default/containers/$(CONTAINER)`.

### Configure AAD Pod Identity

The following step will create a new `AzureIdentity` resource in Kubernetes in the `blob` namespace. If your namespace is something different then change NAMESPACE to match the namespace you want to deploy into.

```sh
NAMESPACE=blob
kubectl create namespace $NAMESPACE

cat << EOF | kubectl apply -f -
apiVersion: "aadpodidentity.k8s.io/v1"
kind: AzureIdentity
metadata:
  name: $IDENTITY_NAME
  namespace: $NAMESPACE
spec:
  type: 0
  resourceID: $IDENTITY_RESOURCE_ID
  clientID: $IDENTITY_CLIENT_ID
EOF
```

Now we'll create the `AzureIdentityBinding` and specify the selector.

```sh
POD_LABEL_SELECTOR=$IDENTITY_NAME

cat << EOF | kubectl apply -f -
apiVersion: aadpodidentity.k8s.io/v1
kind: AzureIdentityBinding
metadata:
  name: $IDENTITY_NAME-binding
  namespace: $NAMESPACE
spec: 
  azureIdentity: $IDENTITY_NAME
  selector: $POD_LABEL_SELECTOR
EOF
```

## Build

In the root of this project contains a `Dockerfile`. All you need to do is build the container and push to Azure Container Registry.

```sh
ACR=<YOUR_ACR>
IMG=$(ACR).azurecr.io/azure-storage-example:1

az acr login --name $ACR

docker build -t $IMG .
docker push $IMG
```

## Deploy

```sh
cat << EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: demo-blob-deployment
  namespace: $NAMESPACE
spec:
  replicas: 1
  selector:
    matchLabels:
      name: demo-blob
  template:
    metadata: 
      name: demo-blob
      labels:
        name: demo-blob
        aadpodidbinding: $POD_LABEL_SELECTOR
    spec:
      containers:
      - name: demo-blob
        image: $IMG
        env:
          - name: AZURE_CLIENT_ID
            value: $IDENTITY_CLIENT_ID
          - name: BLOB_ACCOUNT_NAME
            value: $STORAGE_ACCOUNT_NAME
          - name: BLOB_CONTAINER_NAME
            value: $CONTAINER
      nodeSelector:
        kubernetes.io/os: linux
---
apiVersion: v1
kind: Service
metadata:
  name: demo-blob-service
  namespace: $NAMESPACE
spec:
  type: NodePort
  selector:
    name: demo-blob
  ports:
    - port: 8080
      targetPort: 8080
EOF
```
## Test

To interact with the `demo-blob` pod running in AKS, we first need a pod with cURL installed. The following command create a new pod called `curl` using the `curlimages/curl` image, and then execute into the pod.

```sh
kubectl run curl --rm -i --tty --image=curlimages/curl:7.73.0 -n blob -- sh
```

Once you have access into the `curl` pod, run the following command to upload a new blob into the Azure Storage Account:

```sh
curl -d 'new message' -H 'Content-Type: text/plain' demo-blob-service:8080/
```

Assuming everything works, this will create a new `.txt` file the container created earlier with the text `new message`.

You should receive the following message:

```sh
file quickstart-8fea77d9-d133-4cb1-8f16-391dc8e4e3f7.txt was uploaded
```

To retrieve the contents of the blob, assuming the file was saved under `quickstart-8fea77d9-d133-4cb1-8f16-391dc8e4e3f7.txt`, make the following `GET` request:

```sh
curl -X GET http://demo-blob-service:8080/?fileName=quickstart-8fea77d9-d133-4cb1-8f16-391dc8e4e3f7.txt
```

You should get back the following:

```sh
new message
```
