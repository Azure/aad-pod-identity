#!/bin/bash

set -x

# Give SP ID and secret, subscription ID

# Test 1
# create a k8s cluster using acs-engine
# az login -> create user-assigned identity on Azure
# Assign role 'Managed Identity Operator' to cluster SP
# Create an identity crd + identity binding
# Create a pod with labels that will trigger MIC to create a AzureAssignedIdentity
# Test the identity within the pod

# Setup k8s CRD, Azure Identity, assign roles to identity to SP
kubectl apply -f https://raw.githubusercontent.com/Azure/aad-pod-identity/master/deploy/infra/deployment-rbac.yaml

readonly SUBSCRIPTION_ID=''
readonly AZURE_CLIENT_ID=''
readonly RESOURCE_GROUP=''
readonly IDENTITY_NAME=''

az identity create -g $RESOURCE_GROUP -n $IDENTITY_NAME
readonly IDENTITY_PRINCIPAL_ID=$(az identity show -g aad-pod-identity-e2e -n "$IDENTITY_NAME" --query 'principalId' -otsv)
readonly IDENTITY_CLIENT_ID=$(az identity show -g aad-pod-identity-e2e -n "$IDENTITY_NAME" --query 'clientId' -otsv)
az role assignment create --role Reader --assignee "$IDENTITY_PRINCIPAL_ID" --scope "/subscriptions/$SUBSCRIPTION_ID/resourceGroups/$RESOURCE_GROUP"
az role assignment create --role 'Managed Identity Operator' --assignee "$AZURE_CLIENT_ID" --scope "/subscriptions/$SUBSCRIPTION_ID/resourcegroups/$RESOURCE_GROUP/providers/Microsoft.ManagedIdentity/userAssignedIdentities/$IDENTITY_NAME"

# Deploy a pod identity
cat <<EOF | kubectl apply -f -
apiVersion: "aadpodidentity.k8s.io/v1"
kind: AzureIdentity
metadata:
  name: $IDENTITY_NAME
spec:
  type: 0
  ResourceID: /subscriptions/$SUBSCRIPTION_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.ManagedIdentity/userAssignedIdentities/$IDENTITY_NAME
  ClientID: $IDENTITY_CLIENT_ID
EOF

# Deploy an identity binding
cat <<EOF | kubectl apply -f -
apiVersion: "aadpodidentity.k8s.io/v1"
kind: AzureIdentityBinding
metadata:
  name: $IDENTITY_NAME-binding
spec:
  AzureIdentity: $IDENTITY_NAME
  Selector: $IDENTITY_NAME
EOF

# Deploy a demo app
cat <<EOF | kubectl apply -f -
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  labels:
    app: demo
    aadpodidbinding: $IDENTITY_NAME
  name: demo
  namespace: default
spec:
  template:
    metadata:
      labels:
        app: demo
        aadpodidbinding: $IDENTITY_NAME
    spec:
      containers:
      - name: demo
        image: "mcr.microsoft.com/k8s/aad-pod-identity/demo:1.2"
        imagePullPolicy: Always
        args:
          - "--subscriptionid=$SUBSCRIPTION_ID"
          - "--clientid=$IDENTITY_CLIENT_ID"
          - "--resourcegroup=$RESOURCE_GROUP"
        env:
        - name: MY_POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: MY_POD_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: MY_POD_IP
          valueFrom:
            fieldRef:
              fieldPath: status.podIP
EOF

# Wait until demo pod is running
kubectl get AzureIdentity
kubectl get AzureIdentityBinding
kubectl get AzureAssignedIdentity
