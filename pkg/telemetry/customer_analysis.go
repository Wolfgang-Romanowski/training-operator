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

package telemetry

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CustomerInfo represents customer classification data for telemetry analysis.
// It captures metadata to differentiate real customers from test/demo usage
// without collecting any personally identifiable information.
type CustomerInfo struct {
	CustomerType     string   // enterprise, development, demo, test
	UsageSource      string   // rhoai-ui, cli, api-direct, external-tool
	NamespacePattern string   // production, development, demo, test
	TenantHints      []string // organization indicators (deprecated for privacy)
}

// ResourceInfo represents resource utilization analysis for capacity planning.
// It categorizes resource usage patterns to understand workload characteristics
// and infrastructure requirements.
type ResourceInfo struct {
	CPUCategory    string // small, medium, large, xlarge
	MemoryCategory string // small, medium, large, xlarge
	GPUCategory    string // none, single, multi, massive
	StorageType    string // local, network, distributed
}

// ClassifyCustomerUsage analyzes job metadata to classify customer type.
// This function is privacy-compliant, collecting no personally identifiable information
// and providing only binary enterprise/non-enterprise classification per Red Hat requirements.
func ClassifyCustomerUsage(namespace string, job interface{}) *CustomerInfo {
	customerInfo := &CustomerInfo{
		CustomerType:     "non-enterprise", // Default to non-enterprise for privacy
		UsageSource:      "unknown",
		NamespacePattern: "unknown",
		TenantHints:      []string{}, // Kept for interface compatibility but not used
	}

	// Extract metadata accessor from job
	var annotations map[string]string
	var labels map[string]string

	if metaAccessor, ok := job.(metav1.Object); ok {
		annotations = metaAccessor.GetAnnotations()
		labels = metaAccessor.GetLabels()
	}

	// Perform binary classification: enterprise vs non-enterprise for privacy compliance
	customerInfo.CustomerType = determineSimpleCustomerType(namespace, annotations, labels)

	return customerInfo
}

// classifyNamespacePattern analyzes namespace naming patterns to identify environment type.
// It uses common naming conventions to infer whether the namespace represents
// production, development, demo, or test environments.
func classifyNamespacePattern(namespace string) string {
	namespaceLower := strings.ToLower(namespace)

	// Production environment patterns
	productionPatterns := []string{
		"prod", "production", "live", "release", "stable",
		"main", "master", "deploy", "runtime",
	}

	// Development environment patterns
	developmentPatterns := []string{
		"dev", "develop", "development", "staging", "stage",
		"test", "testing", "qa", "uat", "integration",
		"int", "pre-prod", "preprod",
	}

	// Demo/example patterns
	demoPatterns := []string{
		"demo", "example", "sample", "tutorial", "workshop",
		"training", "learn", "playground", "sandbox",
		"trial", "evaluation", "eval",
	}

	// Production patterns have highest priority for accurate classification
	for _, pattern := range productionPatterns {
		if strings.Contains(namespaceLower, pattern) {
			return "production"
		}
	}

	// Demo patterns indicate non-production test or evaluation usage
	for _, pattern := range demoPatterns {
		if strings.Contains(namespaceLower, pattern) {
			return "demo"
		}
	}

	// Development patterns suggest pre-production environments
	for _, pattern := range developmentPatterns {
		if strings.Contains(namespaceLower, pattern) {
			return "development"
		}
	}

	// Complex namespace names often indicate enterprise usage patterns
	if len(namespace) > 20 || strings.Contains(namespace, "-") {
		return "enterprise" // Complex namespace names suggest enterprise usage
	}

	return "unknown"
}

// determineSimpleCustomerType provides binary enterprise/non-enterprise classification.
// This function is Red Hat privacy-compliant, avoiding collection of personally
// identifiable information or detailed customer identification.
func determineSimpleCustomerType(namespace string, annotations, labels map[string]string) string {
	// Enterprise indicators include RHOAI UI usage and production patterns
	if annotations != nil {
		// RHOAI UI created jobs indicate enterprise usage
		if source, exists := annotations["rhods.openshiftai.io/source"]; exists {
			if source == "dashboard" || source == "ui" || source == "workbench" {
				return "enterprise" // UI usage indicates real customers
			}
		}

		// Notebook integration indicates enterprise usage
		if _, exists := annotations["notebooks.openshiftai.io/notebook-name"]; exists {
			return "enterprise"
		}
	}

	if labels != nil {
		// Production environment labels
		if env, exists := labels["environment"]; exists {
			if env == "production" || env == "prod" {
				return "enterprise"
			}
		}
	}

	// Namespace-based classification provides additional signal
	namespaceLower := strings.ToLower(namespace)

	// Production patterns suggest enterprise usage
	if strings.Contains(namespaceLower, "prod") || strings.Contains(namespaceLower, "production") {
		return "enterprise"
	}

	// Conservative default to non-enterprise ensures privacy compliance
	return "non-enterprise"
}

// Note: The following functions were removed for production readiness:
// - extractTenantHints: Removed to ensure privacy compliance (no PII collection)
// - analyzeJobResources: Removed as it contained only placeholder values
// - identifyUsageSource: Removed as it was never called in production code
// Customer classification is handled by the main ClassifyCustomerUsage function above.
