package analyzers

import (
    "regexp"
    "strings"
)

type ImageAnalysisResult struct {
    ImageSource  string
    RHOAIVersion string
}

var (
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
    
    tensorflow210Patterns = []*regexp.Regexp{
        regexp.MustCompile(`quay\.io/modh/tensorflow:.*2\.10`),
        regexp.MustCompile(`registry\.redhat\.io/rhoai/tensorflow:.*2\.10`),
    }
)

func AnalyzeContainerImage(image string) ImageAnalysisResult {
    if image == "" {
        return ImageAnalysisResult{
            ImageSource:  "unknown",
            RHOAIVersion: "none",
        }
    }
    
    imageLower := strings.ToLower(image)
    
    for _, pattern := range pytorch25Patterns {
        if pattern.MatchString(imageLower) {
            return ImageAnalysisResult{
                ImageSource:  "rhoai_official",
                RHOAIVersion: "2.5",
            }
        }
    }
    
    for _, pattern := range pytorch24Patterns {
        if pattern.MatchString(imageLower) {
            return ImageAnalysisResult{
                ImageSource:  "rhoai_official",
                RHOAIVersion: "2.4",
            }
        }
    }
    
    for _, pattern := range pytorch23Patterns {
        if pattern.MatchString(imageLower) {
            return ImageAnalysisResult{
                ImageSource:  "rhoai_official",
                RHOAIVersion: "2.3",
            }
        }
    }
    
    for _, pattern := range tensorflow210Patterns {
        if pattern.MatchString(imageLower) {
            return ImageAnalysisResult{
                ImageSource:  "rhoai_official",
                RHOAIVersion: "tf-2.10",
            }
        }
    }
    
    if strings.Contains(imageLower, "pytorch/pytorch") || 
       strings.Contains(imageLower, "tensorflow/tensorflow") ||
       strings.Contains(imageLower, "docker.io/pytorch") {
        return ImageAnalysisResult{
            ImageSource:  "community",
            RHOAIVersion: "none",
        }
    }
    
    return ImageAnalysisResult{
        ImageSource:  "custom",
        RHOAIVersion: "none",
    }
}