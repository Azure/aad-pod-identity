
---
title: "Getting Started"
linkTitle: "Getting Started"
weight: 1
date: 2020-10-04
description: >
  It is recommended to get familiar with the AAD Pod Identity ecosystem before diving into the demo. It consists of the Managed Identity Controller (MIC) deployment, the Node Managed Identity (NMI) DaemonSet, and several standard and custom resources.
---


## Role Assignment

Your cluster will need the correct role assignment configuration to perform Azure-related operations such as assigning and un-assigning the identity on the underlying VM/VMSS. You can run the following commands to help you set up the appropriate role assignments for your cluster identity before deploying aad-pod-identity (assuming you are running an AKS cluster):

```bash
export SUBSCRIPTION_ID="<SubscriptionID>"
export RESOURCE_GROUP="<AKSResourceGroup>"
export CLUSTER_NAME="<AKSClusterName>"
export CLUSTER_LOCATION="<AKSClusterLocation>"

# if you are planning to deploy your user-assigned identities in a separate resource group
export IDENTITY_RESOURCE_GROUP="<IdentityResourceGroup>"

./hack/role-assignment.sh
```

> Note: `<AKSResourceGroup>` is where your AKS cluster is deployed to.

For more details, please refer to the [role assignment](./docs/readmes/README.role-assignment.md) documentation.


## What's next?

* [Add content and customize your site](/docs/adding-content/)
* Get some ideas from our [Example Site](https://github.com/google/docsy-example) and other [Examples](/docs/examples/).
* [Publish your site](/docs/deployment/).



