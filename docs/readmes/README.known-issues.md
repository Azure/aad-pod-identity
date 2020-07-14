# Known Issues

This section lists the major known issues with aad-pod-identity. For a complete list of issues, please check our [GitHub issues page](https://github.com/Azure/aad-pod-identity/issues) or [file a new issue](https://github.com/Azure/aad-pod-identity/issues/new?assignees=&labels=bug&template=bug_report.md&title=) if your issue is not listed.

- NMI pods not yet running during cluster autoscale event

## NMI pods not yet running during cluster autoscale event

NMI redirects Instance Metadata Service (IMDS) requests to itself by setting up iptables rules after it starts running on the node. When a cluster is scaled up, there **might** be a scenario where the `kube-scheduler` schedules the workload pod before the NMI pod on the new nodes. In such scenario, the token request will be directly sent to IMDS instead of being intercepted by NMI. What this means is that the workload pod that runs before the NMI pod is run on the node can access identities that it doesn't have access to.

There is currently no solution in Kubernetes where a node can be set to `NoSchedule` until critical addons are deployed to the cluster. There was a KEP for this particular enhancement - [kubernetes/enhancements#1003](https://github.com/kubernetes/enhancements/pull/1003) which is now closed.