// pkg/telemetry/collectors/crd_collector.go
// Bridge collector that coordinates image version tracking for deprecation decisions
package collectors

import (
	"context"
	"strings"
	"sync"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	"github.com/kubeflow/training-operator/pkg/telemetry/analyzers"
	"github.com/kubeflow/training-operator/pkg/telemetry/metrics"
	"k8s.io/klog/v2"
)

// CRDCollector bridges events to the metric tracking system
type CRDCollector struct {
	initialized bool
	mu          sync.RWMutex
}

// JobEventData represents job event information
type JobEventData struct {
	EventType    string
	Framework    string
	Job          interface{}
	JobName      string
	JobNamespace string
	Metadata     map[string]string
}

var (
	crdCollector *CRDCollector
	crdOnce      sync.Once
)

// InitializeCRDCollector sets up the CRD collector
func InitializeCRDCollector() *CRDCollector {
	crdOnce.Do(func() {
		crdCollector = &CRDCollector{
			initialized: true,
		}

		// Initialize the metrics system
		metrics.InitializeMetrics()

		klog.Info("CRD collector initialized for image version tracking")
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

// Initialize initializes all collectors (for backward compatibility)
func Initialize() error {
	InitializeCRDCollector()
	return nil
}

// ProcessJobCreation handles job creation events for version tracking
func (c *CRDCollector) ProcessJobCreation(ctx context.Context, event JobEventData) error {
	if !c.initialized {
		klog.Warning("CRD collector not initialized")
		return nil
	}

	// Extract container image for version analysis
	image := c.extractContainerImage(event.Job, event.Framework)
	if image == "" {
		klog.V(3).Infof("Could not extract image for job %s/%s", event.JobNamespace, event.JobName)
		image = "unknown"
	}

	// Analyze image to determine version and source
	imageAnalysis := analyzers.AnalyzeContainerImage(image)

	// Classify customer type for adoption tracking
	customerInfo := metrics.ClassifyCustomer(event.JobNamespace, event.Job)

	// Record job creation with image version tracking
	// This is the key for deprecation decisions
	metrics.RecordJobCreation(
		event.Framework,
		imageAnalysis.RHOAIVersion, // e.g., "pytorch-2.4", "tensorflow-2.15"
		imageAnalysis.ImageSource,  // "rhoai_official", "community", "custom"
		customerInfo.CustomerType,  // "enterprise" or "non-enterprise"
		event.JobNamespace,
		event.JobName,
	)

	klog.V(2).Infof("Tracked job creation: %s/%s - version: %s, source: %s, customer: %s",
		event.JobNamespace, event.JobName,
		imageAnalysis.RHOAIVersion, imageAnalysis.ImageSource, customerInfo.CustomerType)

	return nil
}

// ProcessJobDeletion handles job deletion events
func (c *CRDCollector) ProcessJobDeletion(ctx context.Context, event JobEventData) error {
	if !c.initialized {
		return nil
	}

	// Extract image to determine version for cleanup
	image := c.extractContainerImage(event.Job, event.Framework)
	if image != "" {
		imageAnalysis := analyzers.AnalyzeContainerImage(image)

		// Record job deletion to update version tracking
		metrics.RecordJobDeletion(
			event.Framework,
			imageAnalysis.RHOAIVersion,
			event.JobNamespace,
			event.JobName,
		)

		klog.V(3).Infof("Tracked job deletion: %s/%s - version: %s",
			event.JobNamespace, event.JobName, imageAnalysis.RHOAIVersion)
	}

	return nil
}

// ProcessJobCompletion handles job completion events (for future use)
func (c *CRDCollector) ProcessJobCompletion(ctx context.Context, event JobEventData, succeeded bool) error {
	// Currently not tracking completion metrics, but kept for interface compatibility
	status := "failed"
	if succeeded {
		status = "succeeded"
	}

	klog.V(4).Infof("Job %s/%s completed with status: %s",
		event.JobNamespace, event.JobName, status)

	return nil
}

// ProcessJobStart handles job start events (for future use)
func (c *CRDCollector) ProcessJobStart(ctx context.Context, event JobEventData) error {
	// Currently not tracking start events, but kept for interface compatibility
	klog.V(4).Infof("Job %s/%s started", event.JobNamespace, event.JobName)
	return nil
}

// extractContainerImage extracts the container image from various job types
func (c *CRDCollector) extractContainerImage(job interface{}, framework string) string {
	frameworkLower := strings.ToLower(framework)

	switch frameworkLower {
	case "pytorch":
		return c.extractPyTorchImage(job)
	case "tensorflow":
		return c.extractTensorFlowImage(job)
	case "mpi":
		return c.extractMPIImage(job)
	case "xgboost":
		return c.extractXGBoostImage(job)
	case "paddle":
		return c.extractPaddleImage(job)
	case "jax":
		return c.extractJAXImage(job)
	default:
		klog.V(4).Infof("Unknown framework for image extraction: %s", framework)
		return ""
	}
}

// extractPyTorchImage extracts image from PyTorchJob
func (c *CRDCollector) extractPyTorchImage(job interface{}) string {
	pytorchJob, ok := job.(*kubeflowv1.PyTorchJob)
	if !ok {
		return ""
	}

	if pytorchJob.Spec.PyTorchReplicaSpecs != nil {
		// Try Master first, then Worker
		for _, replicaType := range []kubeflowv1.ReplicaType{
			kubeflowv1.PyTorchJobReplicaTypeMaster,
			kubeflowv1.PyTorchJobReplicaTypeWorker,
		} {
			if replica, ok := pytorchJob.Spec.PyTorchReplicaSpecs[replicaType]; ok {
				if len(replica.Template.Spec.Containers) > 0 {
					return replica.Template.Spec.Containers[0].Image
				}
			}
		}
	}
	return ""
}

// extractTensorFlowImage extracts image from TFJob
func (c *CRDCollector) extractTensorFlowImage(job interface{}) string {
	tfJob, ok := job.(*kubeflowv1.TFJob)
	if !ok {
		return ""
	}

	if tfJob.Spec.TFReplicaSpecs != nil {
		// Try Chief first, then Worker
		for _, replicaType := range []kubeflowv1.ReplicaType{
			kubeflowv1.TFJobReplicaTypeChief,
			kubeflowv1.TFJobReplicaTypeWorker,
		} {
			if replica, ok := tfJob.Spec.TFReplicaSpecs[replicaType]; ok {
				if len(replica.Template.Spec.Containers) > 0 {
					return replica.Template.Spec.Containers[0].Image
				}
			}
		}
	}
	return ""
}

// extractMPIImage extracts image from MPIJob
func (c *CRDCollector) extractMPIImage(job interface{}) string {
	mpiJob, ok := job.(*kubeflowv1.MPIJob)
	if !ok {
		return ""
	}

	if mpiJob.Spec.MPIReplicaSpecs != nil {
		if launcher, ok := mpiJob.Spec.MPIReplicaSpecs[kubeflowv1.MPIJobReplicaTypeLauncher]; ok {
			if len(launcher.Template.Spec.Containers) > 0 {
				return launcher.Template.Spec.Containers[0].Image
			}
		}
	}
	return ""
}

// extractXGBoostImage extracts image from XGBoostJob
func (c *CRDCollector) extractXGBoostImage(job interface{}) string {
	xgboostJob, ok := job.(*kubeflowv1.XGBoostJob)
	if !ok {
		return ""
	}

	if xgboostJob.Spec.XGBReplicaSpecs != nil {
		if master, ok := xgboostJob.Spec.XGBReplicaSpecs[kubeflowv1.XGBoostJobReplicaTypeMaster]; ok {
			if len(master.Template.Spec.Containers) > 0 {
				return master.Template.Spec.Containers[0].Image
			}
		}
	}
	return ""
}

// extractPaddleImage extracts image from PaddleJob
func (c *CRDCollector) extractPaddleImage(job interface{}) string {
	paddleJob, ok := job.(*kubeflowv1.PaddleJob)
	if !ok {
		return ""
	}

	if paddleJob.Spec.PaddleReplicaSpecs != nil {
		if master, ok := paddleJob.Spec.PaddleReplicaSpecs[kubeflowv1.PaddleJobReplicaTypeMaster]; ok {
			if len(master.Template.Spec.Containers) > 0 {
				return master.Template.Spec.Containers[0].Image
			}
		}
	}
	return ""
}

// extractJAXImage extracts image from JAXJob
func (c *CRDCollector) extractJAXImage(job interface{}) string {
	jaxJob, ok := job.(*kubeflowv1.JAXJob)
	if !ok {
		return ""
	}

	if jaxJob.Spec.JAXReplicaSpecs != nil {
		if worker, ok := jaxJob.Spec.JAXReplicaSpecs[kubeflowv1.JAXJobReplicaTypeWorker]; ok {
			if len(worker.Template.Spec.Containers) > 0 {
				return worker.Template.Spec.Containers[0].Image
			}
		}
	}
	return ""
}

// GetActiveInstanceCount returns the number of active jobs being tracked
func (c *CRDCollector) GetActiveInstanceCount() int {
	// Delegate to metric_definitions.go
	return metrics.GetActiveJobCount()
}

// GetActiveInstancesByFramework returns version distribution
func (c *CRDCollector) GetActiveInstancesByFramework() map[string]int {
	// Delegate to metric_definitions.go
	return metrics.GetVersionDistribution()
}

// GetMetricsSummary returns a summary of all metrics
func (c *CRDCollector) GetMetricsSummary() map[string]interface{} {
	return metrics.GetMetricsSummary()
}
