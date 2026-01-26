# Video Test Suite

Complete end-to-end testing suite for video advertising functionality.

## Quick Start

Run the complete test suite:

```bash
./scripts/run_video_tests.sh
```

## Test Categories

### 1. Unit Tests

**VAST Library Tests** (`pkg/vast/`)
- `vast_test.go` - VAST parsing and generation
- `validator_test.go` - VAST validation rules
- `builder_test.go` - Fluent builder API (if exists)

```bash
go test ./pkg/vast/... -v -cover
```

**Video Handler Tests** (`internal/endpoints/`)
- `video_handler_test.go` - Endpoint tests (if exists)

```bash
go test ./internal/endpoints/video_* -v
```

### 2. Integration Tests

**Outbound VAST** (`tests/integration/video_outbound_test.go`)
- GET /video/vast endpoint
- POST /video/openrtb endpoint
- VAST structure validation
- Tracking URL generation
- Macro handling
- No-bid scenarios

**Inbound VAST** (`tests/integration/video_inbound_test.go`)
- VAST XML parsing (inline, wrapper, empty)
- Wrapper unwrapping
- Tracking event extraction
- Media file selection
- Companion ads parsing

**Video Adapters** (`tests/integration/video_adapters_test.go`)
- Multiple demand partner integration
- Parallel bidding
- Timeout handling
- Format preferences
- Performance benchmarks

**Video Caching** (`tests/integration/video_cache_test.go`)
- Redis caching for VAST responses
- Creative URL caching
- Cache expiration
- Performance improvements

**Error Handling** (`tests/integration/video_error_handling_test.go`)
- Missing parameters
- Invalid VAST from demand
- Timeouts
- Network failures
- Malformed data

**Tracking Pixels** (`tests/integration/video_tracking_test.go`)
- All tracking events (impression, start, quartiles, complete, etc.)
- Event sequencing
- Concurrent tracking
- POST and GET methods

**OpenRTB Compliance** (`tests/integration/openrtb_video_compliance_test.go`)
- Required video fields
- Protocol enumeration
- API frameworks
- Placement types
- All OpenRTB 2.x video parameters

```bash
# All integration tests
go test -tags=integration ./tests/integration/video_* -v

# Specific test
go test -tags=integration ./tests/integration/video_outbound_test.go -v

# With race detection
go test -tags=integration -race ./tests/integration/video_* -v
```

### 3. Performance Benchmarks

**Benchmarks** (`tests/benchmark/video_benchmark_test.go`)
- VAST generation (target: < 1ms)
- VAST parsing (target: < 2ms)
- Complete auction cycle (target: < 100ms)
- Concurrent operations
- Memory allocations

```bash
# Run all benchmarks
go test -bench=. ./tests/benchmark/video_benchmark_test.go

# With memory profiling
go test -bench=. -benchmem ./tests/benchmark/video_benchmark_test.go

# Specific benchmark
go test -bench=BenchmarkVASTGeneration ./tests/benchmark/video_benchmark_test.go
```

## Test Fixtures

### Bid Requests (`tests/fixtures/video_bid_requests.json`)
- `basic_video_request` - Standard in-stream video
- `instream_video_request` - In-stream with companions
- `outstream_video_request` - Out-stream video
- `ctv_video_request` - Connected TV (4K)
- `video_pod_request` - Ad pod (3 sequential ads)
- `mobile_video_request` - Mobile video
- `vpaid_video_request` - VPAID interactive
- `vast_4_only_request` - VAST 4.0 protocols

### Bid Responses (`tests/fixtures/video_bid_responses.json`)
- `basic_video_response` - Inline VAST
- `wrapper_video_response` - Wrapper VAST
- `multiple_bids_response` - Multiple bidders
- `ad_pod_response` - Ad pod responses
- `companion_ads_response` - With companion banners
- `skippable_video_response` - Skippable ads
- `vpaid_response` - VPAID creative
- `no_bid_response` - No bid scenario
- `ctv_4k_response` - 4K video
- `extended_tracking_response` - Full tracking events

