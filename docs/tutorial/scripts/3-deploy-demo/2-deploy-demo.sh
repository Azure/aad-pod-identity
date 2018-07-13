#!/bin/bash

cd "${0%/*}"

set -e
echo "To deploy the demo app, update the /deploy/demo.deployment.yaml arguments with \
your subscription, clientID and resource group. \
Make sure your identity with the client ID has reader permission to the resource group provided in the input."

read -p "Press enter to continue"


# aad-pod-identity/deploy/demo/deployment.yaml is the original file 
set -x

kubectl apply -f ../../../../deploy/demo/deployment.yaml
