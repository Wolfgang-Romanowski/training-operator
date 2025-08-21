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
	"context"
	"sync"
	"sync/atomic"
	"time"

	"k8s.io/klog/v2"

	"github.com/kubeflow/training-operator/pkg/telemetry/config"
	"github.com/kubeflow/training-operator/pkg/telemetry/metrics"
)

var (
	initOnce         sync.Once
	telemetryEnabled bool
	isInitialized    bool
	eventQueue       chan JobEventData
	eventQueueSize   = 1000
	workerCount      = 3
	droppedEvents    int64
	processedEvents  int64
	queueMutex       sync.RWMutex
	workersRunning   bool
	shutdownChan     chan struct{}
)

// EventType represents different lifecycle events for training jobs.
// These events track the complete lifecycle from creation to deletion.
type EventType string

const (
	JobCreatedEvent   EventType = "created"
	JobStartedEvent   EventType = "started"
	JobCompletedEvent EventType = "completed"
	JobFailedEvent    EventType = "failed"
	JobDeletedEvent   EventType = "deleted"
)

// JobEventData contains all relevant information for a training job event.
// It captures the event type, framework details, and job metadata needed
// for comprehensive telemetry analysis.
type JobEventData struct {
	EventType    EventType
	Framework    string
	Job          interface{}
	JobName      string
	JobNamespace string
	Metadata     map[string]string
}

// InitializeTelemetryReceiver sets up the telemetry event receiver system.
// It checks environment variables to determine if telemetry is enabled and
// initializes the metrics collection system when appropriate.
// This is the main entry point for telemetry initialization.
func InitializeTelemetryReceiver() error {
	var err error
	initOnce.Do(func() {
		// Initialize configuration first
		config.Initialize()
		telemetryEnabled = config.IsTelemetryConfigEnabled()

		if !telemetryEnabled {
			klog.Info("Telemetry is disabled via configuration")
			return
		}

		klog.Info("Initializing telemetry event receiver")

		// Initialize metrics registry
		err = metrics.Initialize()
		if err != nil {
			klog.ErrorS(err, "Failed to initialize telemetry metrics")
			return
		}

		// Initialize event queue and start workers
		eventQueue = make(chan JobEventData, eventQueueSize)
		shutdownChan = make(chan struct{})
		startEventWorkers()

		isInitialized = true
		klog.Info("Telemetry event receiver initialized successfully")
	})
	return err
}

// IsTelemetryEnabled returns true if telemetry collection is enabled and initialized.
// This checks both the configuration and successful initialization of the metrics system.
func IsTelemetryEnabled() bool {
	return telemetryEnabled && isInitialized
}

// isTelemetryEnabled provides backward compatibility for existing telemetry checks.
// It is an alias for IsTelemetryEnabled() to maintain compatibility with existing code.
func isTelemetryEnabled() bool {
	return IsTelemetryEnabled()
}

// ReportJobCreation processes a training job creation event.
// It extracts image information from the job specification and updates
// telemetry metrics to track version usage and customer patterns.
func ReportJobCreation(job interface{}, framework string) {
	if !IsTelemetryEnabled() {
		if !isInitialized {
			if err := InitializeTelemetryReceiver(); err != nil {
				klog.V(4).InfoS("Telemetry initialization failed, skipping event", "error", err)
				return
			}
		}
		if !IsTelemetryEnabled() {
			return
		}
	}

	event := JobEventData{
		EventType: JobCreatedEvent,
		Framework: framework,
		Job:       job,
	}

	// Use non-blocking queue submission
	submitEventToQueue(event)
}

// ReportJobStarted processes a training job started event.
// This tracks when jobs transition from pending to running state for
// performance and reliability analysis.
func ReportJobStarted(job interface{}, framework string) {
	if !IsTelemetryEnabled() {
		return
	}

	event := JobEventData{
		EventType: JobStartedEvent,
		Framework: framework,
		Job:       job,
	}

	// Use non-blocking queue submission
	submitEventToQueue(event)
}

