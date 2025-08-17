package telemetry

import (
	"context"
	"os"
	"strings"
	"sync"
)

type JobEventData struct {
	EventType EventType
	Framework string
	Job       interface{}
	Metadata  map[string]string
}

type EventType string

const (
	JobCreatedEvent   EventType = "created"
	JobStartedEvent   EventType = "started"
	JobCompletedEvent EventType = "completed"
	JobFailedEvent    EventType = "failed"
	JobDeletedEvent   EventType = "deleted"
)

var (
	telemetryEnabled bool
	enabledOnce      sync.Once
)

func isTelemetryEnabled() bool {
	enabledOnce.Do(func() {
		enabled := os.Getenv("TELEMETRY_ENABLED")
		telemetryEnabled = strings.ToLower(enabled) != "false"
	})
	return telemetryEnabled
}

func ReceiveJobEvent(ctx context.Context, event JobEventData) {
	if !isTelemetryEnabled() {
		return
	}

	convertEventToMetrics(ctx, event)
}

func ReportJobCreation(job interface{}, framework string) {
	ReceiveJobEvent(context.Background(), JobEventData{
		EventType: JobCreatedEvent,
		Framework: framework,
		Job:       job,
	})
}

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

func ReportJobDeletion(job interface{}, framework string) {
	ReceiveJobEvent(context.Background(), JobEventData{
		EventType: JobDeletedEvent,
		Framework: framework,
		Job:       job,
	})
}
