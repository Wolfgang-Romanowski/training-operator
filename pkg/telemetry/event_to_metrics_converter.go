package telemetry

import (
	"context"
	"fmt"
	"sync"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	"github.com/kubeflow/training-operator/pkg/telemetry/analyzers"
	"github.com/kubeflow/training-operator/pkg/telemetry/metrics"
)

var (
	namespaceVersions = make(map[string]string)
	nvMutex           sync.RWMutex
)

type JobDetails struct {
	ContainerImage string
	Namespace      string
	JobName        string
	Framework      string
	ReplicaCount   int
	GPURequested   int
}

// The ctx parameter is kept for future extensibility
func convertEventToMetrics(ctx context.Context, event JobEventData) {
	jobDetails := extractJobDetails(event.Job)
	jobDetails.Framework = event.Framework // Ensure framework is set

	switch event.EventType {
	case JobCreatedEvent:
		recordJobCreationMetrics(jobDetails, event.Framework)
	case JobCompletedEvent:
		recordJobCompletionMetrics(jobDetails, event.Framework, true)
	case JobFailedEvent:
		recordJobCompletionMetrics(jobDetails, event.Framework, false)
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

		for replicaType, replicaSpec := range v.Spec.PyTorchReplicaSpecs {
			if replicaSpec.Replicas != nil {
				details.ReplicaCount += int(*replicaSpec.Replicas)
			}

			if replicaType == kubeflowv1.PyTorchJobReplicaTypeMaster || replicaType == kubeflowv1.PyTorchJobReplicaTypeWorker {
				if len(replicaSpec.Template.Spec.Containers) > 0 {
					if details.ContainerImage == "" {
						details.ContainerImage = replicaSpec.Template.Spec.Containers[0].Image
					}

					for _, container := range replicaSpec.Template.Spec.Containers {
						if container.Resources.Limits != nil {
							if gpuQuantity, ok := container.Resources.Limits["nvidia.com/gpu"]; ok {
								details.GPURequested += int(gpuQuantity.Value())
							}
						}
					}
				}
			}
		}

	case *kubeflowv1.TFJob:
		details.JobName = v.Name
		details.Namespace = v.Namespace
		details.ReplicaCount = 0

		// Priority: Chief > Worker > PS
		if chief, ok := v.Spec.TFReplicaSpecs[kubeflowv1.TFJobReplicaTypeChief]; ok {
			if len(chief.Template.Spec.Containers) > 0 {
				details.ContainerImage = chief.Template.Spec.Containers[0].Image
			}
		}

		if details.ContainerImage == "" {
			if worker, ok := v.Spec.TFReplicaSpecs[kubeflowv1.TFJobReplicaTypeWorker]; ok {
				if len(worker.Template.Spec.Containers) > 0 {
					details.ContainerImage = worker.Template.Spec.Containers[0].Image
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

		if v.Spec.MPIReplicaSpecs != nil {
			if launcherSpec, ok := v.Spec.MPIReplicaSpecs[kubeflowv1.MPIJobReplicaTypeLauncher]; ok {
				if len(launcherSpec.Template.Spec.Containers) > 0 {
					details.ContainerImage = launcherSpec.Template.Spec.Containers[0].Image
				}
			}
			if workerSpec, ok := v.Spec.MPIReplicaSpecs[kubeflowv1.MPIJobReplicaTypeWorker]; ok {
				if workerSpec.Replicas != nil {
					details.ReplicaCount = int(*workerSpec.Replicas)
				}
			}
		}

	case *kubeflowv1.XGBoostJob:
		details.JobName = v.Name
		details.Namespace = v.Namespace

		// Priority: Master > Worker
		if master, ok := v.Spec.XGBReplicaSpecs[kubeflowv1.XGBoostJobReplicaTypeMaster]; ok {
			if len(master.Template.Spec.Containers) > 0 {
				details.ContainerImage = master.Template.Spec.Containers[0].Image
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

		// Priority: Worker > any other
		if worker, ok := v.Spec.JAXReplicaSpecs[kubeflowv1.JAXJobReplicaTypeWorker]; ok {
			if len(worker.Template.Spec.Containers) > 0 {
				details.ContainerImage = worker.Template.Spec.Containers[0].Image
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

		// Priority: Master > Worker
		if master, ok := v.Spec.PaddleReplicaSpecs[kubeflowv1.PaddleJobReplicaTypeMaster]; ok {
			if len(master.Template.Spec.Containers) > 0 {
				details.ContainerImage = master.Template.Spec.Containers[0].Image
			}
		}

		if details.ContainerImage == "" {
			if worker, ok := v.Spec.PaddleReplicaSpecs[kubeflowv1.PaddleJobReplicaTypeWorker]; ok {
				if len(worker.Template.Spec.Containers) > 0 {
					details.ContainerImage = worker.Template.Spec.Containers[0].Image
				}
			}
		}

		for _, replicaSpec := range v.Spec.PaddleReplicaSpecs {
			if replicaSpec.Replicas != nil {
				details.ReplicaCount += int(*replicaSpec.Replicas)
			}
		}
	}

	return details
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

	// Compliance metric: total jobs
	metrics.RecordJobCreated(framework)

	// Track job start time
	jobKey := fmt.Sprintf("%s/%s", details.Namespace, details.JobName)
	metrics.RecordJobStart(jobKey)
}

func recordJobCompletionMetrics(details JobDetails, framework string, succeeded bool) {
	imageAnalysis := analyzers.AnalyzeContainerImage(details.ContainerImage)

	// Decrement active RHOAI version count
	if imageAnalysis.ImageSource == "rhoai_official" {
		metrics.DecrementActiveRHOAIVersion(framework, imageAnalysis.RHOAIVersion)
	}

	// Compliance metric: job completion
	status := "failed"
	if succeeded {
		status = "succeeded"
	}
	metrics.RecordJobCompletion(framework, status)

	// Clean up tracking
	jobKey := fmt.Sprintf("%s/%s", details.Namespace, details.JobName)
	metrics.RemoveJobStartTime(jobKey)
}

func recordJobDeletionMetrics(details JobDetails, framework string) {
	imageAnalysis := analyzers.AnalyzeContainerImage(details.ContainerImage)

	// Ensure we decrement active counts if job was using RHOAI
	if imageAnalysis.ImageSource == "rhoai_official" {
		// Safe to call even if already decremented
		metrics.DecrementActiveRHOAIVersion(framework, imageAnalysis.RHOAIVersion)
	}

	// Clean up any tracking
	jobKey := fmt.Sprintf("%s/%s", details.Namespace, details.JobName)
	metrics.RemoveJobStartTime(jobKey)
}
