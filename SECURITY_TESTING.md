# Security Testing Documentation

This document describes the comprehensive security tests implemented for the Prebid Ad Exchange.

## Overview

We have implemented extensive security tests covering the three major categories of web application vulnerabilities:

1. **SQL Injection** - Tests in `internal/storage/security_test.go`
2. **Cross-Site Scripting (XSS)** - Tests in `internal/endpoints/xss_security_test.go`
3. **Denial of Service (DoS)** - Tests in `internal/middleware/dos_protection_test.go`

## Running the Tests

```bash
# Run all security tests
go test -v ./internal/storage -run TestSQLInjection
go test -v ./internal/endpoints -run TestXSS
go test -v ./internal/middleware -run TestDoS

# Run specific test suites
go test -v ./internal/storage/security_test.go
go test -v ./internal/endpoints/xss_security_test.go
go test -v ./internal/middleware/dos_protection_test.go
```

## 1. SQL Injection Protection

### Location
`internal/storage/security_test.go`

### Protection Mechanisms

1. **Parameterized Queries**: All database operations use parameterized statements (`$1`, `$2`, etc.)
2. **PostgreSQL Wire Protocol**: Parameters sent separately from SQL text
3. **No String Concatenation**: User input never concatenated into SQL strings
4. **JSONB Safety**: JSONB operations use parameterized operators (`->$2`)

### Test Coverage

#### TestSQLInjection_PublisherID
Tests injection attempts via publisher ID parameter:
- OR clause injection (`' OR '1'='1`)
- UNION injection (`' UNION SELECT * FROM users--`)
- DROP TABLE attacks (`'; DROP TABLE publishers; --`)
- Comment injection (`admin'--`)
- Stacked queries (`pub123'; INSERT INTO ...`)
- Time-based blind injection (`' OR SLEEP(5)--`)
- Hex encoding attempts

#### TestSQLInjection_GetBidderParams
Tests JSONB query injection:
- Injection via bidder code parameter
- JSONB operator exploitation attempts
- Publisher ID injection in JSONB context

#### TestSQLInjection_SpecialCharacters
Tests handling of SQL special characters:
- Single quotes (`O'Reilly Media`)
- Double quotes (`Company "Best" LLC`)
- Backslashes, semicolons, percent signs
- Null bytes, Unicode characters
- Combined attack strings

### Verification

All tests verify that:
1. Malicious input is treated as data, not SQL code
2. Query uses parameterized statements
3. No SQL injection can modify query logic
4. Data integrity is maintained

## 2. Cross-Site Scripting (XSS) Protection

### Location
`internal/endpoints/xss_security_test.go`

### Protection Mechanisms

1. **Content-Type Header**: All responses set `Content-Type: application/json`
2. **JSON Encoding**: Go's `json.Marshal()` escapes HTML special characters
3. **No HTML Rendering**: Pure JSON API with no HTML templates
4. **CRLF Protection**: Go's `net/http` strips `\r` and `\n` from headers

### Test Coverage

#### TestXSS_JSONEncodingPreventsScriptExecution
Tests that JSON encoding escapes dangerous characters:
- Script tags (`<script>alert(1)</script>`)
- Event handlers (`<img onerror=alert(1)>`)
- JavaScript protocol (`javascript:alert(1)`)
- Various HTML injection attempts
- All special characters properly escaped

#### TestXSS_ValidationErrorEncoding
Tests XSS prevention in validation error messages:
- Malicious field names
- Malicious error messages
- Combined attacks in validation responses

#### TestXSS_ContentTypeHeader
Documents how `Content-Type: application/json` prevents browser script execution

#### TestXSS_HeaderInjectionPrevention
Tests CRLF injection prevention:
- Newline injection attempts
- Header injection attacks
- Verifies Go's automatic sanitization

#### TestXSS_RealWorldScenarios
Tests realistic attack scenarios:
- Stored XSS (malicious data in database)
- Reflected XSS (malicious URL/input)
- DOM XSS (client-side injection)
- Mutation XSS (HTML parser confusion)
- Polyglot XSS (universal payload)

### Character Escaping

JSON encoding automatically escapes:
- `<` becomes `\u003c`
- `>` becomes `\u003e`
- `&` becomes `\u0026`
- `'` becomes `\u0027`
- `"` becomes `\"`

## 3. Denial of Service (DoS) Protection

### Location
`internal/middleware/dos_protection_test.go`

### Protection Mechanisms

1. **Rate Limiting**: Token bucket algorithm, configurable RPS and burst
2. **Request Size Limits**: Body size (1MB) and URL length (8KB) limits
3. **Timeout Protection**: Server-level ReadTimeout, WriteTimeout, IdleTimeout
4. **IP Spoofing Prevention**: Trusted proxy validation for X-Forwarded-For
5. **Concurrent Protection**: Thread-safe rate limiting with mutex

### Test Coverage

