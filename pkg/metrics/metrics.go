package metrics

import (
	"context"
	"time"

	log "github.com/Azure/aad-pod-identity/pkg/logger"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

// This const block defines the metric names.
const (
	assignedIdentityAdditionDurationName                = "assigned_identity_addition_duration_seconds"
	assignedIdentityAdditionCountName                   = "assigned_identity_addition_count"
	assignedIdentityDeletionDurationName                = "assigned_identity_deletion_duration_seconds"
	assignedIdentityDeletionCountName                   = "assigned_identity_deletion_count"
	nodeManagedIdentityOperationsDurationName           = "nodemanagedidentity_operations_duration_seconds"
	managedIdentityControllerCycleDurationName          = "managedidentitycontroller_cycle_duration_seconds"
	managedIdentityControllerCycleCountName             = "managedidentitycontroller_cycle_count"
	managedIdentityControllerNewLeaderElectionCountName = "managedidentitycontroller_new_leader_election_count"
	cloudProviderOperationsErrorsCountName              = "cloud_provider_operations_errors_count"
	cloudProviderOperationsDurationName                 = "cloud_provider_operations_duration_seconds"
	kubernetesAPIOperationsErrorsCountName              = "kubernetes_api_operations_errors_count"
)

// The following variables are measures
var (

	// AssignedIdentityAdditionDurationM is a measure that tracks the duration in seconds of assigned_identity_addition operations.
	AssignedIdentityAdditionDurationM = stats.Float64(
		assignedIdentityAdditionDurationName,
		"Duration in seconds of assigned identity addition operations",
		stats.UnitMilliseconds)

	// AssignedIdentityAdditionCountM is a measure that tracks the cumulative number of assigned identity addition operations.
	AssignedIdentityAdditionCountM = stats.Int64(
		assignedIdentityAdditionCountName,
		"Total number of assigned identity addition operations",
		stats.UnitDimensionless)

	// AssignedIdentityDeletionDurationM is a measure that tracks the duration in seconds of assigned_identity_deletion operations.
	AssignedIdentityDeletionDurationM = stats.Float64(
		assignedIdentityDeletionDurationName,
		"Duration in seconds of assigned identity deletion operations",
		stats.UnitMilliseconds)

	// AssignedIdentityDeletionCountM is a measure that tracks the cumulative number of assigned identity deletion operations.
	AssignedIdentityDeletionCountM = stats.Int64(assignedIdentityDeletionCountName,
		"Total number of assigned identity deletion operations",
		stats.UnitDimensionless)

	// NodeManagedIdentityOperationsDurationM is a measure that tracks the duration in seconds of nodemanagedidentity operations.
	NodeManagedIdentityOperationsDurationM = stats.Float64(
		nodeManagedIdentityOperationsDurationName,
		"Duration in seconds of node managed identity operations",
		stats.UnitMilliseconds)
	// "operation_type", "status_code"

	// ManagedIdentityControllerCycleDurationM is a measure that tracks the duration in seconds of single cycle in Managed Identity Controller.
	ManagedIdentityControllerCycleDurationM = stats.Float64(
		managedIdentityControllerCycleDurationName,
		"Duration in seconds of single cycle in managed identity controller",
		stats.UnitMilliseconds)

	// ManagedIdentityControllerCycleCountM is a measure that tracks the cumulative number of cycles executed in managed identity controller.
	ManagedIdentityControllerCycleCountM = stats.Int64(
		managedIdentityControllerCycleCountName,
		"Total number of cycles executed in managed identity controller",
		stats.UnitDimensionless)

	// ManagedIdentityControllerCycleCountM is a measure that tracks the cumulative number of new leader election in managed identity controller.
	ManagedIdentityControllerNewLeaderElectionCountM = stats.Int64(
		managedIdentityControllerNewLeaderElectionCountName,
		"Total number of new leader election in managed identity controller",
		stats.UnitDimensionless)

	// CloudProviderOperationsErrorsCountM is a measure that tracks the cumulative number of errors in cloud provider operations.
	CloudProviderOperationsErrorsCountM = stats.Int64(
		cloudProviderOperationsErrorsCountName,
		"Total number of errors in cloud provider operations",
		stats.UnitDimensionless)
	// operation_type

	// CloudProviderOperationsDurationM is a measure that tracks the duration in seconds of CloudProviderOperations operations.
	CloudProviderOperationsDurationM = stats.Float64(
		cloudProviderOperationsDurationName,
		"Duration in seconds of cloudprovider operations",
		stats.UnitMilliseconds)
	// operation_type

	// KubernetesAPIOperationsErrorsCountM is a measure that tracks the cumulative number of errors in cloud provider operations.
	KubernetesAPIOperationsErrorsCountM = stats.Int64(
		kubernetesAPIOperationsErrorsCountName,
		"Total number of errors in kubernetes api operations",
		stats.UnitDimensionless)
	// operation_type
)

var (
	operationTypeKey = tag.MustNewKey("operation_type")
	statusCodeKey    = tag.MustNewKey("status_code")
	namespaceKey     = tag.MustNewKey("namespace")
	resourceKey      = tag.MustNewKey("resource")
)

const componentNamespace = "aadpodidentity"

// SinceInSeconds gets the time since the specified start in seconds.
func SinceInSeconds(start time.Time) float64 {
	return time.Since(start).Seconds()
}

// registerViews register views to be collected by exporter
func registerViews() error {

	views := []*view.View{
		&view.View{
			Description: AssignedIdentityAdditionDurationM.Description(),
			Measure:     AssignedIdentityAdditionDurationM,
			Aggregation: view.Distribution(0.01, 0.02, 0.05, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1, 2, 3, 4, 5, 10),
		},
		&view.View{
			Description: AssignedIdentityAdditionCountM.Description(),
			Measure:     AssignedIdentityAdditionCountM,
			Aggregation: view.Count(),
		},
		&view.View{
			Description: AssignedIdentityDeletionDurationM.Description(),
			Measure:     AssignedIdentityDeletionDurationM,
			Aggregation: view.Distribution(0.01, 0.02, 0.05, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1, 2, 3, 4, 5, 10),
		},
		&view.View{
			Description: AssignedIdentityDeletionCountM.Description(),
			Measure:     AssignedIdentityDeletionCountM,
			Aggregation: view.Count(),
		},
		&view.View{
			Description: NodeManagedIdentityOperationsDurationM.Description(),
			Measure:     NodeManagedIdentityOperationsDurationM,
			Aggregation: view.Distribution(0.01, 0.02, 0.05, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1, 2, 3, 4, 5, 10),
			TagKeys:     []tag.Key{operationTypeKey, statusCodeKey, namespaceKey, resourceKey},
		},
		&view.View{
			Description: ManagedIdentityControllerCycleDurationM.Description(),
			Measure:     ManagedIdentityControllerCycleDurationM,
			Aggregation: view.Distribution(0.01, 0.02, 0.05, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1, 2, 3, 4, 5, 10),
		},
		&view.View{
			Description: ManagedIdentityControllerCycleCountM.Description(),
			Measure:     ManagedIdentityControllerCycleCountM,
			Aggregation: view.Count(),
		},
		&view.View{
			Description: ManagedIdentityControllerNewLeaderElectionCountM.Description(),
			Measure:     ManagedIdentityControllerNewLeaderElectionCountM,
			Aggregation: view.Count(),
		},
		&view.View{
			Description: CloudProviderOperationsErrorsCountM.Description(),
			Measure:     CloudProviderOperationsErrorsCountM,
			Aggregation: view.Count(),
			TagKeys:     []tag.Key{operationTypeKey},
		},
		&view.View{
			Description: CloudProviderOperationsDurationM.Description(),
			Measure:     CloudProviderOperationsDurationM,
			Aggregation: view.Distribution(0.01, 0.02, 0.05, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1, 2, 3, 4, 5, 10),
			TagKeys:     []tag.Key{operationTypeKey},
		},
		&view.View{
			Description: KubernetesAPIOperationsErrorsCountM.Description(),
			Measure:     KubernetesAPIOperationsErrorsCountM,
			Aggregation: view.Count(),
			TagKeys:     []tag.Key{operationTypeKey},
		},
	}
	err := view.Register(views...)
	return err
}

// record records the given measure
func record(ctx context.Context, ms ...stats.Measurement) {
	stats.Record(ctx, ms...)
}

// Reporter is stats reporter in the context
type Reporter struct {
	ctx context.Context
}

// NewReporter creates a reporter with new context
func NewReporter() (*Reporter, error) {
	ctx, err := tag.New(
		context.Background(),
	)
	if err != nil {
		return nil, err
	}
	return &Reporter{ctx: ctx}, nil
}

// Report records the given measure
func (r *Reporter) Report(ms ...stats.Measurement) {
	record(r.ctx, ms...)
}

// ReportOperationAndStatus records given measurements by operation type, status code for the given namespace and resource.
func (r *Reporter) ReportOperationAndStatus(operationType string, statusCode string, namespace string, resource string, ms ...stats.Measurement) error {
	ctx, err := tag.New(
		r.ctx,
		tag.Insert(operationTypeKey, operationType),
		tag.Insert(statusCodeKey, statusCode),
		tag.Insert(namespaceKey, namespace),
		tag.Insert(resourceKey, resource),
	)
	if err != nil {
		return err
	}
	record(ctx, ms...)
	return nil
}

// ReportOperation records given measurement by operation type.
func (r *Reporter) ReportOperation(operationType string, measurement stats.Measurement) error {
	ctx, err := tag.New(
		r.ctx,
		tag.Insert(operationTypeKey, operationType),
	)
	if err != nil {
		return err
	}
	record(ctx, measurement)
	return nil
}

// RegisterAndExport register the views for the measures and expose via prometheus exporter
func RegisterAndExport(port string, log log.Logger) error {
	err := registerViews()
	if err != nil {
		log.Errorf("Failed to register views for metrics. error:%v", err)
		return err
	}
	log.Infof("Registered views for metric")
	exporter, err := newPrometheusExporter(componentNamespace, port, log)
	if err != nil {
		log.Errorf("Prometheus exporter error: %+v", err)
		return err
	}
	view.RegisterExporter(exporter)
	log.Infof("Registered and exported metrics on port %s", port)
	return nil
}
