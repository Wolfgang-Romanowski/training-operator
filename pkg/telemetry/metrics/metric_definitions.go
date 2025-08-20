// pkg/telemetry/metrics/metric_definitions.go
package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// =================================================================
	// RED HAT COMPLIANT TELEMETRY METRICS - 3 METRICS MAXIMUM
	// Fully compliant with RHOAISTRAT-575 cardinality limits
	// =================================================================

	// NOTE: CRD instance tracking metrics are defined in crd_instance_tracking.go
	// This file only contains internal operational metrics not exported to telemetry

	// =================================================================
	// INTERNAL METRICS - NOT EXPORTED TO TELEMETRY
	// =================================================================

	// Version tracking for internal use only
	internalVersionDistribution = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "training_internal_version_distribution",
			Help: "Internal metric for version distribution analysis",
		},
		[]string{"framework", "version"},
	)

	// Detailed failure tracking for debugging
	internalFailureReasons = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "training_internal_failure_reasons",
			Help: "Internal metric for failure analysis",
		},
		[]string{"framework", "reason"},
	)

	// Controller health metrics
	ReconcileErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "training_operator_reconcile_errors_total",
			Help: "Total reconciliation errors",
		},
		[]string{"controller"},
	)

	ReconcileDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "training_operator_reconcile_duration_seconds",
			Help:    "Reconciliation duration",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"controller"},
	)

	// =================================================================
	// INTERNAL STATE TRACKING
	// =================================================================

	jobTracker = &JobTracker{
		startTimes: make(map[string]time.Time),
		versions:   make(map[string]string),
		maxAge:     24 * time.Hour,
		maxEntries: 10000,
	}

	initOnce sync.Once
)

// JobTracker provides thread-safe tracking with automatic cleanup
type JobTracker struct {
	mu         sync.RWMutex
	startTimes map[string]time.Time
	versions   map[string]string
	maxAge     time.Duration
	maxEntries int
}

// InitMetrics registers all metrics with the controller-runtime registry
func InitMetrics() {
	initOnce.Do(func() {
		// Register only internal operational metrics (not exported to telemetry)
		metrics.Registry.MustRegister(
			// Controller health metrics (internal only, not exported)
			ReconcileErrors,
			ReconcileDuration,
			// Internal metrics for debugging (not exported)
			internalVersionDistribution,
			internalFailureReasons,
		)

		// Initialize CRD instance tracking
		InitCRDInstanceTracking()

		// Start cleanup routine for memory management
		go jobTracker.cleanupRoutine()
	})
}

// EnsureInitialized ensures metrics are initialized (idempotent)
func EnsureInitialized() {
	InitMetrics()
}

// Thread-safe job tracking methods
func (jt *JobTracker) SetStartTime(key string, t time.Time) {
	jt.mu.Lock()
	defer jt.mu.Unlock()

	// Prevent unbounded growth
	if len(jt.startTimes) >= jt.maxEntries {
		jt.removeOldestLocked()
	}

	jt.startTimes[key] = t
}

func (jt *JobTracker) GetStartTime(key string) (time.Time, bool) {
	jt.mu.RLock()
	defer jt.mu.RUnlock()

	t, exists := jt.startTimes[key]
	return t, exists
}

func (jt *JobTracker) Delete(key string) {
	jt.mu.Lock()
	defer jt.mu.Unlock()

	delete(jt.startTimes, key)
	delete(jt.versions, key)
}

func (jt *JobTracker) SetVersion(key, version string) {
	jt.mu.Lock()
	defer jt.mu.Unlock()

	jt.versions[key] = version
}

func (jt *JobTracker) GetVersion(key string) (string, bool) {
	jt.mu.RLock()
	defer jt.mu.RUnlock()

	v, exists := jt.versions[key]
	return v, exists
}

// Automatic cleanup routine
func (jt *JobTracker) cleanupRoutine() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		jt.cleanup()
	}
}

func (jt *JobTracker) cleanup() {
	jt.mu.Lock()
	defer jt.mu.Unlock()

	now := time.Now()
	for key, startTime := range jt.startTimes {
		if now.Sub(startTime) > jt.maxAge {
			delete(jt.startTimes, key)
			delete(jt.versions, key)
		}
	}
}

func (jt *JobTracker) removeOldestLocked() {
	var oldestKey string
	var oldestTime time.Time

	for key, t := range jt.startTimes {
		if oldestKey == "" || t.Before(oldestTime) {
			oldestKey = key
			oldestTime = t
		}
	}

	if oldestKey != "" {
		delete(jt.startTimes, oldestKey)
		delete(jt.versions, oldestKey)
	}
}

// Helper functions for metric updates
func RecordJobStart(jobKey string) {
	jobTracker.SetStartTime(jobKey, time.Now())
}

func GetJobStartTime(jobKey string) (time.Time, bool) {
	return jobTracker.GetStartTime(jobKey)
}

func RemoveJobTracking(jobKey string) {
	jobTracker.Delete(jobKey)
}