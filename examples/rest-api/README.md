# Using AAD Pod Identity with a custom REST API

We will look at the scenario where a client pod is calling a service (REST API) with an AAD Pod Identity.

The client is a bash script which will request a token, pass it as authorization HTTP header to query a REST API.  The client has a corresponding user managed identity which will be exposed as an AAD Pod Identity.

The REST API is implemented in C# / .NET Core.  It simply validates it receives a bearer token in the authorization header of each request.  The REST API has a corresponding Azure AD Application.  The client requests a token with that AAD application as the *resource*.

**<span style="background:yellow">TODO:  simple diagram showing client / service pods + identities involved ; everything is name so no surprised in the script</span>**

## Prerequisites

* An AKS cluster with [AAD Pod Identity installed on it](https://github.com/Azure/aad-pod-identity/blob/master/README.md)

## Identity

In this section, we'll create the user managed identity used for the client.

First, let's define those variable:

```bash
rg=<name of the resource group where AKS is>
cluster=<name of the AKS cluster>
```

Then, let's create the user managed identity:

```bash
az identity create -g $rg -n client-principal \
    --query "{ClientId: clientId, ManagedIdentityId: id, TenantId:  tenantId}" -o jsonc
```

This returns three values in a JSON document.  We will use those values later on.

We need to give the Service Principal running the cluster the *Managed Identity Operator* role on the user managed identity:

```bash
aksPrincipalId=$(az aks show -g $rg -n $cluster --query "servicePrincipalProfile.clientId" -o tsv)
managedId=$(az identity show -g $rg -n client-principal \
    --query "id" -o tsv)
az role assignment create --role "Managed Identity Operator" --assignee $aksPrincipalId --scope $managedId
```

The first line acquires the AKS service principal client ID.  The second line acquires the client ID of the user managed identity (the *ManagedIdentityId* returned in the JSON above).  The third line performs the role assignment.

## Identity & Binding in Kubernetes

In this section, we'll configure AAD pod identity with the user managed identity.

We'll create a Kubernetes namespace to put all our resources.  It makes it easier to clean up afterwards.

```bash
kubectl create namespace pir
kubectl label namespace/pir description=PodIdentityRestApi
```

Let's customize [identity.yaml](TODO):

```yaml
apiVersion: "aadpodidentity.k8s.io/v1"
kind: AzureIdentity
metadata:
 name: client-principal
spec:
 type: 0
 ResourceID: <resource-id of client-principal>
 ClientID: <client-id of client-principal>
```

*ResourceID* should be set to the value of *ManagedIdentityId* in the JSON from the previous section.  That is the resource ID of the user managed identity.

*ClientID* should be set to the value of *ClientId* in the JSON from the previous section.  That is the client id of the user managed identity.

We do not need to customize [binding.yaml](TODO).

```yaml
apiVersion: "aadpodidentity.k8s.io/v1"
kind: AzureIdentityBinding
metadata:
 name: client-principal-binding
spec:
 AzureIdentity: client-principal
 Selector:  client-principal-pod-binding
```

We can now deploy those two files in the *pir* namespace:

```bash
kubectl apply -f identity.yaml --namespace pir
kubectl apply -f binding.yaml --namespace pir
```

We can check the resources got deployed:

```bash
$ kubectl get AzureIdentity --namespace pir

NAME               AGE
client-principal   12s

$ kubectl get AzureIdentityBinding --namespace pir

NAME                       AGE
client-principal-binding   32s
```

## Application

In this section, we will simply create the Azure AD application corresponding to the REST API Service:

```bash
appId=$(az ad app create --display-name myapi \
    --identifier-uris http://myapi.restapi.aad-pod-identity \
    --query "appId" -o tsv)
echo $appId
```

The application's name is *myapi*.  The identifier uri is irrelevant but required.

## Client

In [client-pod.yaml](client-pod.yaml), we need to customize the value of the environment variable *RESOURCE* to the value of *$appId* computed above, i.e. the client id of the Azure AD application:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: aad-id-client-pod
  labels:
    app: aad-id-client
    platform: cli
    aadpodidbinding:  client-principal-pod-binding
spec:
  containers:
  - name: main-container
    image: vplauzon/aad-pod-id-client
    env:
    - name: RESOURCE
      value: <Application Id>
    - name: SERVICE_URL
      value: http://aad-id-service
```

The client will use that when requesting a token.

The client's code is packaged in a container.  The core of the code is [script.sh](client/script.sh):

```bash
#!/bin/sh

echo "Hello ${RESOURCE}"

i=0
while true
do
    echo "Iteration $i"

    jwt=$(curl -sS http://169.254.169.254/metadata/identity/oauth2/token/?resource=$RESOURCE)
    echo "Full token:  $jwt"
    token=$(echo $jwt | jq -r '.access_token')
    echo "Access token:  $token"
    curl -v -H 'Accept: application/json' -H "Authorization: Bearer ${token}" $SERVICE_URL

    i=$((i+1))
    sleep 1
done
```

Every second the script queries a token from http://169.254.169.254 which is exposed by AAD Pod Identity (more specifically the *Node Managed Identity* component, or NMI).

We use the [jq](https://stedolan.github.io/jq/) tool to parse the JSON of the token.  We extract the access token element which we then pass as in an HTTP header using *curl*.

## Service

In [service.yaml](service.yaml), we need to customize the value for APPLICATION_ID & TENANT_ID.

APPLICATION_ID's value is the same as the RESOURCE from previous section, i.e. *$appId*.  TENANT_ID is the ID of the tenant owning the identities.  It was given in the JSON as the output of the user managed identity.

```yaml
apiVersion: v1
kind: Service
metadata:
  name: aad-id-service
spec:
  type: ClusterIP
  ports:
  - port: 80
  selector:
    app: aad-id-service
---
apiVersion: v1
kind: Pod
metadata:
  name: aad-id-service-pod
  labels:
    app: aad-id-service
    platform: csharp
spec:
  containers:
  - name: api-container
    image: vplauzon/aad-pod-id-svc
    ports:
    - containerPort: 80
    env:
    - name: TENANT_ID
      value: <ID of TENANT owning the identities>
    - name: APPLICATION_ID
      value: <Application Id>
```

Here we defined a service and a pod implementing the service.  The service is available on port 80.

[The code for the service](service) is packaged in a container.  The core of it is [Startup.cs](service/MyApiSolution/MyApi/Startup.cs), more specifically its *ConfigureServices* method:

```csharp
public void ConfigureServices(IServiceCollection services)
{
    services
        .AddAuthentication()
        .AddJwtBearer(options =>
        {
            options.Audience = _applicationId;
            options.Authority = $"https://sts.windows.net/{_tenantId}/";
        });
    services.AddAuthorization(options =>
    {
        var defaultAuthorizationPolicyBuilder = new AuthorizationPolicyBuilder(
            JwtBearerDefaults.AuthenticationScheme);

        defaultAuthorizationPolicyBuilder =
            defaultAuthorizationPolicyBuilder.RequireAuthenticatedUser();
        options.DefaultPolicy = defaultAuthorizationPolicyBuilder.Build();
    });
    services.AddMvc().SetCompatibilityVersion(CompatibilityVersion.Version_2_1);
}
```

Here we specify that a [Json Web Token](https://en.wikipedia.org/wiki/JSON_Web_Token) (JWT) is required for authentication as default.  I.e. we do not need to put a *[Authentication]* attribute on controllers.

## Test

Let's deploy the service & client.

```bash
kubectl apply -f service.yaml --namespace pir
kubectl apply -f client-pod.yaml --namespace pir
kubectl get AzureAssignedIdentity --all-namespaces
```

The last command should return an entry.  This is showing that Azure Pod Identity did bind an identity to a pod.

We can then monitor the client:

```bash
kubectl logs aad-id-client-pod --namespace pir  -f
```

```bash
kubectl delete AzureIdentityBinding client-principal-binding --namespace pir
```

```bash
kubectl apply -f binding.yaml --namespace pir
```

## Clean up

```bash
kubectl delete namespace pir
az identity delete -g $rg -n client-principal
az ad app delete --id $appId
```

#	Build client docker container
sudo docker build -t vplauzon/aad-pod-id-client .

#	Publish client image
sudo docker push vplauzon/aad-pod-id-client

#kubectl run aad-id-rest-client --image=vplauzon/aad-pod-id-svc --generator=run-pod/v1
kubectl apply -f pod.yaml

kubectl logs test-svc-pod

kubectl delete pod test-svc-pod

#	Build service docker container
sudo docker build -t vplauzon/aad-pod-id-svc .

#	Publish service image
sudo docker push vplauzon/aad-pod-id-svc
