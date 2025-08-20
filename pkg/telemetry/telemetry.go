// pkg/telemetry/telemetry.go
// Main telemetry package - single entry point for all telemetry operations
package telemetry

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/kubeflow/training-operator/pkg/telemetry/collectors"
	"github.com/kubeflow/training-operator/pkg/telemetry/config"
	"github.com/kubeflow/training-operator/pkg/telemetry/events"
	"github.com/kubeflow/training-operator/pkg/telemetry/metrics"
	"k8s.io/klog/v2"
)

var (
	initOnce     sync.Once
	isInitialized bool
)

// Initialize sets up all telemetry components
func Initialize() error {
	var err error
	initOnce.Do(func() {
		klog.Info("Initializing telemetry system")
		
		// Initialize configuration
		config.Initialize()
		
		// Initialize metrics registry
		err = metrics.Initialize()
		if err != nil {
			klog.Errorf("Failed to initialize metrics: %v", err)
			return
		}
		
		// Initialize collectors
		err = collectors.Initialize()
		if err != nil {
			klog.Errorf("Failed to initialize collectors: %v", err)
			return
		}
		
		// Initialize event processing
		events.Initialize()
		
		isInitialized = true
		klog.Info("Telemetry system initialized successfully")
	})
	return err
}

// IsEnabled returns whether telemetry is enabled
func IsEnabled() bool {
	return config.IsEnabled()
}

// IsInitialized returns whether telemetry has been initialized
func IsInitialized() bool {
	return isInitialized
}

// Public API functions for controllers to use

// ReportJobCreation reports a training job creation event
func ReportJobCreation(job interface{}, framework string) {
	if !IsEnabled() || !IsInitialized() {
		return
	}
	events.ReportJobEvent(events.JobCreatedEvent, job, framework, nil)
}

// ReportJobStarted reports a training job started event
func ReportJobStarted(job interface{}, framework string) {
	if !IsEnabled() || !IsInitialized() {
		return
	}
	events.ReportJobEvent(events.JobStartedEvent, job, framework, nil)
}

// ReportJobCompletion reports a training job completion event
func ReportJobCompletion(job interface{}, framework string, succeeded bool) {
	if !IsEnabled() || !IsInitialized() {
		return
	}
	
	eventType := events.JobCompletedEvent
	if !succeeded {
		eventType = events.JobFailedEvent
	}
	
	events.ReportJobEvent(eventType, job, framework, nil)
}

// ReportJobFailure reports a training job failure with reason
func ReportJobFailure(job interface{}, framework string, reason string) {
	if !IsEnabled() || !IsInitialized() {
		return
	}
	
	metadata := map[string]string{"reason": reason}
	events.ReportJobEvent(events.JobFailedEvent, job, framework, metadata)
}

// ReportJobDeletion reports a training job deletion event
func ReportJobDeletion(job interface{}, framework string) {
	if !IsEnabled() || !IsInitialized() {
		return
	}
	events.ReportJobEvent(events.JobDeletedEvent, job, framework, nil)
}

// Internal helper for backward compatibility
func convertEventToMetrics(ctx context.Context, event events.JobEventData) {
	// This function is kept for backward compatibility with existing code
	// but now delegates to the new event processing system
	events.ProcessEvent(ctx, event)
}