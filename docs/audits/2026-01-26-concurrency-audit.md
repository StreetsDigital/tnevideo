# Concurrency Safety Audit Report

**Date**: 2026-01-26  
**Auditor**: Concurrency Cop (AI Agent)  
**Codebase**: tnevideo/tnevideo (Prebid Ad Exchange)  
**Focus Areas**: internal/exchange/, internal/adapters/, pkg/

---

## Executive Summary

This audit analyzed the Go codebase for concurrency safety issues including data races, goroutine leaks, deadlocks, and improper synchronization. The codebase demonstrates generally good concurrency practices with proper mutex usage and bounded goroutine patterns. Several areas require attention, ranked by likelihood of production incident.

---

## Findings by Severity

### CRITICAL - Immediate Production Risk

**(None Found)** - The codebase appears to have addressed major concurrency issues with recent fixes (P0-1 through P3-1 comments visible in code).

---

### HIGH - Potential Production Issues

#### 1. IVTDetector sync.Once Reset Without Lock Synchronization
**File**: `/Users/andrewstreets/tnevideo/tnevideo/internal/middleware/ivt_detector.go`  
**Lines**: 550-557

```go
func (d *IVTDetector) SetConfig(config *IVTConfig) {
    d.mu.Lock()
    defer d.mu.Unlock()
    d.config = config

    // Reset compiled patterns to force recompile
    d.patternsOnce = sync.Once{}  // <-- ISSUE
}
```

**Issue**: Resetting `sync.Once` while other goroutines may be executing `compilePatterns()` can cause data races. The `patternsOnce.Do()` in `compilePatterns()` acquires `mu.RLock()` which doesn't block `SetConfig()`.

**Impact**: Pattern compilation could be corrupted if SetConfig is called during validation.

**Severity**: HIGH - Can cause invalid IVT detection during config hot-reload.

**Recommendation**: Use a separate mutex for pattern compilation or ensure SetConfig blocks until all ongoing validations complete.

---

#### 2. Circuit Breaker Callback Goroutine Leak Potential
**File**: `/Users/andrewstreets/tnevideo/tnevideo/pkg/idr/circuitbreaker.go`  
**Lines**: 179-206

```go
func (cb *CircuitBreaker) setState(newState string) {
    // ...
    if cb.config.OnStateChange != nil {
        cb.callbackWg.Add(1)
        go func(from, to string) {
            defer cb.callbackWg.Done()
            
            ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
            defer cancel()
            
            done := make(chan struct{})
            go func() {  // <-- NESTED GOROUTINE
                defer close(done)
                cb.config.OnStateChange(from, to)
            }()
            
            select {
            case <-done:
            case <-ctx.Done():
                // Callback timed out - log warning but continue
            }
        }(oldState, newState)
    }
}
```

**Issue**: If `OnStateChange` callback blocks forever, the nested goroutine leaks. The outer goroutine completes after 5s timeout, but the inner goroutine persists.

**Impact**: Memory leak proportional to number of blocked callbacks.

**Severity**: HIGH - Each stuck callback leaks a goroutine indefinitely.

**Recommendation**: Ensure callbacks are non-blocking or add panic recovery with logging.

---

#### 3. PublisherAuth Multiple Lock Ordering Risk
**File**: `/Users/andrewstreets/tnevideo/tnevideo/internal/middleware/publisher_auth.go`  
**Lines**: 97-105

```go
type PublisherAuth struct {
    mu             sync.RWMutex       // Lock A

    rateLimits     map[string]*rateLimitEntry
    rateLimitsMu   sync.RWMutex       // Lock B

    publisherCache   map[string]*publisherCacheEntry
    publisherCacheMu sync.RWMutex     // Lock C
}
```

**Issue**: Three separate RWMutex protecting related state. If lock ordering is violated between concurrent operations, deadlock is possible.

**Analysis**: Current code appears to maintain consistent ordering (mu -> rateLimitsMu, mu -> publisherCacheMu), but this is fragile.

**Severity**: HIGH - Deadlock risk if future code changes violate ordering.

**Recommendation**: Document required lock ordering explicitly or consider consolidating to fewer locks.

---

### MEDIUM - Potential Issues Under Load

#### 4. Dashboard Global Metrics - Lock Contention
**File**: `/Users/andrewstreets/tnevideo/tnevideo/internal/endpoints/dashboard.go`  
**Lines**: 41-46, 49-92

```go
var globalMetrics = &DashboardMetrics{...}

func LogAuction(...) {
    globalMetrics.mu.Lock()
    defer globalMetrics.mu.Unlock()
    // Extensive work under lock including slice operations
    globalMetrics.RecentAuctions = append([]AuctionLog{auctionLog}, globalMetrics.RecentAuctions...)
    if len(globalMetrics.RecentAuctions) > 100 {
        globalMetrics.RecentAuctions = globalMetrics.RecentAuctions[:100]
    }
}
```

**Issue**: Every auction holds a write lock on global metrics, including slice prepend operations that allocate memory.

**Impact**: Under high load (1000+ QPS), lock contention could add latency to auction responses.

**Severity**: MEDIUM - Performance degradation under load.

**Recommendation**: Use atomic counters for simple metrics, consider ring buffer for recent auctions.

---

#### 5. EventRecorder Buffer Slice Operations
**File**: `/Users/andrewstreets/tnevideo/tnevideo/pkg/idr/events.go`  
**Lines**: 180-207

