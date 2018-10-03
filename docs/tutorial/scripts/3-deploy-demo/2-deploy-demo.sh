#!/bin/bash

cd "${0%/*}"

file="../../../../deploy/demo/deployment.yaml"

set -e

client_id=$(az identity show -n ${AZ_IDENTITY_NAME} -g ${MC_RG} | jq -r .clientId)
subscription_id=$(az account show | jq -r .id)
binding_name="${AZ_IDENTITY_NAME}-binding"
binding_selector="${binding_name}-selector"

perl -pi -e "s/CLIENT_ID/${client_id}/" ${file}
perl -pi -e "s/SUBSCRIPTION_ID/${subscription_id}/" ${file}
perl -pi -e "s/RESOURCE_GROUP/${MC_RG}/" ${file}
perl -pi -e "s/SELECTOR_VALUE/${binding_selector}/" ${file}



# aad-pod-identity/deploy/demo/deployment.yaml is the original file
set -x

kubectl apply -f ${file}
