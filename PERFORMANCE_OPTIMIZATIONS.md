# Performance Optimization Report

This document summarizes the three critical performance bottlenecks that were identified and fixed in the Prebid ad exchange codebase.

## Executive Summary

Three major performance bottlenecks were identified and resolved:

1. **Bid Slice Allocations** - 11x faster, 100% reduction in allocations
2. **FPD JSON Marshaling** - 3.4x faster, 43% less memory
3. **Cookie Trim Algorithm** - 5.2x faster, 80% less memory

Combined, these optimizations significantly reduce GC pressure, improve request latency, and increase auction throughput.

---

## Fix #1: Add sync.Pool for Bid Allocations

### Location
- **File**: `internal/exchange/exchange.go` (line 1451-1452)
- **New File**: `internal/exchange/pools.go`

### Problem
Every auction allocated new slices for `validBids` and `validationErrors`, causing excessive GC pressure in high-throughput scenarios. With thousands of auctions per second, these allocations accumulated significantly.

### Solution
Implemented `sync.Pool` to reuse bid and error slices across auctions:

```go
// Pools for reusing bid and error slices
var (
    validBidsPool = sync.Pool{
        New: func() interface{} {
            s := make([]ValidatedBid, 0, 32)
            return &s
        },
    }
    
    validationErrorsPool = sync.Pool{
        New: func() interface{} {
            s := make([]error, 0, 8)
            return &s
        },
    }
)
```

Usage in `RunAuction`:
```go
// Get slices from pool
validBidsPtr := getValidBidsSlice()
defer putValidBidsSlice(validBidsPtr)
validBids := *validBidsPtr

validationErrorsPtr := getValidationErrorsSlice()
defer putValidationErrorsSlice(validationErrorsPtr)
validationErrors := *validationErrorsPtr
```

### Benchmark Results

```
BenchmarkBidSliceAllocation/DirectAllocation-24    3044252   785.3 ns/op   1312 B/op   5 allocs/op
BenchmarkBidSliceAllocation/PooledAllocation-24   39993482    71.42 ns/op     0 B/op   0 allocs/op
```

**Impact:**
- **11x faster** (785ns → 71ns per operation)
- **100% reduction** in allocations (5 → 0)
- **100% reduction** in memory allocated (1312 B → 0 B)

### Files Modified
- `internal/exchange/pools.go` (new file)
- `internal/exchange/exchange.go` (lines 1450-1459)
- `internal/exchange/pools_bench_test.go` (new file)

---

## Fix #2: Pre-marshal FPD Bidder Configs

### Location
- **File**: `internal/fpd/processor.go` (lines 164-176)

### Problem
The `applyBidderConfig` function marshaled ORTB2 configs (Site, App, User) **per bidder** in a hot loop. For a request with 5 bidders and 3 configs, this resulted in 15 JSON marshal operations instead of 3.

**Old code:**
```go
for _, bidder := range bidders {
    // Apply bidder-specific config
    if config.BidderConfigEnabled && prebidExt != nil {
        // This calls json.Marshal 3x per bidder!
        bidderFPD = p.applyBidderConfig(bidderFPD, bidder, prebidExt.BidderConfig)
    }
}
```

### Solution
Pre-marshal bidder configs **once** when parsing the request:

1. Created `BidderConfigCached` struct to hold pre-marshaled JSON:
```go
type BidderConfigCached struct {
    BidderConfig
    SiteJSON json.RawMessage  // Pre-marshaled
    AppJSON  json.RawMessage  // Pre-marshaled
    UserJSON json.RawMessage  // Pre-marshaled
}
```

2. Added `PrepareBidderConfigs` to marshal once:
```go
func PrepareBidderConfigs(configs []BidderConfig) []BidderConfigCached {
    cached := make([]BidderConfigCached, 0, len(configs))
    for _, config := range configs {
        // Marshal once, reuse for all bidders
        if ortb2.Site != nil {
            siteJSON, _ := json.Marshal(ortb2.Site)
            c.SiteJSON = siteJSON
        }
        // Same for App and User
    }
    return cached
}
```

3. Updated `ProcessRequest` to use cached configs:
```go
// Pre-marshal once
cachedBidderConfigs := PrepareBidderConfigs(prebidExt.BidderConfig)

for _, bidder := range bidders {
    // Apply cached configs (no marshal needed!)
    bidderFPD = p.applyBidderConfigCached(bidderFPD, bidder, cachedBidderConfigs)
}
```

### Benchmark Results

```
BenchmarkBidderConfigMarshal/OldApproach_MarshalPerBidder-24    241334   9569 ns/op   1681 B/op   25 allocs/op
BenchmarkBidderConfigMarshal/NewApproach_PreMarshal-24          866017   2775 ns/op    960 B/op   14 allocs/op
```

**Impact:**
- **3.4x faster** (9569ns → 2775ns per 5-bidder request)
- **43% less memory** (1681 B → 960 B)
- **44% fewer allocations** (25 → 14)

### Files Modified
- `internal/fpd/processor.go` (ProcessRequest function and new functions)
- `internal/fpd/processor_bench_test.go` (new file)

---

## Fix #3: Optimize Cookie trimToFit

### Location
- **File**: `internal/usersync/cookie.go` (lines 194-218)

