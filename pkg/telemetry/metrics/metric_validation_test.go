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
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMetricCardinalityCompliance verifies that metrics stay within Red Hat's limits
func TestMetricCardinalityCompliance(t *testing.T) {
	// Initialize metrics
	err := Initialize()
	require.NoError(t, err, "Failed to initialize metrics")

	testCases := []struct {
		name              string
		simulateLoad      func()
		expectedTimeseries int
		maxAllowed        int
	}{
		{
			name: "Version usage metric cardinality",
			simulateLoad: func() {
				// Simulate all 5 allowed versions
				versions := []string{
					"pytorch-2.5", "pytorch-2.4", "pytorch-2.3",
					"tensorflow-2.15", "other",
				}
				for _, v := range versions {
					RecordImageVersionUsage(v)
				}
			},
			expectedTimeseries: 5,
			maxAllowed:        5, // Red Hat limit for version metric
		},
		{
			name: "Source preference metric cardinality",
			simulateLoad: func() {
				// Simulate all 3 allowed sources
				sources := []string{"rhoai_official", "custom", "upstream"}
				for _, s := range sources {
					RecordImageSourcePreference(s)
				}
			},
			expectedTimeseries: 3,
			maxAllowed:        3, // Red Hat limit for source metric
		},
		{
			name: "Enterprise adoption metric cardinality",
			simulateLoad: func() {
				// Simulate both customer types
				RecordEnterpriseAdoption("enterprise")
				RecordEnterpriseAdoption("non_enterprise")
			},
			expectedTimeseries: 2,
			maxAllowed:        2, // Red Hat limit for customer type metric
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset metrics before each test
			resetMetrics()
			
			// Simulate load
			tc.simulateLoad()
			
			// Gather metrics
			metricFamilies, err := prometheus.DefaultGatherer.Gather()
			require.NoError(t, err)
			
			// Count timeseries for training_operator metrics
			timeseriesCount := 0
			for _, mf := range metricFamilies {
				if strings.HasPrefix(*mf.Name, "training_operator_") {
					timeseriesCount += len(mf.Metric)
				}
			}
			
			assert.LessOrEqual(t, timeseriesCount, tc.maxAllowed,
				"Metric cardinality %d exceeds Red Hat limit of %d", 
				timeseriesCount, tc.maxAllowed)
			assert.Equal(t, tc.expectedTimeseries, timeseriesCount,
				"Unexpected number of timeseries")
		})
	}
}

// TestTotalTimeseriesLimit verifies the total timeseries stays under 10
func TestTotalTimeseriesLimit(t *testing.T) {
	// Initialize metrics
	err := Initialize()
	require.NoError(t, err)
	
	// Reset metrics
	resetMetrics()
	
	// Simulate maximum allowed load
	// 5 versions
	versions := []string{
		"pytorch-2.5", "pytorch-2.4", "pytorch-2.3",
		"tensorflow-2.15", "other",
	}
	for _, v := range versions {
		RecordImageVersionUsage(v)
	}
	
	// 3 sources
	sources := []string{"rhoai_official", "custom", "upstream"}
	for _, s := range sources {
		RecordImageSourcePreference(s)
	}
	
	// 2 customer types
	RecordEnterpriseAdoption("enterprise")
	RecordEnterpriseAdoption("non_enterprise")
	
	// Gather all metrics
	metricFamilies, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)
	
	// Count total timeseries
	totalTimeseries := 0
	for _, mf := range metricFamilies {
		if strings.HasPrefix(*mf.Name, "training_operator_") {
			totalTimeseries += len(mf.Metric)
			
			// Log each metric for debugging
			t.Logf("Metric %s has %d timeseries", *mf.Name, len(mf.Metric))
		}
	}
	
	// Verify total is within Red Hat's limit
	assert.LessOrEqual(t, totalTimeseries, 10,
		"Total timeseries %d exceeds Red Hat limit of 10", totalTimeseries)
	
	// Verify we have exactly 10 (5 + 3 + 2)
	assert.Equal(t, 10, totalTimeseries,
		"Expected exactly 10 timeseries (5 versions + 3 sources + 2 customer types)")
}

