package metrics

// Existing RHOAI Adoption Metrics
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

// NEW: GPU/Accelerator Metrics (REQUIRED)
func RecordAcceleratorHoursConsumed(framework, acceleratorType, imageSource string, hours float64) {
	AcceleratorHoursConsumed.WithLabelValues(framework, acceleratorType, imageSource).Add(hours)
}

func IncrementAcceleratorUtilization(framework, acceleratorType string, count float64) {
	AcceleratorUtilization.WithLabelValues(framework, acceleratorType).Add(count)
}

func DecrementAcceleratorUtilization(framework, acceleratorType string, count float64) {
	AcceleratorUtilization.WithLabelValues(framework, acceleratorType).Sub(count)
}

// NEW: Kueue Metrics (REQUIRED)
func RecordKueueManagedJob(framework, queueName, imageSource string) {
	KueueManagedJobs.WithLabelValues(framework, queueName, imageSource).Inc()
}

func IncrementKueueQueueDepth(queueName string) {
	KueueQueueDepth.WithLabelValues(queueName).Inc()
}

func DecrementKueueQueueDepth(queueName string) {
	KueueQueueDepth.WithLabelValues(queueName).Dec()
}

// NEW: Job Lifecycle Metrics
func RecordJobQueueDuration(framework, imageSource string, duration float64) {
	JobQueueDuration.WithLabelValues(framework, imageSource).Observe(duration)
}

func RecordJobRunDuration(framework, imageSource, status string, duration float64) {
	JobRunDuration.WithLabelValues(framework, imageSource, status).Observe(duration)
}

func RecordJobFailureReason(framework, reason, imageSource string) {
	JobFailureReasons.WithLabelValues(framework, reason, imageSource).Inc()
}

// Compliance Metrics
func RecordJobCreated(framework string) {
	TrainingJobsTotal.WithLabelValues(framework).Inc()
}

func RecordJobCompletion(framework, status string) {
	TrainingJobsCompleted.WithLabelValues(framework, status).Inc()
}

func RecordReconcileError(controller string) {
	ReconcileErrors.WithLabelValues(controller).Inc()
}
