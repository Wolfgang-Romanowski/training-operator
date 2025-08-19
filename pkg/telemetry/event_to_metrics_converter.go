// Copyright 2021 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package telemetry

import (
	"context"
	"fmt"
	"sync"
	"time"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	"github.com/kubeflow/training-operator/pkg/telemetry/analyzers"
	"github.com/kubeflow/training-operator/pkg/telemetry/metrics"
	corev1 "k8s.io/api/core/v1"
)

var (
	namespaceVersions = make(map[string]string)
	nvMutex           sync.RWMutex

	// Track GPU hours for cost analysis (REQUIRED)
	jobGPUStartTimes = make(map[string]struct {
		startTime time.Time
		gpuCount  int
		gpuType   string
	})
	gpuMutex sync.RWMutex

	// Track job start times for queue duration
	jobStartTimes = make(map[string]time.Time)
	jsMutex       sync.RWMutex
)

type JobDetails struct {
	ContainerImage  string
	Namespace       string
	JobName         string
	Framework       string
	ReplicaCount    int
	GPURequested    int
	GPUType         string
	MemoryRequested int64
	IsKueueManaged  bool
	KueueName       string
	FailureReason   string
}

// ReportJobStarted is called when job transitions to running
func ReportJobStarted(job interface{}, framework string) {
	ReceiveJobEvent(context.Background(), JobEventData{
		EventType: JobStartedEvent,
		Framework: framework,
		Job:       job,
	})
}

// ReportJobFailure is called when job fails with a reason
func ReportJobFailure(job interface{}, framework string, reason string) {
	ReceiveJobEvent(context.Background(), JobEventData{
		EventType: JobFailedEvent,
		Framework: framework,
		Job:       job,
		Metadata:  map[string]string{"failure_reason": reason},
	})
}

// The ctx parameter is kept for future extensibility
func convertEventToMetrics(ctx context.Context, event JobEventData) {
	jobDetails := extractJobDetails(event.Job)
	jobDetails.Framework = event.Framework

	switch event.EventType {
	case JobCreatedEvent:
		recordJobCreationMetrics(jobDetails, event.Framework)
	case JobStartedEvent:
		recordJobStartedMetrics(jobDetails)
	case JobCompletedEvent:
		recordJobCompletionMetrics(jobDetails, event.Framework, true)
	case JobFailedEvent:
		recordJobCompletionMetrics(jobDetails, event.Framework, false)
		recordJobFailureMetrics(jobDetails, event.Metadata)
	case JobDeletedEvent:
		recordJobDeletionMetrics(jobDetails, event.Framework)
	}
}