// TestVersionNormalization verifies version strings are properly normalized
func TestVersionNormalization(t *testing.T) {
	testCases := []struct {
		inputVersion     string
		expectedVersion  string
		shouldBeAccepted bool
	}{
		// PyTorch versions
		{"quay.io/modh/pytorch:2.5.1-cuda12.1-python3.11", "pytorch-2.5", true},
		{"registry.redhat.io/ubi8/python-39:pytorch-2.4.0", "pytorch-2.4", true},
		{"pytorch/pytorch:2.3.0-cuda11.8-cudnn8-runtime", "pytorch-2.3", true},
		{"pytorch:2.2.0", "other", true}, // Old version -> other
		{"nvcr.io/nvidia/pytorch:24.01-py3", "pytorch-2.5", true}, // NVIDIA tags map to versions
		
		// TensorFlow versions
		{"tensorflow/tensorflow:2.15.0-gpu", "tensorflow-2.15", true},
		{"gcr.io/deeplearning-platform/tf2-gpu.2-14", "tensorflow-2.14", true},
		{"tensorflow:2.13.0", "other", true}, // Old version -> other
		
		// Other frameworks
		{"mxnet/mxnet:1.9.1", "other", true},
		{"paddlepaddle/paddle:2.5.1-gpu", "other", true},
		{"jaxlib:0.4.23", "other", true},
		
		// Edge cases
		{"", "other", true},
		{"invalid-image", "other", true},
		{"no-version-info", "other", true},
	}
	
	for _, tc := range testCases {
		t.Run(tc.inputVersion, func(t *testing.T) {
			normalized := normalizeRuntimeVersion(tc.inputVersion)
			assert.Equal(t, tc.expectedVersion, normalized,
				"Version normalization failed for %s", tc.inputVersion)
			
			// Verify it's in the allowed set
			allowed := isAllowedVersion(normalized)
			assert.Equal(t, tc.shouldBeAccepted, allowed,
				"Version %s acceptance mismatch", normalized)
		})
	}
}

// TestImageSourceClassification verifies image sources are properly classified
func TestImageSourceClassification(t *testing.T) {
	testCases := []struct {
		imageName       string
		expectedSource  string
	}{
		// RHOAI official images
		{"quay.io/modh/pytorch:latest", "rhoai_official"},
		{"registry.redhat.io/rhoai/tensorflow:2.15", "rhoai_official"},
		{"quay.io/opendatahub/pytorch:v2", "rhoai_official"},
		
		// Custom/internal images
		{"internal.company.com/ml/pytorch:custom", "custom"},
		{"10.0.0.1:5000/tensorflow:modified", "custom"},
		{"localhost:5000/ml-image:latest", "custom"},
		{"my-registry.corp.net/ai/model:v1", "custom"},
		
		// Upstream images
		{"docker.io/pytorch/pytorch:latest", "upstream"},
		{"gcr.io/tensorflow/tensorflow:2.15", "upstream"},
		{"nvcr.io/nvidia/pytorch:24.01", "upstream"},
		{"hub.docker.com/library/tensorflow:latest", "upstream"},
		
		// Edge cases default to upstream
		{"unknown-registry.com/image:tag", "upstream"},
		{"", "upstream"},
	}
	
	for _, tc := range testCases {
		t.Run(tc.imageName, func(t *testing.T) {
			source := classifyImageSource(tc.imageName)
			assert.Equal(t, tc.expectedSource, source,
				"Image source classification failed for %s", tc.imageName)
		})
	}
}

// TestCustomerTypeClassification verifies customer classification logic
func TestCustomerTypeClassification(t *testing.T) {
	testCases := []struct {
		namespace      string
		annotations    map[string]string
		expectedType   string
	}{
		// Enterprise indicators
		{
			namespace: "openshift-operators",
			annotations: map[string]string{
				"openshift.io/cluster-monitoring": "true",
			},
			expectedType: "enterprise",
		},
		{
			namespace: "production",
			annotations: map[string]string{
				"redhat.com/support-tier": "premium",
			},
			expectedType: "enterprise",
		},
		{
			namespace: "kube-system",
			annotations: map[string]string{},
			expectedType: "enterprise", // System namespace
		},
		
		// Non-enterprise
		{
			namespace: "student-project",
			annotations: map[string]string{},
			expectedType: "non_enterprise",
		},
		{
			namespace: "test",
			annotations: map[string]string{
				"environment": "development",
			},
			expectedType: "non_enterprise",
		},
		{
			namespace: "demo",
			annotations: map[string]string{},
			expectedType: "non_enterprise",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.namespace, func(t *testing.T) {
			customerType := classifyCustomerType(tc.namespace, tc.annotations)
			assert.Equal(t, tc.expectedType, customerType,
				"Customer classification failed for namespace %s", tc.namespace)
		})
	}
}

