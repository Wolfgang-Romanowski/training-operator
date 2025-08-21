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

// Package telemetry provides telemetry collection for training-operator.
// This file handles conversion of job events to business metrics per RHOAISTRAT-575.
package telemetry

import (
	"context"
	"reflect"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	"github.com/kubeflow/training-operator/pkg/telemetry/analyzers"
	"github.com/kubeflow/training-operator/pkg/telemetry/metrics"
)

// convertEventToMetrics is the main entry point for converting job events to metrics.
// It includes circuit breaker protection and cardinality validation to ensure
// compliance with Red Hat monitoring requirements.
func convertEventToMetrics(ctx context.Context, event JobEventData) {
	if !isTelemetryEnabled() {
		return
	}

	// Circuit breaker protection
	if metrics.IsCircuitBreakerOpen() {
		metrics.RecordInternalFailure("telemetry", "circuit_breaker_open")
		klog.V(5).Info("Circuit breaker open, skipping event conversion to metrics")
		return
	}

	// Pre-processing cardinality validation
	if err := metrics.ValidateMetricCardinality(); err != nil {
		klog.ErrorS(err, "Cardinality violation detected, triggering circuit breaker")
		metrics.CheckCardinalityCircuitBreaker()
		return
	}

	processCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Process event with panic recovery
	func() {
		defer func() {
			if r := recover(); r != nil {
				klog.ErrorS(nil, "Panic in event conversion", "panic", r, "event", event.EventType)
				metrics.RecordInternalFailure("telemetry", "panic_in_conversion")
			}
		}()

		// Process based on event type
		switch event.EventType {
		case JobCreatedEvent:
			processJobCreatedEvent(processCtx, event)
		case JobCompletedEvent:
			processJobCompletionEvent(processCtx, event, true)
		case JobFailedEvent:
			processJobCompletionEvent(processCtx, event, false)
		case JobDeletedEvent:
			processJobDeletedEvent(processCtx, event)
		case JobStartedEvent:
			// Started events are logged but don't update business metrics
			klog.V(4).InfoS("Job started event processed", 
				"namespace", event.JobNamespace, 
				"name", event.JobName,
				"framework", event.Framework)
		default:
			klog.V(4).InfoS("Unknown job event type", "eventType", event.EventType)
		}
	}()

	// Post-processing cardinality check
	metrics.CheckCardinalityCircuitBreaker()
}

// processJobCreatedEvent handles job creation events by extracting and analyzing
// container images to record business metrics for version usage, image source
// preference, and enterprise adoption.
func processJobCreatedEvent(ctx context.Context, event JobEventData) {
	// Extract container images using generic extraction
	containerImages := extractContainerImagesGeneric(event.Job)
	if len(containerImages) == 0 {
		klog.V(4).InfoS("No container images found in job", 
			"framework", event.Framework,
			"namespace", event.JobNamespace)
		return
	}

	// Determine customer type once for all images
	customerInfo := ClassifyCustomerUsage(event.JobNamespace, event.Job)
	customerType := "non-enterprise"
	if customerInfo != nil {
		customerType = customerInfo.CustomerType
	}

	// Process each container image for metrics
	for _, imageSpec := range containerImages {
		if imageSpec == "" {
			continue
		}

		// Analyze the image for version and source
		imageAnalysis := analyzers.AnalyzeContainerImage(imageSpec)
		
		// Record all three business metrics
		metrics.RecordImageVersionUsage(imageAnalysis.RHOAIVersion)
		metrics.RecordImageSourcePreference(imageAnalysis.ImageSource)
		metrics.RecordEnterpriseAdoption(customerType)

		klog.V(3).InfoS("Job creation metrics recorded",
			"framework", event.Framework,
			"namespace", event.JobNamespace,
			"version", imageAnalysis.RHOAIVersion,
			"source", imageAnalysis.ImageSource,
			"customerType", customerType)
	}
}

// processJobCompletionEvent handles job completion events (success or failure).
// Currently logs for observability but doesn't update business metrics as
// completion status isn't part of the 3 required metrics.
func processJobCompletionEvent(ctx context.Context, event JobEventData, succeeded bool) {
	status := "failed"
	if succeeded {
		status = "succeeded"
	}

	klog.V(3).InfoS("Job completion event processed", 
		"namespace", event.JobNamespace,
		"name", event.JobName,
		"status", status,
		"framework", event.Framework)
}

// processJobDeletedEvent handles job deletion events.
// Deletion events are logged for observability but don't affect the
// 3 business metrics which track adoption patterns, not lifecycle.
func processJobDeletedEvent(ctx context.Context, event JobEventData) {
	klog.V(3).InfoS("Job deletion event processed",
		"namespace", event.JobNamespace,
		"name", event.JobName,
		"framework", event.Framework)
}

// extractContainerImagesGeneric extracts all container images from any job type
// using reflection to avoid duplicating code for each framework.
// This replaces 6 nearly identical framework-specific extraction functions.
func extractContainerImagesGeneric(job interface{}) []string {
	if job == nil {
		return []string{}
	}

	var images []string
	jobValue := reflect.ValueOf(job)
	
	// Handle pointer types
	if jobValue.Kind() == reflect.Ptr {
		jobValue = jobValue.Elem()
	}
	
	if !jobValue.IsValid() {
		return []string{}
	}

	// Navigate to Spec field
	specField := jobValue.FieldByName("Spec")
	if !specField.IsValid() {
		return []string{}
	}

	// Look for replica specs field (handles all job types)
	// Patterns: PyTorchReplicaSpecs, TFReplicaSpecs, XGBReplicaSpecs, etc.
	specType := specField.Type()
	for i := 0; i < specField.NumField(); i++ {
		fieldName := specType.Field(i).Name
		if strings.HasSuffix(fieldName, "ReplicaSpecs") {
			replicaSpecs := specField.Field(i)
			images = append(images, extractImagesFromReplicaSpecs(replicaSpecs)...)
		}
	}

	return images
}

// extractImagesFromReplicaSpecs extracts container images from replica specifications
func extractImagesFromReplicaSpecs(replicaSpecs reflect.Value) []string {
	var images []string
	
	if replicaSpecs.Kind() != reflect.Map {
		return images
	}

	// Iterate through map entries (e.g., master, worker replicas)
	for _, key := range replicaSpecs.MapKeys() {
		replicaSpec := replicaSpecs.MapIndex(key)
		
		// Handle pointer types
		if replicaSpec.Kind() == reflect.Ptr {
			if replicaSpec.IsNil() {
				continue
			}
			replicaSpec = replicaSpec.Elem()
		}

		// Navigate to Template.Spec.Containers
		template := replicaSpec.FieldByName("Template")
		if !template.IsValid() {
			continue
		}

		podSpec := template.FieldByName("Spec")
		if !podSpec.IsValid() {
			continue
		}

		containers := podSpec.FieldByName("Containers")
		if !containers.IsValid() || containers.Kind() != reflect.Slice {
			continue
		}

		// Extract images from containers
		for j := 0; j < containers.Len(); j++ {
			container := containers.Index(j)
			imageField := container.FieldByName("Image")
			if imageField.IsValid() && imageField.Kind() == reflect.String {
				if image := imageField.String(); image != "" {
					images = append(images, image)
				}
			}
		}
	}

	return images
}
