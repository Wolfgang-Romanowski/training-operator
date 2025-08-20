// pkg/telemetry/event_to_metrics_converter.go
// FIXED: Properly tracks image versions with 10 timeseries limit compliance
package telemetry

import (
	"context"
	"strings"
	"time"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
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

	// Analyze customer type (binary classification for compliance)
	customerInfo := classifyCustomerUsage(namespace, event.Job)
	customerType := "non-enterprise"
	if customerInfo != nil {
		// Strict binary classification to keep under 10 timeseries
		if strings.Contains(strings.ToLower(customerInfo.CustomerType), "enterprise") ||
			strings.Contains(strings.ToLower(customerInfo.CustomerType), "prod") ||
			strings.Contains(strings.ToLower(namespace), "prod") {
			customerType = "enterprise"
		}
	}

	// Extract and analyze container image
	image := extractContainerImage(event.Job, framework)
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

	// Extract image to determine version for cleanup
	image := extractContainerImage(event.Job, framework)
	if image != "" {
		imageAnalysis := analyzers.AnalyzeContainerImage(image)
		metrics.RemoveImageVersion(framework, imageAnalysis.RHOAIVersion, namespace, name)
	}

	klog.V(3).Infof("Job %s/%s deleted", namespace, name)
}

// extractContainerImage extracts the container image from various job types
func extractContainerImage(job interface{}, framework string) string {
	switch framework {
	case "pytorch":
		if pytorchJob, ok := job.(*kubeflowv1.PyTorchJob); ok {
			return extractPyTorchImage(pytorchJob)
		}
	case "tensorflow":
		if tfJob, ok := job.(*kubeflowv1.TFJob); ok {
			return extractTensorFlowImage(tfJob)
		}
	case "mpi":
		if mpiJob, ok := job.(*kubeflowv1.MPIJob); ok {
			return extractMPIImage(mpiJob)
		}
	case "xgboost":
		if xgboostJob, ok := job.(*kubeflowv1.XGBoostJob); ok {
			return extractXGBoostImage(xgboostJob)
		}
	case "paddle":
		if paddleJob, ok := job.(*kubeflowv1.PaddleJob); ok {
			return extractPaddleImage(paddleJob)
		}
	case "jax":
		if jaxJob, ok := job.(*kubeflowv1.JAXJob); ok {
			return extractJAXImage(jaxJob)
		}
	}

	klog.V(4).Infof("Could not extract image for framework %s", framework)
	return ""
}

// Framework-specific image extraction functions
func extractPyTorchImage(job *kubeflowv1.PyTorchJob) string {
	if job.Spec.PyTorchReplicaSpecs != nil {
		// Try Master first, then Worker
		for _, replicaType := range []kubeflowv1.ReplicaType{
			kubeflowv1.PyTorchJobReplicaTypeMaster,
			kubeflowv1.PyTorchJobReplicaTypeWorker,
		} {
			if replica, ok := job.Spec.PyTorchReplicaSpecs[replicaType]; ok {
				if len(replica.Template.Spec.Containers) > 0 {
					return replica.Template.Spec.Containers[0].Image
				}
			}
		}
	}
	return ""
}

func extractTensorFlowImage(job *kubeflowv1.TFJob) string {
	if job.Spec.TFReplicaSpecs != nil {
		// Try Chief first, then Worker
		for _, replicaType := range []kubeflowv1.ReplicaType{
			kubeflowv1.TFJobReplicaTypeChief,
			kubeflowv1.TFJobReplicaTypeWorker,
		} {
			if replica, ok := job.Spec.TFReplicaSpecs[replicaType]; ok {
				if len(replica.Template.Spec.Containers) > 0 {
					return replica.Template.Spec.Containers[0].Image
				}
			}
		}
	}
	return ""
}

func extractMPIImage(job *kubeflowv1.MPIJob) string {
	if job.Spec.MPIReplicaSpecs != nil {
		if launcher, ok := job.Spec.MPIReplicaSpecs[kubeflowv1.MPIJobReplicaTypeLauncher]; ok {
			if len(launcher.Template.Spec.Containers) > 0 {
				return launcher.Template.Spec.Containers[0].Image
			}
		}
	}
	return ""
}

func extractXGBoostImage(job *kubeflowv1.XGBoostJob) string {
	if job.Spec.XGBReplicaSpecs != nil {
		if master, ok := job.Spec.XGBReplicaSpecs[kubeflowv1.XGBoostJobReplicaTypeMaster]; ok {
			if len(master.Template.Spec.Containers) > 0 {
				return master.Template.Spec.Containers[0].Image
			}
		}
	}
	return ""
}

func extractPaddleImage(job *kubeflowv1.PaddleJob) string {
	if job.Spec.PaddleReplicaSpecs != nil {
		if master, ok := job.Spec.PaddleReplicaSpecs[kubeflowv1.PaddleJobReplicaTypeMaster]; ok {
			if len(master.Template.Spec.Containers) > 0 {
				return master.Template.Spec.Containers[0].Image
			}
		}
	}
	return ""
}

func extractJAXImage(job *kubeflowv1.JAXJob) string {
	if job.Spec.JAXReplicaSpecs != nil {
		if worker, ok := job.Spec.JAXReplicaSpecs[kubeflowv1.JAXJobReplicaTypeWorker]; ok {
			if len(worker.Template.Spec.Containers) > 0 {
				return worker.Template.Spec.Containers[0].Image
			}
		}
	}
	return ""
}
