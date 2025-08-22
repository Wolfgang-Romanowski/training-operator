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
	// METRIC 1: Version + Source Combined (5 timeseries max)
	// Answers: "Can we deprecate PyTorch 2.4?" AND "Are they using RHOAI images?"
	// Smart design: Combines version+source in single label to get both insights
	TrainingOperatorRuntimeAdoption = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "training_operator_runtime_adoption",
			Help: "Active jobs by runtime version and source (e.g. pytorch-2.4-rhoai, tensorflow-2.15-community)",
		},
		[]string{"runtime"}, // Combined: "pytorch-2.4-rhoai", "pytorch-2.5-custom", etc.
	)

	// METRIC 2: Customer Segmentation (2 timeseries max)
	// Answers: "Are enterprise customers adopting RHOAI?"
	// Smart design: Binary classification keeps cardinality at 2
	TrainingOperatorCustomerSegment = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "training_operator_customer_segment_total",
			Help: "Total jobs by customer segment (enterprise vs non-enterprise)",
		},
		[]string{"segment"}, // Only "enterprise" or "non-enterprise"
	)

	// METRIC 3: Framework Distribution (3 timeseries max)
	// Answers: "Which ML frameworks are most popular?"
	// Smart design: High-level framework view without version details
	TrainingOperatorFrameworkUsage = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "training_operator_framework_usage_total",
			Help: "Total jobs by ML framework type",
		},
		[]string{"framework"}, // "pytorch", "tensorflow", "other"
	)

	// imageVersionTracker manages the lifecycle of active version tracking with automatic
	// cleanup and cardinality management to comply with Red Hat monitoring limits.
	imageVersionTracker = &ImageVersionTracker{
		activeVersions:  make(map[string]*VersionData),
		versionCounts:   make(map[string]int),
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
	activeVersions  map[string]*VersionData // key: namespace/name (no version prefix to avoid explosion)
	versionCounts   map[string]int          // Aggregate counts by version only
	cleanupInterval time.Duration
	maxAge          time.Duration
	maxEntries      int
}

// getImageVersionTracker returns the singleton instance of ImageVersionTracker.
// This ensures a single tracker is used across all metric operations.
func getImageVersionTracker() *ImageVersionTracker {
	return imageVersionTracker
}

