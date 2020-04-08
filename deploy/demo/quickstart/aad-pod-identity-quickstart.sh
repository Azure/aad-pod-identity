################################################################################
#! Update default variables in this section (if needed) | @phillipgibson

# Name of Azure Resource Group
RESOURCEGROUPNAME="aad-pod-identity-assets"

# Name of Azure Region
LOCATION="eastus2"

# Name of Azure Managed Service Identity
MSINAME="pod-identity-acct"

# Name of the AAD Pod Identity JSON Manifest for K8 | MUST BE A .json FILE | eg. aadpodidentity.json
PODIDENTITYJSONFILENAME="aadpodidentity.json"

# Name of pod label to use to associate pod to Azure Managed Service Identity 
PODIDENTITYLABEL="use-pod-identity"

# Name of AAD Pod Idenity Binding JSON Manifest for K8 | MUST BE A .json FILE | eg. azureidenitybindings.json
AZUREIDENTITYBINDINGJSONFILENAME="azureidentitybindings-${MSINAME}.json"

#################################################################################

az group create -l $LOCATION -n $RESOURCEGROUPNAME

kubectl apply -f https://raw.githubusercontent.com/Azure/aad-pod-identity/master/deploy/infra/deployment-rbac.yaml

read MSINAME MSICLIENTID MSIRESOURCEID < <(echo $(az identity create -g $RESOURCEGROUPNAME -n $MSINAME -o json | jq -r '.name, .clientId, .id'))

jq -r --arg MSINAME "$MSINAME" '.items[].metadata.name |= $MSINAME' aadpodidentity-template.json . > $PODIDENTITYJSONFILENAME


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
