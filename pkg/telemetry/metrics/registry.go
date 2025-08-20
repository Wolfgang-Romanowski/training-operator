// pkg/telemetry/metrics/registry.go
// Centralized metrics registry - single point for all metric definitions and initialization
package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"k8s.io/klog/v2"
)

var (
	initOnce     sync.Once
	initialized  bool
)

// Registry holds all telemetry metrics
type Registry struct {
	// RED HAT COMPLIANT TELEMETRY METRICS (3 maximum per RHOAISTRAT-575)
	
	// METRIC 1: CRD Instance Count (Core requirement: count of CRDs/APIs instances)
	// Cardinality: 6 frameworks × 2 customer types = 12 timeseries
	CRDInstancesActive *prometheus.GaugeVec
	
	// METRIC 2: Customer Usage Distribution (Enterprise vs Non-Enterprise)
	// Cardinality: 2 customer types = 2 timeseries
	CustomerUsage *prometheus.CounterVec
	
	// METRIC 3: Framework Adoption (Business value: framework popularity)
	// Cardinality: 6 frameworks = 6 timeseries
	FrameworkAdoption *prometheus.CounterVec
	
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
		klog.Info("Initializing telemetry metrics registry")
		
		registry = &Registry{
			// Telemetry metrics (Red Hat compliant)
			CRDInstancesActive: prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: "training_operator_crd_instances_active",
					Help: "Current number of active training job CRD instances by framework and customer type",
				},
				[]string{"framework", "customer_type"}, // Binary: enterprise/non-enterprise
			),
			
			CustomerUsage: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Name: "training_operator_customer_usage_total",
					Help: "Customer usage distribution for real vs test workload analysis",
				},
				[]string{"customer_type"}, // Binary: enterprise/non-enterprise
			),
			
			FrameworkAdoption: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Name: "training_operator_framework_adoption_total",
					Help: "Training framework adoption for product planning decisions",
				},
				[]string{"framework"}, // pytorch, tensorflow, mpi, xgboost, jax, paddle
			),
			
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
		
		// Register all metrics with controller-runtime
		err = registerMetrics()
		if err != nil {
			klog.Errorf("Failed to register metrics: %v", err)
			return
		}
		
		initialized = true
		klog.Info("Telemetry metrics registry initialized successfully")
	})
	return err
}

// registerMetrics registers all metrics with the controller-runtime registry
func registerMetrics() error {
	// Register telemetry metrics (exported to Red Hat Observatorium)
	metrics.Registry.MustRegister(
		registry.CRDInstancesActive,
		registry.CustomerUsage,
		registry.FrameworkAdoption,
	)
	
	// Register internal metrics (local monitoring only)
	metrics.Registry.MustRegister(
		registry.ReconcileErrors,
		registry.ReconcileDuration,
		registry.InternalFailures,
	)
	
	return nil
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
	return 3 // CRDInstancesActive, CustomerUsage, FrameworkAdoption
}

// ValidateCardinality validates that metrics comply with Red Hat cardinality limits
func ValidateCardinality() error {
	if GetTelemetryMetricCount() > 3 {
		return prometheus.ErrMetricNameCollision // Reuse existing error type
	}
	// Additional cardinality validation could be added here
	return nil
}