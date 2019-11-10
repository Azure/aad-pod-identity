package metrics

import (
	"context"
	"time"

	log "github.com/Azure/aad-pod-identity/pkg/logger"
	"github.com/golang/glog"
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

	// NodeManagedIdentityOperationsDurationM is a measure that tracks the duration in seconds of odemanagedidentity operations.
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
			Aggregation: view.Distribution(0, 1, 2, 3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47, 97),
		},
		&view.View{
			Description: AssignedIdentityAdditionCountM.Description(),
			Measure:     AssignedIdentityAdditionCountM,
			Aggregation: view.Count(),
		},
		&view.View{
			Description: AssignedIdentityDeletionDurationM.Description(),
			Measure:     AssignedIdentityDeletionDurationM,
			Aggregation: view.Distribution(0, 1, 2, 3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47, 97),
		},
		&view.View{
			Description: AssignedIdentityDeletionCountM.Description(),
			Measure:     AssignedIdentityDeletionCountM,
			Aggregation: view.Count(),
		},
		&view.View{
			Description: NodeManagedIdentityOperationsDurationM.Description(),
			Measure:     NodeManagedIdentityOperationsDurationM,
			Aggregation: view.Distribution(0, 1, 2, 3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47, 97),
			TagKeys:     []tag.Key{operationTypeKey, statusCodeKey},
		},
		&view.View{
			Description: ManagedIdentityControllerCycleDurationM.Description(),
			Measure:     ManagedIdentityControllerCycleDurationM,
			Aggregation: view.Distribution(0, 1, 2, 3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47, 97),
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

// reporter is stats reporter in the context
type reporter struct {
	ctx context.Context
}

// Report records the given measure
func (r *reporter) Report(ms ...stats.Measurement) {
	record(r.ctx, ms...)
}

// NewReporter creates a reporter with new context
func NewReporter() (*reporter, error) {
	ctx, err := tag.New(
		context.Background(),
	)
	if err != nil {
		return nil, err
	}
	return &reporter{ctx: ctx}, nil
}

// NewReporterAndReport creates a new instance of reporter and report given measurements
func NewReporterAndReport(ms ...stats.Measurement) {

	reporter, reporterError := NewReporter()

	if reporterError != nil {
		glog.Error(reporterError)
	} else {
		reporter.Report(ms...)
	}
}

// ReportOperationStatusCount
func (r *reporter) ReportOperationAndStatus(operationType string, statusCode string, ms ...stats.Measurement) error {

	ctx, err := tag.New(
		r.ctx,
		tag.Insert(operationTypeKey, operationType),
		tag.Insert(statusCodeKey, statusCode),
	)
	if err != nil {
		return err
	}
	record(ctx, ms...)
	return nil
}

// ReportMeasurementWithOperation records given measurement by operation type.
func (r *reporter) ReportMeasurementWithOperation(operationType string, measurement stats.Measurement) error {

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

// RecordK8SAPIOperationError records the error in KubernetesAPIOperationsErrorsCountM
func RecordK8SAPIOperationError(operation string) {

	reporter, reporterError := NewReporter()
	if reporterError != nil {
		glog.Error(reporterError)
	} else {
		reporter.ReportMeasurementWithOperation(operation, KubernetesAPIOperationsErrorsCountM.M(1))
	}
}

// RecordCloudProviderOperationError records the error in CloudProviderOperationsErrorsCountM
func RecordCloudProviderOperationError(operation string) {

	reporter, reporterError := NewReporter()
	if reporterError != nil {
		glog.Error(reporterError)
	} else {
		reporter.ReportMeasurementWithOperation(operation, CloudProviderOperationsErrorsCountM.M(1))
	}
}

// RegisterAndExport register the views for the measures and exposeas  prometheus
func RegisterAndExport(port string, log log.Logger) {

	err := registerViews()

	if err != nil {
		log.Errorf("Failed to register views for metrics. error:%v", err)
	}

	log.Infof("Registered views for metric")
	err = newPrometheusExporter(componentNamespace, port, log)
	if err != nil {
		log.Errorf("Prometheus exporter error: %+v", err)
	} else {
		log.Infof("Exported metrics on port %s", port)
	}

}
