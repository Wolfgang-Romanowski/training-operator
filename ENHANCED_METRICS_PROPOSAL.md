# Enhanced Metrics Proposal for Red Hat Business Requirements

## Current State Issues

### Cardinality Violations Fixed
- **Issue**: Image source could have 4 values (rhoai_official, community, custom, unknown)
- **Fix**: Mapped "unknown" → "custom" to maintain 3 values max
- **Result**: Guaranteed 10 timeseries total (5+3+2)

### Missing Business Intelligence

The current 3 metrics don't provide sufficient business intelligence for Red Hat's needs:

## Proposed Enhanced Metrics Architecture

### Option 1: Replace Current Metrics (Stay within 10 timeseries)

Replace the current 3 metrics with more valuable business metrics:

```yaml
# METRIC 1: GPU vs CPU Usage (2 timeseries)
training_operator_accelerator_usage:
  labels: ["accelerator_type"] # gpu, cpu
  
# METRIC 2: RHOAI Component Usage (4 timeseries)  
training_operator_component_usage:
  labels: ["component"] # training, notebook, ray, other
  
# METRIC 3: Customer Segmentation (4 timeseries)
training_operator_customer_profile:
  labels: ["profile"] # enterprise_gpu, enterprise_cpu, community_gpu, community_cpu
```

**Total: 10 timeseries** ✅

### Option 2: Use Recording Rules for Aggregation (Recommended)

Keep base metrics simple, use recording rules for business intelligence:

#### Base Metrics (10 timeseries):
```yaml
# METRIC 1: Job Activity (1 timeseries - gauge)
training_operator_active_jobs_total

# METRIC 2: Resource Type (2 timeseries)
training_operator_resource_type_total:
  labels: ["type"] # gpu, cpu

# METRIC 3: Image Classification (7 timeseries)
training_operator_image_classification_total:
  labels: ["classification"] 
  # rhoai_pytorch, rhoai_tensorflow, rhoai_ray, 
  # community_pytorch, community_tensorflow, 
  # custom, other
```

#### Recording Rules (Business Intelligence):
```yaml
# Business Metric 1: GPU Adoption Rate
openshift:training_gpu_adoption_percentage:
  expr: |
    (sum(training_operator_resource_type_total{type="gpu"}) / 
     sum(training_operator_resource_type_total)) * 100

# Business Metric 2: RHOAI Market Share
openshift:training_rhoai_market_share:
  expr: |
    (sum(training_operator_image_classification_total{classification=~"rhoai_.*"}) /
     sum(training_operator_image_classification_total)) * 100

# Business Metric 3: Framework Distribution
openshift:training_framework_distribution:
  expr: |
    sum by (framework) (
      label_replace(
        training_operator_image_classification_total,
        "framework", "$1", "classification", ".*_(pytorch|tensorflow|ray).*"
      )
    )

# Business Metric 4: Enterprise GPU Usage (High Value)
openshift:training_enterprise_gpu_jobs:
  expr: |
    training_operator_resource_type_total{type="gpu"} * 
    on() group_left() 
    (training_operator_active_jobs_total > 10)
```

### Option 3: Multi-Dimensional Analysis (Advanced)

Use a single metric with composite labels encoded:

```yaml
# Single Metric with Encoded Dimensions (10 timeseries max)
training_operator_job_profile_total:
  labels: ["profile"]
  # Encoded as: <source>_<accel>_<segment>
  values:
    - "rhoai_gpu_enterprise"      # High value
    - "rhoai_gpu_community"        # Growth opportunity
    - "rhoai_cpu_enterprise"       # Standard enterprise
    - "rhoai_cpu_community"        # Entry level
    - "community_gpu_all"          # Competitive threat
    - "community_cpu_all"          # Low priority
    - "custom_gpu_all"             # Advanced users
    - "custom_cpu_all"             # DIY segment
    - "notebook_all_all"           # Notebook usage
    - "other"                      # Catch-all
```

## Critical Business Questions Answered

### With Current Metrics ❌
1. **GPU vs CPU usage?** - Not tracked
2. **Job size/scale?** - Not tracked
3. **Notebook vs Training?** - Not tracked
4. **Multi-tenancy?** - Not tracked
5. **Resource consumption?** - Not tracked

### With Enhanced Metrics ✅
1. **GPU Adoption**: Clear GPU vs CPU split
2. **Component Usage**: Notebooks vs Training vs Ray
3. **Customer Value**: Enterprise GPU users (highest value)
4. **Competitive Analysis**: RHOAI vs Community adoption
5. **Resource Planning**: Actual resource consumption patterns

## Implementation Strategy

### Phase 1: Fix Current Cardinality Issues ✅
- Map "unknown" → "custom" (DONE)
- Add validation functions (DONE)
- Ensure strict 10 timeseries limit (DONE)

### Phase 2: Enhanced Recording Rules
Add recording rules that extract more business value from existing metrics:

```yaml
# GPU Revenue Potential
openshift:training_gpu_revenue_potential:
  expr: |
    sum(training_operator_image_version_usage{version=~".*gpu.*"}) * 1000

# Deprecation Readiness
openshift:training_deprecation_safety_score:
  expr: |
    (sum(training_operator_image_version_usage{version="other"}) /
     sum(training_operator_image_version_usage)) * 100
```

### Phase 3: Future Architecture (v2)
When Red Hat increases timeseries limit:

```yaml
# Comprehensive Metrics (30 timeseries)
- training_operator_job_details (10 timeseries)
  - framework, version, accelerator, scale, namespace_count
  
- training_operator_resource_consumption (10 timeseries)
  - cpu_cores, memory_gb, gpu_count, storage_gb, duration_hours
  
- training_operator_business_classification (10 timeseries)
  - customer_segment, revenue_tier, support_level, region, cloud_provider
```

## Recommended Approach

**Use Option 2** - Recording Rules for Business Intelligence:
1. Maintains strict 10 timeseries limit
2. Provides rich business intelligence through aggregation
3. Allows evolution without breaking changes
4. Follows Red Hat Monitoring Handbook best practices

## Validation Checklist

- [x] Maximum 10 timeseries guaranteed
- [x] No "unknown" values that create 4th dimension
- [x] Validation functions prevent cardinality creep
- [x] Circuit breaker stops at 10 timeseries
- [x] Recording rules provide business value
- [ ] Add GPU/CPU differentiation
- [ ] Add component type tracking
- [ ] Add resource consumption metrics

## Business Value Summary

Current metrics answer:
- Which versions can we deprecate?
- What % use RHOAI images?
- Enterprise vs community split?

Enhanced metrics would answer:
- **How many GPU workloads?** (pricing tier)
- **What scale of jobs?** (resource planning)
- **Which components used?** (product focus)
- **Resource consumption?** (cost analysis)
- **Growth trends?** (business planning)