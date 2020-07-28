#!/bin/bash
# created by @phillipgibson

# function to validate JSON demo config file input
validate_demo_config_file () {
	echo "Validating the JSON demo config file input..."
	if [ -f $1 ]; then # TODO - Check file exist and last 5 char is .json | && [ ${2: -5} == "*.json"]
		#echo $1
		echo "The JSON config file has been validated."
		echo "Retrieving config file data..."
		ASSETSRESOURCEGROUPNAME=$(jq '.ASSETSRESOURCEGROUPNAME' $1)
		ACCESSRESOURCEGROUPNAME=$(jq '.ACCESSRESOURCEGROUPNAME' $1)
		NOACCESSRESOURCEGROUPNAME=$(jq '.NOACCESSRESOURCEGROUPNAME' $1)
		LOCATION=$(jq '.LOCATION' $1)
		MSINAME=$(jq '.MSINAME' $1)
		PODIDENTITYLABEL=$(jq '.PODIDENTITYLABEL' $1)
		PODIDENTITYJSONFILENAME=$(jq '.PODIDENTITYJSONFILENAME' $1)
		AZUREIDENTITYBINDINGJSONFILENAME=$(jq '.AZUREIDENTITYBINDINGJSONFILENAME' $1)
	else
		echo "The file input of ${1} does not seem to exists. Please verify the file exists."
		echo "Exiting script"
		exit 1
	fi
}

