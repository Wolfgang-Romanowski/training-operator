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
	"k8s.io/klog/v2"
)

// RecordReconcileError increments the error counter for a specific controller.
// This tracks reconciliation failures to identify controllers that may need
// attention or debugging.
func RecordReconcileError(controller string) {
	// Check circuit breaker first
	if IsCircuitBreakerOpen() {
		klog.V(5).Info("Circuit breaker open, skipping metric update")
		return
	}

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

// RecordReconcileDuration records the time taken for a controller reconciliation.
// The duration is measured in seconds and helps identify performance bottlenecks
// in the reconciliation loop.
func RecordReconcileDuration(controller string, duration float64) {
	// Check circuit breaker first
	if IsCircuitBreakerOpen() {
		klog.V(5).Info("Circuit breaker open, skipping metric update")
		return
	}

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

// RecordInternalFailure increments the internal failure counter for debugging.
// It tracks unexpected errors in specific components with detailed reason codes
// to help diagnose systemic issues.
func RecordInternalFailure(component, reason string) {
	// Check circuit breaker first
	if IsCircuitBreakerOpen() {
		klog.V(5).Info("Circuit breaker open, skipping metric update")
		return
	}

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
