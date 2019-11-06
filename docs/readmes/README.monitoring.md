# Azure Identity Monitoring  

## Introduction  

AAD pod identity is a foundational service that other applications depend upon, it is recommended to monitor the same.

Liveliness probe and Prometheus metrics are available in both Managed Identity Controller (MIC) and the Node Managed Identity (NMI) components.
  
## Liveliness Probe

MIC and NMI exposes /healthz endpoint with content of "Active/Not Active" state.
State "Active" is being returned if the component has started successfully and "Not Active" otherwise.  

## Prometheus Metrics 

[Prometheus](https://github.com/prometheus/prometheus) is a systems and service monitoring system. It collects metrics from configured targets at given intervals, evaluates rule expressions,displays the results, and can trigger alerts if some condition is observed to be true.

The following Prometheus metrics are exposed in AAD pod identity system.  

**1. assigned_identity_addition_duration_seconds**

Histogram that tracks the duration (in seconds) it takes to Assigned identity addition operations.

**2. assigned_identity_addition_count**

Counter that tracks the cumulative number of assigned identity addition operations.

**3. assigned_identity_deletion_duration_seconds**

Histogram that tracks the duration (in seconds) it takes to Assigned identity deletion operations.

**4. assigned_identity_deletion_count**

Counter that tracks the cumulative number of assigned identity deletion operations.


**5. nodemanagedidentity_operations_latency_nanoseconds**

Histogram that tracks the latency (in nanoseconds) of Node Managed Identity operations to complete. Broken down by operation type, status code.

**6. managedidentitycontroller_cycle_duration_seconds**

Histogram that tracks the duration (in seconds) it takes for a single cycle in Managed Identity Controller.

**7. managedidentitycontroller_cycle_count**

Counter that tracks the number of cycles executed in Managed Identity Controller.

**8. managedidentitycontroller_cycle_duration_seconds**

Histogram that tracks the duration (in seconds) it takes for a single cycle in Managed Identity Controller.

**9. managedidentitycontroller_cycle_count**

Counter that tracks the number of cycles executed in Managed Identity Controller.

**10. managedidentitycontroller_new_leader_election_count**

Counter that tracks the cumulative number of new leader election in Managed Identity Controller.

**11. cloud_provider_operations_errors_count**

Counter that tracks the cumulative number of cloud provider operations errors.Broken down by operation type.

**12. kubernetes_api_operations_errors_count**

Counter that tracks the cumulative number of kubernetes api operations errors.Broken down by operation type.