# Database Security Fixes - HIGH Severity Issues Resolved

## Summary

This document describes the implementation of two HIGH severity database security fixes:

1. **Optimistic Locking for Concurrent Updates**
2. **Query Timeouts for All Database Operations**

## 1. Optimistic Locking Implementation

### Problem
Lost updates can occur when multiple processes attempt to modify the same database record concurrently. Without version control, the last write wins, potentially overwriting important changes made by other processes.

### Solution
Implemented optimistic locking using version columns and transactional updates with version checking.

### Files Modified
- `/Users/andrewstreets/tnevideo/tnevideo/internal/storage/bidders.go`
- `/Users/andrewstreets/tnevideo/tnevideo/internal/storage/publishers.go`

### Database Changes
**Migration**: `/Users/andrewstreets/tnevideo/tnevideo/deployment/migrations/004_add_version_columns.sql`

```sql
-- Add version columns to both tables
ALTER TABLE publishers ADD COLUMN version INTEGER NOT NULL DEFAULT 1;
ALTER TABLE bidders ADD COLUMN version INTEGER NOT NULL DEFAULT 1;

-- Create trigger to auto-increment version on update
CREATE OR REPLACE FUNCTION increment_version()
RETURNS TRIGGER AS $$
BEGIN
    NEW.version = OLD.version + 1;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_publishers_version
    BEFORE UPDATE ON publishers
    FOR EACH ROW
    EXECUTE FUNCTION increment_version();

CREATE TRIGGER trigger_bidders_version
    BEFORE UPDATE ON bidders
    FOR EACH ROW
    EXECUTE FUNCTION increment_version();
```

### Implementation Details

#### Data Structure Changes
Added `Version int` field to both `Bidder` and `Publisher` structs:

```go
type Bidder struct {
    // ... existing fields ...
    Version          int                    `json:"version"`
    CreatedAt        time.Time              `json:"created_at"`
    UpdatedAt        time.Time              `json:"updated_at"`
}

type Publisher struct {
    // ... existing fields ...
    Version        int                    `json:"version"`
    CreatedAt      time.Time              `json:"created_at"`
    UpdatedAt      time.Time              `json:"updated_at"`
}
```

#### Update Method Changes
The `Update()` methods now use transactions with version checking:

```go
func (s *BidderStore) Update(ctx context.Context, b *Bidder) error {
    // Begin transaction
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()

    // Check current version
    var currentVersion int
    err = tx.QueryRowContext(ctx,
        "SELECT version FROM bidders WHERE bidder_code = $1",
        b.BidderCode).Scan(&currentVersion)

    if err == sql.ErrNoRows {
        return fmt.Errorf("bidder not found: %s", b.BidderCode)
    }

    // Verify version matches (optimistic lock check)
    if currentVersion != b.Version {
        return fmt.Errorf("concurrent modification detected: bidder %s was updated by another process", b.BidderCode)
    }

    // Update with version check in WHERE clause
    query := `
        UPDATE bidders
        SET ... fields ...
        WHERE bidder_code = $15 AND version = $16
    `

    result, err := tx.ExecContext(ctx, query, ..., b.BidderCode, b.Version)

    if rows == 0 {
        return fmt.Errorf("concurrent modification detected: bidder %s version mismatch", b.BidderCode)
    }

    // Commit transaction
    if err := tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit transaction: %w", err)
    }

    // Update version in struct for caller
    b.Version = currentVersion + 1

    return nil
}
```

### Benefits
- **Prevents Lost Updates**: Concurrent modifications are detected and rejected
- **Data Integrity**: Ensures all changes are preserved, no silent overwrites
- **Clear Error Messages**: Applications receive explicit "concurrent modification" errors
- **Automatic Version Management**: Database triggers handle version incrementing

### Error Handling
When concurrent modification is detected, the update fails with:
- Error: `"concurrent modification detected: [entity] was updated by another process"`
- Application must re-fetch the entity and retry the update with the new version

## 2. Query Timeout Implementation

