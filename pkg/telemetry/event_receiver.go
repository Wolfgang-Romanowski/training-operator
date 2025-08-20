package telemetry

import (
	"context"
	"os"
	"strings"
	"sync"

	"k8s.io/klog/v2"

	"github.com/kubeflow/training-operator/pkg/telemetry/metrics"
)

var (
	initOnce         sync.Once
	telemetryEnabled bool
	isInitialized    bool
)

// EventType represents different lifecycle events for training jobs.
type EventType string

const (
	JobCreatedEvent   EventType = "created"
	JobStartedEvent   EventType = "started"
	JobCompletedEvent EventType = "completed"
	JobFailedEvent    EventType = "failed"
	JobDeletedEvent   EventType = "deleted"
)

// JobEventData contains all relevant information for a training job event
// including job metadata and framework details.
type JobEventData struct {
	EventType    EventType
	Framework    string
	Job          interface{}
	JobName      string
	JobNamespace string
	Metadata     map[string]string
}

// Initialize sets up the telemetry system, checking environment variables
// and initializing metrics collection if enabled.
func Initialize() error {
	var err error
	initOnce.Do(func() {
		enabled := os.Getenv("TELEMETRY_ENABLED")
		telemetryEnabled = strings.ToLower(enabled) != "false"

		if !telemetryEnabled {
			klog.Info("Telemetry is disabled via TELEMETRY_ENABLED env var")
			return
		}

		klog.Info("Initializing telemetry event receiver")

		err = metrics.Initialize()
		if err != nil {
			klog.ErrorS(err, "Failed to initialize telemetry metrics")
			return
		}

		isInitialized = true
		klog.Info("Telemetry event receiver initialized successfully")
	})
	return err
}

// IsEnabled returns true if telemetry collection is both enabled via environment
// variable and successfully initialized.
func IsEnabled() bool {
	return telemetryEnabled && isInitialized
}

// isTelemetryEnabled provides backward compatibility for existing telemetry checks.
func isTelemetryEnabled() bool {
	return IsEnabled()
}

// ReportJobCreation processes a training job creation event, extracting
// image information and updating telemetry metrics accordingly.
func ReportJobCreation(job interface{}, framework string) {
	if !IsEnabled() {
		if !isInitialized {
			if err := Initialize(); err != nil {
				klog.V(4).InfoS("Telemetry initialization failed, skipping event", "error", err)
				return
			}
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

	ctx := context.Background()
	convertEventToMetrics(ctx, event)
}

// ReportJobStarted processes a training job started event for telemetry collection.
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

// ReportJobCompletion processes a training job completion event, handling both
// successful completions and failures based on the succeeded parameter.
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

// ReportJobFailure processes a training job failure event with specific failure
// reason for detailed telemetry analysis.
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

// ReportJobDeletion processes a training job deletion event, cleaning up
// associated telemetry tracking for the deleted job.
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

// ReceiveJobEvent processes job events for backward compatibility with existing
// telemetry collection code.
func ReceiveJobEvent(ctx context.Context, event JobEventData) {
	if !IsEnabled() {
		return
	}

	convertEventToMetrics(ctx, event)
}
