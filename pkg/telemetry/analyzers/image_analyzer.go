// pkg/telemetry/analyzers/image_analyzer.go
// Enhanced image analysis for RHOAI runtime version tracking
package analyzers

import (
	"regexp"
	"strings"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
	"k8s.io/klog/v2"
)

// ImageAnalysisResult contains image analysis results
type ImageAnalysisResult struct {
	ImageSource     string // rhoai_official, community, custom
	RHOAIVersion    string // pytorch-2.5, tensorflow-2.16, etc
	AcceleratorType string // nvidia.com/gpu, amd.com/gpu, cpu
}

var (
	// Enhanced patterns to properly extract versions from RHOAI images
	rhoaiVersionPatterns = map[string]*regexp.Regexp{
		// PyTorch versions with better pattern matching
		"pytorch-2.5": regexp.MustCompile(`(?i)(registry\.redhat\.io/rhoai|quay\.io/modh|quay\.io/opendatahub).*pytorch.*2[\.-]5`),
		"pytorch-2.4": regexp.MustCompile(`(?i)(registry\.redhat\.io/rhoai|quay\.io/modh|quay\.io/opendatahub).*pytorch.*2[\.-]4`),
		"pytorch-2.3": regexp.MustCompile(`(?i)(registry\.redhat\.io/rhoai|quay\.io/modh|quay\.io/opendatahub).*pytorch.*2[\.-]3`),
		"pytorch-2.2": regexp.MustCompile(`(?i)(registry\.redhat\.io/rhoai|quay\.io/modh|quay\.io/opendatahub).*pytorch.*2[\.-]2`),
		"pytorch-2.1": regexp.MustCompile(`(?i)(registry\.redhat\.io/rhoai|quay\.io/modh|quay\.io/opendatahub).*pytorch.*2[\.-]1`),
		"pytorch-2.0": regexp.MustCompile(`(?i)(registry\.redhat\.io/rhoai|quay\.io/modh|quay\.io/opendatahub).*pytorch.*2[\.-]0`),

		// TensorFlow versions
		"tensorflow-2.16": regexp.MustCompile(`(?i)(registry\.redhat\.io/rhoai|quay\.io/modh).*tensorflow.*2[\.-]16`),
		"tensorflow-2.15": regexp.MustCompile(`(?i)(registry\.redhat\.io/rhoai|quay\.io/modh).*tensorflow.*2[\.-]15`),
		"tensorflow-2.14": regexp.MustCompile(`(?i)(registry\.redhat\.io/rhoai|quay\.io/modh).*tensorflow.*2[\.-]14`),
		"tensorflow-2.13": regexp.MustCompile(`(?i)(registry\.redhat\.io/rhoai|quay\.io/modh).*tensorflow.*2[\.-]13`),

		// Ray/CodeFlare versions
		"ray-2.9": regexp.MustCompile(`(?i)(registry\.redhat\.io/rhoai|quay\.io/modh).*ray.*2[\.-]9`),
		"ray-2.8": regexp.MustCompile(`(?i)(registry\.redhat\.io/rhoai|quay\.io/modh).*ray.*2[\.-]8`),
		"ray-2.7": regexp.MustCompile(`(?i)(registry\.redhat\.io/rhoai|quay\.io/modh).*ray.*2[\.-]7`),

		// UBI-based images with PyTorch
		"ubi9-pytorch": regexp.MustCompile(`(?i)registry\.redhat\.io/ubi9/python.*pytorch`),

		// Notebook images
		"notebook-pytorch":    regexp.MustCompile(`(?i)(registry\.redhat\.io/rhoai|quay\.io/modh).*notebook.*pytorch`),
		"notebook-tensorflow": regexp.MustCompile(`(?i)(registry\.redhat\.io/rhoai|quay\.io/modh).*notebook.*tensorflow`),
	}

	// Check for RHOAI official registries
	rhoaiRegistries = []string{
		"registry.redhat.io/rhoai",
		"quay.io/modh",
		"quay.io/opendatahub",
		"registry.redhat.io/ubi",
	}

	// Community image patterns
	communityPatterns = []string{
		"pytorch/pytorch",
		"tensorflow/tensorflow",
		"docker.io/pytorch",
		"docker.io/tensorflow",
		"nvcr.io/nvidia",
		"huggingface/",
		"jupyter/",
		"deepset/",
	}
)

