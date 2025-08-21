// Copyright 2021 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License

// Package common provides shared utilities for training-operator controllers.
// The metrics functions serve as a compatibility layer that delegates to the
// telemetry system for RHOAISTRAT-575 compliant business metrics.
package common

import (
	"context"
	"time"
	
	"k8s.io/klog/v2"
	
	"github.com/kubeflow/training-operator/pkg/telemetry"
)

// CreatedJobsCounterInc reports job creation metrics.
// This function is maintained for backward compatibility with existing controllers.
// It delegates to the telemetry system which tracks business-relevant metrics
// with controlled cardinality per RHOAISTRAT-575 requirements.
func CreatedJobsCounterInc(jobNamespace, framework string) {
	if !telemetry.IsTelemetryEnabled() {
		return
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	event := telemetry.JobEventData{
		EventType:    telemetry.JobCreatedEvent,
		Framework:    framework,
		JobNamespace: jobNamespace,
	}
	
	telemetry.ReceiveJobEvent(ctx, event)
}

// DeletedJobsCounterInc reports job deletion metrics.
// Maintained for backward compatibility, delegates to telemetry system.
func DeletedJobsCounterInc(jobNamespace, framework string) {
	if !telemetry.IsTelemetryEnabled() {
		return
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	event := telemetry.JobEventData{
		EventType:    telemetry.JobDeletedEvent,
		Framework:    framework,
		JobNamespace: jobNamespace,
	}
	
	telemetry.ReceiveJobEvent(ctx, event)
}

// SuccessfulJobsCounterInc reports successful job completion metrics.
// Maintained for backward compatibility, delegates to telemetry system.
func SuccessfulJobsCounterInc(jobNamespace, framework string) {
	if !telemetry.IsTelemetryEnabled() {
		return
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	event := telemetry.JobEventData{
		EventType:    telemetry.JobCompletedEvent,
		Framework:    framework,
		JobNamespace: jobNamespace,
		Metadata: map[string]string{
			"succeeded": "true",
		},
	}
	
	telemetry.ReceiveJobEvent(ctx, event)
}

// FailedJobsCounterInc reports failed job metrics.
// Maintained for backward compatibility, delegates to telemetry system.
func FailedJobsCounterInc(jobNamespace, framework string) {
	if !telemetry.IsTelemetryEnabled() {
		return
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	event := telemetry.JobEventData{
		EventType:    telemetry.JobFailedEvent,
		Framework:    framework,
		JobNamespace: jobNamespace,
		Metadata: map[string]string{
			"succeeded": "false",
		},
	}
	
	telemetry.ReceiveJobEvent(ctx, event)
}

// RestartedJobsCounterInc reports job restart metrics.
// Maintained for backward compatibility, delegates to telemetry system.
func RestartedJobsCounterInc(jobNamespace, framework string) {
	if !telemetry.IsTelemetryEnabled() {
		return
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	event := telemetry.JobEventData{
		EventType:    telemetry.JobStartedEvent,
		Framework:    framework,
		JobNamespace: jobNamespace,
		Metadata: map[string]string{
			"restarted": "true",
		},
	}
	
	telemetry.ReceiveJobEvent(ctx, event)
}

// ReportJobWithDetails provides a more detailed reporting interface that controllers
// can use when they have access to the full job object. This enables richer
// telemetry data collection for business metrics.
func ReportJobWithDetails(eventType telemetry.EventType, framework string, job interface{}) {
	if !telemetry.IsTelemetryEnabled() {
		return
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	event := telemetry.JobEventData{
		EventType: eventType,
		Framework: framework,
		Job:       job,
	}
	
	telemetry.ReceiveJobEvent(ctx, event)
}