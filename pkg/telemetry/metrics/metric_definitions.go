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
	TrainingOperatorImageVersionUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "training_operator_image_version_usage",
			Help: "Active training jobs by RHOAI runtime version for deprecation analysis",
		},
		[]string{"version"},
	)

	TrainingOperatorImageSourcePreference = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "training_operator_image_source_preference_total",
			Help: "Total jobs by image source showing customer runtime preferences",
		},
		[]string{"image_source"},
	)

	TrainingOperatorEnterpriseAdoption = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "training_operator_enterprise_adoption_total",
			Help: "Enterprise vs non-enterprise adoption of RHOAI runtimes",
		},
		[]string{"customer_type"},
	)


	imageVersionTracker = &ImageVersionTracker{
		activeVersions:  make(map[string]*VersionData),
		topVersions:     make(map[string]int),
		mu:              sync.RWMutex{},
		cleanupInterval: 1 * time.Hour,
		maxAge:          24 * time.Hour,
		maxEntries:      10000,
	}

	crdTrackingInitOnce sync.Once

	trackedVersions = []string{
		"pytorch-2.4",
		"pytorch-2.3",
		"tensorflow-2.15",
		"tensorflow-2.14",
		"other",
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

// InitializeMetrics initializes all telemetry metrics and starts background
// cleanup routines for tracking active job versions.
func InitializeMetrics() {
	crdTrackingInitOnce.Do(func() {
		metrics.Registry.MustRegister(
			TrainingOperatorImageVersionUsage,
			TrainingOperatorImageSourcePreference,
			TrainingOperatorEnterpriseAdoption,
		)

		for _, v := range trackedVersions {
			imageVersionTracker.topVersions[v] = 0
			TrainingOperatorImageVersionUsage.WithLabelValues(v).Set(0)
		}

		go imageVersionTracker.cleanupRoutine()

		klog.Info("Telemetry metrics initialized successfully")
	})
}

// InitCRDInstanceTracking initializes metrics for backward compatibility with existing code.
func InitCRDInstanceTracking() {
	InitializeMetrics()
}

// RecordJobCreation records metrics when a new training job is created,
// tracking image version usage, source preferences, and enterprise adoption.
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

// TrackImageVersion records image version usage for backward compatibility with existing code.
func TrackImageVersion(framework, version, imageSource, customerType, namespace, name string) {
	RecordJobCreation(framework, version, imageSource, customerType, namespace, name)
}

// RecordJobDeletion decrements job tracking metrics when a training job is deleted.
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

// RemoveImageVersion removes image version tracking for backward compatibility with existing code.
func RemoveImageVersion(framework, version, namespace, name string) {
	RecordJobDeletion(framework, version, namespace, name)
}

// ClassifyCustomer analyzes job metadata and namespace patterns to determine
// whether the job represents enterprise or non-enterprise usage.
func ClassifyCustomer(namespace string, job interface{}) *CustomerInfo {
	customerInfo := &CustomerInfo{
		CustomerType:     "non-enterprise",
		UsageSource:      "unknown",
		NamespacePattern: "unknown",
		TenantHints:      []string{},
		CachedAt:         time.Now(),
	}

	var annotations map[string]string
	var labels map[string]string

	if metaAccessor, ok := job.(metav1.Object); ok {
		annotations = metaAccessor.GetAnnotations()
		labels = metaAccessor.GetLabels()
	}

	if annotations != nil {
		if source, exists := annotations["rhods.openshiftai.io/source"]; exists {
			if source == "dashboard" || source == "ui" || source == "workbench" {
				customerInfo.CustomerType = "enterprise"
				customerInfo.UsageSource = "rhoai-ui"
			}
		}

		if _, exists := annotations["notebooks.openshiftai.io/notebook-name"]; exists {
			customerInfo.CustomerType = "enterprise"
			customerInfo.UsageSource = "rhoai-notebook"
		}
	}

	if labels != nil {
		if env, exists := labels["environment"]; exists {
			if env == "production" || env == "prod" {
				customerInfo.CustomerType = "enterprise"
			}
		}
	}

	namespaceLower := strings.ToLower(namespace)
	if strings.Contains(namespaceLower, "prod") ||
		strings.Contains(namespaceLower, "production") {
		customerInfo.CustomerType = "enterprise"
		customerInfo.NamespacePattern = "production"
	}

	return customerInfo
}

// classifyCustomerUsage provides backward compatibility for existing customer classification code.
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

// normalizeVersionForTracking maps framework versions to tracked categories
// to maintain metric cardinality limits.
func normalizeVersionForTracking(framework, version string) string {
	for _, tracked := range trackedVersions[:len(trackedVersions)-1] {
		if version == tracked {
			return tracked
		}
	}

	if strings.Contains(strings.ToLower(version), "pytorch") {
		if strings.Contains(version, "2.4") || strings.Contains(version, "2-4") {
			return "pytorch-2.4"
		}
		if strings.Contains(version, "2.3") || strings.Contains(version, "2-3") {
			return "pytorch-2.3"
		}
	}

	if strings.Contains(strings.ToLower(version), "tensorflow") {
		if strings.Contains(version, "2.15") || strings.Contains(version, "2-15") {
			return "tensorflow-2.15"
		}
		if strings.Contains(version, "2.14") || strings.Contains(version, "2-14") {
			return "tensorflow-2.14"
		}
	}

	return "other"
}

// simplifyCustomerTypeForClassification normalizes customer types to binary
// enterprise/non-enterprise classification for consistent metrics.
func simplifyCustomerTypeForClassification(customerType string) string {
	customerTypeLower := strings.ToLower(customerType)
	if customerTypeLower == "enterprise" ||
		strings.Contains(customerTypeLower, "prod") ||
		customerTypeLower == "production" {
		return "enterprise"
	}
	return "non-enterprise"
}

// analyzeJobResources provides placeholder resource analysis for future extension.
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