// AnalyzeContainerImage analyzes a container image for telemetry
func AnalyzeContainerImage(image string) ImageAnalysisResult {
	if image == "" {
		return ImageAnalysisResult{
			ImageSource:     "unknown",
			RHOAIVersion:    "none",
			AcceleratorType: "cpu",
		}
	}

	imageLower := strings.ToLower(image)
	result := ImageAnalysisResult{
		AcceleratorType: detectAcceleratorType(imageLower),
	}

	// First check if it's from an RHOAI registry
	isRHOAI := false
	for _, registry := range rhoaiRegistries {
		if strings.Contains(imageLower, strings.ToLower(registry)) {
			isRHOAI = true
			break
		}
	}

	if isRHOAI {
		result.ImageSource = "rhoai_official"

		// Try to extract specific version
		versionFound := false
		for version, pattern := range rhoaiVersionPatterns {
			if pattern.MatchString(imageLower) {
				result.RHOAIVersion = version
				versionFound = true
				break
			}
		}

		if !versionFound {
			// RHOAI image but version not recognized - still valuable to track
			result.RHOAIVersion = "other"
		}

		return result
	}

	// Check if it's a community image
	for _, pattern := range communityPatterns {
		if strings.Contains(imageLower, pattern) {
			result.ImageSource = "community"
			result.RHOAIVersion = "none"
			return result
		}
	}

	// Default to custom
	result.ImageSource = "custom"
	result.RHOAIVersion = "none"
	return result
}

// detectAcceleratorType detects the accelerator type from image name
func detectAcceleratorType(imageLower string) string {
	acceleratorPatterns := map[string]string{
		"cuda":   "nvidia.com/gpu",
		"nvidia": "nvidia.com/gpu",
		"rocm":   "amd.com/gpu",
		"amd":    "amd.com/gpu",
		"habana": "habana.ai/gaudi",
		"gaudi":  "habana.ai/gaudi",
		"intel":  "intel.com/gpu",
		"gpu":    "nvidia.com/gpu", // Default GPU to NVIDIA
	}

	// Check each accelerator pattern
	for keyword, acceleratorType := range acceleratorPatterns {
		if strings.Contains(imageLower, keyword) {
			return acceleratorType
		}
	}

	return "cpu"
}

