// pkg/telemetry/event_receiver.go
// Main entry point for receiving job events from controllers
package telemetry

import (
	"context"
	"os"
	"strings"
	"sync"

	"github.com/kubeflow/training-operator/pkg/telemetry/metrics"
	"k8s.io/klog/v2"
)

var (
	initOnce         sync.Once
	telemetryEnabled bool
	isInitialized    bool
)

// EventType represents the type of job event
type EventType string

const (
	JobCreatedEvent   EventType = "created"
	JobStartedEvent   EventType = "started"
	JobCompletedEvent EventType = "completed"
	JobFailedEvent    EventType = "failed"
	JobDeletedEvent   EventType = "deleted"
)

// JobEventData contains event information
type JobEventData struct {
	EventType    EventType
	Framework    string
	Job          interface{}
	JobName      string
	JobNamespace string
	Metadata     map[string]string
}

// Initialize sets up the telemetry system
func Initialize() error {
	var err error
	initOnce.Do(func() {
		// Check if telemetry is enabled
		enabled := os.Getenv("TELEMETRY_ENABLED")
		telemetryEnabled = strings.ToLower(enabled) != "false"

		if !telemetryEnabled {
			klog.Info("Telemetry is disabled via TELEMETRY_ENABLED env var")
			return
		}

		klog.Info("Initializing telemetry event receiver")

		// Initialize metrics registry (this handles all metric registration)
		err = metrics.Initialize()
		if err != nil {
			klog.Errorf("Failed to initialize metrics: %v", err)
			return
		}

		isInitialized = true
		klog.Info("Telemetry event receiver initialized successfully")
	})
	return err
}

// IsEnabled returns whether telemetry is enabled
func IsEnabled() bool {
	return telemetryEnabled && isInitialized
}

// isTelemetryEnabled checks if telemetry is enabled (for backward compatibility)
func isTelemetryEnabled() bool {
	return IsEnabled()
}

// ReportJobCreation reports a training job creation event
func ReportJobCreation(job interface{}, framework string) {
	if !IsEnabled() {
		if !isInitialized {
			// Try to initialize if not done yet
			Initialize()
		}
		if !IsEnabled() {
			return
		}
	}

	event := JobEventData{
		EventType: JobCreatedEvent,
		Framework: framework,
		Job:       job,
	}

	// Process synchronously for metrics accuracy
	ctx := context.Background()
	convertEventToMetrics(ctx, event)
}

// ReportJobStarted reports a training job started event
func ReportJobStarted(job interface{}, framework string) {
	if !IsEnabled() {
		return
	}

	event := JobEventData{
		EventType: JobStartedEvent,
		Framework: framework,
		Job:       job,
	}

	ctx := context.Background()
	convertEventToMetrics(ctx, event)
}

// ReportJobCompletion reports a training job completion event
func ReportJobCompletion(job interface{}, framework string, succeeded bool) {
	if !IsEnabled() {
		return
	}

	eventType := JobCompletedEvent
	if !succeeded {
		eventType = JobFailedEvent
	}

	event := JobEventData{
		EventType: eventType,
		Framework: framework,
		Job:       job,
	}

	ctx := context.Background()
	convertEventToMetrics(ctx, event)
}

// ReportJobFailure reports a training job failure with reason
func ReportJobFailure(job interface{}, framework string, reason string) {
	if !IsEnabled() {
		return
	}

	event := JobEventData{
		EventType: JobFailedEvent,
		Framework: framework,
		Job:       job,
		Metadata: map[string]string{
			"reason": reason,
		},
	}

	ctx := context.Background()
	convertEventToMetrics(ctx, event)
}

// ReportJobDeletion reports a training job deletion event
func ReportJobDeletion(job interface{}, framework string) {
	if !IsEnabled() {
		return
	}

	event := JobEventData{
		EventType: JobDeletedEvent,
		Framework: framework,
		Job:       job,
	}

	ctx := context.Background()
	convertEventToMetrics(ctx, event)
}

// ReceiveJobEvent processes a job event (for backward compatibility)
func ReceiveJobEvent(ctx context.Context, event JobEventData) {
	if !IsEnabled() {
		return
	}

	convertEventToMetrics(ctx, event)
}