```go
func (r *EventRecorder) RecordBidResponse(...) {
    r.mu.Lock()
    r.buffer = append(r.buffer, event)
    shouldFlush := len(r.buffer) >= r.bufferSize
    var eventsToFlush []BidEvent
    if shouldFlush {
        eventsToFlush = make([]BidEvent, len(r.buffer))
        copy(eventsToFlush, r.buffer)
        r.buffer = make([]BidEvent, 0, r.bufferSize)  // <-- Allocates under lock
    }
    r.mu.Unlock()
    // ...
}
```

**Issue**: Memory allocation (`make()`) occurs while holding the lock.

**Impact**: GC pressure could cause lock holders to pause, blocking all event recording.

**Severity**: MEDIUM - Latency spikes during GC.

**Recommendation**: Pre-allocate buffers or use sync.Pool for buffer reuse.

---

#### 6. HTTP Client Response Read Goroutine
**File**: `/Users/andrewstreets/tnevideo/tnevideo/internal/adapters/adapter.go`  
**Lines**: 252-300

```go
func (c *DefaultHTTPClient) Do(ctx context.Context, req *RequestData, timeout time.Duration) (*ResponseData, error) {
    // ...
    readCh := make(chan readResult, 1)

    go func() {
        defer resp.Body.Close()
        limitedReader := io.LimitReader(resp.Body, maxResponseSize+1)
        data, err := io.ReadAll(limitedReader)
        readCh <- readResult{data: data, err: err}
    }()

    select {
    case <-ctx.Done():
        resp.Body.Close()  // <-- Double close if goroutine also closes
        result := <-readCh
        // ...
```

**Issue**: On context cancellation, `resp.Body.Close()` is called, then the goroutine may also call it via defer.

**Analysis**: http.Response.Body.Close() is documented as idempotent, so this is safe but wasteful.

**Severity**: MEDIUM - Unnecessary work, potential confusion.

**Recommendation**: Already appears to be handled correctly (goroutine drains channel), but consider restructuring to avoid double-close pattern.

---

### LOW - Minor Issues / Best Practice Violations

#### 7. Rate Limiter Map Growth
**File**: `/Users/andrewstreets/tnevideo/tnevideo/internal/middleware/ratelimit.go`  
**Lines**: 88-91

```go
type RateLimiter struct {
    clients map[string]*clientState
    mu      sync.Mutex
    stopCh  chan struct{}
}
```

**Issue**: Cleanup only runs periodically (every minute). Under attack with unique IPs, map could grow significantly.

**Current Mitigation**: Cleanup removes entries not seen in last minute.

**Severity**: LOW - Already has cleanup, but could improve with LRU cache.

---

#### 8. Missing context.Context Propagation in Some Paths
**File**: Various

**Issue**: Some internal functions don't propagate context cancellation, though they call context-aware methods downstream.

**Example**: `cleanupExpiredCacheEntries()` in auth.go runs without context awareness.

**Severity**: LOW - Cleanup operations are short-lived.

---

## Concurrency Patterns Assessment

### Well-Implemented Patterns

1. **sync.Map for Concurrent Bidder Results** (`exchange.go:1525`)
   - Correctly used for write-heavy concurrent access from bidder goroutines

2. **Semaphore Pattern** (`exchange.go:1530-1532`)
   - Proper bounded concurrency using buffered channel

3. **Worker Pool with Graceful Shutdown** (`events.go:66-89`)
   - Clean WaitGroup + stopCh pattern for worker lifecycle

4. **RWMutex for Config Snapshots** (`exchange.go:1177-1181`)
   - Correct pattern: read lock, snapshot, unlock, then work

5. **Deferred Unlocks** (throughout)
   - Consistent use of `defer mu.Unlock()` prevents missed unlocks

### Areas for Improvement

1. **Atomic Package Usage**: Could use atomic types more for simple counters instead of mutex
2. **sync.Pool**: Not used but could help with buffer allocation
3. **Error Channel Buffering**: Most error channels are properly buffered

---

## Race Detector Compatibility

The code structure appears compatible with `-race` testing:
- No obvious data races from code inspection
- Proper use of synchronization primitives
- No raw map access without locks

**Recommendation**: Run `go test -race ./...` in CI pipeline.

---

## Goroutine Lifecycle Summary

| Component | Goroutines | Cleanup Method | Risk |
|-----------|------------|----------------|------|
| EventRecorder | 2 workers | Close() waits | LOW |
| RateLimiter | 1 cleanup | Stop() closes channel | LOW |
| Auth | 1 cleanup | Shutdown() waits | LOW |
| PauseAdTracker | 1 cleanup | Shutdown() waits | LOW |
| CircuitBreaker | per-callback | Close() WaitGroup | MEDIUM |
| HTTPClient | per-request | Context cancellation | LOW |

---

## Recommendations Summary

1. **Immediate**: Review IVT detector sync.Once reset for thread safety
2. **Short-term**: Audit circuit breaker callback timeout behavior
3. **Medium-term**: Document lock ordering for PublisherAuth
4. **Long-term**: Consider atomic counters for dashboard metrics
5. **Ongoing**: Run `-race` tests in CI

---

## Conclusion

The codebase demonstrates good concurrency hygiene with proper synchronization. The main risks are:
1. sync.Once misuse in IVT detector (HIGH)
2. Potential goroutine leaks in circuit breaker callbacks (HIGH)
3. Lock ordering complexity in PublisherAuth (HIGH)

All other issues are medium to low severity and represent standard Go concurrency patterns used correctly.

---

*Report generated by Concurrency Cop AI Agent*
