#!/bin/bash

set -euo pipefail

[[ ! -z "${RESOURCE_GROUP:-}" ]]          || (echo 'Must specify RESOURCE_GROUP' && exit -1)

vmss_name=$(az vmss list -g $RESOURCE_GROUP -o json | jq '.[0].name' | sed 's/"//g')
echo "Found VMSS: " $vmss_name
vmss_sys_id=$(az vmss identity show -g $RESOURCE_GROUP -n $vmss_name -o json | jq '.principalId' | sed 's/"//g')
echo "System MSI principal id: " $vmss_sys_id
full_resourceid=$(az group show -n $RESOURCE_GROUP -o json | jq '.id' | sed 's/"//g')
echo $full_resourceid
echo "Assigning system assigned MSI 'Contributor' role to the resource group $RESOURCE_GROUP for MIC assignments"
#az role assignment create --role "Contributor" --assignee $vmss_sys_id --scope $full_resourceid 
echo "Please set the following before executing setup.sh"
echo "export AZURE_CLIENT_ID=$vmss_sys_id"
