package metrics

import (
	"time"

	"k8s.io/component-base/metrics"
	"k8s.io/component-base/metrics/legacyregistry"
)

// This const block defines the metric names for the kubelet metrics.
const (
	AssignedIdentityAdditionDurationKey                = "assigned_identity_addition_duration_seconds"
	AssignedIdentityAdditionCountKey                   = "assigned_identity_addition_count"
	AssignedIdentityDeletionDurationKey                = "assigned_identity_deletion_duration_seconds"
	AssignedIdentityDeletionCountKey                   = "assigned_identity_deletion_count"
	NodeManagedIdentityOperationsLatencyKey            = "nodemanagedidentity_operations_latency_nanoseconds"
	ManagedIdentityControllerCycleDurationKey          = "managedidentitycontroller_cycle_duration_seconds"
	ManagedIdentityControllerCycleCountKey             = "managedidentitycontroller_cycle_count"
	ManagedIdentityControllerNewLeaderElectionCountKey = "managedidentitycontroller_new_leader_election_count"
	CloudProviderOperationsErrorsCountKey              = "cloud_provider_operations_errors_count"
	KubernetesAPIOperationsErrorsCountKey              = "kubernetes_api_operations_errors_count"
)

var (

	// AssignedIdentityAdditionDuration is a Histogram that tracks the duration (in seconds) it takes to assigned_identity_addition operations.
	AssignedIdentityAdditionDuration = metrics.NewHistogramVec(
		&metrics.HistogramOpts{
			Name:           AssignedIdentityAdditionDurationKey,
			Help:           "Duration in seconds of the assigned_identity_addition operations.",
			StabilityLevel: metrics.ALPHA,
		},
		[]string{"namespace"},
	)

	// AssignedIdentityAdditionCount is a Counter that tracks the cumulative number of assigned identity addition operations.
	AssignedIdentityAdditionCount = metrics.NewCounterVec(
		&metrics.CounterOpts{
			Name:           AssignedIdentityAdditionCountKey,
			Help:           "Cumulative number of assigned identity addition operations.",
			StabilityLevel: metrics.ALPHA,
		},
		[]string{"namespace"},
	)

	// AssignedIdentityDeletionDuration is a Histogram that tracks the duration (in seconds) it takes to assigned_identity_deletion operations.
	AssignedIdentityDeletionDuration = metrics.NewHistogramVec(
		&metrics.HistogramOpts{
			Name:           AssignedIdentityDeletionDurationKey,
			Help:           "Duration in seconds of the assigned_identity_deletion operations.",
			StabilityLevel: metrics.ALPHA,
		},
		[]string{"namespace"},
	)

	// AssignedIdentityDeletionCount is a Counter that tracks the cumulative number of assigned identity deletion operations.
	AssignedIdentityDeletionCount = metrics.NewCounterVec(
		&metrics.CounterOpts{
			Name:           AssignedIdentityDeletionCountKey,
			Help:           "Cumulative number of assigned identity deletion operations.",
			StabilityLevel: metrics.ALPHA,
		},
		[]string{"namespace"},
	)

	// NodeManagedIdentityOperationsLatency is a Histogram that tracks the latency (in nanoseconds) of nodemanagedidentity operations
	// to complete. Broken down by operation type, status code.
	NodeManagedIdentityOperationsLatency = metrics.NewHistogramVec(
		&metrics.HistogramOpts{
			Name:           NodeManagedIdentityOperationsLatencyKey,
			Help:           "Latency in nanoseconds of nodemanagedidentity operations. Broken down by operation type, status code.",
			StabilityLevel: metrics.ALPHA,
		},
		[]string{"operation_type", "status_code"},
	)

	// ManagedIdentityControllerCycleDuration is a Histogram that tracks the duration (in seconds) it takes for a single cycle in Managed Identity Controller.
	ManagedIdentityControllerCycleDuration = metrics.NewHistogram(
		&metrics.HistogramOpts{
			Name:           ManagedIdentityControllerCycleDurationKey,
			Help:           "Duration in seconds for a single cycle in Managed Identity Controller.",
			StabilityLevel: metrics.ALPHA,
		},
	)

	// ManagedIdentityControllerCycleCount is a Counter that tracks the number of cycles executed in Managed Identity Controller.
	ManagedIdentityControllerCycleCount = metrics.NewCounter(
		&metrics.CounterOpts{
			Name:           ManagedIdentityControllerCycleCountKey,
			Help:           "The number of cycles executed in Managed Identity Controller.",
			StabilityLevel: metrics.ALPHA,
		},
	)

	// ManagedIdentityControllerNewLeaderElectionCount is a Counter that tracks the cumulative number of new leader election in Managed Identity Controller.
	ManagedIdentityControllerNewLeaderElectionCount = metrics.NewCounter(
		&metrics.CounterOpts{
			Name:           ManagedIdentityControllerNewLeaderElectionCountKey,
			Help:           "Cumulative number of new leader election in Managed Identity Controller.",
			StabilityLevel: metrics.ALPHA,
		},
	)

	// CloudProviderOperationsErrorsCount is a Counter that tracks the cumulative number of cloud provider operations errors.
	// Broken down by operation type.
	CloudProviderOperationsErrorsCount = metrics.NewCounterVec(
		&metrics.CounterOpts{
			Name:           CloudProviderOperationsErrorsCountKey,
			Help:           "Cumulative number of cloud provider operations errors by operation type.",
			StabilityLevel: metrics.ALPHA,
		},
		[]string{"operation_type"},
	)

	// KubernetesAPIOperationsErrorsCount is a Counter that tracks the cumulative number of kubernetes api operations errors.
	// Broken down by operation type.
	KubernetesAPIOperationsErrorsCount = metrics.NewCounterVec(
		&metrics.CounterOpts{
			Name:           KubernetesAPIOperationsErrorsCountKey,
			Help:           "Cumulative number of kubernetes api operations errors by operation type.",
			StabilityLevel: metrics.ALPHA,
		},
		[]string{"operation_type"},
	)
)

// SinceInSeconds gets the time since the specified start in seconds.
func SinceInSeconds(start time.Time) float64 {
	return time.Since(start).Seconds()
}

// SinceInMicroseconds gets the time since the specified start in microseconds.
func SinceInMicroseconds(start time.Time) float64 {
	return float64(time.Since(start).Nanoseconds() / time.Microsecond.Nanoseconds())
}

// Register prometheus metrics.
func Register() {
	// TODO : conditional registeration depending on MIC or NMI
	legacyregistry.MustRegister(AssignedIdentityAdditionDuration)
	legacyregistry.MustRegister(AssignedIdentityAdditionCount)
	legacyregistry.MustRegister(AssignedIdentityDeletionDuration)
	legacyregistry.MustRegister(AssignedIdentityDeletionCount)
	legacyregistry.MustRegister(NodeManagedIdentityOperationsLatency)
	legacyregistry.MustRegister(ManagedIdentityControllerCycleDuration)
	legacyregistry.MustRegister(ManagedIdentityControllerCycleCount)
	legacyregistry.MustRegister(ManagedIdentityControllerNewLeaderElectionCount)
	legacyregistry.MustRegister(CloudProviderOperationsErrorsCount)
	legacyregistry.MustRegister(KubernetesAPIOperationsErrorsCount)
}
