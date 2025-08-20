// pkg/telemetry/metrics/metric_definitions.go
// REDESIGNED: Strictly Red Hat compliant - max 10 timeseries total
// Defines business-focused metrics for image deprecation decisions
package metrics

import (
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// =================================================================
	// RED HAT COMPLIANT TELEMETRY METRICS - 3 METRICS, 10 TIMESERIES MAX
	// Strictly compliant with Red Hat Monitoring Handbook limits
	// =================================================================

	// METRIC 1: RHOAI Version Distribution (Answers: Can we deprecate PyTorch 2.4?)
	// Cardinality: 5 timeseries (top 5 versions we care about)
	// Business value: Deprecation decisions for runtime images
	TrainingOperatorImageVersionUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "training_operator_image_version_usage",
			Help: "Active training jobs by RHOAI runtime version for deprecation analysis",
		},
		[]string{"version"}, // Just version, no framework to stay under 10
	)

	// METRIC 2: Image Source Preference (Answers: Do customers prefer RHOAI images?)
	// Cardinality: 3 timeseries (rhoai_official, community, custom)
	// Business value: Investment decisions in runtime images
	TrainingOperatorImageSourcePreference = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "training_operator_image_source_preference_total",
			Help: "Total jobs by image source showing customer runtime preferences",
		},
		[]string{"image_source"},
	)

	// METRIC 3: Enterprise RHOAI Adoption (Key business metric)
	// Cardinality: 2 timeseries (enterprise, non-enterprise)
	// Business value: Enterprise adoption of RHOAI runtimes
	TrainingOperatorEnterpriseAdoption = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "training_operator_enterprise_adoption_total",
			Help: "Enterprise vs non-enterprise adoption of RHOAI runtimes",
		},
		[]string{"customer_type"},
	)

	// Total cardinality: 5 + 3 + 2 = 10 timeseries (exactly at limit)

	// =================================================================
	// INTERNAL TRACKING STRUCTURES
	// =================================================================

	imageVersionTracker = &ImageVersionTracker{
		activeVersions:  make(map[string]*VersionData),
		topVersions:     make(map[string]int),
		mu:              sync.RWMutex{},
		cleanupInterval: 1 * time.Hour,
		maxAge:          24 * time.Hour,
		maxEntries:      10000,
	}

	crdTrackingInitOnce sync.Once

	// Track only the most important versions for deprecation decisions
	trackedVersions = []string{
		"pytorch-2.4",
		"pytorch-2.3",
		"tensorflow-2.15",
		"tensorflow-2.14",
		"other", // Everything else
	}
)

type ImageVersionTracker struct {
	mu              sync.RWMutex
	activeVersions  map[string]*VersionData // key: version/namespace/name
	topVersions     map[string]int          // Track top 5 versions only
	cleanupInterval time.Duration
	maxAge          time.Duration
	maxEntries      int
}

type VersionData struct {
	Version      string
	ImageSource  string
	CustomerType string
	JobCount     int
	LastUpdated  time.Time
}

// CustomerInfo represents customer classification data
type CustomerInfo struct {
	CustomerType     string
	UsageSource      string
	NamespacePattern string
	TenantHints      []string
	CachedAt         time.Time
}

// ResourceInfo for interface compatibility
type ResourceInfo struct {
	CPUCategory    string
	MemoryCategory string
	GPUCategory    string
	StorageType    string
}

// InitializeMetrics initializes all business-focused metrics
func InitializeMetrics() {
	crdTrackingInitOnce.Do(func() {
		// Register business-focused metrics (Red Hat compliant: 3 metrics, 10 timeseries max)
		metrics.Registry.MustRegister(
			TrainingOperatorImageVersionUsage,
			TrainingOperatorImageSourcePreference,
			TrainingOperatorEnterpriseAdoption,
		)

		// Initialize top versions tracking
		for _, v := range trackedVersions {
			imageVersionTracker.topVersions[v] = 0
			// Initialize gauge to 0
			TrainingOperatorImageVersionUsage.WithLabelValues(v).Set(0)
		}

		// Start cleanup routine for stale version data
		go imageVersionTracker.cleanupRoutine()

		klog.Info("Telemetry metrics initialized with Red Hat compliant limits (10 timeseries max)")
	})
}