### Problem
Database operations without timeouts can hang indefinitely, causing resource exhaustion and degraded system performance under adverse conditions (slow queries, network issues, deadlocks).

### Solution
Implemented automatic timeout wrapping for all database operations with a default 5-second timeout.

### Files Created
**Helper Module**: `/Users/andrewstreets/tnevideo/tnevideo/internal/storage/context.go`

```go
package storage

import (
    "context"
    "time"
)

// DefaultDBTimeout is the default timeout for database operations
const DefaultDBTimeout = 5 * time.Second

// withTimeout wraps a context with a default timeout if it doesn't already have a deadline
func withTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
    // Check if context already has a deadline
    if _, hasDeadline := ctx.Deadline(); hasDeadline {
        // Return context as-is with a no-op cancel function
        return ctx, func() {}
    }

    // Add default timeout
    return context.WithTimeout(ctx, timeout)
}
```

### Implementation
Added timeout wrapping to ALL database operations in both files:

```go
func (s *BidderStore) GetByCode(ctx context.Context, bidderCode string) (*Bidder, error) {
    ctx, cancel := withTimeout(ctx, DefaultDBTimeout)
    defer cancel()

    // ... rest of query ...
}

func (s *PublisherStore) List(ctx context.Context) ([]*Publisher, error) {
    ctx, cancel := withTimeout(ctx, DefaultDBTimeout)
    defer cancel()

    // ... rest of query ...
}
```

### Methods Updated
**Bidders**:
- `GetByCode()`
- `ListActive()`
- `GetForPublisher()`
- `List()`
- `Create()`
- `Update()`
- `Delete()`
- `SetEnabled()`
- `GetCapabilities()`

**Publishers**:
- `GetByPublisherID()`
- `List()`
- `Create()`
- `Update()`
- `Delete()`
- `GetBidderParams()`

### Benefits
- **Prevents Resource Exhaustion**: Operations automatically timeout instead of hanging
- **Predictable Behavior**: 5-second maximum wait for any database operation
- **Respects Existing Deadlines**: If caller provides a context with timeout, that takes precedence
- **Graceful Degradation**: System can shed load under adverse conditions

### Configuration
Default timeout: **5 seconds** (`DefaultDBTimeout`)

To override the default timeout, pass a context with a deadline:
```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
store.GetByCode(ctx, "bidder-code")
```

## Testing

### Test Files Created

1. **Optimistic Locking Tests**:
   - `/Users/andrewstreets/tnevideo/tnevideo/internal/storage/bidders_concurrent_test.go` (10 tests)
   - `/Users/andrewstreets/tnevideo/tnevideo/internal/storage/publishers_concurrent_test.go` (8 tests)

2. **Timeout Tests**:
   - `/Users/andrewstreets/tnevideo/tnevideo/internal/storage/timeout_test.go` (9 tests)

3. **Context Helper Tests**:
   - `/Users/andrewstreets/tnevideo/tnevideo/internal/storage/context_test.go` (5 tests)

### Test Coverage

#### Optimistic Locking Tests
- ✅ Successful update with correct version
- ✅ Version mismatch detection (concurrent modification)
- ✅ Update of non-existent entity
- ✅ Zero rows affected (race condition)
- ✅ Transaction begin failure
- ✅ Commit failure
- ✅ Version included in GET operations
- ✅ Version returned on CREATE operations

#### Timeout Tests
- ✅ Query timeout enforcement (5s default)
- ✅ Timeout applies to all operation types (GET, LIST, CREATE, UPDATE)
- ✅ Existing deadline is preserved and not overridden
- ✅ Shorter existing deadline takes precedence
- ✅ Timeout enforced in transaction operations

#### Context Helper Tests
- ✅ Adds timeout when no deadline exists
- ✅ Preserves existing deadline
- ✅ Returns no-op cancel for existing deadline
- ✅ Cancel function works correctly
- ✅ Default timeout value is correct

