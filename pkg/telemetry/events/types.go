// pkg/telemetry/events/types.go
// Event type definitions for telemetry system
package events

import "time"

// EventType represents different job lifecycle events
type EventType string

const (
	JobCreatedEvent   EventType = "created"
	JobStartedEvent   EventType = "started"
	JobCompletedEvent EventType = "completed"
	JobFailedEvent    EventType = "failed"
	JobDeletedEvent   EventType = "deleted"
)

// JobEventData contains all information about a job event
type JobEventData struct {
	// Event metadata
	EventType   EventType
	Timestamp   time.Time
	Framework   string
	
	// Job information
	Job         interface{}
	JobName     string
	JobNamespace string
	
	// Additional context
	Metadata    map[string]string
	
	// Internal tracking
	ProcessedAt *time.Time
}

// CustomerInfo represents customer classification data
type CustomerInfo struct {
	CustomerType     string   // enterprise, non-enterprise
	UsageSource      string   // rhoai-ui, cli, api-direct, etc.
	NamespacePattern string   // production, development, demo, test
	TenantHints      []string // For interface compatibility (not used for privacy)
}

// JobMetadata contains extracted job metadata for telemetry
type JobMetadata struct {
	Name         string
	Namespace    string
	Framework    string
	CustomerInfo *CustomerInfo
	Labels       map[string]string
	Annotations  map[string]string
}

// IsValid checks if the event data is valid for processing
func (e *JobEventData) IsValid() bool {
	return e.EventType != "" && 
		   e.Framework != "" && 
		   e.Job != nil &&
		   !e.Timestamp.IsZero()
}

// IsProcessed returns whether this event has been processed
func (e *JobEventData) IsProcessed() bool {
	return e.ProcessedAt != nil
}

// MarkProcessed marks the event as processed
func (e *JobEventData) MarkProcessed() {
	now := time.Now()
	e.ProcessedAt = &now
}

// GetAge returns how long ago this event occurred
func (e *JobEventData) GetAge() time.Duration {
	return time.Since(e.Timestamp)
}