#### TestDoS_RateLimitFlood
Tests protection against request flooding:
- 100 rapid requests from single IP
- Verifies burst size allows initial requests
- Confirms rate limiting blocks excessive requests
- Checks for proper 429 (Too Many Requests) responses

#### TestDoS_ConcurrentFlood
Tests protection against concurrent attacks:
- 50 goroutines × 10 requests each
- Verifies thread-safe rate limiting
- Tests mutex protection during concurrent access

#### TestDoS_OversizedRequest
Tests body size limits:
- Small requests pass through
- Requests at limit are allowed
- Oversized requests rejected (413 Payload Too Large)
- Very large requests (10MB) blocked

#### TestDoS_OversizedURL
Tests URL length limits:
- Normal URLs pass through
- URLs at limit allowed
- Oversized URLs rejected (414 URI Too Long)
- Very long URLs (10KB+) blocked

#### TestDoS_UnknownContentLength
Tests rejection of unknown content length:
- Requests with `Content-Length: -1` rejected
- Prevents memory exhaustion from unbounded reads

#### TestDoS_IPSpoofingPrevention
Tests IP spoofing protection:
- X-Forwarded-For ignored without trusted proxies
- XFF only trusted from configured CIDR ranges
- Falls back to RemoteAddr for untrusted sources

### Configuration

Environment variables for DoS protection:
```bash
RATE_LIMIT_RPS=1000          # Requests per second
RATE_LIMIT_BURST=2000        # Burst size
MAX_REQUEST_SIZE=1048576     # 1MB body limit
MAX_URL_LENGTH=8192          # 8KB URL limit
TRUSTED_PROXIES=10.0.0.0/8   # Trusted proxy CIDR
```

## Security Test Documentation

Each test file includes comprehensive documentation tests:

- `TestSQLInjection_Documentation` - Documents SQL injection protection
- `TestXSS_Documentation` - Documents XSS protection mechanisms
- `TestDoS_Documentation` - Documents DoS protection layers

Run these to see detailed security documentation:
```bash
go test -v ./internal/storage -run TestSQLInjection_Documentation
go test -v ./internal/endpoints -run TestXSS_Documentation
go test -v ./internal/middleware -run TestDoS_Documentation
```

## Expected Security Behavior

### SQL Injection
✅ All queries use parameterized statements
✅ No string concatenation in SQL
✅ Special characters handled safely
✅ JSONB operations properly parameterized
✅ PostgreSQL driver provides automatic escaping

### XSS Protection
✅ Content-Type: application/json prevents execution
✅ JSON encoding escapes HTML characters
✅ No HTML rendering endpoints
✅ CRLF injection prevented automatically
✅ All user input treated as data, not code

### DoS Protection
✅ Rate limiting active with token bucket algorithm
✅ Request size limits enforced
✅ URL length limits enforced
✅ Unknown content length rejected
✅ IP spoofing prevented with proxy validation
✅ Concurrent requests handled safely

## Attack Vectors Prevented

### SQL Injection
- OR clause injection (bypass WHERE)
- UNION injection (data exfiltration)
- Stacked queries (multiple statements)
- Comment injection (-- or /**/)
- Time-based blind injection
- Special character exploitation

### XSS
- Stored XSS (malicious data in database)
- Reflected XSS (malicious data in URL/input)
- DOM-based XSS (client-side script injection)
- Event handler injection (onerror, onload, etc.)
- JavaScript protocol injection (javascript:)
- CRLF header injection
- Content-Type confusion

### DoS
- Request flooding
- Memory exhaustion
- Slowloris attacks
- Connection exhaustion
- IP spoofing
- Concurrent floods

## Security Principles

1. **Defense in Depth**: Multiple layers of protection
2. **Secure by Default**: Protection enabled without configuration
3. **Fail Secure**: Errors don't bypass security
4. **Least Privilege**: Minimal permissions required
5. **Input Validation**: All input validated before processing

## Compliance

These security tests help ensure compliance with:
- OWASP Top 10 (2021)
- PCI DSS (Payment Card Industry Data Security Standard)
- GDPR (General Data Protection Regulation)
- CWE (Common Weakness Enumeration)

## Continuous Testing

Security tests should be run:
- On every commit (CI/CD pipeline)
- Before deployments
- After dependency updates
- During security audits
- As part of regular testing

## Reporting Security Issues

If you discover a security vulnerability:
1. DO NOT open a public issue
2. Email security@example.com with details
3. Include steps to reproduce
4. Allow 90 days for patching before disclosure

## References

- OWASP SQL Injection: https://owasp.org/www-community/attacks/SQL_Injection
- OWASP XSS: https://owasp.org/www-community/attacks/xss/
- OWASP DoS: https://owasp.org/www-community/attacks/Denial_of_Service
- CWE-89 (SQL Injection): https://cwe.mitre.org/data/definitions/89.html
- CWE-79 (XSS): https://cwe.mitre.org/data/definitions/79.html
- CWE-400 (Resource Exhaustion): https://cwe.mitre.org/data/definitions/400.html