### Test Results
All tests pass (32 new tests + existing tests):
```bash
go test ./internal/storage/... -v
# PASS
# ok    github.com/thenexusengine/tne_springwire/internal/storage    41.639s
```

### Existing Tests Updated
Updated all existing storage tests to accommodate the new version column:
- Added `version` to all SELECT column lists
- Added version value (1) to all mock row data
- Updated Create tests to expect version in RETURNING clause
- Updated Update tests to expect transaction + version checking pattern

## Migration Instructions

### 1. Run Database Migration
```bash
# Apply the version column migration
psql -U $DB_USER -d $DB_NAME -f deployment/migrations/004_add_version_columns.sql
```

### 2. Update Application Code
The changes are backward compatible for READ operations. However, for UPDATE operations:

**Before** (old code):
```go
bidder, err := store.GetByCode(ctx, "appnexus")
bidder.BidderName = "Updated Name"
err = store.Update(ctx, bidder) // Works, version is preserved from GET
```

**After** (no changes needed):
```go
bidder, err := store.GetByCode(ctx, "appnexus") // Returns with version=1
bidder.BidderName = "Updated Name"
err = store.Update(ctx, bidder) // Uses version for optimistic locking
```

### 3. Handle Concurrent Modification Errors
Applications should handle concurrent modification errors:

```go
err := store.Update(ctx, bidder)
if err != nil && strings.Contains(err.Error(), "concurrent modification detected") {
    // Re-fetch the entity and retry
    bidder, err = store.GetByCode(ctx, bidder.BidderCode)
    if err != nil {
        return err
    }
    // Reapply changes and retry update
    bidder.BidderName = "Updated Name"
    err = store.Update(ctx, bidder)
}
```

## Performance Impact

### Optimistic Locking
- **Overhead**: Minimal - adds one SELECT query per UPDATE (version check)
- **Benefit**: Prevents data corruption and lost updates
- **Trade-off**: Updates may fail and require retry (fail-fast is safer than silent corruption)

### Query Timeouts
- **Overhead**: Negligible - context wrapping is extremely fast
- **Benefit**: Prevents resource exhaustion, improves system stability
- **Trade-off**: Very slow queries (>5s) will timeout (this is intentional)

## Monitoring Recommendations

### Metrics to Track
1. **Concurrent Modification Rate**: Count of "concurrent modification detected" errors
2. **Timeout Rate**: Count of "context deadline exceeded" errors
3. **Query Duration**: P95/P99 query times to ensure <5s

### Alerts
- Alert if concurrent modification rate > 1% of updates
- Alert if timeout rate > 0.1% of queries
- Alert if P99 query time approaches 4 seconds

## Rollback Plan

If issues arise, rollback is straightforward:

1. **Revert Code**: Git revert the changes
2. **Keep Migration**: The version columns are harmless and can remain
3. **Or Remove Migration**: If needed, run:
   ```sql
   ALTER TABLE bidders DROP COLUMN version;
   ALTER TABLE publishers DROP COLUMN version;
   DROP FUNCTION increment_version CASCADE;
   ```

## Security Impact

### Threats Mitigated
- ✅ **Lost Update Problem**: Concurrent modifications now detected and prevented
- ✅ **Resource Exhaustion**: Runaway queries now timeout automatically
- ✅ **Denial of Service**: Slow query attacks limited to 5s per operation

### Risk Reduction
- **HIGH severity → LOW severity**: Both issues are now fully mitigated
- **Data Integrity**: Improved through optimistic locking
- **System Stability**: Improved through automatic timeouts

## Conclusion

These two HIGH severity database security issues have been fully resolved with:

1. ✅ **Optimistic locking** prevents lost updates from concurrent modifications
2. ✅ **Query timeouts** prevent resource exhaustion and improve stability
3. ✅ **Comprehensive tests** ensure correctness (32 new tests, all passing)
4. ✅ **Backward compatible** changes minimize deployment risk
5. ✅ **Database migration** provided for version column additions

The implementation follows Go best practices and maintains consistency with the existing codebase architecture.
