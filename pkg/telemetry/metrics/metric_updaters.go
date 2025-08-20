package metrics

import (
	"k8s.io/klog/v2"
)

// RecordImageVersionUsage increments the usage count for a specific image version
// in telemetry metrics tracking.
func RecordImageVersionUsage(version string) {
	if !IsInitialized() {
		if err := Initialize(); err != nil {
			klog.ErrorS(err, "Failed to initialize metrics")
			return
		}
	}

	TrainingOperatorImageVersionUsage.WithLabelValues(version).Inc()
	klog.V(4).InfoS("Recorded image version usage", "version", version)
}

// RecordImageSourcePreference increments the counter for customer preference
// of a specific image source (rhoai_official, community, custom).
func RecordImageSourcePreference(imageSource string) {
	if !IsInitialized() {
		if err := Initialize(); err != nil {
			klog.ErrorS(err, "Failed to initialize metrics")
			return
		}
	}

	TrainingOperatorImageSourcePreference.WithLabelValues(imageSource).Inc()
	klog.V(4).InfoS("Recorded image source preference", "imageSource", imageSource)
}

// RecordEnterpriseAdoption increments the enterprise adoption counter
// for the specified customer type (enterprise or non-enterprise).
func RecordEnterpriseAdoption(customerType string) {
	if !IsInitialized() {
		if err := Initialize(); err != nil {
			klog.ErrorS(err, "Failed to initialize metrics")
			return
		}
	}

	TrainingOperatorEnterpriseAdoption.WithLabelValues(customerType).Inc()
	klog.V(4).InfoS("Recorded enterprise adoption", "customerType", customerType)
}

// RecordReconcileError increments the error counter for a specific controller
// when reconciliation operations fail.
func RecordReconcileError(controller string) {
	reg := Get()
	if reg == nil {
		klog.Warning("Metrics registry not available for recording reconcile error")
		return
	}
	if reg.ReconcileErrors == nil {
		klog.Warning("ReconcileErrors metric not initialized")
		return
	}

	reg.ReconcileErrors.WithLabelValues(controller).Inc()
	klog.V(4).InfoS("Recorded reconcile error", "controller", controller)
}

// RecordReconcileDuration records the time taken for a controller reconciliation
// operation in seconds.
func RecordReconcileDuration(controller string, duration float64) {
	reg := Get()
	if reg == nil {
		klog.Warning("Metrics registry not available for recording reconcile duration")
		return
	}
	if reg.ReconcileDuration == nil {
		klog.Warning("ReconcileDuration metric not initialized")
		return
	}

	reg.ReconcileDuration.WithLabelValues(controller).Observe(duration)
	klog.V(4).InfoS("Recorded reconcile duration", "controller", controller, "duration", duration)
}

// RecordInternalFailure increments the internal failure counter for debugging
// purposes when unexpected errors occur in specific components.
func RecordInternalFailure(component, reason string) {
	reg := Get()
	if reg == nil {
		klog.Warning("Metrics registry not available for recording internal failure")
		return
	}
	if reg.InternalFailures == nil {
		klog.Warning("InternalFailures metric not initialized")
		return
	}

	reg.InternalFailures.WithLabelValues(component, reason).Inc()
	klog.V(4).InfoS("Recorded internal failure", "component", component, "reason", reason)
}
