// pkg/telemetry/event_receiver.go
package telemetry

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	"k8s.io/klog/v2"
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
	EventType EventType
	Framework string
	Job       interface{}
	Metadata  map[string]string
}

var (
	telemetryEnabled bool
	enabledOnce      sync.Once
)

// isTelemetryEnabled checks if telemetry is enabled
func isTelemetryEnabled() bool {
	enabledOnce.Do(func() {
		enabled := os.Getenv("TELEMETRY_ENABLED")
		// Default to enabled unless explicitly disabled
		telemetryEnabled = strings.ToLower(enabled) != "false"
		
		if telemetryEnabled {
			klog.Info("Telemetry metrics collection is enabled")
		} else {
			klog.Info("Telemetry metrics collection is disabled")
		}
	})
	return telemetryEnabled
}

// ReceiveJobEvent processes a job event
func ReceiveJobEvent(ctx context.Context, event JobEventData) {
	if !isTelemetryEnabled() {
		return
	}

	// Process event asynchronously to avoid blocking controller
	go func() {
		// Add timeout to prevent hanging
		processCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		
		convertEventToMetrics(processCtx, event)
	}()
}

// ReportJobCreation reports job creation event
func ReportJobCreation(job interface{}, framework string) {
	ReceiveJobEvent(context.Background(), JobEventData{
		EventType: JobCreatedEvent,
		Framework: framework,
		Job:       job,
	})
}

// ReportJobStarted reports job started event
func ReportJobStarted(job interface{}, framework string) {
	ReceiveJobEvent(context.Background(), JobEventData{
		EventType: JobStartedEvent,
		Framework: framework,
		Job:       job,
	})
}

// ReportJobCompletion reports job completion event
func ReportJobCompletion(job interface{}, framework string, succeeded bool) {
	eventType := JobCompletedEvent
	if !succeeded {
		eventType = JobFailedEvent
	}

	ReceiveJobEvent(context.Background(), JobEventData{
		EventType: eventType,
		Framework: framework,
		Job:       job,
	})
}

// ReportJobFailure reports job failure with reason
func ReportJobFailure(job interface{}, framework string, reason string) {
	ReceiveJobEvent(context.Background(), JobEventData{
		EventType: JobFailedEvent,
		Framework: framework,
		Job:       job,
		Metadata: map[string]string{
			"reason": reason,
		},
	})
}

// ReportJobDeletion reports job deletion event
func ReportJobDeletion(job interface{}, framework string) {
	ReceiveJobEvent(context.Background(), JobEventData{
		EventType: JobDeletedEvent,
		Framework: framework,
		Job:       job,
	})
}