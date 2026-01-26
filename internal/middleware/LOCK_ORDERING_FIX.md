# Publisher Auth Lock Ordering Fix

## Summary

Fixed MEDIUM severity lock ordering risk in `internal/middleware/publisher_auth.go` by documenting and enforcing consistent lock acquisition ordering across all methods.

## Issue

The `PublisherAuth` struct had 3 separate mutexes protecting related state:
- `mu` - protects config, redisClient, publisherStore
- `publisherCacheMu` - protects publisherCache
- `rateLimitsMu` - protects rateLimits

Without documented lock ordering, this created a risk of deadlocks if locks were acquired in inconsistent orders across different methods.

## Solution

### 1. Documented Lock Ordering

Added comprehensive lock ordering documentation to the `PublisherAuth` struct definition:

```go
// LOCK ORDERING: To prevent deadlocks, locks MUST be acquired in this order:
//   1. mu (config lock) - protects config, redisClient, publisherStore
//   2. publisherCacheMu - protects publisherCache
//   3. rateLimitsMu - protects rateLimits
//
// RULES:
//   - Never acquire locks in reverse order
//   - Release locks as soon as possible (use RLock when possible)
//   - Never hold multiple locks across I/O operations (Redis, PostgreSQL)
//   - Document any method that acquires multiple locks
```

### 2. Method-Level Documentation

Added lock ordering comments to all methods that acquire locks:

- **checkRateLimit**: Documents `mu → rateLimitsMu` ordering, releases `mu` before acquiring `rateLimitsMu`
- **validatePublisher**: Documents `mu` only, releases before I/O operations or calling other methods
- **cachePublisher**: Documents `publisherCacheMu` only
- **getCachedPublisher**: Documents `publisherCacheMu` only
- **cleanupExpiredCache**: Documents caller must hold `publisherCacheMu.Lock()`
- **cleanupStaleRateLimits**: Documents caller must hold `rateLimitsMu.Lock()`

### 3. Fixed Race Condition

Fixed a race condition in `validatePublisher` where the `registeredPubs` map was copied by reference but could be modified concurrently:

**Before:**
```go
registeredPubs := p.config.RegisteredPubs
```

**After:**
```go
// Make a copy of the map to avoid race conditions when map is modified concurrently
var registeredPubs map[string]string
if p.config.RegisteredPubs != nil {
    registeredPubs = make(map[string]string, len(p.config.RegisteredPubs))
    for k, v := range p.config.RegisteredPubs {
        registeredPubs[k] = v
    }
}
```

### 4. Consistent Pattern

Enforced the pattern:
1. Acquire lock
2. Copy needed data
3. Release lock
4. Perform I/O or call other methods
5. Acquire next lock if needed (following ordering rules)

This ensures:
- Locks are never held across I/O operations
- Multiple locks are never held simultaneously (except when necessary and in documented order)
- Deadlocks are prevented through consistent ordering

## Testing

Created comprehensive test suite in `publisher_auth_lock_ordering_test.go`:

### Test Coverage

1. **TestLockOrdering_NoDeadlock** - Stress test with 10 goroutines performing mixed operations
   - Result: 47+ million operations without deadlock

2. **TestLockOrdering_ConcurrentConfigAndRateLimit** - Tests `mu → rateLimitsMu` ordering
   - Result: 3.8+ million operations without deadlock

3. **TestLockOrdering_ConcurrentCacheOperations** - Tests concurrent cache reads/writes
   - Result: 1.5+ million operations without deadlock

4. **TestLockOrdering_ValidatePublisherWithRateLimiting** - Tests most complex interaction
   - Result: 2.9+ million operations without deadlock

5. **TestLockOrdering_StressTest** - Intensive test with 50 goroutines and all operations
   - Result: 14.6+ million operations without deadlock

6. **TestLockOrdering_RaceDetector** - Detects race conditions with concurrent reads/writes
   - Result: PASS (found and fixed race condition in validatePublisher)

7. **TestLockOrdering_CleanupOperations** - Tests cleanup operations under contention
   - Result: PASS

8. **TestLockOrdering_DocumentedBehavior** - Verifies documented lock ordering
   - Result: PASS

### Test Results

All tests pass:
```
PASS: TestLockOrdering_NoDeadlock (5.00s) - 47,373,096 ops
PASS: TestLockOrdering_ConcurrentConfigAndRateLimit (2.00s) - 3,869,968 ops
PASS: TestLockOrdering_ConcurrentCacheOperations (1.00s) - 1,529,570 ops
PASS: TestLockOrdering_ValidatePublisherWithRateLimiting (2.00s) - 2,907,023 ops
PASS: TestLockOrdering_StressTest (5.00s) - 14,611,932 ops
PASS: TestLockOrdering_RaceDetector (0.50s)
PASS: TestLockOrdering_CleanupOperations (0.22s)
PASS: TestLockOrdering_DocumentedBehavior (0.00s)
```

### Race Detector

All tests pass with `-race` flag enabled:
```
go test -race -run TestLockOrdering ./internal/middleware/
ok  	github.com/thenexusengine/tne_springwire/internal/middleware	17.381s
```

### Performance

Benchmark shows minimal overhead:
```
BenchmarkLockOrdering_Contention-24    11,770,900    301.9 ns/op
```

Even under heavy contention (24 goroutines), operations complete in ~300ns.

## Files Modified

1. **internal/middleware/publisher_auth.go**
   - Added lock ordering documentation to struct
   - Added method-level lock ordering comments
   - Fixed race condition in `validatePublisher`
   - No functional changes to business logic

2. **internal/middleware/publisher_auth_lock_ordering_test.go** (NEW)
   - Comprehensive test suite with 8 test cases
   - Stress tests, race detection, benchmark
   - Tests all lock ordering scenarios

3. **internal/middleware/dos_protection_test.go**
   - Fixed unrelated compilation error (unused import, wrong IPNet type)

## Verification

To verify the fix:

```bash
# Run lock ordering tests
go test -v -run TestLockOrdering ./internal/middleware/

# Run with race detector
go test -race -run TestLockOrdering ./internal/middleware/

# Run benchmark
go test -bench=BenchmarkLockOrdering ./internal/middleware/ -benchtime=3s

# Run all publisher auth tests to ensure no regression
go test -v -run TestPublisherAuth ./internal/middleware/
```

## Impact

- **Security**: MEDIUM severity issue resolved - no risk of deadlocks
- **Performance**: Minimal overhead (~300ns per operation under contention)
- **Reliability**: Comprehensive test coverage prevents future regressions
- **Maintainability**: Clear documentation helps future developers
- **Compatibility**: No API changes, fully backward compatible

## Future Improvements

Consider these optional enhancements:

1. Add runtime lock ordering verification (debug mode)
2. Add metrics to track lock contention
3. Consider using sync.Map for rateLimits if contention becomes an issue
4. Add deadlock detection in test suite (e.g., using timeout + goroutine dump)

## References

- Go sync package best practices
- Lock ordering patterns from Linux kernel (spin_lock hierarchy)
- Database lock ordering (e.g., PostgreSQL tuple-level locks)
- Effective Go: Concurrency patterns