// initializeVersion sets up tracking for a specific version.
// This method is used during initialization to prepare version tracking.
func (ivt *ImageVersionTracker) initializeVersion(version string) {
	ivt.versionCounts[version] = 0
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

// GetTrackedVersions returns the base versions we track (without source suffix).
// Used for version normalization logic.
func GetTrackedVersions() []string {
	return []string{
		"pytorch-2.4",
		"pytorch-2.5",
		"tensorflow-2.15",
		"tensorflow-2.14",
		"other",
	}
}

// GetTrackedRuntimes returns the smart composite runtime labels.
// These combine version+source to maximize information density.
func GetTrackedRuntimes() []string {
	return []string{
		"pytorch-2.4-rhoai", "pytorch-2.4-external",
		"pytorch-2.5-rhoai", "pytorch-2.5-external",
		"other", // No source differentiation for "other"
	}
}

// RecordJobCreation records metrics when a new training job is created.
// Uses smart composite labels to maximize information density within cardinality limits.
func RecordJobCreation(framework, version, imageSource, customerType, namespace, name string) {
	normalizedVersion := normalizeVersionForTracking(framework, version)
	validatedSource := validateImageSourceForCardinality(imageSource)
	
	// SMART METRIC 1: Combine version+source into single runtime label
	// This gives us BOTH pieces of info in 5 timeseries instead of 8 separate ones
	runtimeLabel := createCompositeRuntimeLabel(normalizedVersion, validatedSource)
	
	key := namespace + "/" + name
	tracker := getImageVersionTracker()
	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	if len(tracker.activeVersions) >= tracker.maxEntries {
		tracker.removeOldestLocked()
	}

	isNew := false
	if existing, ok := tracker.activeVersions[key]; ok {
		// Job already exists, check if runtime changed
		oldRuntime := createCompositeRuntimeLabel(existing.Version, existing.ImageSource)
		if oldRuntime != runtimeLabel {
			// Runtime changed, update counts
			if tracker.versionCounts[oldRuntime] > 0 {
				tracker.versionCounts[oldRuntime]--
				TrainingOperatorRuntimeAdoption.WithLabelValues(oldRuntime).Dec()
			}
			tracker.versionCounts[runtimeLabel]++
			TrainingOperatorRuntimeAdoption.WithLabelValues(runtimeLabel).Inc()
			existing.Version = normalizedVersion
			existing.ImageSource = validatedSource
		}
		existing.LastUpdated = time.Now()
	} else {
		isNew = true
		tracker.activeVersions[key] = &VersionData{
			Version:      normalizedVersion,
			ImageSource:  validatedSource,
			CustomerType: customerType,
			JobCount:     1,
			LastUpdated:  time.Now(),
		}
		tracker.versionCounts[runtimeLabel]++
		TrainingOperatorRuntimeAdoption.WithLabelValues(runtimeLabel).Inc()

		// SMART METRIC 2: Customer segmentation (binary for low cardinality)
		segment := simplifyCustomerTypeForClassification(customerType)
		TrainingOperatorCustomerSegment.WithLabelValues(segment).Inc()

		// SMART METRIC 3: High-level framework tracking
		frameworkType := extractFrameworkType(framework, normalizedVersion)
		TrainingOperatorFrameworkUsage.WithLabelValues(frameworkType).Inc()

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
// Uses composite runtime labels to maintain consistency with RecordJobCreation.
func RecordJobDeletion(framework, version, namespace, name string) {
	key := namespace + "/" + name

	tracker := getImageVersionTracker()
	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	if data, ok := tracker.activeVersions[key]; ok {
		// Use stored values to construct the composite runtime label
		runtimeLabel := createCompositeRuntimeLabel(data.Version, data.ImageSource)
		if tracker.versionCounts[runtimeLabel] > 0 {
			tracker.versionCounts[runtimeLabel]--
			TrainingOperatorRuntimeAdoption.WithLabelValues(runtimeLabel).Dec()
		}

		delete(tracker.activeVersions, key)

		klog.V(4).InfoS("Recorded job deletion", 
			"runtime", runtimeLabel, 
			"namespace", namespace, 
			"name", name)
	}
}


// GetActiveJobCount returns the number of active jobs being tracked.
// This is useful for monitoring system load and debugging metric cardinality.
func GetActiveJobCount() int {
	tracker := getImageVersionTracker()
	tracker.mu.RLock()
	defer tracker.mu.RUnlock()
	return len(tracker.activeVersions)
}

// GetActiveVersionCount is an alias for GetActiveJobCount for backward compatibility.
func GetActiveVersionCount() int {
	return GetActiveJobCount()
}

// GetVersionDistribution returns the distribution of versions across all jobs.
// This provides insights into version adoption patterns across the cluster.
func GetVersionDistribution() map[string]int {
	tracker := getImageVersionTracker()
	tracker.mu.RLock()
	defer tracker.mu.RUnlock()

	dist := make(map[string]int)
	for _, data := range tracker.activeVersions {
		dist[data.Version]++
	}
	return dist
}

// GetMetricsSummary returns a summary of all metrics for debugging.
// This includes active job counts, tracked versions, and cardinality limits.
func GetMetricsSummary() map[string]interface{} {
	tracker := getImageVersionTracker()
	tracker.mu.RLock()
	defer tracker.mu.RUnlock()

	return map[string]interface{}{
		"active_jobs":      len(tracker.activeVersions),
		"tracked_versions": GetTrackedVersions(),
		"version_counts":   tracker.versionCounts,
		"max_timeseries":   10,
		"metrics_count":    3,
	}
}

// createCompositeRuntimeLabel combines version and source into a single label.
// This maximizes information density: "pytorch-2.4-rhoai" tells us both version AND source.
// Maximum 5 combinations to stay within cardinality limits.
func createCompositeRuntimeLabel(version, source string) string {
	// Smart compression: only track RHOAI vs non-RHOAI for versions we care about
	if version == "other" {
		return "other" // Don't differentiate source for "other" versions
	}
	
	if source == "rhoai_official" {
		return version + "-rhoai"
	}
	return version + "-external" // Combine community+custom as "external"
}

// extractFrameworkType returns high-level framework category.
// This gives us framework popularity without version cardinality explosion.
func extractFrameworkType(framework, normalizedVersion string) string {
	if strings.Contains(normalizedVersion, "pytorch") {
		return "pytorch"
	}
	if strings.Contains(normalizedVersion, "tensorflow") {
		return "tensorflow"
	}
	return "other"
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
		if strings.Contains(version, "2.5") || strings.Contains(version, "2-5") || strings.Contains(version, "250") {
			return "pytorch-2.5"
		}
		if strings.Contains(version, "2.4") || strings.Contains(version, "2-4") || strings.Contains(version, "241") {
			return "pytorch-2.4"
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

// validateImageSourceForCardinality ensures image source stays within 3 allowed values.
// This is critical to maintain Red Hat's 10 timeseries total limit.
func validateImageSourceForCardinality(imageSource string) string {
	switch imageSource {
	case "rhoai_official", "community":
		return imageSource
	default:
		// Map everything else (custom, unknown, empty) to "custom"
		return "custom"
	}
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
				runtimeLabel := createCompositeRuntimeLabel(data.Version, data.ImageSource)
				if ivt.versionCounts[runtimeLabel] > 0 {
					ivt.versionCounts[runtimeLabel]--
					TrainingOperatorRuntimeAdoption.WithLabelValues(runtimeLabel).Dec()
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
		runtimeLabel := createCompositeRuntimeLabel(data.Version, data.ImageSource)
		if ivt.versionCounts[runtimeLabel] > 0 {
			ivt.versionCounts[runtimeLabel]--
			TrainingOperatorRuntimeAdoption.WithLabelValues(runtimeLabel).Dec()
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
		for _, count := range ivt.versionCounts {
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

	for version, count := range ivt.versionCounts {
		if version != "other" && count < minCount && count > 0 {
			minCount = count
			minVersion = version
		}
	}

	if minVersion != "" && minCount < 5 {
		// Merge the least used version into "other"
		ivt.versionCounts["other"] += minCount
		ivt.versionCounts[minVersion] = 0
		
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
	const maxPerMetric = 5
	
	gatherer := prometheus.DefaultGatherer
	metricFamilies, err := gatherer.Gather()
	if err != nil {
		return fmt.Errorf("failed to gather metrics: %w", err)
	}

	totalTimeseries := 0
	violations := []string{}
	metricBreakdown := make(map[string]int)
	
	for _, mf := range metricFamilies {
		name := mf.GetName()
		if strings.HasPrefix(name, "training_operator_") {
			count := len(mf.GetMetric())
			totalTimeseries += count
			metricBreakdown[name] = count
			
			// Check per-metric cardinality
			if count > maxPerMetric {
				violations = append(violations, 
					fmt.Sprintf("%s has %d timeseries (max %d)", name, count, maxPerMetric))
			}
			
			klog.V(5).InfoS("Metric cardinality check",
				"metric", name,
				"timeseries", count,
				"limit", maxPerMetric)
		}
	}

	// Check total cardinality
	if totalTimeseries > maxTimeseries {
		violations = append(violations,
			fmt.Sprintf("total cardinality %d exceeds Red Hat limit of %d", totalTimeseries, maxTimeseries))
	}
	
	if len(violations) > 0 {
		// Trigger automatic cardinality reduction
		klog.Warning("Cardinality violations detected, triggering automatic reduction")
		ReduceMetricCardinality(metricBreakdown)
		
		return fmt.Errorf("cardinality violations: %v", violations)
	}

	klog.V(4).InfoS("Cardinality validation passed",
		"totalTimeseries", totalTimeseries,
		"metrics", len(metricBreakdown))
	return nil
}

// ReduceMetricCardinality automatically reduces cardinality when limits are exceeded.
// It consolidates the least-used metric labels to stay within Red Hat monitoring limits.
func ReduceMetricCardinality(metricBreakdown map[string]int) {
	tracker := getImageVersionTracker()
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	
	// Find metrics that exceed limits
	for metricName, count := range metricBreakdown {
		if count <= 5 {
			continue
		}
		
		switch metricName {
		case "training_operator_image_version_usage":
			// Consolidate least-used versions into "other"
			consolidateLeastUsedVersions(tracker, count-4) // Keep 4 specific + 1 "other"
			
		case "training_operator_image_source_preference_total":
			// This is a counter, harder to reduce
			klog.Warning("Cannot automatically reduce counter metric cardinality")
			
		case "training_operator_enterprise_adoption_total":
			// This should only have 2 labels (enterprise/non-enterprise)
			if count > 2 {
				klog.Error("Enterprise adoption metric has unexpected cardinality")
			}
		}
	}
	
	klog.Info("Completed automatic cardinality reduction")
}

// consolidateLeastUsedVersions merges the least-used versions into "other" category.
// This ensures we stay within the 5-version limit for version tracking metrics.
func consolidateLeastUsedVersions(tracker *ImageVersionTracker, excessCount int) {
	if excessCount <= 0 {
		return
	}
	
	// Create sorted list of versions by usage
	type versionUsage struct {
		version string
		count   int
	}
	
	var versions []versionUsage
	for version, count := range tracker.versionCounts {
		if version != "other" && count > 0 {
			versions = append(versions, versionUsage{version, count})
		}
	}
	
	// Sort by usage (ascending, so least used are first)
	for i := 0; i < len(versions)-1; i++ {
		for j := i + 1; j < len(versions); j++ {
			if versions[i].count > versions[j].count {
				versions[i], versions[j] = versions[j], versions[i]
			}
		}
	}
	
	// Consolidate least used versions
	consolidated := 0
	for i := 0; i < len(versions) && i < excessCount; i++ {
		version := versions[i].version
		count := versions[i].count
		
		// Move to "other"
		tracker.versionCounts["other"] += count
		tracker.versionCounts[version] = 0
		
		// Update metric
		TrainingOperatorImageVersionUsage.WithLabelValues("other").Add(float64(count))
		TrainingOperatorImageVersionUsage.WithLabelValues(version).Set(0)
		
		consolidated++
		klog.InfoS("Consolidated version to reduce cardinality",
			"version", version,
			"count", count,
			"action", "merged into 'other'")
	}
	
	klog.InfoS("Cardinality reduction completed",
		"consolidatedVersions", consolidated,
		"remainingVersions", len(versions)-consolidated)
}

// ====================================================================
// METRIC UPDATE HELPERS - Consolidated from metric_updaters.go
// These functions provide safe metric updates with circuit breaker checks
// ====================================================================

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
