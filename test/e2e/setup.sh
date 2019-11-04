#!/bin/bash

set -euo pipefail

# Check pre-requisites
[[ ! -z "${SUBSCRIPTION_ID:-}" ]]         || (echo 'Must specify SUBSCRIPTION_ID' && exit -1)
[[ ! -z "${RESOURCE_GROUP:-}" ]]          || (echo 'Must specify RESOURCE_GROUP' && exit -1)
[[ ! -z "${KEYVAULT_NAME:-}" ]]           || (echo 'Must specify KEYVAULT_NAME' && exit -1)
[[ ! -z "${KEYVAULT_SECRET_NAME:-}" ]]    || (echo 'Must specify KEYVAULT_SECRET_NAME' && exit -1)
[[ -z "$(hash az)" ]]                     || (echo 'Azure CLI not found' && exit -1)
[[ -z "$(hash kubectl)" ]]                || (echo 'kubectl not found' && exit -1)


retry_cmd() {
    set +e
    local retval=0
    for i in {0..10}; do
        out=$(eval $1 2>&1)
        retval=$?
        [[ "${out}" == *"already exists"* ]] && retval=0
        [ $retval -eq 0 ] && break
        sleep 6
    done
    set -e
    if [ $retval -ne  0 ]; then
      >&2 echo $2
      return 1
    fi
}

# TODO: Handle multiple pools.
# TODO: Handle system msi and VMA clusters.
if [ ! -z ${SYSTEM_MSI_CLUSTER:-} ]; then
  if "$SYSTEM_MSI_CLUSTER" == "true" ]; then
    echo "System MSI cluster flag enabled"
    full_resourceid=$(az group show -n $RESOURCE_GROUP -o json | jq '.id' | sed 's/"//g')
    echo "Resource id for the group: $full_resourceid"
    vmss_name=$(az vmss list -g $RESOURCE_GROUP -o json | jq '.[0].name' | sed 's/"//g')
    if [ "$vmss_name" == "" ]; then
      echo "Could not find VMSS node pool. Currently e2e support is only for single VMSS pool + System MSI"
      exit 1
    fi
    echo "Found VMSS: " $vmss_name
    vmss_sys_id=$(az vmss identity show -g $RESOURCE_GROUP -n $vmss_name -o json | jq '.principalId' | sed 's/"//g')
    echo "System MSI principal id: " $vmss_sys_id
    echo "Assigning system assigned MSI 'Contributor' role to the resource group $RESOURCE_GROUP for MIC assignments"
    az role assignment create --role "Contributor" --assignee $vmss_sys_id --scope $full_resourceid
    # set the AZURE_CLIENT_ID to be the system assigned identity of the vmss pool
    unset AZURE_CLIENT_ID
    echo "Setting the $vmss_sys_id as the AZURE_CLIENT_ID"
    AZURE_CLIENT_ID=$vmss_sys_id
  fi
fi

[[ ! -z "${AZURE_CLIENT_ID:-}" ]]         || (echo 'Must specify AZURE_CLIENT_ID' && exit -1)

# Create a keyvault
az keyvault create -g "$RESOURCE_GROUP" -n "$KEYVAULT_NAME"
az keyvault secret set --vault-name "$KEYVAULT_NAME" -n "$KEYVAULT_SECRET_NAME" --value test-value

echo 'Creating a keyvault-identity and assign appropriate roles...'
az identity create -g "$RESOURCE_GROUP" -n keyvault-identity
IDENTITY_CLIENT_ID=$(az identity show -g "$RESOURCE_GROUP" -n keyvault-identity  --query 'clientId' -otsv)
IDENTITY_PRINCIPAL_ID=$(az identity show -g "$RESOURCE_GROUP" -n keyvault-identity --query 'principalId' -otsv)
az role assignment create --role 'Managed Identity Operator' --assignee "$AZURE_CLIENT_ID" --scope "/subscriptions/$SUBSCRIPTION_ID/resourcegroups/$RESOURCE_GROUP/providers/Microsoft.ManagedIdentity/userAssignedIdentities/keyvault-identity" || true
retry_cmd "az keyvault set-policy -n "$KEYVAULT_NAME" --secret-permissions get list --spn "$IDENTITY_CLIENT_ID"" "set policy failed"
# The following command might need a couple of retries to succeed
retry_cmd "az role assignment create --role Reader --assignee "$IDENTITY_PRINCIPAL_ID" --scope "/subscriptions/$SUBSCRIPTION_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.KeyVault/vaults/$KEYVAULT_NAME"" "role assignment failed"