func extractJobDetails(job interface{}) JobDetails {
	details := JobDetails{}

	switch v := job.(type) {
	case *kubeflowv1.PyTorchJob:
		details.JobName = v.Name
		details.Namespace = v.Namespace
		details.ReplicaCount = 0

		// Check for Kueue integration
		details.IsKueueManaged, details.KueueName = analyzers.DetectKueueIntegration(v.Annotations)

		for replicaType, replicaSpec := range v.Spec.PyTorchReplicaSpecs {
			if replicaSpec.Replicas != nil {
				details.ReplicaCount += int(*replicaSpec.Replicas)
			}

			if replicaType == kubeflowv1.PyTorchJobReplicaTypeMaster || replicaType == kubeflowv1.PyTorchJobReplicaTypeWorker {
				if len(replicaSpec.Template.Spec.Containers) > 0 {
					container := replicaSpec.Template.Spec.Containers[0]
					if details.ContainerImage == "" {
						details.ContainerImage = container.Image
					}

					// Extract all accelerator types
					if container.Resources.Limits != nil {
						extractAcceleratorResources(&details, container.Resources.Limits)
					}

					// Extract memory for model complexity estimation
					if container.Resources.Requests != nil {
						if memory, ok := container.Resources.Requests[corev1.ResourceMemory]; ok {
							details.MemoryRequested = memory.Value()
						}
					}
				}
			}
		}

	case *kubeflowv1.TFJob:
		details.JobName = v.Name
		details.Namespace = v.Namespace
		details.ReplicaCount = 0

		// Kueue integration
		details.IsKueueManaged, details.KueueName = analyzers.DetectKueueIntegration(v.Annotations)

		// Priority: Chief > Worker > PS
		if chief, ok := v.Spec.TFReplicaSpecs[kubeflowv1.TFJobReplicaTypeChief]; ok {
			if len(chief.Template.Spec.Containers) > 0 {
				details.ContainerImage = chief.Template.Spec.Containers[0].Image
				extractResourceDetails(&details, chief.Template.Spec.Containers[0])
			}
		}

		if details.ContainerImage == "" {
			if worker, ok := v.Spec.TFReplicaSpecs[kubeflowv1.TFJobReplicaTypeWorker]; ok {
				if len(worker.Template.Spec.Containers) > 0 {
					details.ContainerImage = worker.Template.Spec.Containers[0].Image
					extractResourceDetails(&details, worker.Template.Spec.Containers[0])
				}
			}
		}

		for _, replicaSpec := range v.Spec.TFReplicaSpecs {
			if replicaSpec.Replicas != nil {
				details.ReplicaCount += int(*replicaSpec.Replicas)
			}
		}

	case *kubeflowv1.MPIJob:
		details.JobName = v.Name
		details.Namespace = v.Namespace

		details.IsKueueManaged, details.KueueName = analyzers.DetectKueueIntegration(v.Annotations)

		if v.Spec.MPIReplicaSpecs != nil {
			if launcherSpec, ok := v.Spec.MPIReplicaSpecs[kubeflowv1.MPIJobReplicaTypeLauncher]; ok {
				if len(launcherSpec.Template.Spec.Containers) > 0 {
					details.ContainerImage = launcherSpec.Template.Spec.Containers[0].Image
					extractResourceDetails(&details, launcherSpec.Template.Spec.Containers[0])
				}
			}
			if workerSpec, ok := v.Spec.MPIReplicaSpecs[kubeflowv1.MPIJobReplicaTypeWorker]; ok {
				if workerSpec.Replicas != nil {
					details.ReplicaCount = int(*workerSpec.Replicas)
				}
				// Extract GPU resources from workers
				if len(workerSpec.Template.Spec.Containers) > 0 {
					extractResourceDetails(&details, workerSpec.Template.Spec.Containers[0])
				}
			}
		}

	case *kubeflowv1.XGBoostJob:
		details.JobName = v.Name
		details.Namespace = v.Namespace

		details.IsKueueManaged, details.KueueName = analyzers.DetectKueueIntegration(v.Annotations)

		// Priority: Master > Worker
		if master, ok := v.Spec.XGBReplicaSpecs[kubeflowv1.XGBoostJobReplicaTypeMaster]; ok {
			if len(master.Template.Spec.Containers) > 0 {
				details.ContainerImage = master.Template.Spec.Containers[0].Image
				extractResourceDetails(&details, master.Template.Spec.Containers[0])
			}
		}

		for _, replicaSpec := range v.Spec.XGBReplicaSpecs {
			if replicaSpec.Replicas != nil {
				details.ReplicaCount += int(*replicaSpec.Replicas)
			}

			if details.ContainerImage == "" && len(replicaSpec.Template.Spec.Containers) > 0 {
				details.ContainerImage = replicaSpec.Template.Spec.Containers[0].Image
			}
		}

	case *kubeflowv1.JAXJob:
		details.JobName = v.Name
		details.Namespace = v.Namespace

		details.IsKueueManaged, details.KueueName = analyzers.DetectKueueIntegration(v.Annotations)

		// Priority: Worker > any other
		if worker, ok := v.Spec.JAXReplicaSpecs[kubeflowv1.JAXJobReplicaTypeWorker]; ok {
			if len(worker.Template.Spec.Containers) > 0 {
				details.ContainerImage = worker.Template.Spec.Containers[0].Image
				extractResourceDetails(&details, worker.Template.Spec.Containers[0])
			}
			if worker.Replicas != nil {
				details.ReplicaCount = int(*worker.Replicas)
			}
		}

		// Fallback to any replica if no worker
		if details.ContainerImage == "" {
			for _, replicaSpec := range v.Spec.JAXReplicaSpecs {
				if len(replicaSpec.Template.Spec.Containers) > 0 {
					details.ContainerImage = replicaSpec.Template.Spec.Containers[0].Image
					break
				}
			}
		}

		// Count all replicas
		for _, replicaSpec := range v.Spec.JAXReplicaSpecs {
			if replicaSpec.Replicas != nil {
				details.ReplicaCount += int(*replicaSpec.Replicas)
			}
		}

	case *kubeflowv1.PaddleJob:
		details.JobName = v.Name
		details.Namespace = v.Namespace

		details.IsKueueManaged, details.KueueName = analyzers.DetectKueueIntegration(v.Annotations)

		// Priority: Master > Worker
		if master, ok := v.Spec.PaddleReplicaSpecs[kubeflowv1.PaddleJobReplicaTypeMaster]; ok {
			if len(master.Template.Spec.Containers) > 0 {
				details.ContainerImage = master.Template.Spec.Containers[0].Image
				extractResourceDetails(&details, master.Template.Spec.Containers[0])
			}
		}

		if details.ContainerImage == "" {
			if worker, ok := v.Spec.PaddleReplicaSpecs[kubeflowv1.PaddleJobReplicaTypeWorker]; ok {
				if len(worker.Template.Spec.Containers) > 0 {
					details.ContainerImage = worker.Template.Spec.Containers[0].Image
					extractResourceDetails(&details, worker.Template.Spec.Containers[0])
				}
			}
		}

		for _, replicaSpec := range v.Spec.PaddleReplicaSpecs {
			if replicaSpec.Replicas != nil {
				details.ReplicaCount += int(*replicaSpec.Replicas)
			}
		}
	}

	// Default GPU type if not set
	if details.GPUType == "" {
		details.GPUType = "cpu"
	}

	return details
}

