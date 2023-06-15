# Release Management

## Overview

This document describes AAD Pod Identity project release management, which talks about cadence.

**❗ IMPORTANT**: As mentioned in the [announcement](https://cloudblogs.microsoft.com/opensource/2022/01/18/announcing-azure-active-directory-azure-ad-workload-identity-for-kubernetes/), we are replacing AAD Pod Identity with [Azure Workload Identity](https://azure.github.io/azure-workload-identity). Going forward, we will no longer fix bugs or add new features to this project in favor of Azure Workload Identity. However, we will continue patching security vulnerabilities until September 2023.

## Release Cadence

`1.8` will be the last major and minor release of AAD Pod Identity. We will not release a new major or minor version of AAD Pod Identity. However, we will continue publishing patch releases the **first week of every month** to fix security vulnerabilities until September 2023.

## Security Vulnerabilities

We use [trivy](https://github.com/aquasecurity/trivy) to scan the base image for known vulnerabilities. When a vulnerability is detected and has a fixed version, we will update the image to include the fix. For vulnerabilities that are not in a fixed version, there is nothing that can be done immediately. 
Fixable CVE patches will be part of the patch releases published **first week of every month** until September 2023.
