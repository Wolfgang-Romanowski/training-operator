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
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

// TestTelemetryCardinalityCompliance verifies that our metrics implementation
// stays within Red Hat's 10 timeseries limit as required by RHOAISTRAT-575
func TestTelemetryCardinalityCompliance(t *testing.T) {
	// Initialize metrics
	err := Initialize()
	assert.NoError(t, err, "Failed to initialize metrics")

	// Count timeseries for each metric
	tests := []struct {
		name            string
		metric          prometheus.Collector
		expectedLabels  []string
		maxCardinality  int
	}{
		{
			name:           "RuntimeAdoption",
			metric:         TrainingOperatorRuntimeAdoption,
			expectedLabels: []string{"runtime"},
			maxCardinality: 5, // 5 runtime combinations max
		},
		{
			name:           "CustomerSegment",
			metric:         TrainingOperatorCustomerSegment,
			expectedLabels: []string{"segment"},
			maxCardinality: 2, // enterprise vs non-enterprise
		},
		{
			name:           "FrameworkUsage",
			metric:         TrainingOperatorFrameworkUsage,
			expectedLabels: []string{"framework"},
			maxCardinality: 3, // pytorch, tensorflow, other
		},
	}

	totalCardinality := 0
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get metric families
			metricFamilies, err := prometheus.DefaultGatherer.Gather()
			assert.NoError(t, err)

			// Find our metric
			var found bool
			for _, mf := range metricFamilies {
				if mf.GetName() == getMetricName(tt.metric) {
					found = true
					cardinality := len(mf.GetMetric())
					assert.LessOrEqual(t, cardinality, tt.maxCardinality,
						"Metric %s exceeds cardinality limit: got %d, max %d",
						tt.name, cardinality, tt.maxCardinality)
					totalCardinality += cardinality
					break
				}
			}
			assert.True(t, found, "Metric %s not found in registry", tt.name)
		})
	}

	// Verify total cardinality
	assert.LessOrEqual(t, totalCardinality, 10,
		"Total cardinality exceeds Red Hat limit: got %d, max 10", totalCardinality)
	t.Logf("Total telemetry cardinality: %d/10 timeseries", totalCardinality)
}

// TestBusinessQuestionsAnswered verifies that our smart composite metrics
// can still answer all required business questions despite cardinality limits
func TestBusinessQuestionsAnswered(t *testing.T) {
	// Initialize metrics
	err := Initialize()
	assert.NoError(t, err)

	// Reset metrics for clean test
	TrainingOperatorRuntimeAdoption.Reset()
	TrainingOperatorCustomerSegment.Reset()
	TrainingOperatorFrameworkUsage.Reset()

	// Simulate real-world data
	simulateRealWorldUsage()

	// Test Question 1: Can we deprecate PyTorch 2.4?
	t.Run("DeprecationAnalysis", func(t *testing.T) {
		// Check we can extract version info from runtime labels
		pytorch24RHOAICount := testutil.ToFloat64(
			TrainingOperatorRuntimeAdoption.WithLabelValues("pytorch-2.4-rhoai"))
		pytorch24ExternalCount := testutil.ToFloat64(
			TrainingOperatorRuntimeAdoption.WithLabelValues("pytorch-2.4-external"))

		totalPyTorch24 := pytorch24RHOAICount + pytorch24ExternalCount
		t.Logf("PyTorch 2.4 total usage: %.0f jobs", totalPyTorch24)

		// Verify we can make deprecation decisions
		assert.Greater(t, totalPyTorch24, 0.0, "Should track PyTorch 2.4 usage")
	})

	// Test Question 2: What % use RHOAI images?
	t.Run("RHOAIAdoptionRate", func(t *testing.T) {
		// Calculate RHOAI adoption from runtime labels
		var rhoaiTotal, externalTotal float64

		rhoaiTotal += testutil.ToFloat64(
			TrainingOperatorRuntimeAdoption.WithLabelValues("pytorch-2.4-rhoai"))
		rhoaiTotal += testutil.ToFloat64(
			TrainingOperatorRuntimeAdoption.WithLabelValues("pytorch-2.5-rhoai"))

		externalTotal += testutil.ToFloat64(
			TrainingOperatorRuntimeAdoption.WithLabelValues("pytorch-2.4-external"))
		externalTotal += testutil.ToFloat64(
			TrainingOperatorRuntimeAdoption.WithLabelValues("pytorch-2.5-external"))

		total := rhoaiTotal + externalTotal
		if total > 0 {
			adoptionRate := (rhoaiTotal / total) * 100
			t.Logf("RHOAI adoption rate: %.1f%%", adoptionRate)
			assert.GreaterOrEqual(t, adoptionRate, 0.0)
			assert.LessOrEqual(t, adoptionRate, 100.0)
		}
	})

	// Test Question 3: Enterprise vs Non-Enterprise split
	t.Run("CustomerSegmentation", func(t *testing.T) {
		enterpriseCount := testutil.ToFloat64(
			TrainingOperatorCustomerSegment.WithLabelValues("enterprise"))
		nonEnterpriseCount := testutil.ToFloat64(
			TrainingOperatorCustomerSegment.WithLabelValues("non-enterprise"))

		total := enterpriseCount + nonEnterpriseCount
		if total > 0 {
			enterprisePercentage := (enterpriseCount / total) * 100
			t.Logf("Enterprise percentage: %.1f%%", enterprisePercentage)
			assert.GreaterOrEqual(t, enterprisePercentage, 0.0)
			assert.LessOrEqual(t, enterprisePercentage, 100.0)
		}
	})

	// Test Question 4: Framework distribution
	t.Run("FrameworkDistribution", func(t *testing.T) {
		pytorchCount := testutil.ToFloat64(
			TrainingOperatorFrameworkUsage.WithLabelValues("pytorch"))
		tensorflowCount := testutil.ToFloat64(
			TrainingOperatorFrameworkUsage.WithLabelValues("tensorflow"))
		otherCount := testutil.ToFloat64(
			TrainingOperatorFrameworkUsage.WithLabelValues("other"))

		total := pytorchCount + tensorflowCount + otherCount
		t.Logf("Framework distribution - PyTorch: %.0f, TensorFlow: %.0f, Other: %.0f",
			pytorchCount, tensorflowCount, otherCount)
		assert.Greater(t, total, 0.0, "Should track framework usage")
	})
}