// Helper function to extract resource details
func extractResourceDetails(details *JobDetails, container corev1.Container) {
	if container.Resources.Limits != nil {
		extractAcceleratorResources(details, container.Resources.Limits)
	}

	if container.Resources.Requests != nil {
		if memory, ok := container.Resources.Requests[corev1.ResourceMemory]; ok {
			details.MemoryRequested = memory.Value()
		}
	}
}

// Extract accelerator resources following OBSDA-1087 normalization
func extractAcceleratorResources(details *JobDetails, limits corev1.ResourceList) {
	// Check all GPU types per OBSDA-1087 normalization requirements
	gpuResources := map[string]string{
		"nvidia.com/gpu":  "nvidia.com/gpu",
		"amd.com/gpu":     "amd.com/gpu",
		"habana.ai/gaudi": "habana.ai/gaudi",
		"intel.com/gpu":   "intel.com/gpu",
	}

	for resourceName, gpuType := range gpuResources {
		if gpuQuantity, ok := limits[corev1.ResourceName(resourceName)]; ok {
			details.GPURequested += int(gpuQuantity.Value())
			details.GPUType = gpuType
			break
		}
	}
}

func recordJobCreationMetrics(details JobDetails, framework string) {
	// CRITICAL: Analyze container image for RHOAI adoption metrics
	imageAnalysis := analyzers.AnalyzeContainerImage(details.ContainerImage)

	// METRIC 1: Track image source (RHOAI vs custom vs community)
	metrics.UpdateImageSourceMetric(
		framework,
		imageAnalysis.ImageSource,
		imageAnalysis.RHOAIVersion,
	)

	// METRIC 2: Track active RHOAI versions
	if imageAnalysis.ImageSource == "rhoai_official" {
		metrics.IncrementActiveRHOAIVersion(framework, imageAnalysis.RHOAIVersion)

		// METRIC 3: Track version migrations
		nvMutex.RLock()
		previousVersion := namespaceVersions[details.Namespace]
		nvMutex.RUnlock()

		if previousVersion != "" && previousVersion != imageAnalysis.RHOAIVersion {
			metrics.RecordVersionMigration(framework, previousVersion, imageAnalysis.RHOAIVersion)
		}

		nvMutex.Lock()
		namespaceVersions[details.Namespace] = imageAnalysis.RHOAIVersion
		nvMutex.Unlock()
	}

	// METRIC 4: Kueue integration tracking (REQUIRED)
	if details.IsKueueManaged {
		metrics.RecordKueueManagedJob(framework, details.KueueName, imageAnalysis.ImageSource)
		metrics.IncrementKueueQueueDepth(details.KueueName)
	}

	// METRIC 5: GPU/Accelerator tracking (REQUIRED)
	if details.GPURequested > 0 {
		metrics.IncrementAcceleratorUtilization(framework, details.GPUType, float64(details.GPURequested))

		// Track GPU start time for cost calculation
		gpuMutex.Lock()
		jobGPUStartTimes[fmt.Sprintf("%s/%s", details.Namespace, details.JobName)] = struct {
			startTime time.Time
			gpuCount  int
			gpuType   string
		}{
			startTime: time.Now(),
			gpuCount:  details.GPURequested,
			gpuType:   details.GPUType,
		}
		gpuMutex.Unlock()
	}

	// Compliance metric: total jobs
	metrics.RecordJobCreated(framework)

	// Track job start time for queue duration
	jobKey := fmt.Sprintf("%s/%s", details.Namespace, details.JobName)
	jsMutex.Lock()
	jobStartTimes[jobKey] = time.Now()
	jsMutex.Unlock()
}