// GetImageRegistry extracts the registry from an image URL
func GetImageRegistry(image string) string {
	parts := strings.Split(image, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return "unknown"
}

// IsRHOAIImage checks if an image is an official RHOAI image
func IsRHOAIImage(image string) bool {
	imageLower := strings.ToLower(image)
	for _, registry := range rhoaiRegistries {
		if strings.Contains(imageLower, strings.ToLower(registry)) {
			return true
		}
	}
	return false
}

// ========================================================================
// CONSOLIDATED IMAGE EXTRACTION FUNCTIONS
// Single source of truth for extracting images from all job types
// ========================================================================

// ExtractContainerImage extracts the container image from various job types
// This consolidates all image extraction logic in one place
func ExtractContainerImage(job interface{}, framework string) string {
	frameworkLower := strings.ToLower(framework)

	switch frameworkLower {
	case "pytorch":
		if pytorchJob, ok := job.(*kubeflowv1.PyTorchJob); ok {
			return ExtractPyTorchImage(pytorchJob)
		}
	case "tensorflow":
		if tfJob, ok := job.(*kubeflowv1.TFJob); ok {
			return ExtractTensorFlowImage(tfJob)
		}
	case "mpi":
		if mpiJob, ok := job.(*kubeflowv1.MPIJob); ok {
			return ExtractMPIImage(mpiJob)
		}
	case "xgboost":
		if xgboostJob, ok := job.(*kubeflowv1.XGBoostJob); ok {
			return ExtractXGBoostImage(xgboostJob)
		}
	case "paddle":
		if paddleJob, ok := job.(*kubeflowv1.PaddleJob); ok {
			return ExtractPaddleImage(paddleJob)
		}
	case "jax":
		if jaxJob, ok := job.(*kubeflowv1.JAXJob); ok {
			return ExtractJAXImage(jaxJob)
		}
	}

	klog.V(4).Infof("Could not extract image for framework %s", framework)
	return ""
}

// ExtractPyTorchImage extracts image from PyTorchJob
func ExtractPyTorchImage(job *kubeflowv1.PyTorchJob) string {
	if job == nil || job.Spec.PyTorchReplicaSpecs == nil {
		return ""
	}

	// Try Master first, then Worker for consistency
	for _, replicaType := range []kubeflowv1.ReplicaType{
		kubeflowv1.PyTorchJobReplicaTypeMaster,
		kubeflowv1.PyTorchJobReplicaTypeWorker,
	} {
		if replica, ok := job.Spec.PyTorchReplicaSpecs[replicaType]; ok {
			if replica != nil && len(replica.Template.Spec.Containers) > 0 {
				return replica.Template.Spec.Containers[0].Image
			}
		}
	}
	return ""
}

// ExtractTensorFlowImage extracts image from TFJob
func ExtractTensorFlowImage(job *kubeflowv1.TFJob) string {
	if job == nil || job.Spec.TFReplicaSpecs == nil {
		return ""
	}

	// Try Chief first, then Worker
	for _, replicaType := range []kubeflowv1.ReplicaType{
		kubeflowv1.TFJobReplicaTypeChief,
		kubeflowv1.TFJobReplicaTypeWorker,
	} {
		if replica, ok := job.Spec.TFReplicaSpecs[replicaType]; ok {
			if replica != nil && len(replica.Template.Spec.Containers) > 0 {
				return replica.Template.Spec.Containers[0].Image
			}
		}
	}
	return ""
}

// ExtractMPIImage extracts image from MPIJob
func ExtractMPIImage(job *kubeflowv1.MPIJob) string {
	if job == nil || job.Spec.MPIReplicaSpecs == nil {
		return ""
	}

	if launcher, ok := job.Spec.MPIReplicaSpecs[kubeflowv1.MPIJobReplicaTypeLauncher]; ok {
		if launcher != nil && len(launcher.Template.Spec.Containers) > 0 {
			return launcher.Template.Spec.Containers[0].Image
		}
	}
	return ""
}

// ExtractXGBoostImage extracts image from XGBoostJob
func ExtractXGBoostImage(job *kubeflowv1.XGBoostJob) string {
	if job == nil || job.Spec.XGBReplicaSpecs == nil {
		return ""
	}

	if master, ok := job.Spec.XGBReplicaSpecs[kubeflowv1.XGBoostJobReplicaTypeMaster]; ok {
		if master != nil && len(master.Template.Spec.Containers) > 0 {
			return master.Template.Spec.Containers[0].Image
		}
	}
	return ""
}

// ExtractPaddleImage extracts image from PaddleJob
func ExtractPaddleImage(job *kubeflowv1.PaddleJob) string {
	if job == nil || job.Spec.PaddleReplicaSpecs == nil {
		return ""
	}

	if master, ok := job.Spec.PaddleReplicaSpecs[kubeflowv1.PaddleJobReplicaTypeMaster]; ok {
		if master != nil && len(master.Template.Spec.Containers) > 0 {
			return master.Template.Spec.Containers[0].Image
		}
	}
	return ""
}

// ExtractJAXImage extracts image from JAXJob
func ExtractJAXImage(job *kubeflowv1.JAXJob) string {
	if job == nil || job.Spec.JAXReplicaSpecs == nil {
		return ""
	}

	if worker, ok := job.Spec.JAXReplicaSpecs[kubeflowv1.JAXJobReplicaTypeWorker]; ok {
		if worker != nil && len(worker.Template.Spec.Containers) > 0 {
			return worker.Template.Spec.Containers[0].Image
		}
	}
	return ""
}
