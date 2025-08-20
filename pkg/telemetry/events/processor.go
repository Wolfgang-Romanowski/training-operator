// pkg/telemetry/events/processor.go
// Event processing and routing to appropriate collectors
package events

import (
	"context"
	"time"

	"github.com/kubeflow/training-operator/pkg/telemetry/config"
	"k8s.io/klog/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	eventChannel chan JobEventData
	initialized  bool
)

// Initialize sets up the event processing system
func Initialize() {
	if initialized {
		return
	}
	
	cfg := config.Get()
	eventChannel = make(chan JobEventData, cfg.EventBufferSize)
	
	// Start event processing goroutine
	go processEvents()
	
	initialized = true
	klog.Info("Event processing system initialized")
}

// ReportJobEvent reports a job event for processing
func ReportJobEvent(eventType EventType, job interface{}, framework string, metadata map[string]string) {
	if !initialized {
		klog.Warning("Event system not initialized, dropping event")
		return
	}
	
	event := JobEventData{
		EventType: eventType,
		Timestamp: time.Now(),
		Framework: framework,
		Job:       job,
		Metadata:  metadata,
	}
	
	// Extract job metadata
	if metaObj, ok := job.(metav1.Object); ok {
		event.JobName = metaObj.GetName()
		event.JobNamespace = metaObj.GetNamespace()
	}
	
	// Non-blocking send
	select {
	case eventChannel <- event:
		// Event queued successfully
	default:
		klog.Warning("Event channel full, dropping event")
	}
}

// ProcessEvent processes a single event (for backward compatibility)
func ProcessEvent(ctx context.Context, event JobEventData) {
	if !event.IsValid() {
		klog.Warning("Invalid event data, skipping processing")
		return
	}
	
	// Add timeout protection
	processCtx, cancel := context.WithTimeout(ctx, config.GetProcessingTimeout())
	defer cancel()
	
	// Process the event
	if err := processJobEvent(processCtx, event); err != nil {
		klog.Errorf("Failed to process event: %v", err)
	}
}

// processEvents is the main event processing loop
func processEvents() {
	klog.Info("Starting event processing loop")
	
	for event := range eventChannel {
		if !event.IsValid() {
			klog.Warning("Received invalid event, skipping")
			continue
		}
		
		// Process with timeout protection
		ctx, cancel := context.WithTimeout(context.Background(), config.GetProcessingTimeout())
		
		if err := processJobEvent(ctx, event); err != nil {
			klog.Errorf("Failed to process event %s for %s/%s: %v", 
				event.EventType, event.Framework, event.JobName, err)
		}
		
		cancel()
	}
}

// processJobEvent processes a single job event
func processJobEvent(ctx context.Context, event JobEventData) error {
	// Import collectors here to avoid circular dependencies
	collectors := getCollectors()
	if collectors == nil {
		return nil // Collectors not available, skip processing
	}
	
	// Route event to appropriate collectors based on event type
	switch event.EventType {
	case JobCreatedEvent:
		return collectors.ProcessJobCreation(ctx, event)
	case JobStartedEvent:
		return collectors.ProcessJobStart(ctx, event)
	case JobCompletedEvent:
		return collectors.ProcessJobCompletion(ctx, event, true)
	case JobFailedEvent:
		return collectors.ProcessJobCompletion(ctx, event, false)
	case JobDeletedEvent:
		return collectors.ProcessJobDeletion(ctx, event)
	default:
		klog.Warningf("Unknown event type: %s", event.EventType)
		return nil
	}
}

// getCollectors returns the collectors interface (to avoid circular imports)
// This is a temporary solution - in a real implementation, you'd use dependency injection
func getCollectors() interface{} {
	// This would be properly implemented with dependency injection
	// For now, return nil to avoid circular import issues
	return nil
}

// Shutdown gracefully shuts down the event processing system
func Shutdown() {
	if eventChannel != nil {
		close(eventChannel)
	}
	initialized = false
	klog.Info("Event processing system shut down")
}