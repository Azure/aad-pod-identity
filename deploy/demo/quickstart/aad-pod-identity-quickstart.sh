################################################################################
#! Update default variables in this section (if needed) | @phillipgibson

# Name of Azure Resource Group to hold demo assets
ASSETSRESOURCEGROUPNAME="aad-pod-identity-assets"

# Name of Azure Resource Group to test AAD Pod Identity access
ACCESSRESOURCEGROUPNAME="aad-pod-identity-access"

# Name of Azure Resource Group to test AAD Pod Identity no access
NOACCESSRESOURCEGROUPNAME="aad-pod-identity-noaccess"

# Name of Azure Region
LOCATION="eastus2"

# Name of Azure Managed Service Identity | THIS MUST BE A UNIQUE ACCOUNT IN THE AZURE TENENT
MSINAME="pod-identity-acct"

# Name of the AAD Pod Identity JSON Manifest for K8 | MUST BE A .json FILE | eg. aadpodidentity.json
PODIDENTITYJSONFILENAME="aadpodidentity.json"

# Name of pod label to use to associate pod to Azure Managed Service Identity 
PODIDENTITYLABEL="use-pod-identity"

# Name of AAD Pod Idenity Binding JSON Manifest for K8 | MUST BE A .json FILE | eg. azureidenitybindings.json
AZUREIDENTITYBINDINGJSONFILENAME="azureidentitybindings-${MSINAME}.json"

#################################################################################

az group create -l $LOCATION -n $ASSETSRESOURCEGROUPNAME -o tsv

kubectl apply -f https://raw.githubusercontent.com/Azure/aad-pod-identity/master/deploy/infra/deployment-rbac.yaml

read MSINAME MSICLIENTID MSIRESOURCEID MSIPRINCIPALID < <(echo $(az identity create -g $ASSETSRESOURCEGROUPNAME -n $MSINAME -o json | jq -r '.name, .clientId, .id, .principalId'))
echo "Created MSI account ${MSINAME}, and allowing account to propagate the AAD directory..."
sleep 30

jq -r --arg MSINAME "$MSINAME" '.items[].metadata.name |= $MSINAME' aadpodidentity-template.json . > $PODIDENTITYJSONFILENAME

ACCESSRESOURCEGROUPID=$(az group create -l $LOCATION -n $ACCESSRESOURCEGROUPNAME --query id -o tsv)
az role assignment create --assignee-object-id $MSIPRINCIPALID --scope $ACCESSRESOURCEGROUPID --role Contributor

NOACCESSRESOURCEGROUPID=$(az group create -l $LOCATION -n $NOACCESSRESOURCEGROUPNAME --query id -o tsv)
az role assignment create --assignee-object-id $MSIPRINCIPALID --scope $NOACCESSRESOURCEGROUPID --role Reader

CLIENTID=$(jq -r --arg MSICLIENTID "$MSICLIENTID" '.items[].spec.ClientID |= $MSICLIENTID' $PODIDENTITYJSONFILENAME . | grep ClientID)
sed -i "11s/.*/$CLIENTID/g" $PODIDENTITYJSONFILENAME

RESOURCEID=$(jq -r --arg MSIRESOURCEID "$MSIRESOURCEID" '.items[].spec.ResourceID |= $MSIRESOURCEID' $PODIDENTITYJSONFILENAME . | grep ResourceID)
sed -i "12s#.*#$RESOURCEID#g" $PODIDENTITYJSONFILENAME

kubectl apply -f $PODIDENTITYJSONFILENAME

jq -r --arg MSINAME "$MSINAME" '.items[].metadata.name |= $MSINAME' azureidentitybindings-template.json . > $AZUREIDENTITYBINDINGJSONFILENAME

AZUREIDENTITY=$(jq -r --arg MSINAME "$MSINAME" '.items[].spec.AzureIdentity |= $MSINAME' $AZUREIDENTITYBINDINGJSONFILENAME . | grep -E "Identity\W")
sed -i "12s/.*/$AZUREIDENTITY/g" $AZUREIDENTITYBINDINGJSONFILENAME

SELECTOR=$(jq -r --arg PODIDENTITYLABEL "$PODIDENTITYLABEL" '.items[].spec.Selector |= $PODIDENTITYLABEL' $AZUREIDENTITYBINDINGJSONFILENAME . | grep Selector)
sed -i "13s/.*/$SELECTOR/g" $AZUREIDENTITYBINDINGJSONFILENAME

kubectl apply -f $AZUREIDENTITYBINDINGJSONFILENAME
