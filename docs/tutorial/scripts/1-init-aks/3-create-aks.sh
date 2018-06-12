#!/bin/bash

set -e

if [ -z "$RG" ]
then
      echo "Resource Group Name Not Set. Set the env variable with the following command:"
      echo "export RG=\"rg-name\" "
      return 1
else
     echo "Creating AKS cluster named clusterFrank with 1 node. This may take a while..."
fi

set -x

az aks create --resource-group $RG --name clusterFrank --node-count 1 --generate-ssh-keys
