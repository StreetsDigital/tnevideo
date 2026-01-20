# TNE Catalyst API Reference

**Version:** 1.0
**Base URL:** `https://catalyst.springwire.ai`
**Protocol:** OpenRTB 2.5

---

## Table of Contents

1. [Authentication](#authentication)
2. [Endpoints](#endpoints)
3. [Auction Endpoint](#auction-endpoint)
4. [Health Checks](#health-checks)
5. [Metrics](#metrics)
6. [Error Codes](#error-codes)
7. [Rate Limiting](#rate-limiting)

---

## Authentication

### API Key Authentication

All auction requests require authentication via API key.

**Method:** HTTP Header
**Header:** `X-API-Key`
**Format:** 32-character alphanumeric string

**Example:**
```bash
curl -X POST https://catalyst.springwire.ai/openrtb2/auction \
  -H "X-API-Key: your-api-key-here" \
  -H "Content-Type: application/json" \
  -d @bid-request.json
```

**Publisher Signup:**
Contact your account manager or email: publishers@springwire.ai

---

## Endpoints

### Core Endpoints

| Endpoint | Method | Auth | Description |
|----------|--------|------|-------------|
| `/openrtb2/auction` | POST | Required | Submit bid request |
| `/health` | GET | None | Basic health check |
| `/health/ready` | GET | None | Readiness probe |
| `/metrics` | GET | None | Prometheus metrics |

---

## Auction Endpoint

### POST /openrtb2/auction

Submit a bid request to the ad exchange.

**Request:**
```http
POST /openrtb2/auction HTTP/1.1
Host: catalyst.springwire.ai
Content-Type: application/json
X-API-Key: your-api-key-here

{
  "id": "auction-123",
  "imp": [
    {
      "id": "1",
      "banner": {
        "w": 300,
        "h": 250
      },
      "bidfloor": 0.50,
      "bidfloorcur": "USD"
    }
  ],
  "site": {
    "id": "site-123",
    "domain": "example.com",
    "page": "https://example.com/article"
  },
  "device": {
    "ua": "Mozilla/5.0...",
    "ip": "192.0.2.1",
    "devicetype": 2
  },
  "user": {
    "id": "user-456"
  }
}
```

**Response (Success):**
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "id": "auction-123",
  "seatbid": [
    {
      "seat": "bidder-1",
      "bid": [
        {
          "id": "bid-789",
          "impid": "1",
          "price": 1.25,
          "adm": "<html>...</html>",
          "adomain": ["advertiser.com"],
          "crid": "creative-123",
          "w": 300,
          "h": 250
        }
      ]
    }
  ],
  "bidid": "bid-response-123",
  "cur": "USD"
}
```

**Response (No Bid):**
```http
HTTP/1.1 204 No Content
```

**Response (Error):**
```http
HTTP/1.1 400 Bad Request
Content-Type: application/json

{
  "error": "Invalid bid request",
  "details": "Missing required field: imp",
  "nbr": 2
}
```

### Privacy Compliance

**GDPR (EU/EEA):**
```json
{
  "regs": {
    "gdpr": 1
  },
  "user": {
    "consent": "CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA"
  }
}
```

**CCPA (California):**
```json
{
  "regs": {
    "us_privacy": "1YNN"
  }
}
```

**COPPA (Children):**
```json
{
  "regs": {
    "coppa": 1
  }
}
```

### Geo Targeting

**Device Geo:**
```json
{
  "device": {
    "geo": {
      "country": "USA",
      "region": "CA",
      "city": "Los Angeles",
      "zip": "90001",
      "lat": 34.0522,
      "lon": -118.2437
    }
  }
}
```

### Supported Ad Formats

- **Banner:** 300x250, 728x90, 160x600, 320x50, 970x250
- **Video:** VAST 2.0/3.0 (planned)
- **Native:** IAB Native 1.2 (planned)

---

## Health Checks

### GET /health

Basic liveness check. Returns 200 if service is running.

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2026-01-19T23:30:00Z"
}
```

**Status Codes:**
- `200 OK` - Service is alive
- `503 Service Unavailable` - Service is down

---

### GET /health/ready

Readiness check. Verifies all dependencies are healthy.

**Response:**
```json
{
  "ready": true,
  "timestamp": "2026-01-19T23:30:00Z",
  "checks": {
    "database": {
      "status": "healthy"
    },
    "redis": {
      "status": "healthy"
    },
    "idr": {
      "status": "disabled"
    }
  }
}
```

**Status Codes:**
- `200 OK` - Service is ready to accept traffic
- `503 Service Unavailable` - Service is not ready

---

## Metrics

### GET /metrics

Prometheus-formatted metrics for monitoring.

**Response:**
```prometheus
# HELP catalyst_auctions_total Total number of auctions
# TYPE catalyst_auctions_total counter
catalyst_auctions_total{result="success"} 12345

# HELP catalyst_bids_received_total Total bids received from bidders
# TYPE catalyst_bids_received_total counter
catalyst_bids_received_total 45678

# HELP catalyst_http_request_duration_seconds HTTP request latency
# TYPE catalyst_http_request_duration_seconds histogram
catalyst_http_request_duration_seconds_bucket{endpoint="/openrtb2/auction",le="0.1"} 8234
catalyst_http_request_duration_seconds_bucket{endpoint="/openrtb2/auction",le="0.2"} 9876
catalyst_http_request_duration_seconds_sum{endpoint="/openrtb2/auction"} 1234.56
catalyst_http_request_duration_seconds_count{endpoint="/openrtb2/auction"} 10000
```

**Key Metrics:**
- `catalyst_auctions_total` - Total auctions processed
- `catalyst_bids_received_total` - Bids received from bidders
- `catalyst_revenue_usd_total` - Total revenue in USD
- `catalyst_http_request_duration_seconds` - Request latency
- `catalyst_privacy_violations_total` - Privacy compliance violations
- `catalyst_database_connections_*` - Database pool metrics

---

## Error Codes

### HTTP Status Codes

| Code | Meaning | Description |
|------|---------|-------------|
| 200 | OK | Bid response with bids |
| 204 | No Content | No bids returned |
| 400 | Bad Request | Invalid bid request |
| 401 | Unauthorized | Missing or invalid API key |
| 403 | Forbidden | API key valid but access denied |
| 429 | Too Many Requests | Rate limit exceeded |
| 500 | Internal Server Error | Server-side error |
| 503 | Service Unavailable | Service is down or not ready |

### OpenRTB No-Bid Reason Codes

| Code | Reason |
|------|--------|
| 0 | Unknown error |
| 1 | Technical error |
| 2 | Invalid request |
| 3 | Known web spider |
| 4 | Suspected non-human traffic |
| 5 | Cloud, data center, or proxy IP |
| 6 | Unsupported device |
| 7 | Blocked publisher or site |
| 8 | Unmatched user |
| 9 | Privacy compliance (GDPR/CCPA) |

---

## Rate Limiting

### Limits

**Per Publisher:**
- **Requests:** 10,000 per minute
- **Burst:** 100 requests

**Headers:**
```http
X-RateLimit-Limit: 10000
X-RateLimit-Remaining: 9543
X-RateLimit-Reset: 1674172800
```

### Rate Limit Exceeded

**Response:**
```http
HTTP/1.1 429 Too Many Requests
Content-Type: application/json
Retry-After: 60

{
  "error": "Rate limit exceeded",
  "limit": 10000,
  "window": "1 minute",
  "retry_after": 60
}
```

---

## Request Examples

### Minimal Banner Request

```json
{
  "id": "80ce30c53c16e6ede735f123ef6e32361bfc7b22",
  "imp": [{
    "id": "1",
    "banner": {
      "w": 300,
      "h": 250
    }
  }],
  "site": {
    "page": "https://example.com"
  },
  "device": {
    "ua": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
    "ip": "192.0.2.1"
  }
}
```

### Banner with GDPR

```json
{
  "id": "auction-123",
  "imp": [{
    "id": "1",
    "banner": {
      "w": 728,
      "h": 90,
      "pos": 1
    },
    "bidfloor": 0.50,
    "bidfloorcur": "USD"
  }],
  "site": {
    "id": "site-456",
    "domain": "publisher.com",
    "page": "https://publisher.com/news/article",
    "cat": ["IAB12"]
  },
  "device": {
    "ua": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
    "ip": "2.125.160.216",
    "geo": {
      "country": "DEU",
      "city": "Berlin"
    },
    "devicetype": 2,
    "language": "de"
  },
  "user": {
    "id": "user-789",
    "consent": "CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA"
  },
  "regs": {
    "gdpr": 1
  }
}
```

### Mobile App Request

```json
{
  "id": "mobile-auction-123",
  "imp": [{
    "id": "1",
    "banner": {
      "w": 320,
      "h": 50
    },
    "instl": 0
  }],
  "app": {
    "id": "app-123",
    "name": "My Mobile App",
    "bundle": "com.example.app",
    "storeurl": "https://play.google.com/store/apps/details?id=com.example.app",
    "cat": ["IAB1"]
  },
  "device": {
    "ua": "Mozilla/5.0 (iPhone; CPU iPhone OS 15_0 like Mac OS X)",
    "ip": "192.0.2.1",
    "geo": {
      "country": "USA",
      "region": "CA"
    },
    "devicetype": 4,
    "make": "Apple",
    "model": "iPhone13,2",
    "os": "iOS",
    "osv": "15.0",
    "ifa": "AEBE52E7-03EE-455A-B3C4-E57283966239"
  },
  "user": {
    "id": "user-mobile-456"
  }
}
```

---

## Best Practices

### Performance
- Keep bid requests under 10KB
- Cache publisher/site configuration
- Use HTTP/2 or HTTP/3 if available
- Enable keep-alive connections

### Privacy
- Always include GDPR consent for EU users
- Include US Privacy string for California users
- Set COPPA flag for child-directed content
- Include geo data for proper enforcement

### Reliability
- Implement exponential backoff for retries
- Monitor 429 responses (rate limiting)
- Handle 503 gracefully (service unavailable)
- Set reasonable timeouts (< 200ms recommended)

---

## Support

**Technical Support:** tech@springwire.ai
**Account Management:** publishers@springwire.ai
**Documentation:** https://docs.springwire.ai
**Status Page:** https://status.springwire.ai

---

**Last Updated:** 2026-01-19
**API Version:** 1.0
**OpenRTB Version:** 2.5
