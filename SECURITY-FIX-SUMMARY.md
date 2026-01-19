# Security Fix: Header Injection Auth Bypass (CRITICAL)

## Date: 2026-01-19

## Vulnerability Fixed
**CRITICAL**: Authentication bypass via X-Publisher-ID header injection

### What Was Wrong
The authentication system had a fallback that trusted client-provided `X-Publisher-ID` headers:
- Attackers could send `X-Publisher-ID: fakepub123` to bypass auth
- Rate limiting could be bypassed by rotating fake publisher IDs
- IVT detection could be bypassed by spoofing X-Forwarded-For

### What We Changed
**Removed all header-based authentication**. Now uses secure context-only approach:

1. **auction.go** - Removed header fallback in `hasAPIKey()`
2. **publisher_auth.go** - Sets publisher ID in context, not headers
3. **auth.go** - Sets publisher ID in context, not headers
4. **ratelimit.go** - Reads publisher ID from context, not headers
5. **Tests** - Updated to use `NewContextWithPublisherID()` helper

### Files Changed
- internal/endpoints/auction.go
- internal/endpoints/auction_test.go
- internal/middleware/publisher_auth.go
- internal/middleware/publisher_auth_test.go
- internal/middleware/auth.go
- internal/middleware/auth_test.go
- internal/middleware/ratelimit.go

### Security Impact
- **Before**: Any client could inject headers to bypass auth ❌
- **After**: Only middleware can set publisher ID (secure) ✅

### How It Works Now
```go
// Middleware validates publisher and sets context
ctx := context.WithValue(r.Context(), "publisher_id", validatedPublisherID)

// Downstream code reads from context (can't be spoofed)
publisherID := PublisherIDFromContext(r.Context())
```

### Testing
All tests updated and passing:
- `go test ./internal/middleware` ✅
- `go test ./internal/endpoints` ✅

### Notes for Load Tests
Load test scripts that send `X-Publisher-ID` headers will now have them ignored.
Publisher ID must come from the request body (OpenRTB standard) and be validated
by middleware before being placed in context.

---
**Verified by:** Option 2 (Context-Only) Implementation
**Status:** FIXED ✅
