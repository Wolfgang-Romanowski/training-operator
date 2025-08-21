# Prometheus Legacy Code Removal Plan

## 🚨 Legacy Prometheus Code Found

Despite moving to OTEL-first architecture per RHOAISTRAT-575, there are still direct Prometheus dependencies that should be removed for full compliance.

## 1. Legacy Metrics Still Present ❌

### File: `pkg/common/metrics.go`
**Issue**: Contains OLD operational metrics that expose high-cardinality labels
```go
// These metrics violate Red Hat requirements:
training_operator_jobs_created_total{job_namespace="...", framework="..."}
training_operator_jobs_deleted_total{job_namespace="...", framework="..."}
training_operator_jobs_successful_total{job_namespace="...", framework="..."}
training_operator_jobs_failed_total{job_namespace="...", framework="..."}
training_operator_jobs_restarted_total{job_namespace="...", framework="..."}
```

**Problems**:
- ❌ **High cardinality**: `job_namespace` label can have unlimited values
- ❌ **Not business-focused**: Operational metrics, not business insights
- ❌ **Exceeds limits**: Would add 5 more metrics (total 8, exceeds limit of 3)
- ❌ **Direct Prometheus dependency**: Uses prometheus/client_golang directly

**Action Required**:
1. Remove this entire file
2. Remove all calls to these metrics from controllers
3. These operational metrics should be collected via OpenTelemetry spans/traces instead

## 2. Service Still Has Prometheus Annotations ❌

### File: `manifests/base/service.yaml`
```yaml
annotations:
  prometheus.io/path: /metrics
  prometheus.io/scrape: "true"
  prometheus.io/port: "8080"
```

**Problem**: These annotations could cause direct Prometheus scraping if a Prometheus instance is configured to honor them

**Fix Required**:
```yaml
apiVersion: v1
kind: Service
metadata:
  labels:
    app: training-operator
  name: training-operator
  # REMOVE all prometheus.io annotations
spec:
  ports:
    - name: monitoring-port
      port: 8080
      targetPort: 8080
```

## 3. Controllers Still Call Legacy Metrics ❌

Found in multiple controllers:
- `pkg/controller.v1/mpi/mpijob_controller.go`
- `pkg/controller.v1/jax/jaxjob_controller.go`
- `pkg/controller.v1/xgboost/xgboostjob_controller.go`
- `pkg/controller.v1/paddlepaddle/paddlepaddle_controller.go`
- `pkg/controller.v1/pytorch/pytorchjob_controller.go`
- `pkg/controller.v1/tensorflow/tfjob_controller.go`

**Calls to Remove**:
```go
trainingoperatorcommon.CreatedJobsCounterInc(...)
trainingoperatorcommon.DeletedJobsCounterInc(...)
trainingoperatorcommon.SuccessfulJobsCounterInc(...)
trainingoperatorcommon.FailedJobsCounterInc(...)
trainingoperatorcommon.RestartedJobsCounterInc(...)
```

## 4. Direct Prometheus Client Usage ❌

### Files with Direct Dependencies:
- `pkg/telemetry/metrics/metric_definitions.go` - Uses prometheus.NewGaugeVec directly
- `pkg/telemetry/metrics/metric_registry.go` - Uses prometheus.MustRegister

**Issue**: Per RHOAISTRAT-575, metrics should be exposed via OpenTelemetry SDK, not Prometheus client directly

## Recommended Full OTEL-First Implementation

### Option 1: OpenTelemetry Metrics SDK (Preferred) ✅

Replace Prometheus client with OpenTelemetry SDK:

```go
// pkg/telemetry/metrics/otel_metrics.go
package metrics

import (
    "context"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
    "go.opentelemetry.io/otel/exporters/prometheus"
    sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

var (
    meter metric.Meter
    versionGauge metric.Int64UpDownCounter
    sourceCounter metric.Int64Counter
    customerCounter metric.Int64Counter
)

func InitializeOTELMetrics() error {
    // Create Prometheus exporter for OTEL metrics
    exporter, err := prometheus.New()
    if err != nil {
        return err
    }
    
    // Create meter provider with the exporter
    provider := sdkmetric.NewMeterProvider(
        sdkmetric.WithReader(exporter),
    )
    
    // Register as global provider
    otel.SetMeterProvider(provider)
    
    // Get meter
    meter = provider.Meter("training-operator")
    
    // Create metrics using OTEL API
    versionGauge, err = meter.Int64UpDownCounter(
        "training_operator_image_version_usage",
        metric.WithDescription("Current runtime versions in use"),
    )
    
    sourceCounter, err = meter.Int64Counter(
        "training_operator_image_source_preference_total",
        metric.WithDescription("Image source preferences"),
    )
    
    customerCounter, err = meter.Int64Counter(
        "training_operator_enterprise_adoption_total",
        metric.WithDescription("Customer type classification"),
    )
    
    return nil
}

// Update metrics using OTEL API
func RecordImageVersionUsage(version string) {
    ctx := context.Background()
    versionGauge.Add(ctx, 1, metric.WithAttributes(
        attribute.String("version", version),
    ))
}
```

### Option 2: Keep Prometheus BUT Remove High-Cardinality Metrics

If we must keep Prometheus client for now:

1. **Delete `pkg/common/metrics.go`** completely
2. **Keep ONLY the 3 business metrics** in telemetry package
3. **Remove all prometheus.io annotations** from services
4. **Ensure labels are limited**:
   - version: 5 values max
   - image_source: 3 values max  
   - customer_type: 2 values max

## Migration Steps

### Phase 1: Remove Legacy Metrics (Immediate)
```bash
# 1. Delete the legacy metrics file
rm pkg/common/metrics.go

# 2. Remove prometheus annotations from service
# Edit manifests/base/service.yaml

# 3. Remove metric calls from controllers
# Update all controller files to remove Counter Inc calls
```

### Phase 2: Update Controllers
Replace operational metric calls with telemetry events:
```go
// OLD (remove):
trainingoperatorcommon.CreatedJobsCounterInc(job.Namespace, framework)

// NEW (already implemented):
telemetry.ReportJobCreation(job, framework)
```

### Phase 3: (Optional) Migrate to OTEL SDK
```bash
# Add OpenTelemetry dependencies
go get go.opentelemetry.io/otel
go get go.opentelemetry.io/otel/metric
go get go.opentelemetry.io/otel/exporters/prometheus

# Update metric implementation to use OTEL SDK
```

## Compliance Impact

### Current State (With Legacy Code):
- ❌ 8 total metrics (5 legacy + 3 telemetry)
- ❌ High cardinality from `job_namespace` label
- ❌ Direct Prometheus scraping possible via annotations
- ❌ Mixed metrics paradigm (operational + business)

### After Removal:
- ✅ 3 total metrics (business-focused only)
- ✅ 10 total timeseries (within limit)
- ✅ No high-cardinality labels
- ✅ Pure OTEL collection path
- ✅ Full RHOAISTRAT-575 compliance

## Summary

To achieve 100% compliance with RHOAISTRAT-575:

1. **MUST**: Remove `pkg/common/metrics.go` and all its usage
2. **MUST**: Remove prometheus.io annotations from service.yaml
3. **SHOULD**: Migrate to OpenTelemetry Metrics SDK
4. **KEEP**: Only the 3 business-focused telemetry metrics

The legacy Prometheus metrics are:
- Violating cardinality limits
- Not providing business value
- Creating confusion with dual metric systems
- Enabling potential duplicate scraping

Removing them will complete the transition to a pure OTEL-first architecture as mandated by the Red Hat platform team.