package metrics

// =================================================================
// RHOAI Adoption Metrics - Core Business Intelligence
// These metrics answer: Are customers using RHOAI images? Which versions?
// =================================================================

func UpdateImageSourceMetric(framework, imageSource, rhoaiVersion string) {
	TrainingJobsByImageSource.WithLabelValues(framework, imageSource, rhoaiVersion).Inc()
}

func IncrementActiveRHOAIVersion(framework, version string) {
	if version != "" && version != "none" {
		ActiveRHOAIVersionDistribution.WithLabelValues(framework, version).Inc()
	}
}

func DecrementActiveRHOAIVersion(framework, version string) {
	if version != "" && version != "none" {
		ActiveRHOAIVersionDistribution.WithLabelValues(framework, version).Dec()
	}
}

func RecordVersionMigration(framework, fromVersion, toVersion string) {
	if fromVersion != "" && toVersion != "" && fromVersion != toVersion {
		RHOAIVersionMigration.WithLabelValues(framework, fromVersion, toVersion).Inc()
	}
}

// =================================================================
// Compliance Metrics - Required for Platform Monitoring
// =================================================================

func RecordJobCreated(framework string) {
	TrainingJobsTotal.WithLabelValues(framework).Inc()
}

func RecordJobCompletion(framework, status string) {
	TrainingJobsCompleted.WithLabelValues(framework, status).Inc()
}

func RecordReconcileError(controller string) {
	ReconcileErrors.WithLabelValues(controller).Inc()
}
