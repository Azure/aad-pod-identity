# AAD Pod Identity

## 📣 Announcement

**❗ IMPORTANT**: As of Monday 10/24/2022, AAD Pod Identity is **deprecated**. As mentioned in the [announcement](https://cloudblogs.microsoft.com/opensource/2022/01/18/announcing-azure-active-directory-azure-ad-workload-identity-for-kubernetes/), AAD Pod Identity has been replaced with [Azure Workload Identity](https://azure.github.io/azure-workload-identity). Going forward, we will no longer add new features or bug fixes to this project in favor of Azure Workload Identity, which reached [General Availability (GA) in AKS](https://azure.microsoft.com/en-us/updates/ga-azure-active-directory-workload-identity-with-aks-2/). We will provide CVE patches until September 2023, at which time the project will be archived. **There will be no new releases after September 2023.**

[![GoDoc](https://godoc.org/github.com/Azure/aad-pod-identity?status.svg)](https://godoc.org/github.com/Azure/aad-pod-identity)
[![Go Report Card](https://goreportcard.com/badge/github.com/Azure/aad-pod-identity)](https://goreportcard.com/report/github.com/Azure/aad-pod-identity)

AAD Pod Identity enables Kubernetes applications to access cloud resources securely with [Azure Active Directory](https://azure.microsoft.com/en-us/services/active-directory/).

Using Kubernetes primitives, administrators configure identities and bindings to match pods. Then without any code modifications, your containerized applications can leverage any resource in the cloud that depends on AAD as an identity provider.

## Getting Started

Setup the correct [role assignments](https://azure.github.io/aad-pod-identity/docs/getting-started/role-assignment/) on Azure and install AAD Pod Identity through [Helm](https://azure.github.io/aad-pod-identity/docs/getting-started/installation/#helm) or [YAML deployment files](https://azure.github.io/aad-pod-identity/docs/getting-started/installation/#quick-install). Get familiar with our [CRDs](https://azure.github.io/aad-pod-identity/docs/concepts/azureidentity/) and [core components](https://azure.github.io/aad-pod-identity/docs/concepts/mic/).

Try our [walkthrough](https://azure.github.io/aad-pod-identity/docs/demo/standard_walkthrough/) to get a better understanding of the application workflow.

## Release

Currently, AAD Pod Identity releases on a monthly basis to patch security vulnerabilities, targeting the **first** week of the month. Refer to [Release Cadence](docs/RELEASE.md) for more details.

## Code of Conduct

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/). For more information, see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq) or contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.

## Support

aad-pod-identity is an open source project that is [**not** covered by the Microsoft Azure support policy](https://support.microsoft.com/en-us/help/2941892/support-for-linux-and-open-source-technology-in-azure). [Please search open issues here](https://github.com/Azure/aad-pod-identity/issues), and if your issue isn't already represented please [open a new one](https://github.com/Azure/aad-pod-identity/issues/new/choose). The project maintainers will respond to the best of their abilities.
