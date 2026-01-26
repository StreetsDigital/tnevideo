# Concurrency Audit Checkpoint

**Project:** tnevideo
**Date:** 2026-01-26
**Agent:** Concurrency Cop

## Completed Work

### IVT Detector sync.Once Race Condition Fix

**Status:** COMPLETED

**Issue:** HIGH severity race condition in `internal/middleware/ivt_detector.go:550-557`

The original code had:
```go
func (d *IVTDetector) SetConfig(config *IVTConfig) {
    d.mu.Lock()
    defer d.mu.Unlock()
    d.config = config
    
    // RACE CONDITION: Resetting sync.Once while other goroutines may be executing compilePatterns()
    d.patternsOnce = sync.Once{}
}
```

**Problem:** `sync.Once` cannot be safely reset while other goroutines are executing the `Do()` function. The `compilePatterns()` method only takes an `RLock` on the config mutex, but `SetConfig` takes a `Lock` - however, this doesn't prevent the race because the `sync.Once` reset happens outside of proper synchronization with `patternsOnce.Do()`.

**Solution Applied:** Version counter approach

1. Added `patternsVersion atomic.Uint64` to track config changes atomically
2. Added `patternsMu sync.RWMutex` to protect pattern state
3. Added `loadedVersion uint64` to track which version patterns were compiled for
4. Modified `compilePatterns()` to check version before recompiling (double-checked locking pattern)
5. Modified `SetConfig()` to atomically increment version instead of resetting sync.Once

**Files Modified:**
- `/Users/andrewstreets/tnevideo/tnevideo/internal/middleware/ivt_detector.go`
- `/Users/andrewstreets/tnevideo/tnevideo/internal/middleware/ivt_detector_test.go`

**Tests Added:**
1. `TestIVTDetector_ConcurrentConfigReload` - 50 readers, 10 writers, 100 iterations each
2. `TestIVTDetector_ConcurrentConfigReload_RaceDetector` - Tight synchronization for race detector
3. `TestIVTDetector_PatternRecompilation` - Verifies patterns are recompiled after config change
4. `TestIVTDetector_VersionIncrement` - Verifies version counter increments correctly

**Test Results:**
- All concurrent tests PASS
- Race detector finds NO races
- Pattern recompilation works correctly

## Next Steps

- Continue concurrency audit of other areas (pbs/internal/exchange/, pbs/pkg/idr/)
- Check for other sync.Once patterns that may have similar issues
- Review HTTP client timeout and cancellation patterns
- Check channel usage patterns

## Known Pre-existing Issues (Not Addressed)

- `TestIVTDetector_RefererValidation/Valid_subdomain` fails - subdomain matching logic issue (not concurrency related)
