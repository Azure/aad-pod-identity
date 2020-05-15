# Troubleshooting

## Logging

Below is a list of commands you can use to view relevant logs of aad-pod-identity [components](../../README.md#components).

### Isolate errors from logs

You can use `grep ^E` and `--since` flag from `kubectl` to isolate any errors occurred after a given duration.

```bash
kubectl logs -l component=mic --since=1h | grep ^E
kubectl logs -l component=nmi --since=1h | grep ^E
```

> It is always a good idea to include relevant logs from MIC and NMI when opening a new [issue](https://github.com/Azure/aad-pod-identity/issues).

### Ensure that iptables rule exists

To ensure that the correct iptables rule is injected to each node via the [NMI](../../README.md#node-managed-identity) pods, the following command ensures that on a given node, there exists an iptables rule where all packets with a destination IP of 169.254.169.254 (IMDS endpoint) are routed to port 2579 of the host network.

```bash
NMI_POD=$(kubectl get pod -l component=nmi -ojsonpath='{.items[?(@.spec.nodeName=="<NodeName>")].metadata.name}')
kubectl exec $NMI_POD -- iptables -t nat -S aad-metadata
```

The expected output should be:

```log
-N aad-metadata
-A aad-metadata ! -s 127.0.0.1/32 -d 169.254.169.254/32 -p tcp -m tcp --dport 80 -j DNAT --to-destination 10.240.0.34:2579
-A aad-metadata -j RETURN
```

## Common Issues

Common issues or questions that users have run into when using pod identity are detailed below.

### Ignoring azure identity \<podns\>/\<podname\>, error: Invalid resource id: "", must match /subscriptions/\<subid\>/resourcegroups/\<resourcegroup\>/providers/Microsoft.ManagedIdentity/userAssignedIdentities/\<name\>

If you are using MIC v1.6.0+, you will need to ensure the correct capitalization of `AzureIdentity` and `AzureIdentityBinding` fields. For more information, please refer to [this section](../../README.md#v160-breaking-change).

### LinkedAuthorizationFailed

If you received the following error message in MIC:

```log
Code="LinkedAuthorizationFailed" Message="The client '<ClientID>' with object id '<ObjectID>' has permission to perform action 'Microsoft.Compute/<VMType>/write' on scope '<VM/VMSS scope>'; however, it does not have permission to perform action 'Microsoft.ManagedIdentity/userAssignedIdentities/assign/action' on the linked scope(s) '<UserAssignedIdentityScope>' or the linked scope(s) are invalid."
```

It means that your cluster service principal / managed identity does not have the correct role assignment to assign the chosen user-assigned identities to the VM/VMSS. For more information, please follow this [documentation](README.role-assignment.md) to allow your cluster service principal / managed identity to perform identity-related operation.

Past issues:

- https://github.com/Azure/aad-pod-identity/issues/585
