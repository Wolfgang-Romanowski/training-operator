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
	eventQueue       *EventQueue
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

// EventQueue provides backpressure handling and monitoring for event processing.
// It tracks queue metrics and implements exponential backoff for overload scenarios.
type EventQueue struct {
	queue          chan JobEventData
	maxSize        int
	droppedCounter int64
	processedCounter int64
	queueSizeGauge int64
	lastDropTime   time.Time
	backoffDuration time.Duration
	mu             sync.RWMutex
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

		// Initialize event queue with backpressure handling
		eventQueue = &EventQueue{
			queue:           make(chan JobEventData, eventQueueSize),
			maxSize:         eventQueueSize,
			backoffDuration: 100 * time.Millisecond,
		}
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
		case event := <-eventQueue.queue:
			processEventWithRetry(event, workerID)
			eventQueue.recordProcessed()

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
	if eventQueue == nil {
		klog.Warning("Event queue not initialized")
		return
	}
	eventQueue.Submit(event)
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
	if eventQueue == nil {
		return map[string]int64{
			"queueSize":       0,
			"queueCapacity":   int64(eventQueueSize),
			"processedEvents": 0,
			"droppedEvents":   0,
		}
	}
	return eventQueue.GetStats()
}

// Submit adds an event to the queue with backpressure handling.
// It implements exponential backoff when the queue is full to avoid overwhelming the system.
func (eq *EventQueue) Submit(event JobEventData) {
	eq.mu.RLock()
	currentSize := len(eq.queue)
	eq.mu.RUnlock()

	// Try to submit without blocking
	select {
	case eq.queue <- event:
		atomic.AddInt64(&eq.queueSizeGauge, 1)
		return
	default:
		// Queue is full, handle backpressure
	}

	// Check if we should apply backoff
	eq.mu.Lock()
	now := time.Now()
	if now.Sub(eq.lastDropTime) < eq.backoffDuration {
		// Still in backoff period, drop the event
		atomic.AddInt64(&eq.droppedCounter, 1)
		eq.mu.Unlock()
		
		if atomic.LoadInt64(&eq.droppedCounter)%100 == 0 {
			klog.WarningS("Events being dropped due to queue overflow with backpressure",
				"droppedTotal", atomic.LoadInt64(&eq.droppedCounter),
				"queueSize", currentSize,
				"backoffDuration", eq.backoffDuration)
		}
		return
	}
	eq.mu.Unlock()

	// Try one more time with a short timeout
	timer := time.NewTimer(10 * time.Millisecond)
	defer timer.Stop()
	
	select {
	case eq.queue <- event:
		atomic.AddInt64(&eq.queueSizeGauge, 1)
	case <-timer.C:
		// Timeout, drop the event and increase backoff
		eq.mu.Lock()
		atomic.AddInt64(&eq.droppedCounter, 1)
		eq.lastDropTime = now
		eq.backoffDuration = min(eq.backoffDuration*2, 5*time.Second)
		eq.mu.Unlock()
		
		klog.V(4).InfoS("Event dropped after timeout",
			"eventType", event.EventType,
			"framework", event.Framework,
			"newBackoff", eq.backoffDuration)
	}
}

// recordProcessed updates the processed event counter and adjusts queue size.
func (eq *EventQueue) recordProcessed() {
	atomic.AddInt64(&eq.processedCounter, 1)
	atomic.AddInt64(&eq.queueSizeGauge, -1)
	
	// Reset backoff if queue is healthy
	eq.mu.Lock()
	if len(eq.queue) < eq.maxSize/2 && eq.backoffDuration > 100*time.Millisecond {
		eq.backoffDuration = 100 * time.Millisecond
	}
	eq.mu.Unlock()
}

// GetStats returns current queue statistics for monitoring.
func (eq *EventQueue) GetStats() map[string]int64 {
	eq.mu.RLock()
	defer eq.mu.RUnlock()
	
	return map[string]int64{
		"queueSize":       int64(len(eq.queue)),
		"queueCapacity":   int64(eq.maxSize),
		"processedEvents": atomic.LoadInt64(&eq.processedCounter),
		"droppedEvents":   atomic.LoadInt64(&eq.droppedCounter),
	}
}

// min returns the minimum of two durations.
func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
