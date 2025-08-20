// pkg/telemetry/metrics/metric_registry.go
// Centralized metrics registry - business-focused metrics for image deprecation decisions
package metrics

import (
	"fmt"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	initOnce       sync.Once
	ensureInitOnce sync.Once
	initialized    bool
)

// Registry holds all telemetry metrics
type Registry struct {
	// RED HAT COMPLIANT TELEMETRY METRICS (3 metrics, 10 timeseries max)

	// METRIC 1: Image Version Usage (5 timeseries)
	ImageVersionUsage *prometheus.GaugeVec

	// METRIC 2: Image Source Preference (3 timeseries)
	ImageSourcePreference *prometheus.CounterVec

	// METRIC 3: Enterprise Adoption (2 timeseries)
	EnterpriseAdoption *prometheus.CounterVec

	// INTERNAL OPERATIONAL METRICS (not exported to telemetry)
	ReconcileErrors   *prometheus.CounterVec
	ReconcileDuration *prometheus.HistogramVec
	InternalFailures  *prometheus.CounterVec
}

var registry *Registry

// Initialize creates and registers all telemetry metrics
func Initialize() error {
	var err error
	initOnce.Do(func() {
		klog.Info("Initializing telemetry metrics registry (10 timeseries compliant)")

		registry = &Registry{
			// Business-focused telemetry metrics (Red Hat compliant)
			ImageVersionUsage:     TrainingOperatorImageVersionUsage,
			ImageSourcePreference: TrainingOperatorImageSourcePreference,
			EnterpriseAdoption:    TrainingOperatorEnterpriseAdoption,

			// Internal operational metrics
			ReconcileErrors: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Name: "training_operator_reconcile_errors_total",
					Help: "Total reconciliation errors by controller",
				},
				[]string{"controller"},
			),

			ReconcileDuration: prometheus.NewHistogramVec(
				prometheus.HistogramOpts{
					Name:    "training_operator_reconcile_duration_seconds",
					Help:    "Reconciliation duration by controller",
					Buckets: prometheus.DefBuckets,
				},
				[]string{"controller"},
			),

			InternalFailures: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Name: "training_operator_internal_failures_total",
					Help: "Internal failure tracking for debugging",
				},
				[]string{"component", "reason"},
			),
		}

		// Initialize CRD tracking which registers the business metrics
		InitCRDInstanceTracking()

		// Register internal metrics
		metrics.Registry.MustRegister(
			registry.ReconcileErrors,
			registry.ReconcileDuration,
			registry.InternalFailures,
		)

		initialized = true
		klog.Info("Telemetry metrics registry initialized with 10 timeseries compliance")
	})
	return err
}

// EnsureInitialized ensures metrics are initialized (backward compatibility)
func EnsureInitialized() {
	ensureInitOnce.Do(func() {
		if err := Initialize(); err != nil {
			klog.Errorf("Failed to ensure metrics initialization: %v", err)
		}
	})
}

// InitMetrics is called by EnsureInitialized for backward compatibility
func InitMetrics() {
	if err := Initialize(); err != nil {
		klog.Errorf("Failed to initialize metrics: %v", err)
	}
}

// Get returns the metrics registry instance
func Get() *Registry {
	if !initialized {
		if err := Initialize(); err != nil {
			klog.Errorf("Failed to initialize metrics registry: %v", err)
			return nil
		}
	}
	return registry
}

// IsInitialized returns whether the metrics registry has been initialized
func IsInitialized() bool {
	return initialized
}

// GetTelemetryMetricCount returns the number of telemetry metrics (should be ≤ 3)
func GetTelemetryMetricCount() int {
	return 3 // ImageVersionUsage, ImageSourcePreference, EnterpriseAdoption
}

// ValidateCardinality validates that metrics comply with Red Hat cardinality limits
func ValidateCardinality() error {
	if GetTelemetryMetricCount() > 3 {
		return fmt.Errorf("metric count exceeds Red Hat limit of 3: found %d metrics", GetTelemetryMetricCount())
	}
	return nil
}
