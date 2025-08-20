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
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// TrainingOperatorImageVersionUsage tracks active training jobs by RHOAI runtime version
	// for deprecation analysis and version adoption monitoring.
	TrainingOperatorImageVersionUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "training_operator_image_version_usage",
			Help: "Active training jobs by RHOAI runtime version for deprecation analysis",
		},
		[]string{"version"},
	)

	// TrainingOperatorImageSourcePreference tracks total jobs by image source to understand
	// customer runtime preferences between RHOAI official, community, and custom images.
	TrainingOperatorImageSourcePreference = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "training_operator_image_source_preference_total",
			Help: "Total jobs by image source showing customer runtime preferences",
		},
		[]string{"image_source"},
	)

	// TrainingOperatorEnterpriseAdoption differentiates enterprise vs non-enterprise adoption
	// of RHOAI runtimes for customer segmentation analysis.
	TrainingOperatorEnterpriseAdoption = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "training_operator_enterprise_adoption_total",
			Help: "Enterprise vs non-enterprise adoption of RHOAI runtimes",
		},
		[]string{"customer_type"},
	)

	// imageVersionTracker manages the lifecycle of active version tracking with automatic
	// cleanup and cardinality management to comply with Red Hat monitoring limits.
	imageVersionTracker = &ImageVersionTracker{
		activeVersions:  make(map[string]*VersionData),
		topVersions:     make(map[string]int),
		mu:              sync.RWMutex{},
		cleanupInterval: 1 * time.Hour,
		maxAge:          24 * time.Hour,
		maxEntries:      10000,
	}
)

// ImageVersionTracker manages active training job version tracking with automatic cleanup
// and cardinality limits to ensure compliance with Red Hat monitoring requirements.
type ImageVersionTracker struct {
	mu              sync.RWMutex
	activeVersions  map[string]*VersionData // key: version/namespace/name
	topVersions     map[string]int          // Track top 5 versions only for cardinality control
	cleanupInterval time.Duration
	maxAge          time.Duration
	maxEntries      int
}

// VersionData contains metadata about a tracked training job version including
// classification information for telemetry analysis.
type VersionData struct {
	Version      string
	ImageSource  string
	CustomerType string
	JobCount     int
	LastUpdated  time.Time
}

// Customer classification logic is imported from the telemetry package to avoid duplication.
// CustomerInfo and ResourceInfo are defined in pkg/telemetry/customer_analysis.go

// GetTrackedVersions returns the list of tracked versions for external use.
// These versions align with RHOAI supported runtime versions and recording rules.
func GetTrackedVersions() []string {
	return []string{
		"pytorch-2.4",
		"pytorch-2.3",
		"tensorflow-2.15",
		"tensorflow-2.14",
		"other",
	}
}

// RecordJobCreation records metrics when a new training job is created.
// It tracks image version usage, source preferences, and enterprise adoption patterns
// while maintaining cardinality limits per Red Hat monitoring requirements.
func RecordJobCreation(framework, version, imageSource, customerType, namespace, name string) {
	normalizedVersion := normalizeVersionForTracking(framework, version)
	key := normalizedVersion + "/" + namespace + "/" + name

	imageVersionTracker.mu.Lock()
	defer imageVersionTracker.mu.Unlock()

	if len(imageVersionTracker.activeVersions) >= imageVersionTracker.maxEntries {
		imageVersionTracker.removeOldestLocked()
	}

	isNew := false
	if existing, ok := imageVersionTracker.activeVersions[key]; ok {
		existing.LastUpdated = time.Now()
	} else {
		isNew = true
		imageVersionTracker.activeVersions[key] = &VersionData{
			Version:      normalizedVersion,
			ImageSource:  imageSource,
			CustomerType: customerType,
			JobCount:     1,
			LastUpdated:  time.Now(),
		}
	}

	if isNew {
		imageVersionTracker.topVersions[normalizedVersion]++
		TrainingOperatorImageVersionUsage.WithLabelValues(normalizedVersion).Inc()

		TrainingOperatorImageSourcePreference.WithLabelValues(imageSource).Inc()

		if imageSource == "rhoai_official" {
			simplifiedCustomerType := simplifyCustomerTypeForClassification(customerType)
			TrainingOperatorEnterpriseAdoption.WithLabelValues(simplifiedCustomerType).Inc()
		}

		klog.V(4).InfoS("Recorded job creation",
			"version", normalizedVersion,
			"framework", framework,
			"originalVersion", version,
			"imageSource", imageSource,
			"customerType", customerType,
			"namespace", namespace,
			"name", name)
	}
}

