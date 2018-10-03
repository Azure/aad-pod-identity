#!/bin/bash

cd "${0%/*}"

file="../../../../deploy/demo/aadpodidentity.yaml"

set -e

client_id=$(az identity show -n ${AZ_IDENTITY_NAME} -g ${MC_RG} | jq -r .clientId)
resource_id=$(az identity show -n ${AZ_IDENTITY_NAME} -g ${MC_RG} | jq -r .id)

perl -pi -e "s/AZURE_IDENTITY_NAME/${AZ_IDENTITY_NAME}/" ${file}
perl -pi -e "s/CLIENT_ID/${client_id}/" ${file}
perl -pi -e "s/RESOURCE_ID/${resource_id//\//\\/}/" ${file}

set -x

kubectl apply -f ${file}
