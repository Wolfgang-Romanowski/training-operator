// pkg/telemetry/metrics/metric_updaters.go
// Helper functions for updating business-focused metrics
package metrics

import (
	"k8s.io/klog/v2"
)

// RecordImageVersionUsage records a job using a specific image version
func RecordImageVersionUsage(version string) {
	if !IsInitialized() {
		EnsureInitialized()
	}

	TrainingOperatorImageVersionUsage.WithLabelValues(version).Inc()
	klog.V(4).Infof("Recorded image version usage: %s", version)
}

// RecordImageSourcePreference records customer preference for image source
func RecordImageSourcePreference(imageSource string) {
	if !IsInitialized() {
		EnsureInitialized()
	}

	TrainingOperatorImageSourcePreference.WithLabelValues(imageSource).Inc()
	klog.V(4).Infof("Recorded image source preference: %s", imageSource)
}

// RecordEnterpriseAdoption records enterprise adoption patterns
func RecordEnterpriseAdoption(customerType string) {
	if !IsInitialized() {
		EnsureInitialized()
	}

	TrainingOperatorEnterpriseAdoption.WithLabelValues(customerType).Inc()
	klog.V(4).Infof("Recorded enterprise adoption: %s", customerType)
}

// RecordReconcileError records reconciliation errors
func RecordReconcileError(controller string) {
	reg := Get()
	if reg != nil && reg.ReconcileErrors != nil {
		reg.ReconcileErrors.WithLabelValues(controller).Inc()
	}
}

// RecordReconcileDuration records reconciliation duration
func RecordReconcileDuration(controller string, duration float64) {
	reg := Get()
	if reg != nil && reg.ReconcileDuration != nil {
		reg.ReconcileDuration.WithLabelValues(controller).Observe(duration)
	}
}

// RecordInternalFailure records internal failures for debugging
func RecordInternalFailure(component, reason string) {
	reg := Get()
	if reg != nil && reg.InternalFailures != nil {
		reg.InternalFailures.WithLabelValues(component, reason).Inc()
	}
}
