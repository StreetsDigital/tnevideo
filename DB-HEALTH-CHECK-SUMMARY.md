# Database Health Check Added

## Date: 2026-01-19

## Issue Fixed
**CRITICAL**: Missing database health check in `/health/ready` endpoint

### What Was Wrong
The readiness endpoint only checked Redis and IDR, but NOT PostgreSQL:
- Kubernetes/load balancers would route traffic to pods with dead DB connections
- No way to detect database outages before serving traffic
- Could cause cascading failures when DB is down

### What Was Changed

**1. Added `Ping()` method to PublisherStore**
- File: `internal/storage/publishers.go`
- Returns error if database connection is dead
- Returns nil if no database configured (graceful)

```go
func (s *PublisherStore) Ping(ctx context.Context) error {
    if s.db == nil {
        return nil // No database configured, not an error
    }
    return s.db.PingContext(ctx)
}
```

**2. Updated `/health/ready` endpoint**
- File: `cmd/server/server.go`
- Added database health check as first check (most critical)
- Returns 503 if database is unhealthy
- Shows status in JSON response

### Health Check Order (by criticality)
1. **Database** - Most critical (publisher/bidder data)
2. **Redis** - Important (caching, rate limiting)
3. **IDR** - Optional (intelligent demand routing)

### Response Format

**All healthy:**
```json
{
  "ready": true,
  "timestamp": "2026-01-19T21:30:00Z",
  "checks": {
    "database": {"status": "healthy"},
    "redis": {"status": "healthy"},
    "idr": {"status": "disabled"}
  }
}
```

**Database down:**
```json
{
  "ready": false,
  "timestamp": "2026-01-19T21:30:00Z",
  "checks": {
    "database": {
      "status": "unhealthy",
      "error": "connection refused"
    },
    "redis": {"status": "healthy"},
    "idr": {"status": "disabled"}
  }
}
```
HTTP Status: 503 Service Unavailable

### Files Changed
- `internal/storage/publishers.go` - Added Ping() method
- `cmd/server/server.go` - Updated readyHandler to check database
- `cmd/server/server_test.go` - Updated test calls

### Testing
```bash
✅ go test ./cmd/server  → PASS (4.913s)
✅ go build ./cmd/server → SUCCESS
```

### Kubernetes Integration

Update your deployment to use the readiness probe:

```yaml
readinessProbe:
  httpGet:
    path: /health/ready
    port: 8000
  initialDelaySeconds: 5
  periodSeconds: 10
  timeoutSeconds: 2
  failureThreshold: 3
```

Now Kubernetes will:
- Wait for DB to be healthy before routing traffic
- Remove pods from load balancer if DB connection dies
- Prevent cascading failures from DB outages

---
**Status:** FIXED ✅
**Critical Blocker:** 2 of 5 resolved