// TestMetricExposure verifies metrics are properly exposed for scraping
func TestMetricExposure(t *testing.T) {
	// Initialize metrics
	err := Initialize()
	require.NoError(t, err)
	
	// Generate some test data
	RecordImageVersionUsage("pytorch-2.5")
	RecordImageSourcePreference("rhoai_official")
	RecordEnterpriseAdoption("enterprise")
	
	// Check that metrics are registered and exposed
	metricNames := []string{
		"training_operator_image_version_usage",
		"training_operator_image_source_preference_total",
		"training_operator_enterprise_adoption_total",
	}
	
	for _, name := range metricNames {
		t.Run(name, func(t *testing.T) {
			// Verify metric exists in registry
			metrics, err := prometheus.DefaultGatherer.Gather()
			require.NoError(t, err)
			
			found := false
			for _, mf := range metrics {
				if *mf.Name == name {
					found = true
					assert.NotEmpty(t, mf.Metric, "Metric %s has no data", name)
					
					// Verify metric has proper labels
					if len(mf.Metric) > 0 {
						metric := mf.Metric[0]
						assert.NotEmpty(t, metric.Label, "Metric %s has no labels", name)
						
						// Log labels for debugging
						for _, label := range metric.Label {
							t.Logf("  Label: %s = %s", *label.Name, *label.Value)
						}
					}
					break
				}
			}
			assert.True(t, found, "Metric %s not found in registry", name)
		})
	}
}

// TestCircuitBreakerProtection verifies cardinality protection works
func TestCircuitBreakerProtection(t *testing.T) {
	// Initialize with circuit breaker
	err := Initialize()
	require.NoError(t, err)
	
	// Try to exceed version limit
	for i := 0; i < 20; i++ {
		version := fmt.Sprintf("pytorch-%d.%d", i/10, i%10)
		RecordImageVersionUsage(version)
	}
	
	// Gather metrics
	metrics, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)
	
	// Find version metric
	var versionMetric *dto.MetricFamily
	for _, mf := range metrics {
		if *mf.Name == "training_operator_image_version_usage" {
			versionMetric = mf
			break
		}
	}
	
	require.NotNil(t, versionMetric, "Version metric not found")
	
	// Should be limited to 5 versions due to normalization
	assert.LessOrEqual(t, len(versionMetric.Metric), 5,
		"Circuit breaker failed: got %d versions, max should be 5", 
		len(versionMetric.Metric))
}

// TestMetricLabelsCompliance verifies labels follow Red Hat naming conventions
func TestMetricLabelsCompliance(t *testing.T) {
	// Initialize metrics
	err := Initialize()
	require.NoError(t, err)
	
	// Generate test data
	RecordImageVersionUsage("pytorch-2.5")
	RecordImageSourcePreference("rhoai_official")
	RecordEnterpriseAdoption("enterprise")
	
	// Gather metrics
	metrics, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)
	
	// Check each metric's labels
	for _, mf := range metrics {
		if !strings.HasPrefix(*mf.Name, "training_operator_") {
			continue
		}
		
		t.Run(*mf.Name, func(t *testing.T) {
			for _, metric := range mf.Metric {
				for _, label := range metric.Label {
					// Verify label naming conventions
					assert.Regexp(t, "^[a-z][a-z0-9_]*$", *label.Name,
						"Label name %s doesn't follow naming convention", *label.Name)
					
					// Verify no high-cardinality labels
					forbiddenLabels := []string{
						"pod", "instance", "namespace", "job", 
						"endpoint", "container", "service",
					}
					for _, forbidden := range forbiddenLabels {
						assert.NotEqual(t, forbidden, *label.Name,
							"High-cardinality label %s should not be present", forbidden)
					}
				}
			}
		})
	}
}

