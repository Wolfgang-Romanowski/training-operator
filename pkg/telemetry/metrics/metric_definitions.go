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
	// GPU/ACCELERATOR METRICS - REQUIRED by RHOAISTRAT-575 page 8
	// =================================================================

	AcceleratorHoursConsumed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "training_accelerator_hours_consumed_total",
			Help: "Total accelerator hours consumed by training jobs normalized per OBSDA-1087",
		},
		[]string{"framework", "accelerator_type", "image_source"},
	)

	AcceleratorUtilization = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "training_accelerator_active_count",
			Help: "Current accelerators in use by training jobs",
		},
		[]string{"framework", "accelerator_type"},
	)

	// =================================================================
	// KUEUE INTEGRATION METRICS - REQUIRED by RHOAISTRAT-575 page 8
	// =================================================================

	KueueManagedJobs = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "training_kueue_managed_jobs_total",
			Help: "Jobs managed through Kueue workload manager",
		},
		[]string{"framework", "queue_name", "image_source"},
	)

	KueueQueueDepth = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "training_kueue_queue_depth",
			Help: "Current depth of Kueue queues for training jobs",
		},
		[]string{"queue_name"},
	)

	// =================================================================
	// JOB LIFECYCLE METRICS - For SLO Monitoring
	// =================================================================

	JobQueueDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "training_job_queue_duration_seconds",
			Help:    "Time from job creation to first pod scheduling",
			Buckets: prometheus.ExponentialBuckets(10, 2, 10), // 10s to 5120s
		},
		[]string{"framework", "image_source"},
	)

	JobRunDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "training_job_run_duration_seconds",
			Help:    "Time from job start to completion",
			Buckets: prometheus.ExponentialBuckets(60, 2, 12), // 1min to 68hours
		},
		[]string{"framework", "image_source", "status"},
	)

	JobFailureReasons = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "training_job_failure_reasons_total",
			Help: "Job failures categorized by reason",
		},
		[]string{"framework", "reason", "image_source"},
	)

	// =================================================================
	// COMPLIANCE METRICS - Required by RHOAISTRAT-575
	// =================================================================

	TrainingJobsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "training_jobs_created_total",
			Help: "Total training jobs created",
		},
		[]string{"framework"},
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

		// GPU/Accelerator metrics - REQUIRED
		AcceleratorHoursConsumed,
		AcceleratorUtilization,

		// Kueue metrics - REQUIRED
		KueueManagedJobs,
		KueueQueueDepth,

		// Job lifecycle metrics
		JobQueueDuration,
		JobRunDuration,
		JobFailureReasons,

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

func GetJobStartTime(jobKey string) (time.Time, bool) {
	jobMutex.RLock()
	defer jobMutex.RUnlock()
	startTime, exists := jobStartTimes[jobKey]
	return startTime, exists
}

func RemoveJobStartTime(jobKey string) {
	jobMutex.Lock()
	defer jobMutex.Unlock()
	delete(jobStartTimes, jobKey)
}
