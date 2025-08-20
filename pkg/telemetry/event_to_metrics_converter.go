// pkg/telemetry/event_to_metrics_converter.go
// FIXED: Properly tracks image versions with 10 timeseries limit compliance
package telemetry

import (
	"context"
	"strings"
	"time"

	"github.com/kubeflow/training-operator/pkg/telemetry/analyzers"
	"github.com/kubeflow/training-operator/pkg/telemetry/metrics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

// convertEventToMetrics is the main entry point for event processing
func convertEventToMetrics(ctx context.Context, event JobEventData) {
	if !isTelemetryEnabled() {
		return
	}

	// Add timeout protection
	processCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	switch event.EventType {
	case JobCreatedEvent:
		convertJobCreatedToMetrics(processCtx, event)
	case JobStartedEvent:
		klog.V(4).Infof("Job started: %s/%s", event.JobNamespace, event.JobName)
	case JobCompletedEvent:
		convertJobCompletedToMetrics(processCtx, event, true)
	case JobFailedEvent:
		convertJobCompletedToMetrics(processCtx, event, false)
	case JobDeletedEvent:
		convertJobDeletedToMetrics(processCtx, event)
	default:
		klog.V(4).Infof("Unknown event type: %s", event.EventType)
	}
}

// convertJobCreatedToMetrics handles job creation events with 10 timeseries limit
func convertJobCreatedToMetrics(ctx context.Context, event JobEventData) {
	// Extract job metadata
	metaObj, ok := event.Job.(metav1.Object)
	if !ok {
		klog.Warning("Job does not implement metav1.Object")
		return
	}

	namespace := metaObj.GetNamespace()
	name := metaObj.GetName()
	framework := strings.ToLower(event.Framework)

	// Analyze customer type using centralized classification
	// This uses the ClassifyCustomer from metrics package which provides
	// binary classification (enterprise/non-enterprise) for Red Hat compliance
	customerInfo := metrics.ClassifyCustomer(namespace, event.Job)
	customerType := "non-enterprise"
	if customerInfo != nil {
		// Already normalized to binary classification
		customerType = customerInfo.CustomerType
	}

	// Extract and analyze container image using centralized extraction
	image := analyzers.ExtractContainerImage(event.Job, framework)
	if image == "" {
		klog.V(3).Infof("Could not extract image for job %s/%s", namespace, name)
		image = "unknown"
	}

	imageAnalysis := analyzers.AnalyzeContainerImage(image)

	// Track metrics with strict cardinality limits
	// Total must not exceed 10 timeseries across all metrics
	metrics.TrackImageVersion(
		framework,
		imageAnalysis.RHOAIVersion,
		imageAnalysis.ImageSource,
		customerType,
		namespace,
		name,
	)

	klog.V(2).Infof("Job %s/%s created - Framework: %s, Version: %s, Source: %s, Customer: %s",
		namespace, name, framework, imageAnalysis.RHOAIVersion,
		imageAnalysis.ImageSource, customerType)
}

// convertJobCompletedToMetrics handles job completion events
func convertJobCompletedToMetrics(ctx context.Context, event JobEventData, succeeded bool) {
	metaObj, ok := event.Job.(metav1.Object)
	if !ok {
		return
	}

	namespace := metaObj.GetNamespace()
	name := metaObj.GetName()

	status := "failed"
	if succeeded {
		status = "succeeded"
	}

	klog.V(3).Infof("Job %s/%s completed with status: %s", namespace, name, status)
}

// convertJobDeletedToMetrics handles job deletion events
func convertJobDeletedToMetrics(ctx context.Context, event JobEventData) {
	metaObj, ok := event.Job.(metav1.Object)
	if !ok {
		return
	}

	namespace := metaObj.GetNamespace()
	name := metaObj.GetName()
	framework := strings.ToLower(event.Framework)

	// Extract image to determine version for cleanup using centralized extraction
	image := analyzers.ExtractContainerImage(event.Job, framework)
	if image != "" {
		imageAnalysis := analyzers.AnalyzeContainerImage(image)
		metrics.RemoveImageVersion(framework, imageAnalysis.RHOAIVersion, namespace, name)
	}

	klog.V(3).Infof("Job %s/%s deleted", namespace, name)
}

// All image extraction functions have been moved to analyzers/image_analyzer.go
// This provides a single source of truth for image extraction logic
