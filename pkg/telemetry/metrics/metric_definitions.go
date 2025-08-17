package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// =================================================================
	// PRIMARY BUSINESS METRICS - RHOAI Runtime Adoption
	// These are the CORE metrics for understanding customer behavior
	// =================================================================

	TrainingJobsByImageSource = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "training_jobs_by_image_source_total",
			Help: "Total training jobs created by image source (rhoai_official/community/custom)",
		},
		[]string{"framework", "image_source", "rhoai_version"},
	)

	ActiveRHOAIVersionDistribution = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "training_jobs_active_rhoai_versions",
			Help: "Currently active training jobs by RHOAI version (2.3/2.4/2.5)",
		},
		[]string{"framework", "rhoai_version"},
	)

	RHOAIVersionMigration = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "training_jobs_version_migration_total",
			Help: "Version migration patterns (e.g., 2.4->2.5)",
		},
		[]string{"framework", "from_version", "to_version"},
	)

	// =================================================================
	// COMPLIANCE METRICS - Required by RHOAISTRAT-575
	// These ensure we meet platform monitoring requirements
	// =================================================================

	TrainingJobsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "training_jobs_created_total",
			Help: "Total training jobs created",
		},
		[]string{"framework"}, // Minimal labels for cardinality
	)

	TrainingJobsCompleted = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "training_jobs_completed_total",
			Help: "Total training jobs completed by status",
		},
		[]string{"framework", "status"},
	)

	ReconcileErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "training_operator_reconcile_errors_total",
			Help: "Total reconciliation errors",
		},
		[]string{"controller"},
	)

	// =================================================================
	// INTERNAL TRACKING - Not exported to Telemetry
	// These support metric calculation but stay local
	// =================================================================

	jobStartTimes = make(map[string]time.Time)
	jobMutex      sync.RWMutex
)

func InitMetrics() {
	metrics.Registry.MustRegister(
		// Business metrics - RHOAI adoption
		TrainingJobsByImageSource,
		ActiveRHOAIVersionDistribution,
		RHOAIVersionMigration,

		// Compliance metrics
		TrainingJobsTotal,
		TrainingJobsCompleted,
		ReconcileErrors,
	)
}

// Helper functions for tracking
func RecordJobStart(jobKey string) {
	jobMutex.Lock()
	defer jobMutex.Unlock()
	jobStartTimes[jobKey] = time.Now()
}

func RemoveJobStartTime(jobKey string) {
	jobMutex.Lock()
	defer jobMutex.Unlock()
	delete(jobStartTimes, jobKey)
}