# function to validate existence of Azure resource group and create if it doesn't exist
validate_azure_resource_group () {
	# ${1//\"/} is used to remove quotes from string, which is needed for the az commands
	echo "Validating the existence of Azure resource group ${1}..."
	RESOURCEGROUP=$(az group show -n ${1//\"/} --query name -o tsv 2>/dev/null)
	if [ "$RESOURCEGROUP" == ${1//\"/} ]; then
		echo "Azure resource group ${1} already exists. No action needed."
	else
		echo "Azure resource group ${1} does not exist. Creating resource group now..."
		validate_azure_location $LOCATION
		# TODO - create resource group
		CREATEDRESOURCEGROUP=$(az group create -l ${2//\"/} -n ${1//\"/} -o tsv 2>/dev/null)
		# TODO - error handling on RG creation
	fi
}

# function to validate Azure location/region shortname
validate_azure_location () {
	# ${1//\"/} is used to removed quotes from string, which is needed for the az commands
	echo "Validating the Azure location specified for the resource group creation..."
	LOCATIONEXISTS="false"
	AZURELOCATIONS=$(az account list-locations --query "[].name" -o tsv 2>/dev/null)
	for location in $AZURELOCATIONS; do
    	if [ $location == ${1//\"/} ]; then
        	LOCATIONEXISTS="true"
        	break
    	fi
	done

	if [ $LOCATIONEXISTS == "true" ]; then
		echo "The Azure location ${1} is a valid Azure location."
	else
		echo "The Azure location ${1} is not a valide Azure location."
		echo "Please read the Azure documentation for specifying the correct Azure location shortname."
		echo "For a listing of Azure location shortnames, please use the az account list-locations command."
		echo "Exiting script."
		exit 1
	fi
}

# function to validate Managed Serivce Identity
validate_msi_account () {
	# ${1//\"/} is used to remove quotes from string, which is needed for the az commands
	echo "Validating the existence of the Azure Managed Service Identity ${1}..."
	MSINAMEQUERY=$(az ad sp list --display-name ${1//\"/} -o json | jq -r '.[].displayName')
	# echo "MSINAMEQUERY is ${MSINAMEQUERY}"
	# echo ${1//\"/}
	# exit
	if [ $MSINAMEQUERY == ${1//\"/} ]; then
		echo "Azure Managed Service Identity ${1} already exists. Gathering MSI details..."
		read MSINAME MSICLIENTID MSIRESOURCEID MSIPRINCIPALID < <(echo $(az ad sp list --display-name ${1//\"/} -o json | jq -r '.[].displayName, .[].appId, .[].alternativeNames[1], .[].objectId'))
	else
		echo "Azure Managed Service Identity ${1} does not exist. Creating MSI now..."
		read MSINAME MSICLIENTID MSIRESOURCEID MSIPRINCIPALID < <(echo $(az identity create -g ${2//\"/} -n ${1//\"/} -o json | jq -r '.name, .clientId, .id, .principalId'))
		echo "Created MSI account ${MSINAME}, and allowing account to propagate the AAD directory..."
		sleep 30
	fi
}

echo "Validating action of the script..."
# Determine if demo script is to deploy demo env or clean up demo env
# ${1//\"/} is used to remove quotes from string, which is needed for the az commands
if [ $1 = "deploy" ]; then
	echo "Script is to deploy AAD Pod Idenity demo."
	validate_demo_config_file "$2"
	validate_azure_resource_group $ASSETSRESOURCEGROUPNAME $LOCATION # Creating/Validating the Assets Resource Group
	validate_azure_resource_group $ACCESSRESOURCEGROUPNAME $LOCATION # Creating/Validating the Access Resrouce Group
	validate_azure_resource_group $NOACCESSRESOURCEGROUPNAME $LOCATION # Creating/Validating the No Access Resource Group
	validate_msi_account $MSINAME $ASSETSRESOURCEGROUPNAME # Creating/Validating the Managed Service Identity
	echo "Assigning MSI ${MSINAME} Contributor role access to access resource group ${ACCESSRESOURCEGROUPNAME}..."
	ACCESSRESOURCEGROUPID=$(az group create -l ${LOCATION//\"/} -n ${ACCESSRESOURCEGROUPNAME//\"/} --query id -o tsv)
	ASSIGNCONTRIBUTOR=$(az role assignment create --assignee-object-id ${MSIPRINCIPALID//\"/} --scope ${ACCESSRESOURCEGROUPID//\"/} --role Contributor 2>/dev/null)
	echo "Completed assigning MSI ${MSINAME} Contributor role access to access resource group ${ACCESSRESOURCEGROUPNAME}."
	echo "Assigning MSI ${MSINAME} Reader role access to access resource group ${NOACCESSRESOURCEGROUPNAME}..."
	NOACCESSRESOURCEGROUPID=$(az group create -l ${LOCATION//\"/} -n ${NOACCESSRESOURCEGROUPNAME//\"/} --query id -o tsv)
	ASSIGNREADER=$(az role assignment create --assignee-object-id ${MSIPRINCIPALID//\"/} --scope ${NOACCESSRESOURCEGROUPID//\"/} --role Reader 2>/dev/null)
	echo "Completed assigning MSI ${MSINAME} Contributor role access to access resource group ${ACCESSRESOURCEGROUPNAME}."
	echo "Creating and updating ${PODIDENTITYJSONFILENAME} from template..."
	jq -r --arg MSINAME "$MSINAME" '.items[].metadata.name |= $MSINAME' ./aadpodidentity-template.json . > ${PODIDENTITYJSONFILENAME//\"/}
	CLIENTID=$(jq -r --arg MSICLIENTID "$MSICLIENTID" '.items[].spec.ClientID |= $MSICLIENTID' ${PODIDENTITYJSONFILENAME//\"/} . | grep ClientID)
	sed -i "11s/.*/$CLIENTID/g" ${PODIDENTITYJSONFILENAME//\"/}
	RESOURCEID=$(jq -r --arg MSIRESOURCEID "$MSIRESOURCEID" '.items[].spec.ResourceID |= $MSIRESOURCEID' ${PODIDENTITYJSONFILENAME//\"/} . | grep ResourceID)
	sed -i "12s#.*#$RESOURCEID#g" ${PODIDENTITYJSONFILENAME//\"/}
	echo "Completed creation of ${PODIDENTITYJSONFILENAME}."
	echo "Creating and updating ${AZUREIDENTITYBINDINGJSONFILENAME} from template..."
	jq -r --arg MSINAME "$MSINAME" '.items[].metadata.name |= $MSINAME' ./azureidentitybindings-template.json . > ${AZUREIDENTITYBINDINGJSONFILENAME//\"/}
	AZUREIDENTITY=$(jq -r --arg MSINAME "$MSINAME" '.items[].spec.AzureIdentity |= $MSINAME' ${AZUREIDENTITYBINDINGJSONFILENAME//\"/} . | grep -E "Identity\W")
	sed -i "12s/.*/$AZUREIDENTITY/g" ${AZUREIDENTITYBINDINGJSONFILENAME//\"/}
	SELECTOR=$(jq -r --arg PODIDENTITYLABEL "${PODIDENTITYLABEL//\"/}" '.items[].spec.Selector |= $PODIDENTITYLABEL' ${AZUREIDENTITYBINDINGJSONFILENAME//\"/} . | grep Selector)
	sed -i "13s/.*/$SELECTOR/g" ${AZUREIDENTITYBINDINGJSONFILENAME//\"/}
	echo "Completed creation of ${AZUREIDENTITYBINDINGJSONFILENAME}."
	echo "Applying all necessary Kubernetes (RBAC config) configuration files for Azure AD Pod Identity..."
	kubectl apply -f https://raw.githubusercontent.com/Azure/aad-pod-identity/master/deploy/infra/deployment-rbac.yaml
	sleep 30
	kubectl apply -f ./${PODIDENTITYJSONFILENAME//\"/}
	kubectl apply -f ./${AZUREIDENTITYBINDINGJSONFILENAME//\"/}
	echo "Deployment of AAD Pod Identity demo has been completed."
	echo "Please refer to the documentation's next steps to demo the AAD Pod Identity functionality."
	echo "Script completed."
elif [ $1 = "clean" ]; then
	echo "Script is to clean up AAD Pod Idenity demo."
	echo "Starting AAD Pod Identity demo cleanup..."
	validate_demo_config_file "$2"
	echo "Removing Azure Identity Binding for Managed Service Identity ${MSINAME} from Kubernetes cluster..."
	kubectl delete -f ./${AZUREIDENTITYBINDINGJSONFILENAME//\"/}
	echo "Removing Azure Identity for Managed Service Identity ${MSINAME} from Kubernetes cluster..."
	kubectl delete -f ./${PODIDENTITYJSONFILENAME//\"/}
	echo "Removing the ${ASSETSRESOURCEGROUPNAME} resource group, which will also remove the MSI ${MSINAME} account..."
	REMOVEASSETSRG=$(az group delete --name ${ASSETSRESOURCEGROUPNAME//\"/} --yes 2>/dev/null)
	echo "Removing the ${ACCESSRESOURCEGROUPNAME} resource group..."
	REMOVEACCESSRG=$(az group delete --name ${ACCESSRESOURCEGROUPNAME//\"/} --yes 2>/dev/null)
	echo "Removing the ${NOACCESSRESOURCEGROUPNAME} resource group..."
	REMOVENOACCESSRG=$(az group delete --name ${NOACCESSRESOURCEGROUPNAME//\"/} --yes 2>/dev/null)
	echo "Please remember to remove any pods with AAD Pod Identity dependency from the demo."
	echo "Script completed."
else
	echo "No expected script action was detected for the first parameter needed."
	echo "The script action parameter is expecting either a \"deploy\"" or a \"clean\"" action."
	echo "Please ensure you read the documentation on how to use the AAD Pod Identity demo script."
	echo "Exiting script."
	exit 1
fi
exit 0




