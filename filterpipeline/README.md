# Filter Pipeline

The Filter Pipeline provides a flexible middleware architecture for bid request enrichment and bid response modification in prebid-server. It implements a two-phase filtering system with pre-filters and post-filters.

## Architecture Overview

The pipeline consists of two main filter types:

### Pre-Filters
Pre-filters execute **before** auction processing and can:
- Enrich incoming requests with additional data
- Validate request structure and content
- Add identity information
- Reject invalid requests early
- Transform request data

### Post-Filters
Post-filters execute **after** auction processing and can:
- Modify bid responses
- Enforce business policies
- Filter bids based on rules
- Add tracking or analytics data
- Transform response data

## Key Features

- **Priority-based Execution**: Filters execute in order of priority (lower values first)
- **Configurable Pipeline**: Enable/disable the entire pipeline or individual filters
- **Error Handling**: Choose between fail-fast or continue-on-error modes
- **Timeout Protection**: Set maximum execution time for filter stages
- **Metrics Integration**: Track filter execution time, errors, and rejections
- **Account-level Control**: Enable filters per account

## Usage

### Creating a Pipeline

```go
import "github.com/prebid/prebid-server/v3/filterpipeline"

// Create pipeline with default configuration
config := filterpipeline.DefaultPipelineConfig()
metrics := filterpipeline.NewNoOpFilterMetrics()
pipeline := filterpipeline.NewPipeline(config, metrics)
```

### Custom Configuration

```go
config := filterpipeline.PipelineConfig{
    Enabled:               true,
    MaxPreFilterDuration:  100 * time.Millisecond,
    MaxPostFilterDuration: 100 * time.Millisecond,
    ContinueOnError:       true,
    ParallelExecution:     false,
}
```

### Registering Filters

```go
// Register a pre-filter
identityFilter := filters.NewIdentityEnrichmentFilter(true, 10)
err := pipeline.RegisterPreFilter(identityFilter)

// Register a post-filter
policyFilter := filters.NewPolicyEnforcerFilter(filters.PolicyEnforcerConfig{
    Enabled:     true,
    Priority:    10,
    MaxBidPrice: 100.0,
    MinBidPrice: 0.01,
})
err := pipeline.RegisterPostFilter(policyFilter)
```

### Executing Filters

```go
// Execute pre-filters
enrichedRequest, err := pipeline.ExecutePreFilters(
    ctx,
    request,
    accountID,
    endpoint,
)

// Execute post-filters
filteredResponse, err := pipeline.ExecutePostFilters(
    ctx,
    request,
    response,
    accountID,
    endpoint,
)
```

## Built-in Filters

### Identity Enrichment Filter (Pre-filter)

Enriches requests with identity information and captures metadata about user and device presence.

```go
filter := filters.NewIdentityEnrichmentFilter(
    enabled bool,
    priority int,
)
```

**Features:**
- Detects presence of user and device objects
- Captures user ID if available
- Records device identifiers (UA, IP, IFA)
- Adds identity metadata for downstream use

### Request Validator Filter (Pre-filter)

Validates incoming requests against configured rules.

```go
filter := filters.NewRequestValidatorFilter(filters.RequestValidatorConfig{
    Enabled:        true,
    Priority:       5,
    RequireUser:    true,
    RequireDevice:  true,
    MinImpressions: 1,
})
```

**Features:**
- Enforces required fields (user, device)
- Validates minimum impression count
- Checks impression IDs
- Rejects invalid requests early

### Policy Enforcer Filter (Post-filter)

Enforces business policies on bid responses.

```go
filter := filters.NewPolicyEnforcerFilter(filters.PolicyEnforcerConfig{
    Enabled:        true,
    Priority:       10,
    MaxBidPrice:    100.0,
    MinBidPrice:    0.01,
    AllowedBidders: []string{"bidder1", "bidder2"},
})
```

**Features:**
- Enforces min/max bid price constraints
- Filters bids from disallowed bidders
- Removes empty seat bids
- Reports policy violations

## Creating Custom Filters

### Pre-filter Example

```go
type MyPreFilter struct {
    enabled  bool
    priority int
}

func (f *MyPreFilter) Name() string {
    return "my_pre_filter"
}

func (f *MyPreFilter) Priority() int {
    return f.priority
}

func (f *MyPreFilter) Enabled(accountID string) bool {
    return f.enabled
}

func (f *MyPreFilter) Execute(ctx context.Context, req filterpipeline.PreFilterRequest) (filterpipeline.PreFilterResponse, error) {
    resp := filterpipeline.PreFilterResponse{
        Request:  req.Request,
        Metadata: make(map[string]interface{}),
    }

    // Your filter logic here

    return resp, nil
}
```

### Post-filter Example

```go
type MyPostFilter struct {
    enabled  bool
    priority int
}

func (f *MyPostFilter) Name() string {
    return "my_post_filter"
}

func (f *MyPostFilter) Priority() int {
    return f.priority
}

func (f *MyPostFilter) Enabled(accountID string) bool {
    return f.enabled
}

func (f *MyPostFilter) Execute(ctx context.Context, req filterpipeline.PostFilterRequest) (filterpipeline.PostFilterResponse, error) {
    resp := filterpipeline.PostFilterResponse{
        Response: req.Response,
        Metadata: make(map[string]interface{}),
    }

    // Your filter logic here

    return resp, nil
}
```

## Filter Rejection

Filters can reject requests or responses by setting the `Reject` field:

```go
resp := filterpipeline.PreFilterResponse{
    Reject:       true,
    RejectReason: "Request failed validation",
}
```

When a filter rejects, the pipeline stops execution and returns a `FilterRejectionError`.

## Metadata Passing

Filters can pass data to subsequent filters using metadata:

```go
resp := filterpipeline.PreFilterResponse{
    Request: modifiedRequest,
    Metadata: map[string]interface{}{
        "enriched_fields": []string{"user_id", "device_type"},
        "validation_passed": true,
    },
}
```

Metadata is accumulated across filters and available in the filter context.

## Error Handling

The pipeline supports two error handling modes:

1. **Continue on Error** (default): Pipeline continues even if a filter fails
2. **Fail Fast**: Pipeline stops on first filter error

Configure via `PipelineConfig.ContinueOnError`.

## Performance Considerations

- Set appropriate timeouts for your use case
- Lower priority values execute first
- Disable filters not needed for specific accounts
- Use metrics to identify slow filters
- Consider filter execution cost vs. benefit

## Testing

The package includes comprehensive test coverage:

```bash
go test ./filterpipeline/...
go test ./filterpipeline/filters/...
```

## Integration Points

The filter pipeline is designed to integrate with:
- OpenRTB request/response processing
- Account configuration system
- Metrics collection
- Logging infrastructure

## Future Enhancements

Potential future additions:
- Parallel filter execution
- Filter chaining and composition
- Dynamic filter loading
- Filter result caching
- Advanced routing rules
