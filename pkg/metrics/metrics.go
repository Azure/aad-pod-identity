package metrics

import (
	"time"

	"k8s.io/component-base/metrics"
	"k8s.io/component-base/metrics/legacyregistry"
)

// This const block defines the metric names for the kubelet metrics.
const (
	AssignedIdentityAdditionKey               = "assigned_identity_addition_duration_seconds"
	AssignedIdentityDeletionKey               = "assigned_identity_deletion_duration_seconds"
	NodeManagedIdentityOperationsLatencyKey   = "nodemanagedidentity_operations_latency_nanoseconds"
	ManagedIdentityControllerCycleDurationKey = "managedidentitycontroller_cycle_duration_seconds"
	ManagedIdentityControllerCycleCountKey    = "managedidentitycontroller_cycle_count"
)

var (

	// AssignedIdentityAddition is a Histogram that tracks the duration (in seconds) it takes to assigned_identity_addition operations.
	AssignedIdentityAddition = metrics.NewHistogramVec(
		&metrics.HistogramOpts{
			Name:           AssignedIdentityAdditionKey,
			Help:           "Duration in seconds of the assigned_identity_addition operations.",
			StabilityLevel: metrics.ALPHA,
		},
		[]string{"namespace"},
	)
	// AssignedIdentityDeletion is a Histogram that tracks the duration (in seconds) it takes to assigned_identity_deletion operations.
	AssignedIdentityDeletion = metrics.NewHistogramVec(
		&metrics.HistogramOpts{
			Name:           AssignedIdentityDeletionKey,
			Help:           "Duration in seconds of the assigned_identity_deletion operations.",
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
	legacyregistry.MustRegister(AssignedIdentityAddition)
	legacyregistry.MustRegister(AssignedIdentityDeletion)
	legacyregistry.MustRegister(NodeManagedIdentityOperationsLatency)
	legacyregistry.MustRegister(ManagedIdentityControllerCycleDuration)
	legacyregistry.MustRegister(ManagedIdentityControllerCycleCount)
}
