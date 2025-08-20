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

package telemetry

import (
	"context"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	"github.com/kubeflow/training-operator/pkg/telemetry/analyzers"
	"github.com/kubeflow/training-operator/pkg/telemetry/metrics"
)

// convertEventToMetrics processes training job events and converts them to telemetry metrics.
// It applies appropriate timeout protection to ensure event processing doesn't block
// the main reconciliation loop.
func convertEventToMetrics(ctx context.Context, event JobEventData) {
	if !isTelemetryEnabled() {
		return
	}

	processCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	switch event.EventType {
	case JobCreatedEvent:
		convertJobCreatedToMetrics(processCtx, event)
	case JobStartedEvent:
		klog.V(4).InfoS("Job started event processed", "namespace", event.JobNamespace, "name", event.JobName)
	case JobCompletedEvent:
		convertJobCompletedToMetrics(processCtx, event, true)
	case JobFailedEvent:
		convertJobCompletedToMetrics(processCtx, event, false)
	case JobDeletedEvent:
		convertJobDeletedToMetrics(processCtx, event)
	default:
		klog.V(4).InfoS("Unknown job event type", "eventType", event.EventType)
	}
}

// convertJobCreatedToMetrics processes job creation events.
// It extracts container image information, analyzes customer type, and updates
// telemetry metrics for version tracking and usage pattern analysis.
func convertJobCreatedToMetrics(ctx context.Context, event JobEventData) {
	metaObj, ok := event.Job.(metav1.Object)
	if !ok {
		klog.Warning("Job does not implement metav1.Object interface")
		return
	}

	namespace := metaObj.GetNamespace()
	name := metaObj.GetName()
	framework := strings.ToLower(event.Framework)

	customerInfo := ClassifyCustomerUsage(namespace, event.Job)
	customerType := "non-enterprise"
	if customerInfo != nil {
		customerType = customerInfo.CustomerType
	}

	image := analyzers.ExtractContainerImage(event.Job, framework)
	if image == "" {
		klog.V(3).InfoS("Could not extract container image from job", "namespace", namespace, "name", name, "framework", framework)
		image = "unknown"
	}

	imageAnalysis := analyzers.AnalyzeContainerImage(image)

	metrics.RecordJobCreation(
		framework,
		imageAnalysis.RHOAIVersion,
		imageAnalysis.ImageSource,
		customerType,
		namespace,
		name,
	)

	klog.V(2).InfoS("Job creation metrics recorded",
		"namespace", namespace,
		"name", name,
		"framework", framework,
		"version", imageAnalysis.RHOAIVersion,
		"imageSource", imageAnalysis.ImageSource,
		"customerType", customerType,
		"acceleratorType", imageAnalysis.AcceleratorType)
}

// convertJobCompletedToMetrics processes job completion events.
// It logs the final status of training jobs (succeeded or failed) for
// success rate analysis and reliability tracking.
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

	klog.V(3).InfoS("Job completion event processed", "namespace", namespace, "name", name, "status", status, "framework", event.Framework)
}

// convertJobDeletedToMetrics processes job deletion events.
// It removes associated telemetry tracking for the deleted training job
// to maintain accurate active job counts and prevent metric drift.
func convertJobDeletedToMetrics(ctx context.Context, event JobEventData) {
	metaObj, ok := event.Job.(metav1.Object)
	if !ok {
		return
	}

	namespace := metaObj.GetNamespace()
	name := metaObj.GetName()
	framework := strings.ToLower(event.Framework)

	image := analyzers.ExtractContainerImage(event.Job, framework)
	if image != "" {
		imageAnalysis := analyzers.AnalyzeContainerImage(image)
		metrics.RecordJobDeletion(framework, imageAnalysis.RHOAIVersion, namespace, name)
	}

	klog.V(3).InfoS("Job deletion metrics updated", "namespace", namespace, "name", name, "framework", framework)
}