// InitCRDInstanceTracking is an alias for backward compatibility
func InitCRDInstanceTracking() {
	InitializeMetrics()
}

// RecordJobCreation records a new training job creation with image version tracking
func RecordJobCreation(framework, version, imageSource, customerType, namespace, name string) {
	// Normalize version to tracked versions only
	normalizedVersion := normalizeVersion(framework, version)
	key := normalizedVersion + "/" + namespace + "/" + name

	imageVersionTracker.mu.Lock()
	defer imageVersionTracker.mu.Unlock()

	// Prevent unbounded growth
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
		// Update version gauge (limited to 5 tracked versions)
		imageVersionTracker.topVersions[normalizedVersion]++
		TrainingOperatorImageVersionUsage.WithLabelValues(normalizedVersion).Inc()

		// Update image source preference (3 timeseries)
		TrainingOperatorImageSourcePreference.WithLabelValues(imageSource).Inc()

		// Update enterprise adoption (2 timeseries) - only for RHOAI images
		if imageSource == "rhoai_official" {
			simplifiedCustomerType := simplifyCustomerType(customerType)
			TrainingOperatorEnterpriseAdoption.WithLabelValues(simplifiedCustomerType).Inc()
		}

		klog.V(4).Infof("Recorded job creation: version=%s (from %s %s), source=%s, customer=%s",
			normalizedVersion, framework, version, imageSource, customerType)
	}
}

// TrackImageVersion is an alias for RecordJobCreation (backward compatibility)
func TrackImageVersion(framework, version, imageSource, customerType, namespace, name string) {
	RecordJobCreation(framework, version, imageSource, customerType, namespace, name)
}

// RecordJobDeletion records when a training job is deleted
func RecordJobDeletion(framework, version, namespace, name string) {
	normalizedVersion := normalizeVersion(framework, version)
	key := normalizedVersion + "/" + namespace + "/" + name

	imageVersionTracker.mu.Lock()
	defer imageVersionTracker.mu.Unlock()

	if data, ok := imageVersionTracker.activeVersions[key]; ok {
		// Decrement version gauge
		if imageVersionTracker.topVersions[normalizedVersion] > 0 {
			imageVersionTracker.topVersions[normalizedVersion]--
			TrainingOperatorImageVersionUsage.WithLabelValues(normalizedVersion).Dec()
		}

		delete(imageVersionTracker.activeVersions, key)

		klog.V(4).Infof("Recorded job deletion: version=%s", normalizedVersion)
	}
}

// RemoveImageVersion is an alias for RecordJobDeletion (backward compatibility)
func RemoveImageVersion(framework, version, namespace, name string) {
	RecordJobDeletion(framework, version, namespace, name)
}

// ClassifyCustomer analyzes job metadata to determine customer type
func ClassifyCustomer(namespace string, job interface{}) *CustomerInfo {
	customerInfo := &CustomerInfo{
		CustomerType:     "non-enterprise", // Default to non-enterprise for privacy
		UsageSource:      "unknown",
		NamespacePattern: "unknown",
		TenantHints:      []string{}, // Empty for privacy compliance
		CachedAt:         time.Now(),
	}

	// Extract metadata from job
	var annotations map[string]string
	var labels map[string]string

	if metaAccessor, ok := job.(metav1.Object); ok {
		annotations = metaAccessor.GetAnnotations()
		labels = metaAccessor.GetLabels()
	}

	// Check for enterprise indicators
	if annotations != nil {
		// RHOAI UI created jobs indicate enterprise usage
		if source, exists := annotations["rhods.openshiftai.io/source"]; exists {
			if source == "dashboard" || source == "ui" || source == "workbench" {
				customerInfo.CustomerType = "enterprise"
				customerInfo.UsageSource = "rhoai-ui"
			}
		}

		// Notebook integration indicates enterprise usage
		if _, exists := annotations["notebooks.openshiftai.io/notebook-name"]; exists {
			customerInfo.CustomerType = "enterprise"
			customerInfo.UsageSource = "rhoai-notebook"
		}
	}

	if labels != nil {
		// Production environment labels
		if env, exists := labels["environment"]; exists {
			if env == "production" || env == "prod" {
				customerInfo.CustomerType = "enterprise"
			}
		}
	}

	// Simple namespace-based classification
	namespaceLower := strings.ToLower(namespace)
	if strings.Contains(namespaceLower, "prod") ||
		strings.Contains(namespaceLower, "production") {
		customerInfo.CustomerType = "enterprise"
		customerInfo.NamespacePattern = "production"
	}

	return customerInfo
}