echo 'Creating a cluster-identity and assign appropriate roles...'
az identity create -g "$RESOURCE_GROUP" -n cluster-identity
IDENTITY_CLIENT_ID=$(az identity show -g "$RESOURCE_GROUP" -n cluster-identity  --query 'clientId' -otsv)
IDENTITY_PRINCIPAL_ID=$(az identity show -g "$RESOURCE_GROUP" -n cluster-identity --query 'principalId' -otsv)
az role assignment create --role 'Managed Identity Operator' --assignee "$AZURE_CLIENT_ID" --scope "/subscriptions/$SUBSCRIPTION_ID/resourcegroups/$RESOURCE_GROUP/providers/Microsoft.ManagedIdentity/userAssignedIdentities/cluster-identity" || true
# The following command might need a couple of retries to succeed
retry_cmd "az role assignment create --role Reader --assignee "$IDENTITY_PRINCIPAL_ID" -g "$RESOURCE_GROUP""

# Create 5 keyvault identities for test #6
for i in {0..4}; do
    IDENTITY_NAME="keyvault-identity-$i"
    echo "Creating $IDENTITY_NAME and assign appropriate roles..."
    az identity create -g "$RESOURCE_GROUP" -n "$IDENTITY_NAME"
    IDENTITY_CLIENT_ID=$(az identity show -g "$RESOURCE_GROUP" -n "$IDENTITY_NAME"  --query 'clientId' -otsv)
    IDENTITY_PRINCIPAL_ID=$(az identity show -g "$RESOURCE_GROUP" -n "$IDENTITY_NAME" --query 'principalId' -otsv)
    az role assignment create --role 'Managed Identity Operator' --assignee "$AZURE_CLIENT_ID" --scope "/subscriptions/$SUBSCRIPTION_ID/resourcegroups/$RESOURCE_GROUP/providers/Microsoft.ManagedIdentity/userAssignedIdentities/$IDENTITY_NAME" || true
    retry_cmd "az keyvault set-policy -n "$KEYVAULT_NAME" --secret-permissions get list --spn "$IDENTITY_CLIENT_ID"" "set policy failed"
    # The following command might need a couple of retries to succeed
    retry_cmd "az role assignment create --role Reader --assignee "$IDENTITY_PRINCIPAL_ID" --scope "/subscriptions/$SUBSCRIPTION_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.KeyVault/vaults/$KEYVAULT_NAME""
done

# Create identity with not enough permissions to test failure path
echo 'Creating a keyvault-identity-5 and assign only list policy...'
az identity create -g "$RESOURCE_GROUP" -n keyvault-identity-5
IDENTITY_CLIENT_ID=$(az identity show -g "$RESOURCE_GROUP" -n keyvault-identity-5  --query 'clientId' -otsv)
IDENTITY_PRINCIPAL_ID=$(az identity show -g "$RESOURCE_GROUP" -n keyvault-identity-5 --query 'principalId' -otsv)
az role assignment create --role 'Managed Identity Operator' --assignee "$AZURE_CLIENT_ID" --scope "/subscriptions/$SUBSCRIPTION_ID/resourcegroups/$RESOURCE_GROUP/providers/Microsoft.ManagedIdentity/userAssignedIdentities/keyvault-identity-5" || true
retry_cmd "az keyvault set-policy -n "$KEYVAULT_NAME" --secret-permissions list --spn "$IDENTITY_CLIENT_ID"" "set policy failed"
# The following command might need a couple of retries to succeed
retry_cmd "az role assignment create --role Reader --assignee "$IDENTITY_PRINCIPAL_ID" --scope "/subscriptions/$SUBSCRIPTION_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.KeyVault/vaults/$KEYVAULT_NAME""

# Create immutable identity
IDENTITY_NAME="immutable-identity"
echo "Creating $IDENTITY_NAME and assign appropriate roles..."
az identity create -g "$RESOURCE_GROUP" -n "$IDENTITY_NAME"
IMMUTABLE_IDENTITY_CLIENT_ID=$(az identity show -g "$RESOURCE_GROUP" -n "$IDENTITY_NAME"  --query 'clientId' -otsv)
IDENTITY_PRINCIPAL_ID=$(az identity show -g "$RESOURCE_GROUP" -n "$IDENTITY_NAME" --query 'principalId' -otsv)
az role assignment create --role 'Managed Identity Operator' --assignee "$AZURE_CLIENT_ID" --scope "/subscriptions/$SUBSCRIPTION_ID/resourcegroups/$RESOURCE_GROUP/providers/Microsoft.ManagedIdentity/userAssignedIdentities/$IDENTITY_NAME" || true
retry_cmd "az keyvault set-policy -n "$KEYVAULT_NAME" --secret-permissions get list --spn "$IMMUTABLE_IDENTITY_CLIENT_ID"" "set policy failed"
# The following command might need a couple of retries to succeed
retry_cmd "az role assignment create --role Reader --assignee "$IDENTITY_PRINCIPAL_ID" --scope "/subscriptions/$SUBSCRIPTION_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.KeyVault/vaults/$KEYVAULT_NAME""
