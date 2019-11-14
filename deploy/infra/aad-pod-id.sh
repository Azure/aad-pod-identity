###AAD-POD-IDENTITY DEPLOYMENT SCRIPT###
# This script can be used to deploy AAD-POD_IDENTITY in the default namespace wtaching all namespaces as is the default configuration
# To run the script download this file and make it executble on your local system.
# Prerequisites:
# 1. An AKS Cluster
# 2. Azure CLI
# 3. kubectl with your current context set to cluster to deply to
# 4. Logged in to the az cli with an account that can create MSI and assign roles. Usually Owner and have the proper righst in the AAD Tenant
# 5. Terminal with Bash/ZSH so Linux/MacOS or Windows WSL
#
# to run the script pass 3 parameters
# i.e.
# ./aad-pod-id.sh -a <Name_Of_AKS_Cluster> -r <Resource_Group_Of_AKS_Resource> -m <Name_To_Assign_To_MSI>




#!/bin/sh
while getopts :a:r:m:k: option
do
 case "${option}" in
 a) AKS_NAME=${OPTARG};;
 r) AKS_RG=${OPTARG};;
 m) MSI_NAME=${OPTARG};;
 *) echo "Please refer to usage guide on GitHub" >&2
    exit 1 ;;
 esac
done

if ! command az 2>/dev/null; then
    echo "'az' was not found in PATH. Please install the Azure cli before running this script."
    exit 1
fi

if ! command kubectl 2>/dev/null; then
    echo "'kubectl' was not found in PATH. Please install the kubectl cli before running this script."
    exit 1
fi


echo "creating required variables"
if echo $AKS_NAME > /dev/null 2>&1 && echo $AKS_RG > /dev/null 2>&1; then
    if ! export AKS_RES_ID=$(az aks show -g ${AKS_RG} -n ${AKS_NAME} --query id -o tsv); then
        echo "ERROR: failed to get resource ID or AKS Cluster"
        exit 1
    fi
    echo "AKS Resource ID = $AKS_RES_ID"
fi

if echo $AKS_NAME > /dev/null 2>&1 && echo $AKS_RG > /dev/null 2>&1; then
    if ! export AKS_NODE_RG=$(az aks show -g ${AKS_RG} -n ${AKS_NAME} --query nodeResourceGroup -o tsv); then
        echo "ERROR: failed to get Node Resource Group for AKS Cluster"
        exit 1
    fi
    echo "AKS Node Resource Group = $AKS_NODE_RG"
fi

if echo $AKS_NODE_RG > /dev/null 2>&1; then
    if ! export AKS_NODE_RG_RESID=$(az group show -n ${AKS_NODE_RG} --query id -o tsv); then
        echo "ERROR: failed to get Node Resource Group Resource ID for AKS Cluster"
        exit 1
    fi
    echo "AKS Node Resource Group Full ID = $AKS_NODE_RG_RESID"
fi


echo "creating aad-pod-identity deployment in the default namespace"
if ! kubectl get deploy mic > /dev/null 2>&1; then
    if ! kubectl apply -f https://raw.githubusercontent.com/Azure/aad-pod-identity/master/deploy/infra/deployment-rbac.yaml; then
        echo "ERROR: failed to create kubernetes aad-pod-idenity deployment"
        exit 1
    fi
fi

echo "creating azure user assigned managed identity $MSI_NAME in the $AKS_NODE_RG Resource Group"
if echo $AKS_NODE_RG > /dev/null 2>&1 && echo $MSI_NAME > /dev/null 2>&1; then
    if ! export MSI_PrincID=$(az identity create -g ${AKS_NODE_RG} -n ${MSI_NAME} --query principalId -o tsv); then
        echo "ERROR: failed to create managed identity"
        exit 1
    fi
    echo "Managed Identity Principal ID = $MSI_PrincID"
fi

echo "getting azure user assigned managed identity $MSI_NAME resource ID"
if echo $AKS_NODE_RG > /dev/null 2>&1 && echo $MSI_NAME > /dev/null 2>&1; then
    if ! export MSI_ResID=$(az identity show -g ${AKS_NODE_RG} -n ${MSI_NAME} --query id -o tsv); then
        echo "ERROR: failed to get managed identity resource ID"
        exit 1
    fi
    echo "Managed Identity Resource ID = $MSI_ResID"
fi