## Coverage Requirements

Minimum coverage targets:
- **VAST Package**: 90%
- **Video Endpoints**: 85%
- **Overall Video Code**: 80%

Generate coverage report:

```bash
# Run tests with coverage
go test -cover ./pkg/vast/... -coverprofile=coverage.out

# View in browser
go tool cover -html=coverage.out
```

## Race Condition Detection

Always run with race detector before committing:

```bash
go test -race ./pkg/vast/...
go test -tags=integration -race ./tests/integration/video_*
```

## Performance Targets

| Operation | Target | Benchmark |
|-----------|--------|-----------|
| VAST Generation | < 1ms | `BenchmarkVASTGeneration` |
| VAST Parsing | < 2ms | `BenchmarkVASTParsing` |
| Video Auction | < 100ms | `BenchmarkVASTResponseBuilder` |
| Event Tracking | < 10ms | (Integration test) |
| Cache Hit | < 5ms | (Cache test) |

## Continuous Integration

Example GitHub Actions workflow:

```yaml
name: Video Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      redis:
        image: redis:7
        ports:
          - 6379:6379

    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Run video tests
        run: ./scripts/run_video_tests.sh

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./test-results/combined_coverage.out
```

## Test Fixtures Usage

Load fixtures in tests:

```go
import (
    "encoding/json"
    "os"
)

func loadBidRequest(name string) *openrtb.BidRequest {
    data, _ := os.ReadFile("tests/fixtures/video_bid_requests.json")
    var fixtures map[string]*openrtb.BidRequest
    json.Unmarshal(data, &fixtures)
    return fixtures[name]
}

// Usage
bidReq := loadBidRequest("basic_video_request")
```

## Debugging Tests

Enable verbose output:

```bash
# Verbose mode
go test -v ./tests/integration/video_outbound_test.go

# Show test names only
go test -v ./tests/integration/... | grep -E '(PASS|FAIL|RUN)'

# Run specific test
go test -v -run TestOutboundVAST ./tests/integration/video_outbound_test.go
```

View logs in failing tests:

```go
t.Logf("Debug info: %+v", data)  // Logs only shown if test fails
```

## Common Issues

### Redis Not Available

If cache tests fail:

```bash
# Start Redis
docker run -d -p 6379:6379 redis:7

# Or skip cache tests
go test -tags=integration -skip Cache ./tests/integration/...
```

### Timeout in Integration Tests

Increase timeout:

```bash
go test -tags=integration -timeout 60s ./tests/integration/video_*
```

### Test Data Not Found

Ensure you're running from project root:

```bash
cd /path/to/tnevideo
./scripts/run_video_tests.sh
```

## Best Practices

1. **Always run race detector** before committing
2. **Use table-driven tests** for multiple scenarios
3. **Mock external dependencies** (HTTP servers, databases)
4. **Test error paths** not just happy paths
5. **Use test fixtures** for consistent test data
6. **Measure performance** with benchmarks
7. **Clean up resources** (defer server.Close())
8. **Use subtests** for better organization

## Example Test Structure

```go
func TestFeature(t *testing.T) {
    t.Run("Happy_path", func(t *testing.T) {
        // Test successful scenario
    })

    t.Run("Error_case", func(t *testing.T) {
        // Test error handling
    })

    t.Run("Edge_case", func(t *testing.T) {
        // Test edge cases
    })
}
```

## Reporting Issues

When reporting test failures, include:

1. Full test output (`go test -v`)
2. Go version (`go version`)
3. OS and architecture
4. Redis version (for cache tests)
5. Relevant logs

## Contributing

When adding new video features:

1. Write tests first (TDD)
2. Ensure all existing tests pass
3. Add integration test for new functionality
4. Update fixtures if needed
5. Run full test suite
6. Check coverage didn't decrease
