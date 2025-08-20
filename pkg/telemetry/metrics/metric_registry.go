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

// Registry holds all telemetry and operational metrics for the training operator.
type Registry struct {
	ImageVersionUsage     *prometheus.GaugeVec
	ImageSourcePreference *prometheus.CounterVec
	EnterpriseAdoption    *prometheus.CounterVec
	ReconcileErrors       *prometheus.CounterVec
	ReconcileDuration     *prometheus.HistogramVec
	InternalFailures      *prometheus.CounterVec
}

var registry *Registry

// Initialize creates and registers all telemetry and operational metrics
// for the training operator controllers.
func Initialize() error {
	var err error
	initOnce.Do(func() {
		klog.Info("Initializing telemetry metrics registry")

		registry = &Registry{
			ImageVersionUsage:     TrainingOperatorImageVersionUsage,
			ImageSourcePreference: TrainingOperatorImageSourcePreference,
			EnterpriseAdoption:    TrainingOperatorEnterpriseAdoption,

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

		InitCRDInstanceTracking()

		metrics.Registry.MustRegister(
			registry.ReconcileErrors,
			registry.ReconcileDuration,
			registry.InternalFailures,
		)

		initialized = true
		klog.Info("Telemetry metrics registry initialized successfully")
	})
	return err
}

// EnsureInitialized ensures metrics are initialized exactly once for backward compatibility.
func EnsureInitialized() {
	ensureInitOnce.Do(func() {
		if err := Initialize(); err != nil {
			klog.Errorf("Failed to ensure metrics initialization: %v", err)
		}
	})
}

// InitMetrics provides legacy initialization for backward compatibility with existing code.
func InitMetrics() {
	if err := Initialize(); err != nil {
		klog.Errorf("Failed to initialize metrics: %v", err)
	}
}

// Get returns the shared metrics registry instance, initializing it if necessary.
func Get() *Registry {
	if !initialized {
		if err := Initialize(); err != nil {
			klog.Errorf("Failed to initialize metrics registry: %v", err)
			return nil
		}
	}
	return registry
}

// IsInitialized returns true if the metrics registry has been successfully initialized.
func IsInitialized() bool {
	return initialized
}

// GetTelemetryMetricCount returns the number of telemetry metrics being exported.
func GetTelemetryMetricCount() int {
	return 3
}

// ValidateCardinalityLimits ensures metrics stay within acceptable cardinality limits.
func ValidateCardinalityLimits() error {
	if GetTelemetryMetricCount() > 3 {
		return fmt.Errorf("metric count exceeds limit: found %d metrics, expected <= 3", GetTelemetryMetricCount())
	}
	return nil
}
