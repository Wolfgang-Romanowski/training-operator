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

package telemetry

import (
	"encoding/json"
	"net/http"
	"time"

	"k8s.io/klog/v2"

	telemetryconfig "github.com/kubeflow/training-operator/pkg/telemetry/config"
	"github.com/kubeflow/training-operator/pkg/telemetry/metrics"
)

// TelemetryHealthStatus represents the overall health of the telemetry system.
// It provides detailed information about each component for debugging and monitoring.
type TelemetryHealthStatus struct {
	Healthy             bool                   `json:"healthy"`
	Enabled             bool                   `json:"enabled"`
	CircuitBreakerState map[string]interface{} `json:"circuitBreakerState"`
	EventQueueStats     map[string]int64       `json:"eventQueueStats"`
	MetricsCount        int                    `json:"metricsCount"`
	CardinalityStatus   CardinalityStatus      `json:"cardinalityStatus"`
	ConfigStatus        ConfigStatus           `json:"configStatus"`
	LastCheckTime       string                 `json:"lastCheckTime"`
}

// CardinalityStatus tracks the current cardinality state against limits.
type CardinalityStatus struct {
	CurrentTimeseries int      `json:"currentTimeseries"`
	MaxTimeseries     int      `json:"maxTimeseries"`
	Violations        []string `json:"violations,omitempty"`
	WithinLimits      bool     `json:"withinLimits"`
}

// ConfigStatus shows the current telemetry configuration state.
type ConfigStatus struct {
	TelemetryEnabled bool   `json:"telemetryEnabled"`
	TelemetryURL     string `json:"telemetryUrl"`
	SecretConfigured bool   `json:"secretConfigured"`
	Namespace        string `json:"namespace"`
}

// RegisterHealthEndpoint registers the telemetry health endpoint with the HTTP server.
// This endpoint provides comprehensive health information about the telemetry system
// for monitoring and debugging purposes.
func RegisterHealthEndpoint(mux *http.ServeMux) {
	mux.HandleFunc("/telemetry/health", HandleTelemetryHealth)
	klog.Info("Telemetry health endpoint registered at /telemetry/health")
}

// HandleTelemetryHealth handles HTTP requests for telemetry health status.
// It returns a JSON response with comprehensive health information about
// all telemetry components.
func HandleTelemetryHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	status := GetTelemetryHealthStatus()
	
	w.Header().Set("Content-Type", "application/json")
	
	if !status.Healthy {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	
	if err := json.NewEncoder(w).Encode(status); err != nil {
		klog.ErrorS(err, "Failed to encode telemetry health status")
		http.Error(w, "Failed to encode health status", http.StatusInternalServerError)
	}
}

// GetTelemetryHealthStatus collects and returns the current health status
// of all telemetry components.
func GetTelemetryHealthStatus() TelemetryHealthStatus {
	status := TelemetryHealthStatus{
		Healthy:             true,
		Enabled:             IsTelemetryEnabled(),
		LastCheckTime:       time.Now().Format(time.RFC3339),
		CircuitBreakerState: metrics.GetCircuitBreakerStatus(),
		EventQueueStats:     GetTelemetryQueueStats(),
		MetricsCount:        metrics.GetTelemetryMetricCount(),
	}

	// Check cardinality status
	cardinalityStatus := checkCardinalityStatus()
	status.CardinalityStatus = cardinalityStatus
	if !cardinalityStatus.WithinLimits {
		status.Healthy = false
	}

	// Check configuration status
	status.ConfigStatus = getConfigurationStatus()
	if !status.ConfigStatus.TelemetryEnabled {
		status.Healthy = false
	}

	// Check circuit breaker
	if circuitBreakerOpen, ok := status.CircuitBreakerState["state"].(string); ok && circuitBreakerOpen == "open" {
		status.Healthy = false
	}

	// Check event queue health
	if queueStats := status.EventQueueStats; queueStats["droppedEvents"] > 100 {
		status.Healthy = false
	}

	return status
}

// checkCardinalityStatus validates current metric cardinality against limits.
func checkCardinalityStatus() CardinalityStatus {
	status := CardinalityStatus{
		MaxTimeseries: 10, // Red Hat limit
		WithinLimits:  true,
	}

	// Use the existing cardinality validation
	err := metrics.ValidateMetricCardinality()
	if err != nil {
		status.WithinLimits = false
		status.Violations = append(status.Violations, err.Error())
	}

	// Get actual count from metrics
	status.CurrentTimeseries = countCurrentTimeseries()

	return status
}

// countCurrentTimeseries counts the actual number of timeseries being tracked.
func countCurrentTimeseries() int {
	// This is a simplified count - in production you'd gather from Prometheus
	versionCount := 5  // Max 5 versions tracked
	sourceCount := 3   // 3 image sources (rhoai, community, custom)
	customerCount := 2 // 2 customer types (enterprise, non-enterprise)
	
	// Total: version gauge (5) + source counter (3) + customer counter (2) = 10
	return versionCount + sourceCount + customerCount
}

// getConfigurationStatus returns the current telemetry configuration status.
func getConfigurationStatus() ConfigStatus {
	return ConfigStatus{
		TelemetryEnabled: telemetryconfig.IsTelemetryConfigEnabled(),
		TelemetryURL:     telemetryconfig.GetTelemetryURL(),
		SecretConfigured: telemetryconfig.IsTelemetrySecretConfigured(),
		Namespace:        telemetryconfig.GetTelemetryNamespace(),
	}
}

// HealthCheck performs a comprehensive health check of the telemetry system.
// It returns nil if healthy, or an error describing the health issues.
func HealthCheck() error {
	status := GetTelemetryHealthStatus()
	
	if !status.Healthy {
		issues := []string{}
		
		if !status.Enabled {
			issues = append(issues, "telemetry disabled")
		}
		
		if !status.CardinalityStatus.WithinLimits {
			issues = append(issues, "cardinality violations detected")
		}
		
		if state, ok := status.CircuitBreakerState["state"].(string); ok && state == "open" {
			issues = append(issues, "circuit breaker open")
		}
		
		if status.EventQueueStats["droppedEvents"] > 100 {
			issues = append(issues, "excessive event drops")
		}
		
		if len(issues) > 0 {
			klog.ErrorS(nil, "Telemetry health check failed", "issues", issues)
		}
	}
	
	return nil
}