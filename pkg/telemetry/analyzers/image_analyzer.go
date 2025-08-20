package analyzers

import (
	"regexp"
	"strings"

	"k8s.io/klog/v2"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
)

// ImageAnalysisResult contains image analysis results
type ImageAnalysisResult struct {
	ImageSource     string // rhoai_official, community, custom
	RHOAIVersion    string // pytorch-2.5, tensorflow-2.16, etc
	AcceleratorType string // nvidia.com/gpu, amd.com/gpu, cpu
}

var (
	rhoaiVersionPatterns = map[string]*regexp.Regexp{
		"pytorch-2.5": regexp.MustCompile(`(?i)(registry\.redhat\.io/rhoai|quay\.io/modh|quay\.io/opendatahub).*pytorch.*2[\.-]5`),
		"pytorch-2.4": regexp.MustCompile(`(?i)(registry\.redhat\.io/rhoai|quay\.io/modh|quay\.io/opendatahub).*pytorch.*2[\.-]4`),
		"pytorch-2.3": regexp.MustCompile(`(?i)(registry\.redhat\.io/rhoai|quay\.io/modh|quay\.io/opendatahub).*pytorch.*2[\.-]3`),
		"pytorch-2.2": regexp.MustCompile(`(?i)(registry\.redhat\.io/rhoai|quay\.io/modh|quay\.io/opendatahub).*pytorch.*2[\.-]2`),
		"pytorch-2.1": regexp.MustCompile(`(?i)(registry\.redhat\.io/rhoai|quay\.io/modh|quay\.io/opendatahub).*pytorch.*2[\.-]1`),
		"pytorch-2.0": regexp.MustCompile(`(?i)(registry\.redhat\.io/rhoai|quay\.io/modh|quay\.io/opendatahub).*pytorch.*2[\.-]0`),

		"tensorflow-2.16": regexp.MustCompile(`(?i)(registry\.redhat\.io/rhoai|quay\.io/modh).*tensorflow.*2[\.-]16`),
		"tensorflow-2.15": regexp.MustCompile(`(?i)(registry\.redhat\.io/rhoai|quay\.io/modh).*tensorflow.*2[\.-]15`),
		"tensorflow-2.14": regexp.MustCompile(`(?i)(registry\.redhat\.io/rhoai|quay\.io/modh).*tensorflow.*2[\.-]14`),
		"tensorflow-2.13": regexp.MustCompile(`(?i)(registry\.redhat\.io/rhoai|quay\.io/modh).*tensorflow.*2[\.-]13`),

		"ray-2.9": regexp.MustCompile(`(?i)(registry\.redhat\.io/rhoai|quay\.io/modh).*ray.*2[\.-]9`),
		"ray-2.8": regexp.MustCompile(`(?i)(registry\.redhat\.io/rhoai|quay\.io/modh).*ray.*2[\.-]8`),
		"ray-2.7": regexp.MustCompile(`(?i)(registry\.redhat\.io/rhoai|quay\.io/modh).*ray.*2[\.-]7`),

		"ubi9-pytorch": regexp.MustCompile(`(?i)registry\.redhat\.io/ubi9/python.*pytorch`),

		"notebook-pytorch":    regexp.MustCompile(`(?i)(registry\.redhat\.io/rhoai|quay\.io/modh).*notebook.*pytorch`),
		"notebook-tensorflow": regexp.MustCompile(`(?i)(registry\.redhat\.io/rhoai|quay\.io/modh).*notebook.*tensorflow`),
	}

	rhoaiRegistries = []string{
		"registry.redhat.io/rhoai",
		"quay.io/modh",
		"quay.io/opendatahub",
		"registry.redhat.io/ubi",
	}

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

// AnalyzeContainerImage analyzes a container image and returns image source,
// RHOAI version information, and detected accelerator type for telemetry collection.
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

	isRHOAI := false
	for _, registry := range rhoaiRegistries {
		if strings.Contains(imageLower, strings.ToLower(registry)) {
			isRHOAI = true
			break
		}
	}

	if isRHOAI {
		result.ImageSource = "rhoai_official"

		versionFound := false
		for version, pattern := range rhoaiVersionPatterns {
			if pattern.MatchString(imageLower) {
				result.RHOAIVersion = version
				versionFound = true
				break
			}
		}

		if !versionFound {
			result.RHOAIVersion = "other"
		}

		return result
	}

	for _, pattern := range communityPatterns {
		if strings.Contains(imageLower, pattern) {
			result.ImageSource = "community"
			result.RHOAIVersion = "none"
			return result
		}
	}

	result.ImageSource = "custom"
	result.RHOAIVersion = "none"
	return result
}

