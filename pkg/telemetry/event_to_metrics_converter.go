// pkg/telemetry/event_to_metrics_converter.go
package telemetry

import (
	"context"
	"fmt"
	"strings"
	"time"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	"github.com/kubeflow/training-operator/pkg/telemetry/analyzers"
	"github.com/kubeflow/training-operator/pkg/telemetry/metrics"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

// JobDetails contains extracted job information
type JobDetails struct {
	Name             string
	Namespace        string
	Framework        string
	ImageSource      string
	RHOAIVersion     string
	AcceleratorType  string
	AcceleratorCount float64
	IsKueueManaged   bool
	KueueName        string
	StartTime        *metav1.Time
	CompletionTime   *metav1.Time
}

// JobKey returns a unique identifier for the job
func (d JobDetails) JobKey() string {
	return fmt.Sprintf("%s/%s/%s", d.Framework, d.Namespace, d.Name)
}

// convertEventToMetrics is the main entry point with panic recovery
func convertEventToMetrics(ctx context.Context, event JobEventData) {
	// Critical: Recover from panics to prevent controller crashes
	defer func() {
		if r := recover(); r != nil {
			klog.Errorf("Panic in telemetry converter: %v", r)
			metrics.ReconcileErrors.WithLabelValues("telemetry_converter").Inc()
		}
	}()

	// Extract job details based on framework
	details := extractJobDetails(event.Job, event.Framework)
	if details == nil {
		klog.V(4).Infof("Could not extract job details for framework %s", event.Framework)
		return
	}

	// Validate to prevent cardinality explosion
	if err := validateJobDetails(details); err != nil {
		klog.Warningf("Invalid job details: %v", err)
		return
	}

	// Process event based on type
	switch event.EventType {
	case JobCreatedEvent:
		recordJobCreationMetrics(*details)
	case JobStartedEvent:
		recordJobStartedMetrics(*details)
	case JobCompletedEvent:
		recordJobCompletionMetrics(*details, true)
	case JobFailedEvent:
		recordJobFailureMetrics(*details, event.Metadata["reason"])
	case JobDeletedEvent:
		recordJobDeletionMetrics(*details)
	}
}

// extractJobDetails extracts common details from different job types
func extractJobDetails(job interface{}, framework string) *JobDetails {
	switch framework {
	case "pytorch":
		return extractPyTorchJobDetails(job)
	case "tensorflow":
		return extractTFJobDetails(job)
	case "mpi":
		return extractMPIJobDetails(job)
	case "xgboost":
		return extractXGBoostJobDetails(job)
	case "jax":
		return extractJAXJobDetails(job)
	default:
		return nil
	}
}

// extractPyTorchJobDetails extracts details from PyTorchJob
func extractPyTorchJobDetails(job interface{}) *JobDetails {
	pytorchJob, ok := job.(*kubeflowv1.PyTorchJob)
	if !ok {
		return nil
	}

	details := &JobDetails{
		Name:           pytorchJob.Name,
		Namespace:      pytorchJob.Namespace,
		Framework:      "pytorch",
		StartTime:      pytorchJob.Status.StartTime,
		CompletionTime: pytorchJob.Status.CompletionTime,
	}

	// Analyze container image from Master replica
	if master, ok := pytorchJob.Spec.PyTorchReplicaSpecs[kubeflowv1.PyTorchJobReplicaTypeMaster]; ok {
		if len(master.Template.Spec.Containers) > 0 {
			image := master.Template.Spec.Containers[0].Image
			analysis := analyzers.AnalyzeContainerImage(image)
			details.ImageSource = analysis.ImageSource
			details.RHOAIVersion = analysis.RHOAIVersion
			details.AcceleratorType = analysis.AcceleratorType
		}
	}

	// Check for Kueue integration
	if pytorchJob.Annotations != nil {
		if workload, ok := pytorchJob.Annotations["kueue.x-k8s.io/workload"]; ok && workload != "" {
			details.IsKueueManaged = true
			details.KueueName = pytorchJob.Annotations["kueue.x-k8s.io/queue-name"]
			if details.KueueName == "" {
				details.KueueName = "default"
			}
		}
	}

	// Count accelerators
	details.AcceleratorCount = countAccelerators(pytorchJob.Spec.PyTorchReplicaSpecs)

	return details
}

// extractTFJobDetails extracts details from TFJob
func extractTFJobDetails(job interface{}) *JobDetails {
	tfJob, ok := job.(*kubeflowv1.TFJob)
	if !ok {
		return nil
	}

	details := &JobDetails{
		Name:           tfJob.Name,
		Namespace:      tfJob.Namespace,
		Framework:      "tensorflow",
		StartTime:      tfJob.Status.StartTime,
		CompletionTime: tfJob.Status.CompletionTime,
	}

	// Analyze container image from Chief replica
	if chief, ok := tfJob.Spec.TFReplicaSpecs[kubeflowv1.TFJobReplicaTypeChief]; ok {
		if len(chief.Template.Spec.Containers) > 0 {
			image := chief.Template.Spec.Containers[0].Image
			analysis := analyzers.AnalyzeContainerImage(image)
			details.ImageSource = analysis.ImageSource
			details.RHOAIVersion = analysis.RHOAIVersion
			details.AcceleratorType = analysis.AcceleratorType
		}
	} else if worker, ok := tfJob.Spec.TFReplicaSpecs[kubeflowv1.TFJobReplicaTypeWorker]; ok {
		if len(worker.Template.Spec.Containers) > 0 {
			image := worker.Template.Spec.Containers[0].Image
			analysis := analyzers.AnalyzeContainerImage(image)
			details.ImageSource = analysis.ImageSource
			details.RHOAIVersion = analysis.RHOAIVersion
			details.AcceleratorType = analysis.AcceleratorType
		}
	}

	// Check for Kueue integration
	if tfJob.Annotations != nil {
		if workload, ok := tfJob.Annotations["kueue.x-k8s.io/workload"]; ok && workload != "" {
			details.IsKueueManaged = true
			details.KueueName = tfJob.Annotations["kueue.x-k8s.io/queue-name"]
			if details.KueueName == "" {
				details.KueueName = "default"
			}
		}
	}

	return details
}

// Similar extractors for other job types...
func extractMPIJobDetails(job interface{}) *JobDetails {
	mpiJob, ok := job.(*kubeflowv1.MPIJob)
	if !ok {
		return nil
	}

	details := &JobDetails{
		Name:           mpiJob.Name,
		Namespace:      mpiJob.Namespace,
		Framework:      "mpi",
		StartTime:      mpiJob.Status.StartTime,
		CompletionTime: mpiJob.Status.CompletionTime,
	}

	if launcher, ok := mpiJob.Spec.MPIReplicaSpecs[kubeflowv1.MPIJobReplicaTypeLauncher]; ok {
		if len(launcher.Template.Spec.Containers) > 0 {
			image := launcher.Template.Spec.Containers[0].Image
			analysis := analyzers.AnalyzeContainerImage(image)
			details.ImageSource = analysis.ImageSource
			details.RHOAIVersion = analysis.RHOAIVersion
			details.AcceleratorType = analysis.AcceleratorType
		}
	}

	return details
}

func extractXGBoostJobDetails(job interface{}) *JobDetails {
	xgboostJob, ok := job.(*kubeflowv1.XGBoostJob)
	if !ok {
		return nil
	}

	details := &JobDetails{
		Name:           xgboostJob.Name,
		Namespace:      xgboostJob.Namespace,
		Framework:      "xgboost",
		StartTime:      xgboostJob.Status.StartTime,
		CompletionTime: xgboostJob.Status.CompletionTime,
	}

	if master, ok := xgboostJob.Spec.XGBReplicaSpecs[kubeflowv1.XGBoostJobReplicaTypeMaster]; ok {
		if len(master.Template.Spec.Containers) > 0 {
			image := master.Template.Spec.Containers[0].Image
			analysis := analyzers.AnalyzeContainerImage(image)
			details.ImageSource = analysis.ImageSource
			details.RHOAIVersion = analysis.RHOAIVersion
			details.AcceleratorType = analysis.AcceleratorType
		}
	}

	return details
}

func extractJAXJobDetails(job interface{}) *JobDetails {
	jaxJob, ok := job.(*kubeflowv1.JAXJob)
	if !ok {
		return nil
	}

	details := &JobDetails{
		Name:           jaxJob.Name,
		Namespace:      jaxJob.Namespace,
		Framework:      "jax",
		StartTime:      jaxJob.Status.StartTime,
		CompletionTime: jaxJob.Status.CompletionTime,
	}

	if worker, ok := jaxJob.Spec.JAXReplicaSpecs[kubeflowv1.JAXJobReplicaTypeWorker]; ok {
		if len(worker.Template.Spec.Containers) > 0 {
			image := worker.Template.Spec.Containers[0].Image
			analysis := analyzers.AnalyzeContainerImage(image)
			details.ImageSource = analysis.ImageSource
			details.RHOAIVersion = analysis.RHOAIVersion
			details.AcceleratorType = analysis.AcceleratorType
		}
	}

	return details
}

// validateJobDetails validates job details to prevent cardinality explosion
func validateJobDetails(details *JobDetails) error {
	// Limit label values
	validSources := map[string]bool{
		"rhoai_official": true,
		"community":      true,
		"custom":         true,
		"unknown":        true,
	}

	if !validSources[details.ImageSource] {
		details.ImageSource = "unknown"
	}

	// Limit framework values
	validFrameworks := map[string]bool{
		"pytorch":    true,
		"tensorflow": true,
		"mpi":        true,
		"xgboost":    true,
		"jax":        true,
		"paddle":     true,
	}

	if !validFrameworks[details.Framework] {
		return fmt.Errorf("invalid framework: %s", details.Framework)
	}

	return nil
}

// recordJobCreationMetrics records metrics when a job is created
func recordJobCreationMetrics(details JobDetails) {
	// Update counters
	metrics.TrainingJobsCreated.WithLabelValues(details.Framework).Inc()
	metrics.TrainingJobsByImageSource.WithLabelValues(details.ImageSource).Inc()
	metrics.TrainingJobsActive.WithLabelValues(details.Framework).Inc()

	// Track start time for duration calculation
	metrics.RecordJobStart(details.JobKey())
}

// recordJobStartedMetrics records metrics when a job starts running
func recordJobStartedMetrics(details JobDetails) {
	// Job is already tracked from creation
	if _, exists := metrics.GetJobStartTime(details.JobKey()); !exists {
		metrics.RecordJobStart(details.JobKey())
	}
}

// recordJobCompletionMetrics records metrics when a job completes
func recordJobCompletionMetrics(details JobDetails, succeeded bool) {
	status := "failed"
	if succeeded {
		status = "succeeded"
	}

	// Update completion counter
	metrics.TrainingJobsCompleted.WithLabelValues(status).Inc()

	// Decrement active jobs
	metrics.TrainingJobsActive.WithLabelValues(details.Framework).Dec()

	// Clean up tracking
	metrics.RemoveJobTracking(details.JobKey())
}

// recordJobFailureMetrics records metrics when a job fails
func recordJobFailureMetrics(details JobDetails, reason string) {
	// Record as failed completion
	recordJobCompletionMetrics(details, false)

	// Classify failure for internal tracking (not exported to telemetry)
	classifiedReason := classifyFailureReason(reason)
	metrics.internalFailureReasons.WithLabelValues(details.Framework, classifiedReason).Inc()
}

// recordJobDeletionMetrics records metrics when a job is deleted
func recordJobDeletionMetrics(details JobDetails) {
	// Decrement active jobs if still active
	metrics.TrainingJobsActive.WithLabelValues(details.Framework).Dec()

	// Clean up all tracking (CRITICAL: prevents memory leak)
	metrics.RemoveJobTracking(details.JobKey())
}

// classifyFailureReason categorizes failure reasons
func classifyFailureReason(reason string) string {
	if reason == "" {
		return "unknown"
	}

	reasonLower := strings.ToLower(reason)

	switch {
	case strings.Contains(reasonLower, "oomkilled"):
		return "oom"
	case strings.Contains(reasonLower, "imagepull"):
		return "image_pull"
	case strings.Contains(reasonLower, "insufficient"):
		return "insufficient_resources"
	case strings.Contains(reasonLower, "deadline"):
		return "timeout"
	case strings.Contains(reasonLower, "evicted"):
		return "evicted"
	default:
		return "user_error"
	}
}

// countAccelerators counts total accelerators across all replicas
func countAccelerators(replicaSpecs map[kubeflowv1.ReplicaType]*kubeflowv1.ReplicaSpec) float64 {
	var total float64

	for _, spec := range replicaSpecs {
		if spec == nil || spec.Replicas == nil {
			continue
		}

		replicas := float64(*spec.Replicas)
		for _, container := range spec.Template.Spec.Containers {
			if container.Resources.Requests != nil {
				// Check for GPU resources
				if gpuQty, ok := container.Resources.Requests[corev1.ResourceName("nvidia.com/gpu")]; ok {
					total += replicas * float64(gpuQty.Value())
				}
				if gpuQty, ok := container.Resources.Requests[corev1.ResourceName("amd.com/gpu")]; ok {
					total += replicas * float64(gpuQty.Value())
				}
			}
		}
	}

	return total
}