// TestSmartCompositeMetrics verifies that our composite labels work correctly
func TestSmartCompositeMetrics(t *testing.T) {
	err := Initialize()
	assert.NoError(t, err)

	// Test that runtime labels combine version+source intelligently
	t.Run("RuntimeCompositeLabels", func(t *testing.T) {
		// Set specific values
		TrainingOperatorRuntimeAdoption.WithLabelValues("pytorch-2.4-rhoai").Set(10)
		TrainingOperatorRuntimeAdoption.WithLabelValues("pytorch-2.4-external").Set(5)
		TrainingOperatorRuntimeAdoption.WithLabelValues("pytorch-2.5-rhoai").Set(8)
		TrainingOperatorRuntimeAdoption.WithLabelValues("pytorch-2.5-external").Set(3)
		TrainingOperatorRuntimeAdoption.WithLabelValues("other").Set(2)

		// Verify we can extract both dimensions from the composite label
		metrics, err := prometheus.DefaultGatherer.Gather()
		assert.NoError(t, err)

		for _, mf := range metrics {
			if mf.GetName() == "training_operator_runtime_adoption" {
				assert.Equal(t, 5, len(mf.GetMetric()),
					"Should have exactly 5 runtime combinations")

				// Verify each composite label contains both version and source info
				for _, m := range mf.GetMetric() {
					for _, lp := range m.GetLabel() {
						if lp.GetName() == "runtime" {
							runtime := lp.GetValue()
							// Verify format: version-source or "other"
							assert.Regexp(t, 
								`^(pytorch-2\.[45]-(rhoai|external)|tensorflow-2\.1[45]-(rhoai|external)|other)$`,
								runtime, "Invalid runtime composite label format")
						}
					}
				}
				break
			}
		}
	})
}

// TestCardinalityProtection verifies circuit breaker works correctly
func TestCardinalityProtection(t *testing.T) {
	err := Initialize()
	assert.NoError(t, err)

	tracker := getImageVersionTracker()

	// Simulate adding many jobs to trigger cardinality protection
	t.Run("CircuitBreakerConsolidation", func(t *testing.T) {
		// Add jobs with various versions
		for i := 0; i < 20; i++ {
			jobID := fmt.Sprintf("ns-%d/job-%d", i, i)
			version := fmt.Sprintf("pytorch-2.%d", i)
			source := "external"
			
			tracker.trackImageVersion(jobID, version, source)
		}

		// Check that emergency clearing consolidates to "other"
		if tracker.getTotalCardinality() > 10 {
			EmergencyClearMetrics()
			
			// Verify data was consolidated, not lost
			metrics, _ := prometheus.DefaultGatherer.Gather()
			for _, mf := range metrics {
				if mf.GetName() == "training_operator_runtime_adoption" {
					// Should have consolidated excess into "other"
					var foundOther bool
					for _, m := range mf.GetMetric() {
						for _, lp := range m.GetLabel() {
							if lp.GetName() == "runtime" && lp.GetValue() == "other" {
								foundOther = true
								assert.Greater(t, m.GetGauge().GetValue(), 0.0,
									"'other' should contain consolidated data")
							}
						}
					}
					assert.True(t, foundOther, "Should have 'other' category after consolidation")
				}
			}
		}
	})
}

// simulateRealWorldUsage adds realistic test data
func simulateRealWorldUsage() {
	// Simulate RHOAI-heavy usage (60% RHOAI, 40% external)
	TrainingOperatorRuntimeAdoption.WithLabelValues("pytorch-2.4-rhoai").Set(15)
	TrainingOperatorRuntimeAdoption.WithLabelValues("pytorch-2.4-external").Set(8)
	TrainingOperatorRuntimeAdoption.WithLabelValues("pytorch-2.5-rhoai").Set(12)
	TrainingOperatorRuntimeAdoption.WithLabelValues("pytorch-2.5-external").Set(5)
	TrainingOperatorRuntimeAdoption.WithLabelValues("other").Set(3)

	// Simulate enterprise-heavy usage (70% enterprise)
	TrainingOperatorCustomerSegment.WithLabelValues("enterprise").Add(70)
	TrainingOperatorCustomerSegment.WithLabelValues("non-enterprise").Add(30)

	// Simulate PyTorch dominance (80% PyTorch)
	TrainingOperatorFrameworkUsage.WithLabelValues("pytorch").Add(80)
	TrainingOperatorFrameworkUsage.WithLabelValues("tensorflow").Add(15)
	TrainingOperatorFrameworkUsage.WithLabelValues("other").Add(5)
}

// getMetricName extracts the metric name from a collector
func getMetricName(c prometheus.Collector) string {
	// This is a simplified helper - in production use metric.Desc().String()
	switch c {
	case TrainingOperatorRuntimeAdoption:
		return "training_operator_runtime_adoption"
	case TrainingOperatorCustomerSegment:
		return "training_operator_customer_segment_total"
	case TrainingOperatorFrameworkUsage:
		return "training_operator_framework_usage_total"
	default:
		return ""
	}
}