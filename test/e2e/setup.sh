#!/bin/bash

set -euo pipefail

# Check pre-requisites
[[ ! -z "${SUBSCRIPTION_ID:-}" ]]         || (echo 'Must specify SUBSCRIPTION_ID' && exit -1)
[[ ! -z "${RESOURCE_GROUP:-}" ]]          || (echo 'Must specify RESOURCE_GROUP' && exit -1)
[[ ! -z "${AZURE_CLIENT_ID:-}" ]]         || (echo 'Must specify AZURE_CLIENT_ID' && exit -1)
[[ ! -z "${KEYVAULT_NAME:-}" ]]           || (echo 'Must specify KEYVAULT_NAME' && exit -1)
[[ ! -z "${KEYVAULT_SECRET_NAME:-}" ]]    || (echo 'Must specify KEYVAULT_SECRET_NAME' && exit -1)
[[ -z "$(hash az)" ]]                     || (echo 'Azure CLI not found' && exit -1)
[[ -z "$(hash kubectl)" ]]                || (echo 'kubectl not found' && exit -1)

# Create a keyvault
az keyvault create -g "$RESOURCE_GROUP" -n "$KEYVAULT_NAME"
az keyvault secret set --vault-name "$KEYVAULT_NAME" -n "$KEYVAULT_SECRET_NAME" --value test-value

echo 'Creating a keyvault-identity and assign appropriate roles...'
az identity create -g "$RESOURCE_GROUP" -n keyvault-identity
IDENTITY_CLIENT_ID=$(az identity show -g "$RESOURCE_GROUP" -n keyvault-identity  --query 'clientId' -otsv)
IDENTITY_PRINCIPAL_ID=$(az identity show -g "$RESOURCE_GROUP" -n keyvault-identity --query 'principalId' -otsv)
az role assignment create --role 'Managed Identity Operator' --assignee "$AZURE_CLIENT_ID" --scope "/subscriptions/$SUBSCRIPTION_ID/resourcegroups/$RESOURCE_GROUP/providers/Microsoft.ManagedIdentity/userAssignedIdentities/keyvault-identity" || true
az keyvault set-policy -n "$KEYVAULT_NAME" --secret-permissions get list --spn "$IDENTITY_CLIENT_ID" || true
# The following command might need a couple of retries to succeed
az role assignment create --role Reader --assignee "$IDENTITY_PRINCIPAL_ID" --scope "/subscriptions/$SUBSCRIPTION_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.KeyVault/vaults/$KEYVAULT_NAME" || true

echo 'Creating a cluster-identity and assign appropriate roles...'
az identity create -g "$RESOURCE_GROUP" -n cluster-identity
IDENTITY_CLIENT_ID=$(az identity show -g "$RESOURCE_GROUP" -n cluster-identity  --query 'clientId' -otsv)
IDENTITY_PRINCIPAL_ID=$(az identity show -g "$RESOURCE_GROUP" -n cluster-identity --query 'principalId' -otsv)
az role assignment create --role 'Managed Identity Operator' --assignee "$AZURE_CLIENT_ID" --scope "/subscriptions/$SUBSCRIPTION_ID/resourcegroups/$RESOURCE_GROUP/providers/Microsoft.ManagedIdentity/userAssignedIdentities/cluster-identity" || true
# The following command might need a couple of retries to succeed
az role assignment create --role Reader --assignee "$IDENTITY_PRINCIPAL_ID" -g "$RESOURCE_GROUP" || true

# Create 5 keyvault identities for test #6
for i in {0..4}; do
    IDENTITY_NAME="keyvault-identity-$i"
    echo "Creating $IDENTITY_NAME and assign appropriate roles..."
    az identity create -g "$RESOURCE_GROUP" -n "$IDENTITY_NAME"
    IDENTITY_CLIENT_ID=$(az identity show -g "$RESOURCE_GROUP" -n "$IDENTITY_NAME"  --query 'clientId' -otsv)
    IDENTITY_PRINCIPAL_ID=$(az identity show -g "$RESOURCE_GROUP" -n "$IDENTITY_NAME" --query 'principalId' -otsv)
    az role assignment create --role 'Managed Identity Operator' --assignee "$AZURE_CLIENT_ID" --scope "/subscriptions/$SUBSCRIPTION_ID/resourcegroups/$RESOURCE_GROUP/providers/Microsoft.ManagedIdentity/userAssignedIdentities/$IDENTITY_NAME" || true
    az keyvault set-policy -n "$KEYVAULT_NAME" --secret-permissions get list --spn "$IDENTITY_CLIENT_ID" || true
    # The following command might need a couple of retries to succeed
    az role assignment create --role Reader --assignee "$IDENTITY_PRINCIPAL_ID" --scope "/subscriptions/$SUBSCRIPTION_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.KeyVault/vaults/$KEYVAULT_NAME" || true
done
