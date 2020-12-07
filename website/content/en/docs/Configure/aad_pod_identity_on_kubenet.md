---
title: "Deploy AAD Pod Identity in a Cluster with Kubenet"
linkTitle: "Deploy AAD Pod Identity in a Cluster with Kubenet"
weight: 2
description: >
  AAD Pod Identity is disabled by default on Clusters with Kubenet starting from release v1.7. 
---

> Starting from 1.7 release

## Introduction

AAD Pod Identity is disabled by default on clusters with Kubenet network plugin. The NMI pods will fail to run with error `AAD Pod Identity is not supported for Kubenet`.

## Why this change?

Kubenet network plugin is susceptible to ARP spoofing. This makes it possible for pods to impersonate as a pod with access to an identity. Using `CAP_NET_RAW` capability the attacker pod could then request token as a pod it's impersonating.

Network plugins like Azure CNI, Calico, Cilium prevents ARP Spoofing.

## Mitigation steps to take before running clusters with Kubenet

The recommended steps to take before configuring AAD Pod Identity to run on clusters with Kubenet network plugin

- Add a [`securityContext`](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/) that drops the `NET_RAW` capability by default in your applications.
    ```yaml
    securityContext:
      capabilities:
        drop:
        - NET_RAW
    ```
  
  This shouldn’t affect most applications, since it's only needed for applications that do deep networking inspection/manipulation. Dropping this capability will make sure even if your application code got compromised, the attacker could not perform such network-based attacks on your cluster.

## How to run AAD Pod Identity on clusters with Kubenet

Set the `--allow-network-plugin-kubenet=true` arg in the NMI container (or `--set nmi.allowNetworkPluginKubenet=true` if deploying with Helm) to continue running on clusters with Kubenet.

To mitigate the vulnerability at the cluster level, you can use [OpenPolicyAgent](https://www.openpolicyagent.org/) admission controller together with [Gatekeeper](https://github.com/open-policy-agent/gatekeeper) validating webhook.

Provided you have Gatekeeper already installed in your cluster, add the `ConstraintTemplate` of type `K8sPSPCapabilities`:

```bash
kubectl apply -f https://raw.githubusercontent.com/open-policy-agent/gatekeeper-library/master/library/pod-security-policy/capabilities/template.yaml
```

Add a template to limit the spawning of Pods with the `NET_RAW` capability:

```yaml
apiVersion: constraints.gatekeeper.sh/v1beta1
kind: K8sPSPCapabilities
metadata:
  name: prevent-net-raw
spec:
  match:
    kinds:
      - apiGroups: [""]
        kinds: ["Pod"]
    excludedNamespaces:
      - "kube-system"
  parameters:
    requiredDropCapabilities: ["NET_RAW"]
```

You can either [exclude specific namespaces](https://github.com/open-policy-agent/gatekeeper/blob/master/README.md#exempting-namespaces-from-gatekeeper) like in the example above or explicitly include namespaces with `spec.match.namespaces`.
