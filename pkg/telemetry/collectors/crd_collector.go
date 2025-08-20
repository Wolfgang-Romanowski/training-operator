// pkg/telemetry/collectors/crd_collector.go
// CRD instance tracking collector - manages CRD lifecycle metrics
package collectors

import (
	"context"
	"sync"
	"time"

	"github.com/kubeflow/training-operator/pkg/telemetry/analysis"
	"github.com/kubeflow/training-operator/pkg/telemetry/config"
	"github.com/kubeflow/training-operator/pkg/telemetry/events"
	"github.com/kubeflow/training-operator/pkg/telemetry/metrics"
	"k8s.io/klog/v2"
)

// CRDCollector manages CRD instance tracking and metrics
type CRDCollector struct {
	mu              sync.RWMutex
	activeInstances map[string]*CRDInstanceData
	customerCache   map[string]*events.CustomerInfo
	
	// Configuration
	maxEntries      int
	maxAge          time.Duration
	cleanupInterval time.Duration
	
	// Internal state
	initialized     bool
}

// CRDInstanceData represents an active CRD instance
type CRDInstanceData struct {
	InstanceKey      string
	Framework        string
	Namespace        string
	Name             string
	CustomerType     string
	CreatedAt        time.Time
	LastUpdated      time.Duration
}

var (
	crdCollector *CRDCollector
	crdOnce      sync.Once
)

// InitializeCRDCollector sets up the CRD instance collector
func InitializeCRDCollector() *CRDCollector {
	crdOnce.Do(func() {
		cfg := config.Get()
		
		crdCollector = &CRDCollector{
			activeInstances: make(map[string]*CRDInstanceData),
			customerCache:   make(map[string]*events.CustomerInfo),
			maxEntries:      cfg.MaxEntries,
			maxAge:          cfg.MaxEntryAge,
			cleanupInterval: cfg.CleanupInterval,
			initialized:     true,
		}
		
		// Start cleanup routine
		go crdCollector.startCleanupRoutine()
		
		klog.Info("CRD collector initialized")
	})
	return crdCollector
}

// GetCRDCollector returns the singleton CRD collector instance
func GetCRDCollector() *CRDCollector {
	if crdCollector == nil {
		return InitializeCRDCollector()
	}
	return crdCollector
}

// ProcessJobCreation handles job creation events
func (c *CRDCollector) ProcessJobCreation(ctx context.Context, event events.JobEventData) error {
	if !c.initialized {
		return nil
	}
	
	// Analyze customer information
	customerInfo := analysis.ClassifyCustomer(event.JobNamespace, event.Job)
	if customerInfo == nil {
		klog.Warning("Failed to classify customer, using default")
		customerInfo = &events.CustomerInfo{CustomerType: "non-enterprise"}
	}
	
	// Generate instance key
	instanceKey := c.generateInstanceKey(event)
	
	// Track the instance
	c.trackInstance(instanceKey, event.Framework, event.JobNamespace, event.JobName, customerInfo.CustomerType)
	
	// Update metrics
	c.updateMetricsForCreation(event.Framework, customerInfo.CustomerType)
	
	return nil
}

// ProcessJobDeletion handles job deletion events
func (c *CRDCollector) ProcessJobDeletion(ctx context.Context, event events.JobEventData) error {
	if !c.initialized {
		return nil
	}
	
	instanceKey := c.generateInstanceKey(event)
	
	// Remove from tracking
	c.removeInstance(instanceKey)
	
	return nil
}

// trackInstance adds or updates an instance in the tracking map
func (c *CRDCollector) trackInstance(instanceKey, framework, namespace, name, customerType string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Prevent unbounded growth
	if len(c.activeInstances) >= c.maxEntries {
		c.removeOldestInstanceLocked()
	}
	
	instance := &CRDInstanceData{
		InstanceKey:  instanceKey,
		Framework:    framework,
		Namespace:    namespace,
		Name:         name,
		CustomerType: customerType,
		CreatedAt:    time.Now(),
		LastUpdated:  0,
	}
	
	c.activeInstances[instanceKey] = instance
}

// removeInstance removes an instance from tracking
func (c *CRDCollector) removeInstance(instanceKey string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if instance, exists := c.activeInstances[instanceKey]; exists {
		// Update metrics before removal
		c.updateMetricsForDeletion(instance.Framework, instance.CustomerType)
		delete(c.activeInstances, instanceKey)
	}
}

// updateMetricsForCreation updates metrics when an instance is created
func (c *CRDCollector) updateMetricsForCreation(framework, customerType string) {
	metricsRegistry := metrics.Get()
	if metricsRegistry == nil {
		return
	}
	
	// Update active instances gauge
	metricsRegistry.CRDInstancesActive.WithLabelValues(framework, customerType).Inc()
	
	// Update customer usage counter
	metricsRegistry.CustomerUsage.WithLabelValues(customerType).Inc()
	
	// Update framework adoption counter
	metricsRegistry.FrameworkAdoption.WithLabelValues(framework).Inc()
}

// updateMetricsForDeletion updates metrics when an instance is deleted
func (c *CRDCollector) updateMetricsForDeletion(framework, customerType string) {
	metricsRegistry := metrics.Get()
	if metricsRegistry == nil {
		return
	}
	
	// Decrease active instances gauge
	metricsRegistry.CRDInstancesActive.WithLabelValues(framework, customerType).Dec()
}

// generateInstanceKey creates a unique key for tracking instances
func (c *CRDCollector) generateInstanceKey(event events.JobEventData) string {
	return event.Framework + "/" + event.JobNamespace + "/" + event.JobName
}

// removeOldestInstanceLocked removes the oldest instance (must be called with lock held)
func (c *CRDCollector) removeOldestInstanceLocked() {
	var oldestKey string
	var oldestTime time.Time
	
	for key, instance := range c.activeInstances {
		if oldestKey == "" || instance.CreatedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = instance.CreatedAt
		}
	}
	
	if oldestKey != "" {
		instance := c.activeInstances[oldestKey]
		c.updateMetricsForDeletion(instance.Framework, instance.CustomerType)
		delete(c.activeInstances, oldestKey)
	}
}

// startCleanupRoutine starts the background cleanup process
func (c *CRDCollector) startCleanupRoutine() {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()
	
	for range ticker.C {
		c.cleanup()
	}
}

// cleanup removes old instances and customer cache entries
func (c *CRDCollector) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	now := time.Now()
	
	// Clean up old instances
	for key, instance := range c.activeInstances {
		if now.Sub(instance.CreatedAt) > c.maxAge {
			c.updateMetricsForDeletion(instance.Framework, instance.CustomerType)
			delete(c.activeInstances, key)
		}
	}
	
	// Clean up old customer cache entries
	for namespace, customerInfo := range c.customerCache {
		// Check if customer info is old (assuming TenantHints[0] contains timestamp)
		if len(customerInfo.TenantHints) > 0 {
			// This is a simplified cleanup - in practice you'd store timestamps properly
			delete(c.customerCache, namespace)
		}
	}
}

// GetActiveInstanceCount returns the current number of active instances
func (c *CRDCollector) GetActiveInstanceCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.activeInstances)
}

// GetActiveInstancesByFramework returns active instances grouped by framework
func (c *CRDCollector) GetActiveInstancesByFramework() map[string]int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	counts := make(map[string]int)
	for _, instance := range c.activeInstances {
		counts[instance.Framework]++
	}
	return counts
}