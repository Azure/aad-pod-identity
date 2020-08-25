# Kubernetes Applications With Azure Active Directory Identities

## Abstract

## AAD and Kubernetes

The relationship between kubernetes and AAD is covered in three main areas:

1. **Cluster Identity**: The identity used by the cloud provider running in various kubernetes components to perform operations against Azure, typically against Azure's resource group where the cluster lives. This identity is set during the cluster bring up process. This is not included in the scope of this proposal.

2. **User Identity**: What enables user/operator to authenticate against AAD using AAD before using `kubectl` commands. This is not included in the scope of this proposal.

3. **Application Identities**: Identities that are used by applications running on kubernetes to access any resources that uses AAD as identity provider. These resources can be ARM, Applications running on the same cluster, on azure, or anywhere else. Managing, assigning these identities is the scope of this document.

> This proposal does not cover how application can be configured to use AAD as identity/authentication provider.

## Use cases

1. kubernetes applications depending on other applications that use AAD as an identity provider. These applications include Azure 1st party services (such as ARM, Azure sql or Azure storage) or user applications running on Azure (including same cluster) and or on premises.

    > Azure 1st party services are all moving to use AAD as the primary identity provider.

2. Delegating authorization to user familiar tools such as AAD group memberships.

3. Enable identity rotation without application interruption. 

    > Example: rotating a service principal password/cert without having to edit secrets assigned directly to applications.

4. Provide a framework to enable time-boxed identity assignment. Manually triggered or automated. The same framework can be used for (jit sudo style access with automation tools).

    > Example: a front end application can have access to centralized data store between midnight and 1 AM during business days only.

## Guiding Principles

1. Favor little to no change to how users currently write applications against various editions of [ADAL](https://docs.microsoft.com/en-us/azure/active-directory/develop/active-directory-authentication-libraries). Favor committing changes to SDKs and don't ask users to change applications that are written for Kubernetes.

2. Favor little to no change in the way users create kubernetes application specs (favor declarative approach). This enables users to focus their development and debugging experience in code they wrote, not code imposed on them.

    > Example: favor annotation and labels over side-cars (even dynamically injected).

3. Prep for SPIFFE work currently in progress by the wider community.

4. Separate identities from `identity assignment` applications enables users to swap identities used by the applications.

## Processes

### AAD Identity Management and Assignment (within cluster)

- Cluster operators create instances of `crd:azureIdentity`. Each instance is a kubernetes object representing Azure AAD identity that can be EMSI or service principal (with password).

- Cluster operators create instances of `crd:azureIdentityBinding`. Each instance represents binding between `pod` and `crd:azureIdentity`. The binding initially will use (one of)
    1. Selectors
    2. Explicit assignment to applications (?)
    3. Weight (used in case of a single pod matched by multiple identities)

At later phases, the binding will use:

    1. Expiration time (static: bind until Jan 1/1/2019. Dynamic: for 10 mins).
    2. ...?

    > All identity bindings actions are logged as `events` to `crd:azureIdentity`. Allowing clusters operators to track how the system assigned identities to various apps.

- A Controller will run to create `crd:azureAssignedIdentity` based on `crd:azureIdentityBinding` linking `pod` with `crd:azureIdentity`.

### Acquiring Tokens

> for reference please read [Azure VM Managed Service Identity (MSI)](https://docs.microsoft.com/en-us/azure/active-directory/managed-service-identity/how-to-use-vm-token) and [Assign a Managed Service Identity](https://docs.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/howto-assign-access-portal)

1. Kubernetes applications (pods) will default to use MSI endpoint.
2. All traffic to MSI endpoint is routed via `iptables` to a daemon-set that will use `sourceIp` to identify pod, then find an appropriate `crd:azureIdentityBinding`. The daemon-set mimics all the REST api offered by MSI endpoint. All tokens are presented to pods on MSI endpoint irrespective of the identity used to back this request (EMSI or service principal).

    > As SPIFFE becomes more mature pod identity assertion is expected to use SPIFFE pod identity.

3. All token issuance will be logged as events attached to `crd:azureIdentityBinding` for audit purposes.

## Controllers

A single `azureIdentityController` controller is use to:

    1. Create instances of `azureAssignedIdentity` as a result of `crd:azureIdentityBinding` matching.
    2. Assign EMSI to nodes where pods are scheduled to run. Assignment are logged to `crd:azureIdentity` as events.

> The controller can later expose an endpoint for analysis where operators can request analysis of identities, pod assignment.

## RBAC
<<POD/azureAssignedIdentity>> ??

## Api

### azureIdentity

A custom resource definition object that represents either

1. Explicit Managed Service Identity (EMSI).
2. Service Principal.

```yaml
<<Object details>>
type: <EMSI/Principal>
secretRef: <Principal Password>
```

> Identities are created separately by azure cli. ARM RBAC and data plane specific role assignment is done on each service, application or resource.

### azureIdentityBinding

A custom resource definition object used to describe `azureIdentity` is assigned to pod.

```yaml
<<object details>>
azureIdentityRef: xxx
type: <Explicit/Selector>
weight: ??
timeBox: <<future time box details>>
```

### azureAssignedIdentity

A custom resource definition object. Each instance represents linking `pod` to `azureIdentity`

```yaml
<<object details>>
azureIdentityRef: xxx
podRef: xxx
```

## Plan

### Phase 1

1. Api, controller and daemon set (excluding time boxed assignment).

### Phase 2

1. Time box assignment
2. SPIFFE integration