// Record job started metrics for queue time tracking
func recordJobStartedMetrics(details JobDetails) {
	jobKey := fmt.Sprintf("%s/%s", details.Namespace, details.JobName)

	// Calculate queue duration
	jsMutex.RLock()
	startTime, exists := jobStartTimes[jobKey]
	jsMutex.RUnlock()

	if exists {
		queueDuration := time.Since(startTime).Seconds()
		imageAnalysis := analyzers.AnalyzeContainerImage(details.ContainerImage)

		metrics.RecordJobQueueDuration(details.Framework, imageAnalysis.ImageSource, queueDuration)
	}
}

func recordJobCompletionMetrics(details JobDetails, framework string, succeeded bool) {
	imageAnalysis := analyzers.AnalyzeContainerImage(details.ContainerImage)
	jobKey := fmt.Sprintf("%s/%s", details.Namespace, details.JobName)

	// Decrement active RHOAI version count
	if imageAnalysis.ImageSource == "rhoai_official" {
		metrics.DecrementActiveRHOAIVersion(framework, imageAnalysis.RHOAIVersion)
	}

	// Decrement Kueue queue depth
	if details.IsKueueManaged {
		metrics.DecrementKueueQueueDepth(details.KueueName)
	}

	// Calculate GPU hours consumed (REQUIRED)
	gpuMutex.Lock()
	if gpuInfo, exists := jobGPUStartTimes[jobKey]; exists {
		duration := time.Since(gpuInfo.startTime).Hours()
		gpuHours := duration * float64(gpuInfo.gpuCount)

		metrics.RecordAcceleratorHoursConsumed(framework, gpuInfo.gpuType, imageAnalysis.ImageSource, gpuHours)
		metrics.DecrementAcceleratorUtilization(framework, gpuInfo.gpuType, float64(gpuInfo.gpuCount))

		delete(jobGPUStartTimes, jobKey)
	}
	gpuMutex.Unlock()

	// Track job run duration
	jsMutex.RLock()
	startTime, exists := jobStartTimes[jobKey]
	jsMutex.RUnlock()

	if exists {
		runDuration := time.Since(startTime).Seconds()
		status := "failed"
		if succeeded {
			status = "succeeded"
		}

		metrics.RecordJobRunDuration(framework, imageAnalysis.ImageSource, status, runDuration)
	}

	// Compliance metric: job completion
	status := "failed"
	if succeeded {
		status = "succeeded"
	}
	metrics.RecordJobCompletion(framework, status)

	// Clean up tracking
	jsMutex.Lock()
	delete(jobStartTimes, jobKey)
	jsMutex.Unlock()
}

// Record job failure metrics with classification
func recordJobFailureMetrics(details JobDetails, metadata map[string]string) {
	imageAnalysis := analyzers.AnalyzeContainerImage(details.ContainerImage)

	// Extract failure reason from metadata
	failureReason := "unknown"
	if reason, ok := metadata["failure_reason"]; ok {
		failureReason = analyzers.ClassifyJobFailure(reason)
	}

	metrics.RecordJobFailureReason(details.Framework, failureReason, imageAnalysis.ImageSource)
}

func recordJobDeletionMetrics(details JobDetails, framework string) {
	imageAnalysis := analyzers.AnalyzeContainerImage(details.ContainerImage)
	jobKey := fmt.Sprintf("%s/%s", details.Namespace, details.JobName)

	// Ensure we decrement active counts if job was using RHOAI
	if imageAnalysis.ImageSource == "rhoai_official" {
		// Safe to call even if already decremented
		metrics.DecrementActiveRHOAIVersion(framework, imageAnalysis.RHOAIVersion)
	}

	// Clean up GPU tracking
	gpuMutex.Lock()
	if gpuInfo, exists := jobGPUStartTimes[jobKey]; exists {
		metrics.DecrementAcceleratorUtilization(framework, gpuInfo.gpuType, float64(gpuInfo.gpuCount))
		delete(jobGPUStartTimes, jobKey)
	}
	gpuMutex.Unlock()

	// Clean up any tracking
	jsMutex.Lock()
	delete(jobStartTimes, jobKey)
	jsMutex.Unlock()
}