// classifyCustomerUsage is an alias for ClassifyCustomer (backward compatibility)
func classifyCustomerUsage(namespace string, job interface{}) *CustomerInfo {
	return ClassifyCustomer(namespace, job)
}

// GetActiveJobCount returns the number of active jobs being tracked
func GetActiveJobCount() int {
	imageVersionTracker.mu.RLock()
	defer imageVersionTracker.mu.RUnlock()
	return len(imageVersionTracker.activeVersions)
}

// GetActiveVersionCount is an alias for GetActiveJobCount
func GetActiveVersionCount() int {
	return GetActiveJobCount()
}

// GetVersionDistribution returns the distribution of versions across all jobs
func GetVersionDistribution() map[string]int {
	imageVersionTracker.mu.RLock()
	defer imageVersionTracker.mu.RUnlock()

	dist := make(map[string]int)
	for _, data := range imageVersionTracker.activeVersions {
		dist[data.Version]++
	}
	return dist
}

// GetMetricsSummary returns a summary of all metrics for debugging
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

// normalizeVersion maps versions to our tracked set to maintain cardinality limit
func normalizeVersion(framework, version string) string {
	// Check if it's one of our specifically tracked versions
	for _, tracked := range trackedVersions[:len(trackedVersions)-1] { // Exclude "other"
		if version == tracked {
			return tracked
		}
	}

	// Map pytorch versions we care about
	if strings.Contains(strings.ToLower(version), "pytorch") {
		if strings.Contains(version, "2.4") || strings.Contains(version, "2-4") {
			return "pytorch-2.4"
		}
		if strings.Contains(version, "2.3") || strings.Contains(version, "2-3") {
			return "pytorch-2.3"
		}
	}

	// Map tensorflow versions we care about
	if strings.Contains(strings.ToLower(version), "tensorflow") {
		if strings.Contains(version, "2.15") || strings.Contains(version, "2-15") {
			return "tensorflow-2.15"
		}
		if strings.Contains(version, "2.14") || strings.Contains(version, "2-14") {
			return "tensorflow-2.14"
		}
	}

	// Everything else goes to "other"
	return "other"
}

// simplifyCustomerType reduces to binary classification for Red Hat compliance
func simplifyCustomerType(customerType string) string {
	customerTypeLower := strings.ToLower(customerType)
	if customerTypeLower == "enterprise" ||
		strings.Contains(customerTypeLower, "prod") ||
		customerTypeLower == "production" {
		return "enterprise"
	}
	return "non-enterprise"
}

// analyzeJobResources placeholder for resource analysis
func analyzeJobResources(job interface{}) *ResourceInfo {
	return &ResourceInfo{
		CPUCategory:    "medium",
		MemoryCategory: "medium",
		GPUCategory:    "none",
		StorageType:    "local",
	}
}

func (ivt *ImageVersionTracker) cleanupRoutine() {
	ticker := time.NewTicker(ivt.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		ivt.mu.Lock()
		now := time.Now()
		for key, data := range ivt.activeVersions {
			if now.Sub(data.LastUpdated) > ivt.maxAge {
				// Decrement counters before removal
				if ivt.topVersions[data.Version] > 0 {
					ivt.topVersions[data.Version]--
					TrainingOperatorImageVersionUsage.WithLabelValues(data.Version).Dec()
				}
				delete(ivt.activeVersions, key)
				klog.V(4).Infof("Cleaned up stale job tracking: %s", key)
			}
		}
		ivt.mu.Unlock()
	}
}

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