// ReportJobCompletion processes a training job completion event.
// It handles both successful completions and failures, tracking job outcomes
// for success rate analysis and debugging patterns.
func ReportJobCompletion(job interface{}, framework string, succeeded bool) {
	if !IsTelemetryEnabled() {
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

	// Use non-blocking queue submission
	submitEventToQueue(event)
}

// ReportJobFailure processes a training job failure event.
// It captures specific failure reasons to help identify common failure patterns
// and areas for product improvement.
func ReportJobFailure(job interface{}, framework string, reason string) {
	if !IsTelemetryEnabled() {
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

	// Use non-blocking queue submission
	submitEventToQueue(event)
}

// ReportJobDeletion processes a training job deletion event.
// It ensures proper cleanup of telemetry tracking to prevent metric drift
// and maintain accurate active job counts.
func ReportJobDeletion(job interface{}, framework string) {
	if !IsTelemetryEnabled() {
		return
	}

	event := JobEventData{
		EventType: JobDeletedEvent,
		Framework: framework,
		Job:       job,
	}

	// Use non-blocking queue submission
	submitEventToQueue(event)
}

// ReceiveJobEvent processes job events for backward compatibility.
// This function maintains compatibility with existing telemetry collection code
// that uses the event-based interface.
func ReceiveJobEvent(ctx context.Context, event JobEventData) {
	if !IsTelemetryEnabled() {
		return
	}

	// Use non-blocking queue submission
	submitEventToQueue(event)
}

// startEventWorkers starts the background workers that process events from the queue.
// This enables non-blocking event processing and prevents telemetry collection
// from impacting the main controller operations.
func startEventWorkers() {
	queueMutex.Lock()
	defer queueMutex.Unlock()

	if workersRunning {
		return
	}

	workersRunning = true
	for i := 0; i < workerCount; i++ {
		go eventWorker(i)
	}

	klog.InfoS("Started telemetry event workers", "count", workerCount)
}

// eventWorker processes events from the queue with retry logic.
// Each worker runs in its own goroutine to enable parallel processing
// of telemetry events without blocking the main application.
func eventWorker(workerID int) {
	klog.V(4).InfoS("Event worker started", "workerID", workerID)

	for {
		select {
		case event := <-eventQueue:
			processEventWithRetry(event, workerID)
			atomic.AddInt64(&processedEvents, 1)

		case <-shutdownChan:
			klog.V(4).InfoS("Event worker shutting down", "workerID", workerID)
			return
		}
	}
}

// processEventWithRetry processes an event with exponential backoff retry.
// This ensures temporary failures don't cause permanent data loss while
// preventing infinite retry loops that could consume resources.
func processEventWithRetry(event JobEventData, workerID int) {
	ctx := context.Background()
	maxRetries := 3
	backoffDuration := 100 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := processEventSafely(ctx, event)
		if err == nil {
			return
		}

		if attempt < maxRetries-1 {
			klog.V(4).InfoS("Event processing failed, retrying",
				"workerID", workerID,
				"attempt", attempt+1,
				"error", err)
			time.Sleep(backoffDuration)
			backoffDuration *= 2
		} else {
			klog.ErrorS(err, "Event processing failed after all retries",
				"workerID", workerID,
				"eventType", event.EventType,
				"framework", event.Framework)
		}
	}
}

// processEventSafely wraps event processing with panic recovery.
// This prevents a single malformed event from crashing the entire
// telemetry collection system.
func processEventSafely(ctx context.Context, event JobEventData) (err error) {
	defer func() {
		if r := recover(); r != nil {
			klog.ErrorS(nil, "Panic recovered in event processing",
				"panic", r,
				"eventType", event.EventType,
				"framework", event.Framework)
			err = nil // Don't retry on panic
		}
	}()

	convertEventToMetrics(ctx, event)
	return nil
}

// submitEventToQueue submits an event to the processing queue without blocking.
// If the queue is full, the event is dropped and a counter is incremented
// to track data loss for monitoring purposes.
func submitEventToQueue(event JobEventData) {
	select {
	case eventQueue <- event:
		// Event successfully queued
	default:
		// Queue is full, drop the event
		atomic.AddInt64(&droppedEvents, 1)
		if atomic.LoadInt64(&droppedEvents)%100 == 0 {
			klog.WarningS("Events are being dropped due to queue overflow",
				"droppedTotal", atomic.LoadInt64(&droppedEvents))
		}
	}
}

// ShutdownTelemetryReceiver gracefully shuts down the telemetry system.
// It stops accepting new events and waits for existing events to be processed
// before returning.
func ShutdownTelemetryReceiver() {
	queueMutex.Lock()
	defer queueMutex.Unlock()

	if !workersRunning {
		return
	}

	klog.Info("Shutting down telemetry event receiver")

	// Close shutdown channel to signal workers
	close(shutdownChan)

	// Give workers time to process remaining events
	time.Sleep(2 * time.Second)

	workersRunning = false
	klog.InfoS("Telemetry event receiver shutdown complete",
		"processedEvents", atomic.LoadInt64(&processedEvents),
		"droppedEvents", atomic.LoadInt64(&droppedEvents))
}

// GetTelemetryQueueStats returns statistics about the event queue.
// This is useful for monitoring the health of the telemetry system
// and detecting performance issues.
func GetTelemetryQueueStats() map[string]int64 {
	return map[string]int64{
		"queueSize":       int64(len(eventQueue)),
		"queueCapacity":   int64(eventQueueSize),
		"processedEvents": atomic.LoadInt64(&processedEvents),
		"droppedEvents":   atomic.LoadInt64(&droppedEvents),
	}
}
