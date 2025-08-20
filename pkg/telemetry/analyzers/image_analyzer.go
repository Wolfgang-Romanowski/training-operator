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

// ExtractPyTorchImageFromJob extracts the container image from a PyTorchJob.
// It prioritizes the Master replica image, falling back to Worker replica
// if Master is not defined.
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

// ExtractTensorFlowImageFromJob extracts the container image from a TFJob.
// It prioritizes the Chief replica image, falling back to Worker replica
// if Chief is not defined.
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

// ExtractMPIImageFromJob extracts the container image from an MPIJob.
// It retrieves the image from the Launcher replica which coordinates
// the MPI job execution.
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

// ExtractXGBoostImageFromJob extracts the container image from an XGBoostJob.
// It retrieves the image from the Master replica which coordinates
// the distributed XGBoost training.
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

// ExtractPaddleImageFromJob extracts the container image from a PaddleJob.
// It retrieves the image from the Master replica which coordinates
// the PaddlePaddle distributed training.
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

// ExtractJAXImageFromJob extracts the container image from a JAXJob.
// It retrieves the image from the Worker replica since JAX jobs
// typically use a symmetric worker configuration.
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