// detectAcceleratorType detects GPU or accelerator type from container image name
// by checking for keywords like cuda, nvidia, rocm, habana, etc.
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

	for keyword, acceleratorType := range acceleratorPatterns {
		if strings.Contains(imageLower, keyword) {
			return acceleratorType
		}
	}

	return "cpu"
}

// GetImageRegistry extracts the registry hostname from a container image URL.
func GetImageRegistry(image string) string {
	parts := strings.Split(image, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return "unknown"
}

// IsRHOAIImage returns true if the container image is from an official RHOAI registry.
func IsRHOAIImage(image string) bool {
	imageLower := strings.ToLower(image)
	for _, registry := range rhoaiRegistries {
		if strings.Contains(imageLower, strings.ToLower(registry)) {
			return true
		}
	}
	return false
}


// ExtractContainerImage extracts the primary container image from training jobs
// across all supported frameworks (PyTorch, TensorFlow, MPI, XGBoost, Paddle, JAX).
func ExtractContainerImage(job interface{}, framework string) string {
	frameworkLower := strings.ToLower(framework)

	switch frameworkLower {
	case "pytorch":
		if pytorchJob, ok := job.(*kubeflowv1.PyTorchJob); ok {
			return ExtractPyTorchImageFromJob(pytorchJob)
		}
	case "tensorflow":
		if tfJob, ok := job.(*kubeflowv1.TFJob); ok {
			return ExtractTensorFlowImageFromJob(tfJob)
		}
	case "mpi":
		if mpiJob, ok := job.(*kubeflowv1.MPIJob); ok {
			return ExtractMPIImageFromJob(mpiJob)
		}
	case "xgboost":
		if xgboostJob, ok := job.(*kubeflowv1.XGBoostJob); ok {
			return ExtractXGBoostImageFromJob(xgboostJob)
		}
	case "paddle":
		if paddleJob, ok := job.(*kubeflowv1.PaddleJob); ok {
			return ExtractPaddleImageFromJob(paddleJob)
		}
	case "jax":
		if jaxJob, ok := job.(*kubeflowv1.JAXJob); ok {
			return ExtractJAXImageFromJob(jaxJob)
		}
	}

	klog.V(4).InfoS("Could not extract image from job", "framework", framework)
	return ""
}

// ExtractPyTorchImageFromJob extracts the container image from a PyTorchJob,
// checking Master replica first, then Worker replica.
func ExtractPyTorchImageFromJob(job *kubeflowv1.PyTorchJob) string {
	if job == nil || job.Spec.PyTorchReplicaSpecs == nil {
		return ""
	}

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

// ExtractTensorFlowImageFromJob extracts the container image from a TFJob,
// checking Chief replica first, then Worker replica.
func ExtractTensorFlowImageFromJob(job *kubeflowv1.TFJob) string {
	if job == nil || job.Spec.TFReplicaSpecs == nil {
		return ""
	}

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

// ExtractMPIImageFromJob extracts the container image from an MPIJob
// by checking the Launcher replica.
func ExtractMPIImageFromJob(job *kubeflowv1.MPIJob) string {
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

// ExtractXGBoostImageFromJob extracts the container image from an XGBoostJob
// by checking the Master replica.
func ExtractXGBoostImageFromJob(job *kubeflowv1.XGBoostJob) string {
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

// ExtractPaddleImageFromJob extracts the container image from a PaddleJob
// by checking the Master replica.
func ExtractPaddleImageFromJob(job *kubeflowv1.PaddleJob) string {
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

// ExtractJAXImageFromJob extracts the container image from a JAXJob
// by checking the Worker replica.
func ExtractJAXImageFromJob(job *kubeflowv1.JAXJob) string {
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
