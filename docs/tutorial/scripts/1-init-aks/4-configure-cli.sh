#!/bin/bash

set -e

if [ -z "$RG" ]
then
      echo "Resource Group Name Not Set. Set the env variable with the following command:"
      echo "export RG = \"rg-name\" "
      return 1
fi

set -x

az aks get-credentials --resource-group $RG --name clusterFrank

set +x

echo "kubectl is now configured. Run the following command to see your 1 node"
echo "kubectl get nodes"
