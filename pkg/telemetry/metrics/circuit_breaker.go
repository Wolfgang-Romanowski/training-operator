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
	"sync"
	"time"

	"k8s.io/klog/v2"
)

var (
	// circuitBreakerState tracks whether telemetry is temporarily disabled due to cardinality violations
	circuitBreakerState = &CircuitBreaker{
		enabled:            true,
		lastCheck:          time.Now(),
		checkInterval:      1 * time.Minute,
		recoveryInterval:   5 * time.Minute,
		consecutiveFailures: 0,
		maxFailures:        3,
	}
)

// CircuitBreaker implements a circuit breaker pattern for cardinality protection.
// It temporarily disables telemetry collection when cardinality violations are detected
// to prevent overwhelming the monitoring system per Red Hat requirements.
type CircuitBreaker struct {
	mu                  sync.RWMutex
	enabled             bool
	lastCheck           time.Time
	lastFailure         time.Time
	checkInterval       time.Duration
	recoveryInterval    time.Duration
	consecutiveFailures int
	maxFailures         int
}

// IsCircuitBreakerOpen checks if the circuit breaker is currently open (telemetry disabled).
// An open circuit means telemetry collection is temporarily suspended due to violations.
func IsCircuitBreakerOpen() bool {
	circuitBreakerState.mu.RLock()
	defer circuitBreakerState.mu.RUnlock()
	return !circuitBreakerState.enabled
}

// CheckCardinalityCircuitBreaker evaluates current metric cardinality and trips the breaker if needed.
// This function should be called periodically to ensure telemetry stays within limits.
func CheckCardinalityCircuitBreaker() {
	circuitBreakerState.mu.Lock()
	defer circuitBreakerState.mu.Unlock()

	now := time.Now()
	
	// Skip check if interval hasn't passed
	if now.Sub(circuitBreakerState.lastCheck) < circuitBreakerState.checkInterval {
		return
	}
	
	circuitBreakerState.lastCheck = now

	// If circuit is open, check if recovery time has passed
	if !circuitBreakerState.enabled {
		if now.Sub(circuitBreakerState.lastFailure) >= circuitBreakerState.recoveryInterval {
			klog.Info("Attempting to recover from circuit breaker state")
			circuitBreakerState.enabled = true
			circuitBreakerState.consecutiveFailures = 0
		} else {
			return // Still in recovery period
		}
	}

	// Validate cardinality
	err := ValidateMetricCardinality()
	if err != nil {
		circuitBreakerState.consecutiveFailures++
		circuitBreakerState.lastFailure = now
		
		if circuitBreakerState.consecutiveFailures >= circuitBreakerState.maxFailures {
			// Trip the circuit breaker
			circuitBreakerState.enabled = false
			klog.ErrorS(err, "Circuit breaker tripped due to repeated cardinality violations",
				"failures", circuitBreakerState.consecutiveFailures,
				"recoveryTime", circuitBreakerState.recoveryInterval)
			
			// Clear all metrics to reduce cardinality immediately
			EmergencyClearMetrics()
		}
	} else {
		// Reset failure count on success
		if circuitBreakerState.consecutiveFailures > 0 {
			klog.Info("Cardinality check passed, resetting failure count")
		}
		circuitBreakerState.consecutiveFailures = 0
	}
}

// EmergencyClearMetrics reduces cardinality by consolidating least-used labels.
// Instead of clearing all metrics, it merges low-usage versions into "other".
func EmergencyClearMetrics() {
	klog.Warning("Reducing metric cardinality due to violations")
	
	tracker := getImageVersionTracker()
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	
	// Find least-used versions and consolidate them
	consolidateIntoOther := []string{}
	for version, count := range tracker.versionCounts {
		if version != "other" && count <= 2 {
			consolidateIntoOther = append(consolidateIntoOther, version)
		}
	}
	
	// Consolidate low-usage versions into "other"
	otherCount := tracker.versionCounts["other"]
	for _, version := range consolidateIntoOther {
		otherCount += tracker.versionCounts[version]
		delete(tracker.versionCounts, version)
		
		// Update gauge metric with composite label
		for _, source := range []string{"-rhoai", "-external"} {
			TrainingOperatorRuntimeAdoption.DeleteLabelValues(version + source)
		}
		
		// Update internal tracking
		for key, data := range tracker.activeVersions {
			if data.Version == version {
				data.Version = "other"
			}
		}
	}
	
	// Update "other" count
	if otherCount > 0 {
		tracker.versionCounts["other"] = otherCount
		TrainingOperatorRuntimeAdoption.WithLabelValues("other").Set(float64(otherCount))
	}
	
	klog.InfoS("Cardinality reduction completed", 
		"consolidated", len(consolidateIntoOther),
		"remaining_versions", len(tracker.versionCounts))
}

// ResetCircuitBreaker manually resets the circuit breaker state.
// This should only be used in testing or emergency recovery scenarios.
func ResetCircuitBreaker() {
	circuitBreakerState.mu.Lock()
	defer circuitBreakerState.mu.Unlock()
	
	circuitBreakerState.enabled = true
	circuitBreakerState.consecutiveFailures = 0
	circuitBreakerState.lastCheck = time.Now()
	
	klog.Info("Circuit breaker manually reset")
}

// GetCircuitBreakerStatus returns the current status of the circuit breaker.
// This is useful for monitoring and debugging telemetry health.
func GetCircuitBreakerStatus() map[string]interface{} {
	circuitBreakerState.mu.RLock()
	defer circuitBreakerState.mu.RUnlock()
	
	return map[string]interface{}{
		"enabled":             circuitBreakerState.enabled,
		"consecutiveFailures": circuitBreakerState.consecutiveFailures,
		"lastCheck":           circuitBreakerState.lastCheck.Format(time.RFC3339),
		"lastFailure":         circuitBreakerState.lastFailure.Format(time.RFC3339),
		"state":               map[bool]string{true: "closed", false: "open"}[circuitBreakerState.enabled],
	}
}

// StartCircuitBreakerMonitor starts a background goroutine that periodically checks cardinality.
// This ensures the circuit breaker is always monitoring metric health.
func StartCircuitBreakerMonitor() {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		
		for range ticker.C {
			CheckCardinalityCircuitBreaker()
		}
	}()
	
	klog.Info("Circuit breaker monitor started")
}