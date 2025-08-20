// pkg/telemetry/analyzers/image_analyzer.go
package analyzers

import (
	"regexp"
	"strings"
)

// ImageAnalysisResult contains image analysis results
type ImageAnalysisResult struct {
	ImageSource     string // rhoai_official, community, custom
	RHOAIVersion    string // pytorch-2.5, tensorflow-2.16, etc
	AcceleratorType string // nvidia.com/gpu, amd.com/gpu, cpu
}

var (
	// RHOAI Official Image Patterns
	rhoaiPatterns = map[string]*regexp.Regexp{
		// PyTorch versions
		"pytorch-2.5": regexp.MustCompile(`(?i)(registry\.redhat\.io/rhoai|quay\.io/modh|quay\.io/opendatahub)/.*pytorch.*2\.5`),
		"pytorch-2.4": regexp.MustCompile(`(?i)(registry\.redhat\.io/rhoai|quay\.io/modh|quay\.io/opendatahub)/.*pytorch.*2\.4`),
		"pytorch-2.3": regexp.MustCompile(`(?i)(registry\.redhat\.io/rhoai|quay\.io/modh|quay\.io/opendatahub)/.*pytorch.*2\.3`),
		
		// TensorFlow versions
		"tensorflow-2.16": regexp.MustCompile(`(?i)(registry\.redhat\.io/rhoai|quay\.io/modh)/.*tensorflow.*2\.16`),
		"tensorflow-2.15": regexp.MustCompile(`(?i)(registry\.redhat\.io/rhoai|quay\.io/modh)/.*tensorflow.*2\.15`),
		"tensorflow-2.14": regexp.MustCompile(`(?i)(registry\.redhat\.io/rhoai|quay\.io/modh)/.*tensorflow.*2\.14`),
		
		// UBI-based PyTorch images
		"ubi-pytorch": regexp.MustCompile(`(?i)registry\.redhat\.io/ubi9/python-3\d+.*pytorch`),
		
		// Notebook images that might be used for training
		"notebook": regexp.MustCompile(`(?i)(registry\.redhat\.io/rhoai|quay\.io/modh)/odh-.*notebook`),
		
		// CodeFlare/Ray images
		"ray": regexp.MustCompile(`(?i)(registry\.redhat\.io/rhoai|quay\.io/modh)/ray`),
	}

	// Community image patterns
	communityPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)pytorch/pytorch`),
		regexp.MustCompile(`(?i)tensorflow/tensorflow`),
		regexp.MustCompile(`(?i)docker\.io/(pytorch|tensorflow)`),
		regexp.MustCompile(`(?i)nvcr\.io/nvidia`),
		regexp.MustCompile(`(?i)huggingface/`),
		regexp.MustCompile(`(?i)jupyter/`),
	}

	// Accelerator patterns (normalized per OBSDA-1087)
	acceleratorPatterns = map[string]string{
		"cuda":   "nvidia.com/gpu",
		"nvidia": "nvidia.com/gpu",
		"rocm":   "amd.com/gpu",
		"amd":    "amd.com/gpu",
		"habana": "habana.ai/gaudi",
		"gaudi":  "habana.ai/gaudi",
		"intel":  "intel.com/gpu",
		"cpu":    "cpu",
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

	// Check for RHOAI official images
	for version, pattern := range rhoaiPatterns {
		if pattern.MatchString(imageLower) {
			result.ImageSource = "rhoai_official"
			result.RHOAIVersion = version
			return result
		}
	}

	// Check for community images
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
	// Check each accelerator pattern
	for keyword, acceleratorType := range acceleratorPatterns {
		if strings.Contains(imageLower, keyword) {
			return acceleratorType
		}
	}

	// Check for GPU keyword without specific vendor
	if strings.Contains(imageLower, "gpu") {
		return "nvidia.com/gpu" // Default to NVIDIA if generic GPU
	}

	return "cpu"
}