#!/bin/bash

cd "${0%/*}"

file="../../../../deploy/demo/aadpodidentitybinding.yaml"

set -e

binding_name="${AZ_IDENTITY_NAME}-binding"
binding_selector="${binding_name}-selector"

perl -pi -e "s/BINDING_NAME/${binding_name}/" ${file}
perl -pi -e "s/AZURE_IDENTITY_NAME/${AZ_IDENTITY_NAME}/" ${file}
perl -pi -e "s/SELECTOR_VALUE/${binding_selector}/" ${file}

set -x

kubectl apply -f ../../../../deploy/demo/aadpodidentitybinding.yaml