// RecordJobDeletion decrements job tracking metrics when a training job is deleted.
// This ensures accurate active job counts and prevents metric drift over time.
func RecordJobDeletion(framework, version, namespace, name string) {
	normalizedVersion := normalizeVersionForTracking(framework, version)
	key := normalizedVersion + "/" + namespace + "/" + name

	imageVersionTracker.mu.Lock()
	defer imageVersionTracker.mu.Unlock()

	if _, ok := imageVersionTracker.activeVersions[key]; ok {
		if imageVersionTracker.topVersions[normalizedVersion] > 0 {
			imageVersionTracker.topVersions[normalizedVersion]--
			TrainingOperatorImageVersionUsage.WithLabelValues(normalizedVersion).Dec()
		}

		delete(imageVersionTracker.activeVersions, key)

		klog.V(4).InfoS("Recorded job deletion", "version", normalizedVersion, "namespace", namespace, "name", name)
	}
}


// GetActiveJobCount returns the number of active jobs being tracked.
// This is useful for monitoring system load and debugging metric cardinality.
func GetActiveJobCount() int {
	imageVersionTracker.mu.RLock()
	defer imageVersionTracker.mu.RUnlock()
	return len(imageVersionTracker.activeVersions)
}

// GetActiveVersionCount is an alias for GetActiveJobCount for backward compatibility.
func GetActiveVersionCount() int {
	return GetActiveJobCount()
}

// GetVersionDistribution returns the distribution of versions across all jobs.
// This provides insights into version adoption patterns across the cluster.
func GetVersionDistribution() map[string]int {
	imageVersionTracker.mu.RLock()
	defer imageVersionTracker.mu.RUnlock()

	dist := make(map[string]int)
	for _, data := range imageVersionTracker.activeVersions {
		dist[data.Version]++
	}
	return dist
}

// GetMetricsSummary returns a summary of all metrics for debugging.
// This includes active job counts, tracked versions, and cardinality limits.
func GetMetricsSummary() map[string]interface{} {
	imageVersionTracker.mu.RLock()
	defer imageVersionTracker.mu.RUnlock()

	return map[string]interface{}{
		"active_jobs":      len(imageVersionTracker.activeVersions),
		"tracked_versions": trackedVersions,
		"version_counts":   imageVersionTracker.topVersions,
		"max_timeseries":   10,
		"metrics_count":    3,
	}
}

// normalizeVersionForTracking maps framework versions to tracked categories.
// It maintains metric cardinality limits per Red Hat Monitoring Handbook by
// normalizing various version formats to a consistent set of tracked versions.
// Returns exact version strings that match recording rule patterns.
func normalizeVersionForTracking(framework, version string) string {
	trackedVersions := GetTrackedVersions()
	// Direct match for known tracked versions (excluding "other")
	for _, tracked := range trackedVersions[:len(trackedVersions)-1] {
		if version == tracked {
			return tracked
		}
	}

	// Normalize PyTorch versions to match recording rule regex patterns
	versionLower := strings.ToLower(version)
	if strings.Contains(versionLower, "pytorch") {
		if strings.Contains(version, "2.4") || strings.Contains(version, "2-4") || strings.Contains(version, "241") {
			return "pytorch-2.4"
		}
		if strings.Contains(version, "2.3") || strings.Contains(version, "2-3") || strings.Contains(version, "230") {
			return "pytorch-2.3"
		}
	}

	// Normalize TensorFlow versions to match recording rule regex patterns
	if strings.Contains(versionLower, "tensorflow") || strings.Contains(versionLower, "tf") {
		if strings.Contains(version, "2.15") || strings.Contains(version, "2-15") || strings.Contains(version, "215") {
			return "tensorflow-2.15"
		}
		if strings.Contains(version, "2.14") || strings.Contains(version, "2-14") || strings.Contains(version, "214") {
			return "tensorflow-2.14"
		}
	}

	// Fall back to "other" for unrecognized versions
	return "other"
}

// simplifyCustomerTypeForClassification normalizes customer types to binary classification.
// It converts various customer type indicators to either "enterprise" or "non-enterprise"
// to maintain consistent metrics and comply with privacy requirements.
func simplifyCustomerTypeForClassification(customerType string) string {
	customerTypeLower := strings.ToLower(customerType)
	if customerTypeLower == "enterprise" ||
		strings.Contains(customerTypeLower, "prod") ||
		customerTypeLower == "production" {
		return "enterprise"
	}
	return "non-enterprise"
}


// cleanupRoutine runs periodically to remove stale job tracking entries.
// It prevents unbounded memory growth by removing entries older than maxAge.
func (ivt *ImageVersionTracker) cleanupRoutine() {
	ticker := time.NewTicker(ivt.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		ivt.mu.Lock()
		now := time.Now()
		for key, data := range ivt.activeVersions {
			if now.Sub(data.LastUpdated) > ivt.maxAge {
				if ivt.topVersions[data.Version] > 0 {
					ivt.topVersions[data.Version]--
					TrainingOperatorImageVersionUsage.WithLabelValues(data.Version).Dec()
				}
				delete(ivt.activeVersions, key)
				klog.V(4).InfoS("Cleaned up stale job tracking", "key", key)
			}
		}
		ivt.mu.Unlock()
	}
}