echo "getting azure user assigned managed identity $MSI_NAME client ID"
if echo $AKS_NODE_RG > /dev/null 2>&1 && echo $MSI_NAME > /dev/null 2>&1; then
    if ! export MSI_ClientID=$(az identity show -g ${AKS_NODE_RG} -n ${MSI_NAME} --query clientId -o tsv); then
        echo "ERROR: failed to get managed identity resource ID"
        exit 1
    fi
    echo "Managed Identity Client ID = $MSI_ClientID"
fi

while ! az role assignment create --role Reader --assignee $MSI_PrincID --scope $AKS_NODE_RG_RESID
do
  echo "Sleeping for 10 seconds waiting for AAD Propogation of Identity"
  sleep 10s
done

echo "assigning the managed identity Principal ID $MSI_PrincID reader role to  the $AKS_NODE_RG Resource Group"
if echo $MSI_PrincID > /dev/null 2>&1 && echo $AKS_NODE_RG > /dev/null 2>&1; then
    if ! az role assignment create --role Reader --assignee $MSI_PrincID --scope $AKS_NODE_RG_RESID; then
        echo "ERROR: failed to assign the reader role to the managed identity"
        exit 1
    fi
fi

echo "creating required variables"
if echo $AKS_NAME > /dev/null 2>&1 && echo $AKS_RG >/dev/null 2>&1; then
    if ! export AKS_SPID=$(az aks show -g ${AKS_RG} -n ${AKS_NAME} --query 'servicePrincipalProfile.clientId' -o tsv); then
        echo "ERROR: failed to get Service Principal ID for AKS Cluster"
        exit 1
    fi
    echo "AKS Service Principal ID = $AKS_SPID"
fi

if echo $AKS_NAME > /dev/null 2>&1 && echo $AKS_RG >/dev/null 2>&1; then
    if ! export AKS_SP_ObjID=$(az ad sp show --id ${AKS_SPID} --query objectId -o tsv); then
        echo "ERROR: failed to get Service Principal Object ID for AKS Cluster"
        exit 1
    fi
    echo "AKS Service Principal Object ID = $AKS_SP_ObjID"
fi

echo "assigning the AKS Service Principal Managed Identity Operator rights"
if echo $AKS_SP_ObjID > /dev/null 2>&1 && echo $MSI_ResID > /dev/null 2>&1; then
    if ! az role assignment create --role "Managed Identity Operator" --assignee $AKS_SP_ObjID --scope $MSI_ResID; then
        echo "ERROR: failed to assign the AKS Service Principal Managed Identity Operator rights"
        exit 1
    fi
fi


echo "The aadpodidentity resource will be deployed to cluster $AKS_NAME. It will be saved to your current directory as aadpodidentity.yaml"
cat << EOF > ${MSI_NAME}-aadpodidentity.yaml
apiVersion: "aadpodidentity.k8s.io/v1"
kind: AzureIdentity
metadata:
 name: $MSI_NAME
spec:
 type: 0
 ResourceID: $MSI_ResID
 ClientID: $MSI_ClientID
EOF

echo "deploying aadpodidentity resource into the default namespace"
if kubectl get deploy mic > /dev/null 2>&1; then
    if ! kubectl apply -f ${MSI_NAME}-aadpodidentity.yaml; then
        echo "ERROR: failed to create kubernetes aadpodidenity resource"
        exit 1
    fi
fi

echo "The aadpodidentitybiding resource will be deployed to cluster $AKS_NAME. It will be saved to your current directory as aadpodidentitybinding.yaml"
cat << EOF > ${MSI_NAME}-aadpodidentitybinding.yaml
apiVersion: "aadpodidentity.k8s.io/v1"
kind: AzureIdentityBinding
metadata:
 name: ${MIS_NAME}-binding
spec:
 AzureIdentity: ${MSI_NAME}
 Selector: podid-${MSI_NAME}
EOF

echo "creating deploying aadpodidentitybinding resource into the default namespace"
if kubectl get deploy mic > /dev/null 2>&1; then
    if ! kubectl apply -f ${MSI_NAME}-aadpodidentitybinding.yaml; then
        echo "ERROR: failed to create kubernetes aadpodidenitybinding resource"
        exit 1
    fi
fi

echo "AAD Pod Identity has been deployed to you cluster $AKS_NAME and is using $MSI_NAME for its managed Identity"
echo "Add the label aadpodidentity: podid=${MSI_NAME} to your deployment to assign an aad MSi to those pods"
echo "Now you can configure ${MSI_NAME} to have access rihts to any azure resource based on Azure IAM roles"