### Problem
The `trimToFit` function used an O(n²) approach: marshal the entire cookie, check size, remove one UID, repeat. For 50 UIDs needing to trim 20, this meant **50 marshal operations** instead of the optimal log₂(50) ≈ **6 marshals**.

**Old code:**
```go
for len(c.UIDs) > 0 {
    data, _ := json.Marshal(c)  // Marshal N times!
    encoded := base64.URLEncoding.EncodeToString(data)
    if len(encoded) <= MaxCookieSize {
        break
    }
    // Remove one oldest UID, repeat
}
```

### Solution
Used **binary search** to find the minimum number of UIDs to remove:

```go
func (c *Cookie) trimToFit() {
    // Quick check if we need to trim
    if alreadyFits() {
        return
    }
    
    // Sort UIDs by expiry time
    uidList := sortByExpiry(c.UIDs)
    
    // Binary search: find minimum UIDs to remove
    left, right := 0, len(uidList)
    for left < right {
        mid := (left + right) / 2
        
        // Test if removing 'mid' UIDs is enough
        if testSize(withoutFirst(mid, uidList)) <= MaxCookieSize {
            right = mid  // Try removing fewer
        } else {
            left = mid + 1  // Need to remove more
        }
    }
    
    // Remove the minimum required UIDs
    removeFirst(left, uidList)
}
```

This reduces complexity from **O(n²)** to **O(n log n)**.

### Benchmark Results

```
BenchmarkTrimToFit_Old/50UIDs-24          730   1577102 ns/op   667462 B/op   3207 allocs/op
BenchmarkTrimToFit_New/50UIDs-24         3844    304212 ns/op   128563 B/op    588 allocs/op
```

**Impact:**
- **5.2x faster** (1577μs → 304μs for 50 UIDs)
- **80% less memory** (667 KB → 128 KB)
- **82% fewer allocations** (3207 → 588)

For 100 UIDs, the improvement is even more dramatic (O(n²) vs O(n log n)).

### Files Modified
- `internal/usersync/cookie.go` (trimToFit function)
- `internal/usersync/cookie_bench_test.go` (new file)

---

## Combined Impact

### Memory & GC Pressure
- **Fix #1**: Eliminates 5 allocations per auction (1312 B)
- **Fix #2**: Saves 721 B per multi-bidder request
- **Fix #3**: Reduces cookie trim memory by 80%

At 10,000 auctions/second:
- **Fix #1** saves: ~12.5 MB/s in allocations
- **Fix #2** saves: ~7 MB/s in allocations (typical requests)
- **Combined**: Significant reduction in GC pause time

### Latency Improvements
- **Fix #1**: Reduces auction overhead by 714ns per request
- **Fix #2**: Saves 6.8μs per request with 5 bidders
- **Fix #3**: Saves 1.3ms per cookie sync with 50 UIDs

### Throughput
With reduced GC pressure and lower latency:
- Higher sustained auction rate
- More consistent p99 latency
- Better CPU utilization

---

## Testing & Validation

### Running Benchmarks

```bash
# Fix #1: Bid slice pools
cd internal/exchange
go test -bench=BenchmarkBidSliceAllocation -benchmem

# Fix #2: FPD pre-marshal
cd internal/fpd
go test -bench=BenchmarkBidderConfigMarshal -benchmem

# Fix #3: Cookie trim optimization
cd internal/usersync
go test -bench=BenchmarkTrimToFit -benchmem
```

### Unit Tests
All existing tests continue to pass:
```bash
go test ./internal/exchange/...
go test ./internal/fpd/...
go test ./internal/usersync/...
```

### Production Validation Checklist
- [ ] Monitor GC pause times (expect reduction)
- [ ] Monitor p99 latency (expect improvement)
- [ ] Monitor memory usage (expect lower baseline)
- [ ] Monitor auction throughput (expect higher sustained rate)
- [ ] Verify no behavioral changes in auction results
- [ ] Check cookie sync functionality remains correct

---

## Recommendations

### Immediate Actions
1. **Deploy to staging** and run load tests
2. **Monitor metrics** before/after deployment
3. **Gradual rollout** to production with canary testing

### Future Optimizations
Based on this work, consider:
1. **Add more pools** for other hot-path allocations (e.g., HTTP request/response buffers)
2. **Cache other JSON marshaling** (e.g., OpenRTB request objects)
3. **Profile production** to find next bottlenecks

### Monitoring Metrics
Track these metrics post-deployment:
- `gc_pause_duration_seconds` (p99, p999)
- `auction_duration_milliseconds` (p50, p99)
- `memory_allocated_bytes` (rate and absolute)
- `auction_throughput_per_second`

---

## Conclusion

These three optimizations deliver significant performance improvements:
- **11x faster bid allocation** with zero allocations
- **3.4x faster FPD processing** with 43% less memory
- **5.2x faster cookie trimming** with 80% less memory

Combined, they reduce GC pressure, improve latency, and increase throughput—critical for a real-time bidding system processing thousands of auctions per second.

**Total Lines Changed**: ~350 lines
**Files Added**: 4 (pools.go + 3 benchmark files)
**Breaking Changes**: None (all changes are internal optimizations)

---

## References

- Benchmark files: `*_bench_test.go` in each package
- Go sync.Pool docs: https://pkg.go.dev/sync#Pool
- Binary search complexity: O(n log n) vs O(n²)
