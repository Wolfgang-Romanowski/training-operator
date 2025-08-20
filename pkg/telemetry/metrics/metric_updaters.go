// pkg/telemetry/metrics/metric_updaters.go
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// UpdateImageSourceMetric updates the image source counter
func UpdateImageSourceMetric(framework, imageSource, rhoaiVersion string) {
	// Only track image source for cardinality control
	TrainingJobsByImageSource.WithLabelValues(imageSource).Inc()
}

// IncrementActiveJobs increments active job count
func IncrementActiveJobs(framework string) {
	TrainingJobsActive.WithLabelValues(framework).Inc()
}

// DecrementActiveJobs decrements active job count
func DecrementActiveJobs(framework string) {
	// Ensure we don't go negative
	TrainingJobsActive.WithLabelValues(framework).Dec()
}

// RecordJobCreated records job creation
func RecordJobCreated(framework string) {
	TrainingJobsCreated.WithLabelValues(framework).Inc()
}

// RecordJobCompletion records job completion with status
func RecordJobCompletion(framework, status string) {
	TrainingJobsCompleted.WithLabelValues(status).Inc()
}

// RecordReconcileError records reconciliation errors
func RecordReconcileError(controller string) {
	ReconcileErrors.WithLabelValues(controller).Inc()
}

// RecordReconcileDuration records reconciliation duration
func RecordReconcileDuration(controller string, duration float64) {
	ReconcileDuration.WithLabelValues(controller).Observe(duration)
}