## aad-pod-identity - Known Issues

This section lists the major known issues with aad-pod-identity. For complete list of issues please check our Github issues page. If you notice an issue not listed in Github issues page, please do file an issue on the Github repository.

- NMI pods not yet running during cluster autoscale event

### NMI pods not yet running during cluster autoscale event

NMI redirects requests to IMDS to itself by setting up iptable rules after it starts running on the node. When a cluster is scaled up, there **might** be a scenario where the `kube-scheduler` schedules the workload pod before the NMI pod on the new nodes. In such scenario token request will be directly sent to Instance Metadata Service (IMDS) instead of being intercepted by NMI. What this means is the workload pod that runs before the NMI pod is run on the node can access identities that it doesn't have access to. 

There is currently no solution in kubernetes where node can be set to NoSchedule until critical addons are deployed to the cluster. There was a KEP for this particular enhancement - kubernetes/enhancements#1003 which is now closed.