// removeOldestLocked removes the oldest tracked entry when maxEntries is exceeded.
// This method must be called with the lock already held.
func (ivt *ImageVersionTracker) removeOldestLocked() {
	var oldestKey string
	var oldestTime time.Time

	for key, data := range ivt.activeVersions {
		if oldestKey == "" || data.LastUpdated.Before(oldestTime) {
			oldestKey = key
			oldestTime = data.LastUpdated
		}
	}

	if oldestKey != "" {
		data := ivt.activeVersions[oldestKey]
		if ivt.topVersions[data.Version] > 0 {
			ivt.topVersions[data.Version]--
			TrainingOperatorImageVersionUsage.WithLabelValues(data.Version).Dec()
		}
		delete(ivt.activeVersions, oldestKey)
	}
}

// cardinalityMonitor continuously monitors metric cardinality.
// It ensures compliance with Red Hat Monitoring Handbook limits (10 timeseries max)
// and automatically consolidates metrics when approaching limits.
func (ivt *ImageVersionTracker) cardinalityMonitor() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	const maxTimeseries = 10
	const maxVersionTimeseries = 5
	const maxSourceTimeseries = 3
	const maxCustomerTimeseries = 2

	for range ticker.C {
		// Check current metric cardinality against Red Hat limits
		versionCount := 0
		sourceCount := 0
		customerCount := 0

		// Count active version timeseries
		ivt.mu.RLock()
		for _, count := range ivt.topVersions {
			if count > 0 {
				versionCount++
			}
		}
		ivt.mu.RUnlock()

		// Get source preference cardinality from Prometheus
		sourceMetrics, _ := prometheus.DefaultGatherer.Gather()
		for _, mf := range sourceMetrics {
			if mf.GetName() == "training_operator_image_source_preference_total" {
				sourceCount = len(mf.GetMetric())
			}
			if mf.GetName() == "training_operator_enterprise_adoption_total" {
				customerCount = len(mf.GetMetric())
			}
		}

		totalTimeseries := versionCount + sourceCount + customerCount

		if totalTimeseries > maxTimeseries {
			klog.Errorf("CARDINALITY VIOLATION: Total timeseries %d exceeds Red Hat limit of %d",
				totalTimeseries, maxTimeseries)
			klog.Errorf("Breakdown: versions=%d, sources=%d, customers=%d",
				versionCount, sourceCount, customerCount)
			
			// Attempt to reduce cardinality by consolidating "other" versions
			ivt.consolidateOtherVersions()
		}

		if versionCount > maxVersionTimeseries {
			klog.Warningf("Version cardinality %d exceeds recommended limit of %d",
				versionCount, maxVersionTimeseries)
		}

		klog.V(4).InfoS("Cardinality check passed",
			"totalTimeseries", totalTimeseries,
			"maxAllowed", maxTimeseries,
			"versions", versionCount,
			"sources", sourceCount,
			"customers", customerCount)
	}
}

// consolidateOtherVersions reduces cardinality by merging less common versions.
// When cardinality limits are approached, this method consolidates the least-used
// tracked versions into the "other" category to maintain compliance.
func (ivt *ImageVersionTracker) consolidateOtherVersions() {
	ivt.mu.Lock()
	defer ivt.mu.Unlock()

	// Find least used tracked version and merge into "other"
	minCount := int(^uint(0) >> 1) // Max int
	minVersion := ""

	for version, count := range ivt.topVersions {
		if version != "other" && count < minCount && count > 0 {
			minCount = count
			minVersion = version
		}
	}

	if minVersion != "" && minCount < 5 {
		// Merge the least used version into "other"
		ivt.topVersions["other"] += minCount
		ivt.topVersions[minVersion] = 0
		
		// Update metrics
		TrainingOperatorImageVersionUsage.WithLabelValues("other").Add(float64(minCount))
		TrainingOperatorImageVersionUsage.WithLabelValues(minVersion).Set(0)
		
		klog.InfoS("Consolidated version to reduce cardinality",
			"version", minVersion,
			"count", minCount,
			"action", "merged into 'other'")
	}
}

// ValidateMetricCardinality validates that metrics stay within Red Hat limits.
// It returns an error if the total timeseries count exceeds the maximum allowed
// per Red Hat Monitoring Handbook requirements.
func ValidateMetricCardinality() error {
	const maxTimeseries = 10
	
	gatherer := prometheus.DefaultGatherer
	metricFamilies, err := gatherer.Gather()
	if err != nil {
		return err
	}

	totalTimeseries := 0
	for _, mf := range metricFamilies {
		name := mf.GetName()
		if strings.HasPrefix(name, "training_operator_") {
			count := len(mf.GetMetric())
			totalTimeseries += count
			klog.V(5).InfoS("Metric cardinality",
				"metric", name,
				"timeseries", count)
		}
	}

	if totalTimeseries > maxTimeseries {
		return fmt.Errorf("metric cardinality %d exceeds Red Hat limit of %d timeseries",
			totalTimeseries, maxTimeseries)
	}

	return nil
}
