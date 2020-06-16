#!/bin/bash
# created by @phillipgibson

# function to validate JSON demo config file input
validate_demo_config_file () {
	echo "Validating the JSON demo config file input..."
	if [ -f $1 ]; then # TODO - Check file exist and last 5 char is .json | && [ ${2: -5} == "*.json"]
		#echo $1
		echo "The JSON config file has been validated."
		echo "Retrieving config file data..."
		ASSETS_RESOURCE_GROUP_NAME=$(jq '.assetsResourceGroupName' $1)
		ACCESS_RESOURCE_GROUP_NAME=$(jq '.accessResourceGroupName' $1)
		NO_ACCESS_RESOURCE_GROUP_NAME=$(jq '.noAccessResourceGroupName' $1)
		LOCATION=$(jq '.location' $1)
		MSI_NAME=$(jq '.msiName' $1)
		POD_IDENTITY_LABEL=$(jq '.podIdentityLabel' $1)
		POD_IDENTITY_JSON_FILE_NAME=$(jq '.podIdentityJsonFileName' $1)
		AZURE_IDENTITY_BINDING_JSON_FILE_NAME=$(jq '.azureIdentityBindingJsonFileName' $1)
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
	RESOURCE_GROUP=$(az group show -n ${1//\"/} --query name -o tsv 2>/dev/null)
	if [ "$RESOURCE_GROUP" == ${1//\"/} ]; then
		echo "Azure resource group ${1} already exists. No action needed."
	else
		echo "Azure resource group ${1} does not exist. Creating resource group now..."
		validate_azure_location $LOCATION
		# TODO - create resource group
		CREATED_RESOURCE_GROUP=$(az group create -l ${2//\"/} -n ${1//\"/} -o tsv 2>/dev/null)
		# TODO - error handling on RG creation
	fi
}

# function to validate Azure location/region shortname
validate_azure_location () {
	# ${1//\"/} is used to removed quotes from string, which is needed for the az commands
	echo "Validating the Azure location specified for the resource group creation..."
	LOCATION_EXISTS="false"
	AZURE_LOCATIONS=$(az account list-locations --query "[].name" -o tsv 2>/dev/null)
	for location in $AZURE_LOCATIONS; do
    	if [ $location == ${1//\"/} ]; then
        	LOCATION_EXISTS="true"
        	break
    	fi
	done

	if [ $LOCATION_EXISTS == "true" ]; then
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
	MSI_NAME_QUERY=$(az ad sp list --display-name ${1//\"/} -o json | jq -r '.[].displayName')
	# echo "MSINAMEQUERY is ${MSINAMEQUERY}"
	# echo ${1//\"/}
	# exit
	if [ $MSI_NAME_QUERY == ${1//\"/} ]; then
		echo "Azure Managed Service Identity ${1} already exists. Gathering MSI details..."
		read MSI_NAME MSI_CLIENT_ID MSI_RESOURCE_ID MSI_PRINCIPAL_ID < <(echo $(az ad sp list --display-name ${1//\"/} -o json | jq -r '.[].displayName, .[].appId, .[].alternativeNames[1], .[].objectId'))
	else
		echo "Azure Managed Service Identity ${1} does not exist. Creating MSI now..."
		read MSI_NAME MSI_CLIENT_ID MSI_RESOURCE_ID MSI_PRINCIPAL_ID < <(echo $(az identity create -g ${2//\"/} -n ${1//\"/} -o json | jq -r '.name, .clientId, .id, .principalId'))
		echo "Created MSI account ${MSI_NAME}, and allowing account to propagate the AAD directory..."
		sleep 30
	fi
}

echo "Validating action of the script..."
# Determine if demo script is to deploy demo env or clean up demo env
# ${1//\"/} is used to remove quotes from string, which is needed for the az commands
if [ $1 = "deploy" ]; then
	echo "Script is to deploy AAD Pod Idenity demo."
	validate_demo_config_file "$2"
	validate_azure_resource_group $ASSETS_RESOURCE_GROUP_NAME $LOCATION # Creating/Validating the Assets Resource Group
	validate_azure_resource_group $ACCESS_RESOURCE_GROUP_NAME $LOCATION # Creating/Validating the Access Resrouce Group
	validate_azure_resource_group $NO_ACCESS_RESOURCE_GROUP_NAME $LOCATION # Creating/Validating the No Access Resource Group
	validate_msi_account $MSI_NAME $ASSETS_RESOURCE_GROUP_NAME # Creating/Validating the Managed Service Identity
	echo "Assigning MSI ${MSI_NAME} Contributor role access to access resource group ${ACCESSRESOURCEGROUPNAME}..."
	ACCESS_RESOURCE_GROUP_ID=$(az group create -l ${LOCATION//\"/} -n ${ACCESS_RESOURCE_GROUP_NAME//\"/} --query id -o tsv)
	ASSIGN_CONTRIBUTOR=$(az role assignment create --assignee-object-id ${MSI_PRINCIPAL_ID//\"/} --scope ${ACCESS_RESOURCE_GROUP_ID//\"/} --role Contributor 2>/dev/null)
	echo "Completed assigning MSI ${MSI_NAME} Contributor role access to access resource group ${ACCESS_RESOURCE_GROUP_NAME}."
	echo "Assigning MSI ${MSI_NAME} Reader role access to access resource group ${NO_ACCESS_RESOURCE_GROUP_NAME}..."
	NO_ACCESS_RESOURCE_GROUP_ID=$(az group create -l ${LOCATION//\"/} -n ${NO_ACCESS_RESOURCE_GROUP_NAME//\"/} --query id -o tsv)
	ASSIGN_READER=$(az role assignment create --assignee-object-id ${MSI_PRINCIPAL_ID//\"/} --scope ${NO_ACCESS_RESOURCE_GROUP_ID//\"/} --role Reader 2>/dev/null)
	echo "Completed assigning MSI ${MSI_NAME} Contributor role access to access resource group ${ACCESS_RESOURCE_GROUP_NAME}."
	echo "Creating and updating ${POD_IDENTITY_JSON_FILE_NAME} from template..."
	jq -r --arg MSI_NAME "$MSI_NAME" '.items[].metadata.name |= $MSI_NAME' ./aadpodidentity-template.json . > ${POD_IDENTITY_JSON_FILE_NAME//\"/}
	CLIENT_ID=$(jq -r --arg MSI_CLIENT_ID "$MSI_CLIENT_ID" '.items[].spec.ClientID |= $MSI_CLIENT_ID' ${POD_IDENTITY_JSON_FILE_NAME//\"/} . | grep ClientID)
	sed -i "11s/.*/$CLIENT_ID/g" ${POD_IDENTITY_JSON_FILE_NAME//\"/}
	RESOURCE_ID=$(jq -r --arg MSI_RESOURCE_ID "$MSI_RESOURCE_ID" '.items[].spec.ResourceID |= $MSI_RESOURCE_ID' ${POD_IDENTITY_JSON_FILE_NAME//\"/} . | grep ResourceID)
	sed -i "12s#.*#$RESOURCEID#g" ${POD_IDENTITY_JSON_FILE_NAME//\"/}
	echo "Completed creation of ${POD_IDENTITY_JSON_FILE_NAME}."
	echo "Creating and updating ${AZURE_IDENTITY_BINDING_JSON_FILE_NAME} from template..."
	jq -r --arg MSI_NAME "$MSI_NAME" '.items[].metadata.name |= $MSI_NAME' ./azureidentitybindings-template.json . > ${AZURE_IDENTITY_BINDING_JSON_FILE_NAME//\"/}
	AZURE_IDENTITY=$(jq -r --arg MSI_NAME "$MSI_NAME" '.items[].spec.AzureIdentity |= $MSI_NAME' ${AZURE_IDENTITY_BINDING_JSON_FILE_NAME//\"/} . | grep -E "Identity\W")
	sed -i "12s/.*/$AZURE_IDENTITY/g" ${AZURE_IDENTITY_BINDING_JSON_FILE_NAME//\"/}
	SELECTOR=$(jq -r --arg POD_IDENTITY_LABEL "${POD_IDENTITY_LABEL//\"/}" '.items[].spec.Selector |= $POD_IDENTITY_LABEL' ${AZURE_IDENTITY_BINDING_JSON_FILE_NAME//\"/} . | grep Selector)
	sed -i "13s/.*/$SELECTOR/g" $AZURE_IDENTITY_BINDING_JSON_FILE_NAME//\"/}
	echo "Completed creation of ${AZURE_IDENTITY_BINDING_JSON_FILE_NAME}."
	echo "Applying all necessary Kubernetes (RBAC config) configuration files for Azure AD Pod Identity..."
	kubectl apply -f https://raw.githubusercontent.com/Azure/aad-pod-identity/master/deploy/infra/deployment-rbac.yaml
	sleep 30
	kubectl apply -f ./${POD_IDENTITY_JSON_FILE_NAME//\"/}
	kubectl apply -f ./${AZURE_IDENTITY_BINDING_JSON_FILE_NAME//\"/}
	echo "Deployment of AAD Pod Identity demo has been completed."
	echo "Please refer to the documentation's next steps to demo the AAD Pod Identity functionality."
	echo "Script completed."
	exit 0
elif [ $1 = "clean" ]; then
	echo "Script is to clean up AAD Pod Idenity demo."
	echo "Starting AAD Pod Identity demo cleanup..."
	validate_demo_config_file "$2"
	echo "Removing Azure Identity Binding for Managed Service Identity ${MSI_NAME} from Kubernetes cluster..."
	kubectl delete -f ./${AZURE_IDENTITY_BINDING_JSON_FILE_NAME//\"/}
	echo "Removing Azure Identity for Managed Service Identity ${MSI_NAME} from Kubernetes cluster..."
	kubectl delete -f ./${POD_IDENTITY_JSON_FILE_NAME//\"/}
	echo "Removing the ${ASSETS_RESOURCE_GROUP_NAME} resource group, which will also remove the MSI ${MSI_NAME} account..."
	REMOVE_ASSETS_RG=$(az group delete --name ${ASSETS_RESOURCE_GROUP_NAME//\"/} --yes 2>/dev/null)
	echo "Removing the ${ACCESS_RESOURCE_GROUP_NAME} resource group..."
	REMOVE_ACCESS_RG=$(az group delete --name ${ACCESS_RESOURCE_GROUP_NAME//\"/} --yes 2>/dev/null)
	echo "Removing the ${NO_ACCESS_RESOURCE_GROUP_NAME} resource group..."
	REMOVE_NO_ACCESS_RG=$(az group delete --name ${NO_ACCESS_RESOURCE_GROUP_NAME//\"/} --yes 2>/dev/null)
	echo "Please remember to remove any pods with AAD Pod Identity dependency from the demo."
	echo "Script completed."
	exit 0
else
	echo "No expected script action was detected for the first parameter needed."
	echo "The script action parameter is expecting either a \"deploy\"" or a \"clean\"" action."
	echo "Please ensure you read the documentation on how to use the AAD Pod Identity demo script."
	echo "Exiting script."
	exit 1
fi
exit 0
