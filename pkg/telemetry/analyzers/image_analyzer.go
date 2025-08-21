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

package analyzers

import (
	"regexp"
	"strings"

	"k8s.io/klog/v2"

	kubeflowv1 "github.com/kubeflow/training-operator/pkg/apis/kubeflow.org/v1"
)

// ImageAnalysisResult contains comprehensive image analysis results.
// It identifies the image source, RHOAI version, and hardware acceleration
// requirements for telemetry and usage pattern analysis.
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

// AnalyzeContainerImage analyzes a container image for telemetry collection.
// It identifies the image source (RHOAI official, community, or custom),
// extracts version information, and detects hardware accelerator requirements.
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
				// Normalize version to ensure cardinality compliance
				result.RHOAIVersion = normalizeVersionForCardinality(version)
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

// normalizeVersionForCardinality ensures version strings comply with Red Hat cardinality limits.
// Only 5 version values are allowed to stay within the 10 timeseries total limit.
func normalizeVersionForCardinality(version string) string {
	// Red Hat monitoring allows max 5 version labels to stay within 10 timeseries total
	allowedVersions := map[string]bool{
		"pytorch-2.4":     true,
		"pytorch-2.3":     true,
		"tensorflow-2.15": true,
		"tensorflow-2.14": true,
	}
	
	if allowedVersions[version] {
		return version
	}
	
	// All other versions map to "other" to maintain cardinality compliance
	return "other"
}

// detectAcceleratorType detects GPU or accelerator type from container image name.
// It identifies hardware acceleration requirements by matching keywords
// like cuda, nvidia, rocm, habana in the image name.
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
// This helps identify the source of container images for security and
// compliance tracking.
func GetImageRegistry(image string) string {
	parts := strings.Split(image, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return "unknown"
}

// IsRHOAIImage returns true if the container image is from an official RHOAI registry.
// This function helps identify customers using official Red Hat provided images
// versus community or custom alternatives.
func IsRHOAIImage(image string) bool {
	imageLower := strings.ToLower(image)
	for _, registry := range rhoaiRegistries {
		if strings.Contains(imageLower, strings.ToLower(registry)) {
			return true
		}
	}
	return false
}

// ExtractContainerImage extracts the primary container image from training jobs.
// It supports all Kubeflow training operator frameworks including PyTorch,
// TensorFlow, MPI, XGBoost, Paddle, and JAX.
func ExtractContainerImage(job interface{}, framework string) string {
	// Use the unified extraction function
	return ExtractImageFromTrainingJob(job, framework)
}

// ExtractImageFromTrainingJob extracts the container image from any training job type.
// It uses a generic approach to handle all framework types uniformly, reducing code duplication
// while maintaining framework-specific logic for replica priority.
func ExtractImageFromTrainingJob(job interface{}, framework string) string {
	if job == nil {
		return ""
	}

	switch framework {
	case "pytorch":
		pytorchJob, ok := job.(*kubeflowv1.PyTorchJob)
		if !ok || pytorchJob.Spec.PyTorchReplicaSpecs == nil {
			return ""
		}
		// Check Master first, then Worker
		for _, replicaType := range []kubeflowv1.ReplicaType{
			kubeflowv1.PyTorchJobReplicaTypeMaster,
			kubeflowv1.PyTorchJobReplicaTypeWorker,
		} {
			if replica, exists := pytorchJob.Spec.PyTorchReplicaSpecs[replicaType]; exists {
				if image := extractImageFromReplicaSpec(replica); image != "" {
					return image
				}
			}
		}

	case "tensorflow":
		tfJob, ok := job.(*kubeflowv1.TFJob)
		if !ok || tfJob.Spec.TFReplicaSpecs == nil {
			return ""
		}
		// Check Chief first, then Worker
		for _, replicaType := range []kubeflowv1.ReplicaType{
			kubeflowv1.TFJobReplicaTypeChief,
			kubeflowv1.TFJobReplicaTypeWorker,
		} {
			if replica, exists := tfJob.Spec.TFReplicaSpecs[replicaType]; exists {
				if image := extractImageFromReplicaSpec(replica); image != "" {
					return image
				}
			}
		}

	case "mpi":
		mpiJob, ok := job.(*kubeflowv1.MPIJob)
		if !ok || mpiJob.Spec.MPIReplicaSpecs == nil {
			return ""
		}
		if launcher, exists := mpiJob.Spec.MPIReplicaSpecs[kubeflowv1.MPIJobReplicaTypeLauncher]; exists {
			return extractImageFromReplicaSpec(launcher)
		}

	case "xgboost":
		xgboostJob, ok := job.(*kubeflowv1.XGBoostJob)
		if !ok || xgboostJob.Spec.XGBReplicaSpecs == nil {
			return ""
		}
		if master, exists := xgboostJob.Spec.XGBReplicaSpecs[kubeflowv1.XGBoostJobReplicaTypeMaster]; exists {
			return extractImageFromReplicaSpec(master)
		}

	case "paddle":
		paddleJob, ok := job.(*kubeflowv1.PaddleJob)
		if !ok || paddleJob.Spec.PaddleReplicaSpecs == nil {
			return ""
		}
		if master, exists := paddleJob.Spec.PaddleReplicaSpecs[kubeflowv1.PaddleJobReplicaTypeMaster]; exists {
			return extractImageFromReplicaSpec(master)
		}

	case "jax":
		jaxJob, ok := job.(*kubeflowv1.JAXJob)
		if !ok || jaxJob.Spec.JAXReplicaSpecs == nil {
			return ""
		}
		if worker, exists := jaxJob.Spec.JAXReplicaSpecs[kubeflowv1.JAXJobReplicaTypeWorker]; exists {
			return extractImageFromReplicaSpec(worker)
		}
	}

	return ""
}

// extractImageFromReplicaSpec extracts the container image from a replica specification.
// This helper function reduces duplication across different job types.
func extractImageFromReplicaSpec(replica *kubeflowv1.ReplicaSpec) string {
	if replica != nil && len(replica.Template.Spec.Containers) > 0 {
		return replica.Template.Spec.Containers[0].Image
	}
	return ""
}

