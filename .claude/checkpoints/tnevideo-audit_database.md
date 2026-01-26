# Database and SQL Audit Report - tnevideo

**Audit Date:** 2026-01-26
**Auditor:** Claude Opus 4.5
**Status:** COMPLETE

---

## Executive Summary

This audit examined the database and SQL-related code in the tnevideo Prebid ad exchange codebase, focusing on the storage layer (`internal/storage/`) and all files containing database queries. The codebase uses PostgreSQL for persistent storage and Redis for caching/real-time operations.

---

## Files Analyzed

- `/Users/andrewstreets/tnevideo/tnevideo/internal/storage/bidders.go`
- `/Users/andrewstreets/tnevideo/tnevideo/internal/storage/publishers.go`
- `/Users/andrewstreets/tnevideo/tnevideo/internal/endpoints/publisher_admin.go`
- `/Users/andrewstreets/tnevideo/tnevideo/pkg/redis/client.go`
- `/Users/andrewstreets/tnevideo/tnevideo/cmd/server/server.go`

---

## CRITICAL FINDINGS

### [NONE]
No critical SQL injection or connection leak issues found. The codebase properly uses parameterized queries.

---

## HIGH SEVERITY FINDINGS

### H1: Missing Transaction Handling for Multi-Step Operations
**Location:** `/Users/andrewstreets/tnevideo/tnevideo/internal/storage/bidders.go` (lines 366-413), `/Users/andrewstreets/tnevideo/tnevideo/internal/storage/publishers.go` (lines 206-244)

**Issue:** The `Update()` methods perform a single UPDATE query but do not use transactions for read-modify-write patterns or when multiple related operations need to be atomic.

**Risk:** Race conditions in concurrent update scenarios. If two requests try to update the same record simultaneously, one update may be lost.

**Remediation:** Implement optimistic locking with version column or wrap related operations in transactions using `db.BeginTx()`.

---

### H2: Missing Query Timeouts on Individual Queries
**Location:** `/Users/andrewstreets/tnevideo/tnevideo/internal/storage/bidders.go` (all query methods), `/Users/andrewstreets/tnevideo/tnevideo/internal/storage/publishers.go` (all query methods)

**Issue:** While the database connection test uses a timeout (line 327 in publishers.go), individual query operations rely on the context passed by callers. If a caller passes `context.Background()`, queries could run indefinitely.

**Risk:** Long-running queries could exhaust database connections or cause request timeouts.

**Evidence:**
```go
// publishers.go:80 - No explicit timeout wrapper
err := s.db.QueryRowContext(ctx, query, publisherID).Scan(...)
```

**Remediation:** Consider wrapping all database operations with a default timeout or documenting the requirement that callers must always provide contexts with timeouts.

---

## MEDIUM SEVERITY FINDINGS

### M1: Unbounded Result Sets in List Operations
**Location:** `/Users/andrewstreets/tnevideo/tnevideo/internal/storage/bidders.go` (lines 104-162, 259-316, 466-528), `/Users/andrewstreets/tnevideo/tnevideo/internal/storage/publishers.go` (lines 111-160)

**Issue:** The `List()`, `ListActive()`, and `GetCapabilities()` methods return all matching records without LIMIT clauses.

**Evidence:**
```go
// bidders.go:106-114
query := `
    SELECT id, bidder_code, bidder_name, ...
    FROM bidders
    WHERE enabled = true AND status = 'active'
    ORDER BY bidder_code
`  // No LIMIT clause
```

**Risk:** Could cause memory exhaustion or slow responses if tables grow large. Pre-allocation uses magic numbers (e.g., `make([]*Bidder, 0, 100)` on line 122) that may not reflect actual data volume.

**Remediation:** Add pagination support with LIMIT/OFFSET or cursor-based pagination for list operations.

---

### M2: Missing Prepared Statement Optimization
**Location:** `/Users/andrewstreets/tnevideo/tnevideo/internal/storage/bidders.go`, `/Users/andrewstreets/tnevideo/tnevideo/internal/storage/publishers.go`

**Issue:** Queries are defined as inline strings and executed directly. While parameterized queries prevent SQL injection, repeated execution of the same query pattern doesn't benefit from prepared statement caching.

**Risk:** Slightly increased database overhead for high-frequency operations in a latency-sensitive ad exchange.

**Remediation:** Consider using `db.Prepare()` for frequently-executed queries to improve performance.

---

### M3: Potential Race Condition in Publisher Admin Create/Update
**Location:** `/Users/andrewstreets/tnevideo/tnevideo/internal/endpoints/publisher_admin.go` (lines 157-210, 212-262)

**Issue:** The `createPublisher()` and `updatePublisher()` methods check for existence then perform create/update as separate operations without atomicity.

