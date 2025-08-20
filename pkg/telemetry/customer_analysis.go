// pkg/telemetry/customer_analysis.go
// Customer differentiation logic to distinguish real customers from test/demo usage
// Addresses requirement: "Include metadata to differentiate real customers from other usage sources"

package telemetry

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CustomerInfo represents customer classification data
type CustomerInfo struct {
	CustomerType     string   // enterprise, development, demo, test
	UsageSource      string   // rhoai-ui, cli, api-direct, external-tool
	NamespacePattern string   // production, development, demo, test
	TenantHints      []string // organization indicators
}

// ResourceInfo represents resource utilization analysis
type ResourceInfo struct {
	CPUCategory    string // small, medium, large, xlarge
	MemoryCategory string // small, medium, large, xlarge
	GPUCategory    string // none, single, multi, massive
	StorageType    string // local, network, distributed
}

// classifyCustomerUsage analyzes job metadata to classify customer type (Red Hat compliant)
// PRIVACY COMPLIANT: No PII collection, binary enterprise/non-enterprise classification only
func classifyCustomerUsage(namespace string, job interface{}) *CustomerInfo {
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

	// Simple binary classification: enterprise vs non-enterprise (Red Hat compliant)
	customerInfo.CustomerType = determineSimpleCustomerType(namespace, annotations, labels)

	return customerInfo
}

// classifyNamespacePattern analyzes namespace naming patterns to identify environment type
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

	// Check for production patterns first (highest confidence)
	for _, pattern := range productionPatterns {
		if strings.Contains(namespaceLower, pattern) {
			return "production"
		}
	}

	// Check for demo patterns (clear test usage)
	for _, pattern := range demoPatterns {
		if strings.Contains(namespaceLower, pattern) {
			return "demo"
		}
	}

	// Check for development patterns
	for _, pattern := range developmentPatterns {
		if strings.Contains(namespaceLower, pattern) {
			return "development"
		}
	}

	// Default based on common enterprise naming patterns
	if len(namespace) > 20 || strings.Contains(namespace, "-") {
		return "enterprise" // Complex namespace names suggest enterprise usage
	}

	return "unknown"
}

// determineSimpleCustomerType provides binary enterprise/non-enterprise classification
// RED HAT PRIVACY COMPLIANT: No PII collection, no detailed customer identification
func determineSimpleCustomerType(namespace string, annotations, labels map[string]string) string {
	// Check for clear enterprise indicators (RHOAI UI usage, production patterns)
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

	// Simple namespace-based classification (no detailed pattern analysis)
	namespaceLower := strings.ToLower(namespace)

	// Production patterns suggest enterprise usage
	if strings.Contains(namespaceLower, "prod") || strings.Contains(namespaceLower, "production") {
		return "enterprise"
	}

	// Default to non-enterprise for privacy (includes dev, test, demo, unknown)
	return "non-enterprise"
}

// identifyUsageSource determines how the training job was created
func identifyUsageSource(annotations, labels map[string]string) string {
	if annotations != nil {
		// RHOAI UI/Dashboard source
		if source, exists := annotations["rhods.openshiftai.io/source"]; exists {
			switch source {
			case "dashboard", "ui", "workbench":
				return "rhoai-ui"
			case "notebook":
				return "rhoai-notebook"
			}
		}

		// kubectl usage detection
		if _, exists := annotations["kubectl.kubernetes.io/last-applied-configuration"]; exists {
			return "cli"
		}

		// Helm/operator usage
		if _, exists := annotations["meta.helm.sh/release-name"]; exists {
			return "helm"
		}

		// OpenShift template usage
		if _, exists := annotations["template.openshift.io/instance"]; exists {
			return "openshift-template"
		}

		// Generic Kubernetes API usage
		if managedBy, exists := annotations["app.kubernetes.io/managed-by"]; exists {
			return "managed-" + managedBy
		}
	}

	if labels != nil {
		// Check for common management labels
		if managedBy, exists := labels["app.kubernetes.io/managed-by"]; exists {
			return "managed-" + managedBy
		}

		// Argo/GitOps patterns
		if _, exists := labels["argocd.argoproj.io/instance"]; exists {
			return "argocd"
		}

		// Flux patterns
		if _, exists := labels["kustomize.toolkit.fluxcd.io/name"]; exists {
			return "flux"
		}
	}

	return "api-direct" // Direct API usage (programmatic)
}

// extractTenantHints is deprecated and returns empty hints for privacy compliance
// RED HAT PRIVACY COMPLIANT: No collection of organization/tenant identifying information
func extractTenantHints(namespace string, annotations, labels map[string]string) []string {
	// Return empty hints to prevent PII collection (email addresses, org names, etc.)
	// This maintains interface compatibility while ensuring privacy compliance
	return []string{}
}

// analyzeJobResources analyzes job resource requirements for capacity planning
func analyzeJobResources(job interface{}) *ResourceInfo {
	resourceInfo := &ResourceInfo{
		CPUCategory:    "unknown",
		MemoryCategory: "unknown",
		GPUCategory:    "none",
		StorageType:    "unknown",
	}

	// This would need to be implemented based on specific job types
	// For now, provide a basic implementation that can be extended

	// TODO: Implement resource analysis based on job specifications
	// This would analyze:
	// - CPU requests/limits from pod specs
	// - Memory requests/limits from pod specs
	// - GPU requests from resource requirements
	// - Storage volume configurations

	// Placeholder implementation - should be extended based on actual job specs
	resourceInfo.CPUCategory = "medium"    // Default assumption
	resourceInfo.MemoryCategory = "medium" // Default assumption
	resourceInfo.GPUCategory = "none"      // Conservative default
	resourceInfo.StorageType = "local"     // Common default

	return resourceInfo
}
