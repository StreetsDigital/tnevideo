# Concurrency Audit Report

**Date:** 2026-01-26
**Auditor:** Concurrency Cop
**Scope:** IVT Detector sync.Once race condition fix

---

## Executive Summary

Fixed a HIGH severity race condition in the IVT (Invalid Traffic) detector middleware that could cause data races and undefined behavior in production under concurrent load.

---

## Issue Fixed

### IVT Detector sync.Once Reset Race Condition

**Severity:** HIGH
**File:** `/Users/andrewstreets/tnevideo/tnevideo/internal/middleware/ivt_detector.go`
**Lines:** 550-557 (original), 215-232 (new struct), 274-304 (new implementation)

#### Problem Description

The original `SetConfig()` method reset a `sync.Once` value while other goroutines could be executing the `compilePatterns()` method that uses it:

```go
// ORIGINAL CODE (VULNERABLE)
func (d *IVTDetector) SetConfig(config *IVTConfig) {
    d.mu.Lock()
    defer d.mu.Unlock()
    d.config = config
    
    // RACE: Resetting sync.Once while compilePatterns() may be executing
    d.patternsOnce = sync.Once{}
}

func (d *IVTDetector) compilePatterns() {
    d.patternsOnce.Do(func() {
        d.mu.RLock()
        patterns := d.config.SuspiciousUAPatterns
        d.mu.RUnlock()
        // ... compile patterns
    })
}
```

**Why This Is Dangerous:**

1. `sync.Once` is a struct with internal state (`done` flag and `m` mutex)
2. Assigning a new empty `sync.Once{}` while another goroutine is inside `Do()` causes undefined behavior
3. The Go race detector would flag this as a data race
4. In production, this could cause:
   - Patterns being compiled multiple times concurrently (memory corruption)
   - Patterns never being compiled (nil pointer dereference)
   - Partial state being visible to readers

#### Solution Implemented

Replaced `sync.Once` with a version counter approach using atomic operations:

```go
// FIXED CODE
type IVTDetector struct {
    // ... other fields ...
    
    // Pattern compilation with version-based reloading (thread-safe)
    patternsVersion atomic.Uint64    // Incremented on config change
    patternsMu      sync.RWMutex     // Protects uaPatterns and loadedVersion
    uaPatterns      []*regexp.Regexp // Compiled patterns
    loadedVersion   uint64           // Version when patterns were last compiled
}

func (d *IVTDetector) compilePatterns() {
    // Fast path: check if patterns are already up to date
    currentVersion := d.patternsVersion.Load()
    
    d.patternsMu.RLock()
    needsRecompile := d.loadedVersion != currentVersion
    d.patternsMu.RUnlock()
    
    if !needsRecompile {
        return
    }
    
    // Slow path: recompile with double-checked locking
    d.patternsMu.Lock()
    defer d.patternsMu.Unlock()
    
    currentVersion = d.patternsVersion.Load()
    if d.loadedVersion == currentVersion {
        return // Another goroutine compiled while we waited
    }
    
    // ... compile patterns ...
    d.loadedVersion = currentVersion
}

func (d *IVTDetector) SetConfig(config *IVTConfig) {
    d.mu.Lock()
    d.config = config
    d.mu.Unlock()
    
    // Thread-safe: atomic increment signals need for recompilation
    d.patternsVersion.Add(1)
}
```

**Key Design Decisions:**

1. **Atomic version counter**: `atomic.Uint64` provides lock-free reads for the common case
2. **Double-checked locking**: Prevents redundant compilation when multiple goroutines race
3. **Separate mutex for patterns**: `patternsMu` prevents contention with config reads
4. **No blocking on config update**: `SetConfig()` returns immediately, compilation happens lazily

---

## Tests Added

Four new tests verify the fix:

### 1. TestIVTDetector_ConcurrentConfigReload
- **Purpose:** Stress test concurrent reads and writes
- **Configuration:** 50 reader goroutines, 10 writer goroutines, 100 iterations each
- **Validates:** No panics, no nil results under high concurrency

### 2. TestIVTDetector_ConcurrentConfigReload_RaceDetector
- **Purpose:** Designed to trigger race detector if race exists
- **Configuration:** 2 goroutines with tight synchronization, 1000 iterations
- **Validates:** Race detector finds no data races

### 3. TestIVTDetector_PatternRecompilation
- **Purpose:** Verify patterns update after config change
- **Method:** Change patterns via SetConfig, verify detection behavior changes
- **Validates:** Lazy recompilation works correctly

### 4. TestIVTDetector_VersionIncrement
- **Purpose:** Verify version counter mechanics
- **Method:** Call SetConfig multiple times, check version value
- **Validates:** Version increments correctly from initial value of 1

---

## Test Results

```
$ go test -race -run "TestIVTDetector_Concurrent|TestIVTDetector_Pattern|TestIVTDetector_Version" ./internal/middleware/

=== RUN   TestIVTDetector_ConcurrentConfigReload
--- PASS: TestIVTDetector_ConcurrentConfigReload (0.31s)
=== RUN   TestIVTDetector_ConcurrentConfigReload_RaceDetector
--- PASS: TestIVTDetector_ConcurrentConfigReload_RaceDetector (0.06s)
=== RUN   TestIVTDetector_PatternRecompilation
--- PASS: TestIVTDetector_PatternRecompilation (0.00s)
=== RUN   TestIVTDetector_VersionIncrement
--- PASS: TestIVTDetector_VersionIncrement (0.00s)
PASS
ok      github.com/thenexusengine/tne_springwire/internal/middleware    2.051s
```

All tests pass with the Go race detector enabled.

---

## Recommendations

### Immediate
- [x] Fix applied and tested

### Follow-up
- [ ] Search codebase for other `sync.Once` patterns that may be reset
- [ ] Consider adding a linter rule to flag `sync.Once` assignments outside of initialization
- [ ] Document the version counter pattern in team coding standards

---

## Files Modified

| File | Change |
|------|--------|
| `internal/middleware/ivt_detector.go` | Replaced sync.Once with version counter |
| `internal/middleware/ivt_detector_test.go` | Added 4 concurrent tests |

---

## Notes

A pre-existing test failure `TestIVTDetector_RefererValidation/Valid_subdomain` was observed. This is a logic issue with subdomain matching, not related to concurrency. The test expects `www.example.com` to match `example.com` but the current implementation does exact domain matching.