**Evidence:**
```go
// Check if publisher already exists (line 178-188)
existing, err := h.redisClient.HGet(ctx, publishersHashKey, req.ID)
// ... later ...
// Create publisher in Redis (line 191)
if err := h.redisClient.HSet(ctx, publishersHashKey, req.ID, req.AllowedDomains)
```

**Risk:** Time-of-check to time-of-use (TOCTOU) race condition. Two concurrent requests could both pass the existence check and one would overwrite the other.

**Remediation:** Use Redis HSETNX for creates or implement Redis transactions (MULTI/EXEC) for atomic check-and-set operations.

---

### M4: Missing Connection Pool Idle Timeout Configuration
**Location:** `/Users/andrewstreets/tnevideo/tnevideo/internal/storage/publishers.go` (lines 321-325)

**Issue:** While `SetMaxOpenConns`, `SetMaxIdleConns`, and `SetConnMaxLifetime` are configured, `SetConnMaxIdleTime` is not set.

**Evidence:**
```go
db.SetMaxOpenConns(100)
db.SetMaxIdleConns(25)
db.SetConnMaxLifetime(10 * time.Minute)
// Missing: db.SetConnMaxIdleTime(...)
```

**Risk:** Idle connections may remain open indefinitely, potentially holding stale connections or consuming resources during low-traffic periods.

**Remediation:** Add `db.SetConnMaxIdleTime(5 * time.Minute)` to ensure idle connections are recycled.

---

## LOW SEVERITY FINDINGS

### L1: Silent Failure on NULL JSON Fields
**Location:** `/Users/andrewstreets/tnevideo/tnevideo/internal/storage/bidders.go` (lines 94-99, 151-156), `/Users/andrewstreets/tnevideo/tnevideo/internal/storage/publishers.go` (lines 101-106)

**Issue:** JSON parsing only occurs when `len(bidderParamsJSON) > 0`, silently leaving the map as nil/empty for NULL database values.

**Evidence:**
```go
if len(httpHeadersJSON) > 0 {
    if err := json.Unmarshal(httpHeadersJSON, &b.HTTPHeaders); err != nil {
        return nil, fmt.Errorf("failed to parse http_headers: %w", err)
    }
}
// If httpHeadersJSON is empty, b.HTTPHeaders remains nil
```

**Risk:** May cause nil pointer dereferences elsewhere if code assumes the map is always initialized.

**Remediation:** Initialize maps to empty maps rather than nil, or document that nil is valid.

---

### L2: Hardcoded Connection Pool Values
**Location:** `/Users/andrewstreets/tnevideo/tnevideo/internal/storage/publishers.go` (lines 322-324), `/Users/andrewstreets/tnevideo/tnevideo/pkg/redis/client.go` (lines 40-46)

**Issue:** Connection pool sizes are hardcoded rather than configurable via environment variables or configuration.

**Evidence:**
```go
// PostgreSQL
db.SetMaxOpenConns(100)
db.SetMaxIdleConns(25)

// Redis
PoolSize:     100,
MinIdleConns: 10,
```

**Risk:** Cannot tune for different deployment environments without code changes.

**Remediation:** Move connection pool configuration to environment variables or configuration files.

---

### L3: Missing Index Hints in Query Comments
**Location:** `/Users/andrewstreets/tnevideo/tnevideo/internal/storage/bidders.go`, `/Users/andrewstreets/tnevideo/tnevideo/internal/storage/publishers.go`

**Issue:** Queries filter on `bidder_code`, `publisher_id`, `status`, and `enabled` columns but there's no indication that corresponding indexes exist.

**Risk:** Potential slow queries on large tables if indexes are missing.

**Remediation:** Verify database migration scripts include appropriate indexes for filtered columns.

---

## POSITIVE OBSERVATIONS

1. **SQL Injection Prevention**: All queries use parameterized placeholders ($1, $2, etc.) - no string concatenation in queries.

2. **Row Close Handling**: All `rows` from QueryContext are properly closed with `defer rows.Close()`.

3. **Error Handling**: All database errors are properly wrapped and returned with context.

4. **Connection Testing**: Database connections are tested at startup with a timeout.

5. **ErrNoRows Handling**: `sql.ErrNoRows` is properly checked and returns nil/nil rather than errors for "not found" cases.

6. **Context Propagation**: All query methods accept and use `context.Context` for cancellation.

---

## Summary by Severity

| Severity | Count |
|----------|-------|
| CRITICAL | 0 |
| HIGH | 2 |
| MEDIUM | 4 |
| LOW | 3 |

---

## Next Steps

1. Address HIGH severity issues before production deployment
2. Add pagination to list endpoints
3. Implement transaction handling for multi-step operations
4. Add connection pool idle timeout configuration
5. Consider prepared statements for high-frequency queries

