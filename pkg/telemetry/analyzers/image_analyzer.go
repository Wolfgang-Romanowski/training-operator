package analyzers

import (
	"regexp"
	"strings"
)

type ImageAnalysisResult struct {
	ImageSource     string
	RHOAIVersion    string
	IsNotebook      bool
	IsCodeFlare     bool
	AcceleratorType string
}

var (
	// PyTorch RHOAI images by version
	pytorch25Patterns = []*regexp.Regexp{
		regexp.MustCompile(`quay\.io/modh/pytorch:.*2\.5`),
		regexp.MustCompile(`registry\.redhat\.io/rhoai/pytorch:.*2\.5`),
		regexp.MustCompile(`quay\.io/opendatahub/pytorch:.*2\.5`),
		regexp.MustCompile(`registry\.redhat\.io/ubi9/python-311.*pytorch.*2\.5`),
	}

	pytorch24Patterns = []*regexp.Regexp{
		regexp.MustCompile(`quay\.io/modh/pytorch:.*2\.4`),
		regexp.MustCompile(`registry\.redhat\.io/rhoai/pytorch:.*2\.4`),
		regexp.MustCompile(`quay\.io/opendatahub/pytorch:.*2\.4`),
		regexp.MustCompile(`registry\.redhat\.io/ubi9/python-311.*pytorch.*2\.4`),
	}

	pytorch23Patterns = []*regexp.Regexp{
		regexp.MustCompile(`quay\.io/modh/pytorch:.*2\.3`),
		regexp.MustCompile(`registry\.redhat\.io/rhoai/pytorch:.*2\.3`),
		regexp.MustCompile(`quay\.io/opendatahub/pytorch:.*2\.3`),
	}

	// TensorFlow RHOAI images
	tensorflow216Patterns = []*regexp.Regexp{
		regexp.MustCompile(`quay\.io/modh/tensorflow:.*2\.16`),
		regexp.MustCompile(`registry\.redhat\.io/rhoai/tensorflow:.*2\.16`),
	}

	tensorflow215Patterns = []*regexp.Regexp{
		regexp.MustCompile(`quay\.io/modh/tensorflow:.*2\.15`),
		regexp.MustCompile(`registry\.redhat\.io/rhoai/tensorflow:.*2\.15`),
	}

	tensorflow210Patterns = []*regexp.Regexp{
		regexp.MustCompile(`quay\.io/modh/tensorflow:.*2\.10`),
		regexp.MustCompile(`registry\.redhat\.io/rhoai/tensorflow:.*2\.10`),
	}

	// NEW: RHOAI notebook images
	notebookPatterns = []*regexp.Regexp{
		regexp.MustCompile(`registry\.redhat\.io/rhoai/odh-notebook-.*`),
		regexp.MustCompile(`quay\.io/modh/odh-generic-data-science-notebook.*`),
		regexp.MustCompile(`quay\.io/opendatahub/.*-notebook.*`),
	}

	// NEW: CodeFlare/Ray patterns
	codeflarePatterns = []*regexp.Regexp{
		regexp.MustCompile(`quay\.io/modh/ray:.*rhoai`),
		regexp.MustCompile(`registry\.redhat\.io/rhoai/ray.*`),
		regexp.MustCompile(`quay\.io/project-codeflare/.*`),
	}

	// Community images
	communityPatterns = []*regexp.Regexp{
		regexp.MustCompile(`pytorch/pytorch`),
		regexp.MustCompile(`tensorflow/tensorflow`),
		regexp.MustCompile(`docker\.io/pytorch`),
		regexp.MustCompile(`docker\.io/tensorflow`),
		regexp.MustCompile(`nvcr\.io/nvidia`),
	}

	// NEW: GPU-optimized image patterns
	cudaPatterns = []*regexp.Regexp{
		regexp.MustCompile(`.*cuda.*`),
		regexp.MustCompile(`.*nvidia.*`),
		regexp.MustCompile(`.*gpu.*`),
	}

	rocmPatterns = []*regexp.Regexp{
		regexp.MustCompile(`.*rocm.*`),
		regexp.MustCompile(`.*amd.*gpu.*`),
	}

	habanaPatterns = []*regexp.Regexp{
		regexp.MustCompile(`.*habana.*`),
		regexp.MustCompile(`.*gaudi.*`),
	}
)

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

	// Check for RHOAI notebook origins
	for _, pattern := range notebookPatterns {
		if pattern.MatchString(imageLower) {
			result.IsNotebook = true
			break
		}
	}

	// Check for CodeFlare/Ray
	for _, pattern := range codeflarePatterns {
		if pattern.MatchString(imageLower) {
			result.IsCodeFlare = true
			break
		}
	}

	// Check PyTorch versions
	for _, pattern := range pytorch25Patterns {
		if pattern.MatchString(imageLower) {
			result.ImageSource = "rhoai_official"
			result.RHOAIVersion = "pytorch-2.5"
			return result
		}
	}

	for _, pattern := range pytorch24Patterns {
		if pattern.MatchString(imageLower) {
			result.ImageSource = "rhoai_official"
			result.RHOAIVersion = "pytorch-2.4"
			return result
		}
	}

	for _, pattern := range pytorch23Patterns {
		if pattern.MatchString(imageLower) {
			result.ImageSource = "rhoai_official"
			result.RHOAIVersion = "pytorch-2.3"
			return result
		}
	}

	// Check TensorFlow versions
	for _, pattern := range tensorflow216Patterns {
		if pattern.MatchString(imageLower) {
			result.ImageSource = "rhoai_official"
			result.RHOAIVersion = "tensorflow-2.16"
			return result
		}
	}

	for _, pattern := range tensorflow215Patterns {
		if pattern.MatchString(imageLower) {
			result.ImageSource = "rhoai_official"
			result.RHOAIVersion = "tensorflow-2.15"
			return result
		}
	}

	for _, pattern := range tensorflow210Patterns {
		if pattern.MatchString(imageLower) {
			result.ImageSource = "rhoai_official"
			result.RHOAIVersion = "tensorflow-2.10"
			return result
		}
	}

	// Check community images
	for _, pattern := range communityPatterns {
		if pattern.MatchString(imageLower) {
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
	// Check for Habana Gaudi first (most specific)
	for _, pattern := range habanaPatterns {
		if pattern.MatchString(imageLower) {
			return "habana.ai/gaudi"
		}
	}

	// Check for AMD ROCm
	for _, pattern := range rocmPatterns {
		if pattern.MatchString(imageLower) {
			return "amd.com/gpu"
		}
	}

	// Check for NVIDIA CUDA
	for _, pattern := range cudaPatterns {
		if pattern.MatchString(imageLower) {
			return "nvidia.com/gpu"
		}
	}

	return "cpu"
}

// DetectKueueIntegration checks for Kueue annotations
func DetectKueueIntegration(annotations map[string]string) (bool, string) {
	if annotations == nil {
		return false, ""
	}

	// Check for Kueue workload annotations
	if workload, ok := annotations["kueue.x-k8s.io/workload"]; ok && workload != "" {
		if queueName, ok := annotations["kueue.x-k8s.io/queue-name"]; ok {
			return true, queueName
		}
		return true, "default"
	}

	return false, ""
}

// ClassifyJobFailure categorizes failure reasons for better debugging
func ClassifyJobFailure(message string) string {
	messageLower := strings.ToLower(message)

	switch {
	case strings.Contains(messageLower, "oomkilled"):
		return "oom"
	case strings.Contains(messageLower, "imagepullbackoff"):
		return "image_pull_error"
	case strings.Contains(messageLower, "insufficient nvidia.com/gpu"):
		return "insufficient_gpu"
	case strings.Contains(messageLower, "insufficient amd.com/gpu"):
		return "insufficient_gpu"
	case strings.Contains(messageLower, "insufficient habana.ai/gaudi"):
		return "insufficient_accelerator"
	case strings.Contains(messageLower, "insufficient intel.com/gpu"):
		return "insufficient_gpu"
	case strings.Contains(messageLower, "deadlineexceeded"):
		return "timeout"
	case strings.Contains(messageLower, "node.kubernetes.io/not-ready"):
		return "node_failure"
	case strings.Contains(messageLower, "evicted"):
		return "evicted"
	case strings.Contains(messageLower, "preempted"):
		return "preempted"
	default:
		return "user_error"
	}
}
