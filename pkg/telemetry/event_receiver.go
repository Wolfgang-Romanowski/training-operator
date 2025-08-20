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

// Initialize sets up the telemetry system.
// It checks environment variables to determine if telemetry is enabled and
// initializes the metrics collection system when appropriate.
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

// IsEnabled returns true if telemetry collection is enabled and initialized.
// This checks both the environment variable configuration and successful
// initialization of the metrics system.
func IsEnabled() bool {
	return telemetryEnabled && isInitialized
}

// isTelemetryEnabled provides backward compatibility for existing telemetry checks.
// It is an alias for IsEnabled() to maintain compatibility with existing code.
func isTelemetryEnabled() bool {
	return IsEnabled()
}

// ReportJobCreation processes a training job creation event.
// It extracts image information from the job specification and updates
// telemetry metrics to track version usage and customer patterns.
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

// ReportJobStarted processes a training job started event.
// This tracks when jobs transition from pending to running state for
// performance and reliability analysis.
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

// ReportJobCompletion processes a training job completion event.
// It handles both successful completions and failures, tracking job outcomes
// for success rate analysis and debugging patterns.
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

// ReportJobFailure processes a training job failure event.
// It captures specific failure reasons to help identify common failure patterns
// and areas for product improvement.
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

// ReportJobDeletion processes a training job deletion event.
// It ensures proper cleanup of telemetry tracking to prevent metric drift
// and maintain accurate active job counts.
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

// ReceiveJobEvent processes job events for backward compatibility.
// This function maintains compatibility with existing telemetry collection code
// that uses the event-based interface.
func ReceiveJobEvent(ctx context.Context, event JobEventData) {
	if !IsEnabled() {
		return
	}

	convertEventToMetrics(ctx, event)
}