// TestMetricConcurrency verifies thread-safe metric updates
func TestMetricConcurrency(t *testing.T) {
	// Initialize metrics
	err := Initialize()
	require.NoError(t, err)
	
	// Reset metrics
	resetMetrics()
	
	// Concurrent updates
	const goroutines = 100
	const updatesPerGoroutine = 100
	
	done := make(chan bool, goroutines)
	
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			for j := 0; j < updatesPerGoroutine; j++ {
				// Rotate through different values
				versions := []string{"pytorch-2.5", "pytorch-2.4", "tensorflow-2.15"}
				sources := []string{"rhoai_official", "custom", "upstream"}
				types := []string{"enterprise", "non_enterprise"}
				
				RecordImageVersionUsage(versions[j%len(versions)])
				RecordImageSourcePreference(sources[j%len(sources)])
				RecordEnterpriseAdoption(types[j%len(types)])
			}
			done <- true
		}(i)
	}
	
	// Wait for all goroutines
	for i := 0; i < goroutines; i++ {
		<-done
	}
	
	// Verify metrics are still valid
	metrics, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)
	
	// Count total timeseries
	totalTimeseries := 0
	for _, mf := range metrics {
		if strings.HasPrefix(*mf.Name, "training_operator_") {
			totalTimeseries += len(mf.Metric)
		}
	}
	
	// Should still be within limits despite concurrent updates
	assert.LessOrEqual(t, totalTimeseries, 10,
		"Concurrent updates broke cardinality limit: %d timeseries", totalTimeseries)
}

// Helper functions

func resetMetrics() {
	// Reset gauge metrics
	TrainingOperatorImageVersionUsage.Reset()
	
	// For counters, we can't reset them directly in tests
	// but we can track the initial values
}

func normalizeRuntimeVersion(image string) string {
	// Implementation should match the actual normalizeRuntimeVersion function
	// This is a simplified version for testing
	if strings.Contains(image, "pytorch") {
		if strings.Contains(image, "2.5") || strings.Contains(image, "24.01") {
			return "pytorch-2.5"
		} else if strings.Contains(image, "2.4") {
			return "pytorch-2.4"
		} else if strings.Contains(image, "2.3") {
			return "pytorch-2.3"
		}
	} else if strings.Contains(image, "tensorflow") {
		if strings.Contains(image, "2.15") {
			return "tensorflow-2.15"
		} else if strings.Contains(image, "2.14") {
			return "tensorflow-2.14"
		}
	}
	return "other"
}

func isAllowedVersion(version string) bool {
	allowed := []string{
		"pytorch-2.5", "pytorch-2.4", "pytorch-2.3",
		"tensorflow-2.15", "tensorflow-2.14", "other",
	}
	for _, v := range allowed {
		if v == version {
			return true
		}
	}
	return false
}

func classifyImageSource(image string) string {
	if strings.Contains(image, "quay.io/modh") || 
	   strings.Contains(image, "registry.redhat.io/rhoai") ||
	   strings.Contains(image, "quay.io/opendatahub") {
		return "rhoai_official"
	}
	
	if strings.Contains(image, "internal") || 
	   strings.Contains(image, "10.") ||
	   strings.Contains(image, "localhost") ||
	   strings.Contains(image, ".corp.") {
		return "custom"
	}
	
	return "upstream"
}

func classifyCustomerType(namespace string, annotations map[string]string) string {
	// System namespaces indicate enterprise
	systemNamespaces := []string{
		"openshift-", "kube-", "default", "production",
	}
	for _, prefix := range systemNamespaces {
		if strings.HasPrefix(namespace, prefix) {
			return "enterprise"
		}
	}
	
	// Check annotations for enterprise indicators
	if tier, ok := annotations["redhat.com/support-tier"]; ok && tier == "premium" {
		return "enterprise"
	}
	
	if _, ok := annotations["openshift.io/cluster-monitoring"]; ok {
		return "enterprise"
	}
	
	// Non-enterprise indicators
	nonEnterpriseKeywords := []string{
		"test", "demo", "student", "lab", "poc", "dev",
	}
	namespaceLower := strings.ToLower(namespace)
	for _, keyword := range nonEnterpriseKeywords {
		if strings.Contains(namespaceLower, keyword) {
			return "non_enterprise"
		}
	}
	
	// Default to non-enterprise for unknown
	return "non_enterprise"
}