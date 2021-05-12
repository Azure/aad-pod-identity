#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

[[ ! -z "${SUBSCRIPTION_ID:-}" ]] || (echo 'Must specify SUBSCRIPTION_ID' && exit 1)
[[ ! -z "${RESOURCE_GROUP:-}" ]] || (echo 'Must specify RESOURCE_GROUP' && exit 1)
[[ ! -z "${CLUSTER_NAME:-}" ]] || (echo 'Must specify CLUSTER_NAME' && exit 1)

if ! az account set -s "${SUBSCRIPTION_ID}"; then
  echo "az login as a user and set the appropriate subscription ID"
  az login
  az account set -s "${SUBSCRIPTION_ID}"
fi

if [[ -z "${NODE_RESOURCE_GROUP:-}" ]]; then
  echo "Retrieving your node resource group"
  NODE_RESOURCE_GROUP="$(az aks show -g ${RESOURCE_GROUP} -n ${CLUSTER_NAME} --query nodeResourceGroup -otsv)"
fi

echo "Retrieving your cluster identity ID, which will be used for role assignment"
ID="$(az aks show -g ${RESOURCE_GROUP} -n ${CLUSTER_NAME} --query servicePrincipalProfile.clientId -otsv)"

echo "Checking if the aks cluster is using managed identity"
if [[ "${ID:-}" == "msi" ]]; then
  ID="$(az aks show -g ${RESOURCE_GROUP} -n ${CLUSTER_NAME} --query identityProfile.kubeletidentity.clientId -otsv)"
fi

echo "Assigning 'Managed Identity Operator' role to ${ID}"
az role assignment create --role "Managed Identity Operator" --assignee "${ID}" --scope "/subscriptions/${SUBSCRIPTION_ID}/resourcegroups/${NODE_RESOURCE_GROUP}"

echo "Assigning 'Virtual Machine Contributor' role to ${ID}"
az role assignment create --role "Virtual Machine Contributor" --assignee "${ID}" --scope "/subscriptions/${SUBSCRIPTION_ID}/resourcegroups/${NODE_RESOURCE_GROUP}"

# your resource group that is used to store your user-assigned identities
# assuming it is within the same subscription as your AKS node resource group
if [[ -n "${IDENTITY_RESOURCE_GROUP:-}" ]]; then
  echo "Assigning 'Managed Identity Operator' role to ${ID} with ${IDENTITY_RESOURCE_GROUP} resource group scope"
  az role assignment create --role "Managed Identity Operator" --assignee "${ID}" --scope "/subscriptions/${SUBSCRIPTION_ID}/resourcegroups/${IDENTITY_RESOURCE_GROUP}"
fi
