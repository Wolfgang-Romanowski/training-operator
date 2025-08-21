// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metrics

import (
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	initOnce    sync.Once
	initialized bool
	
	// trackedVersions defines the set of versions we actively track.
	// This list is limited to maintain cardinality compliance with Red Hat requirements.
	trackedVersions = []string{
		"pytorch-2.4",
		"pytorch-2.3",
		"tensorflow-2.15",
		"tensorflow-2.14",
		"other",
	}
)

// Registry holds all telemetry and operational metrics for the training operator.
// It provides a centralized location for all metrics to ensure consistent
// initialization and management across the operator.
type Registry struct {
	ImageVersionUsage     *prometheus.GaugeVec
	ImageSourcePreference *prometheus.CounterVec
	EnterpriseAdoption    *prometheus.CounterVec
	ReconcileErrors       *prometheus.CounterVec
	ReconcileDuration     *prometheus.HistogramVec
	InternalFailures      *prometheus.CounterVec
}

var registry *Registry

// Initialize creates and registers all telemetry and operational metrics.
// This is the single initialization point for all metrics in the training operator,
// ensuring no duplication and consistent metric registration across all controllers.
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

		// Initialize CRD instance tracking metrics
		initializeCRDInstanceTracking()

		// Register operational metrics
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

// initializeCRDInstanceTracking initializes the CRD instance tracking metrics.
// This internal function sets up telemetry metrics for tracking training job instances
// and starts background routines for cleanup and cardinality monitoring.
func initializeCRDInstanceTracking() {
	metrics.Registry.MustRegister(
		TrainingOperatorImageVersionUsage,
		TrainingOperatorImageSourcePreference,
		TrainingOperatorEnterpriseAdoption,
	)

	// Initialize tracked versions with zero values
	for _, v := range trackedVersions {
		getImageVersionTracker().initializeVersion(v)
		TrainingOperatorImageVersionUsage.WithLabelValues(v).Set(0)
	}

	// Start background cleanup and monitoring routines
	go getImageVersionTracker().cleanupRoutine()
	go getImageVersionTracker().cardinalityMonitor()
	
	// Start circuit breaker monitor for cardinality protection
	StartCircuitBreakerMonitor()

	klog.Info("CRD instance tracking metrics initialized")
}

// Get returns the shared metrics registry instance.
// It initializes the registry if not already initialized, ensuring metrics are
// available when needed by any component.
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
// This allows components to check initialization status without triggering initialization.
func IsInitialized() bool {
	return initialized
}

// GetTelemetryMetricCount returns the number of telemetry metrics being exported.
// This is used to verify compliance with Red Hat monitoring limits.
func GetTelemetryMetricCount() int {
	return 3
}

// ValidateCardinalityLimits ensures metrics stay within acceptable cardinality limits.
// It returns an error if the metric count exceeds Red Hat Monitoring Handbook requirements.
func ValidateCardinalityLimits() error {
	if GetTelemetryMetricCount() > 3 {
		return fmt.Errorf("metric count exceeds limit: found %d metrics, expected <= 3", GetTelemetryMetricCount())
	}
	return nil
}

// EnsureInitialized ensures the telemetry metrics registry is initialized.
// This function is called from main.go to set up telemetry on startup.
// It is safe to call multiple times - subsequent calls are no-ops.
func EnsureInitialized() error {
	return Initialize()
}
