# Identity Resolution Module

The Identity Resolution module enriches OpenRTB bid requests with resolved user identities from external identity providers such as LiveRamp, UID2, ID5, or custom identity services.

## Overview

This module:
- Integrates with external identity resolution APIs (LiveRamp, UID2, ID5, or custom providers)
- Caches resolved identities to minimize latency
- Enriches bid requests with Extended IDs (EIDs) in OpenRTB format
- Handles timeouts gracefully to prevent auction delays
- Operates asynchronously to minimize impact on auction performance

## Configuration

### Basic Configuration

```json
{
  "provider": "liveramp",
  "api_key": "your-api-key",
  "enabled": true
}
```

### Full Configuration Options

```json
{
  "provider": "liveramp|uid2|id5|custom",
  "endpoint": "https://custom-identity-api.com/resolve",
  "api_key": "your-api-key",
  "timeout_ms": 500,
  "cache_ttl_seconds": 3600,
  "cache_size": 52428800,
  "enabled": true
}
```

### Configuration Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `provider` | string | Yes | - | Identity provider: `liveramp`, `uid2`, `id5`, or `custom` |
| `endpoint` | string | Conditional | Provider-specific | API endpoint (required for `custom` provider) |
| `api_key` | string | Yes | - | API key for authentication |
| `timeout_ms` | int | No | 500 | Request timeout in milliseconds |
| `cache_ttl_seconds` | int | No | 3600 | Cache TTL in seconds (1 hour) |
| `cache_size` | int | No | 52428800 | Cache size in bytes (50MB) |
| `enabled` | bool | No | false | Enable/disable the module |

### Default Endpoints

| Provider | Default Endpoint |
|----------|-----------------|
| LiveRamp | `https://api.liveramp.com/identity/v1/resolve` |
| UID2 | `https://prod.uidapi.com/v2/identity/map` |
| ID5 | `https://id5-sync.com/api/v1/id` |

## How It Works

### Hook Stages

The module implements three hook stages:

1. **EntrypointHook**: Initializes async request context
2. **RawAuctionHook**: Starts async identity resolution API call
3. **BidderRequestHook**: Enriches bidder requests with resolved identities

### Request Flow

```
┌─────────────────┐
│ Auction Request │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Entrypoint Hook │ ◄─── Initialize async context
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ RawAuction Hook │ ◄─── Start async identity resolution
└────────┬────────┘      (non-blocking)
         │
         ▼
┌─────────────────┐
│ Bidder Request  │ ◄─── Wait for identity data
│      Hook       │      Enrich user.eids
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Bidder Requests │
│   (enriched)    │
└─────────────────┘
```

### Caching

The module implements an in-memory LRU cache using `freecache`:
- Cache key is based on `provider + user_id + buyer_uid + ifa`
- Cache TTL is configurable (default: 1 hour)
- Cache size is configurable (default: 50MB)
- Reduces latency for repeated identity resolutions

### Timeout Handling

- Configurable timeout (default: 500ms)
- Graceful degradation: auction continues even if identity resolution fails
- Analytics tags track success/failure

## Identity Provider Integration

### Request Format

The module sends the following data to the identity provider:

```json
{
  "user_id": "user-123",
  "buyer_uid": "buyer-456",
  "ifa": "device-ifa-789",
  "eids": [...]
}
```

### Response Format

Expected response from identity provider:

```json
{
  "resolved_ids": {
    "liveramp": "rampid-abc123",
    "uid2": "uid2-xyz789"
  },
  "eids": [
    {
      "source": "liveramp.com",
      "uids": [
        {"id": "rampid-abc123"}
      ]
    },
    {
      "source": "uidapi.com",
      "uids": [
        {"id": "uid2-xyz789"}
      ]
    }
  ]
}
```

### OpenRTB Enrichment

Resolved identities are added to the `user.eids` array in the OpenRTB bid request:

```json
{
  "user": {
    "id": "user-123",
    "eids": [
      {
        "source": "liveramp.com",
        "uids": [
          {"id": "rampid-abc123"}
        ]
      },
      {
        "source": "uidapi.com",
        "uids": [
          {"id": "uid2-xyz789"}
        ]
      }
    ]
  }
}
```

## Analytics

The module tracks the following analytics events:

| Activity Name | Status | Description |
|--------------|--------|-------------|
| `HandleRawAuctionHook.identity.AsyncRequest` | error | Failed to get async request context |
| `HandleRawAuctionHook.bidRequest.unmarshal` | error | Failed to parse bid request |
| `HandleBidderRequestHook.identity_fetch` | error | Identity API call failed |
| `HandleBidderRequestHook.identity_enrichment` | success | Successfully enriched request |

Success analytics include:
```json
{
  "eids_added": 2
}
```

## Testing

Run tests:
```bash
go test ./modules/prebid/identity/...
```

Run with coverage:
```bash
go test -cover ./modules/prebid/identity/...
```

## Example Account Configuration

Enable for specific accounts in Prebid Server configuration:

```yaml
accounts:
  - id: "publisher-123"
    hooks:
      enabled: true
      execution_plan:
        entrypoint:
          - module_code: "prebid.identity"
            hook_impl_code: "entrypoint-hook"
        raw_auction_request:
          - module_code: "prebid.identity"
            hook_impl_code: "raw-auction-hook"
        bidder_request:
          - module_code: "prebid.identity"
            hook_impl_code: "bidder-request-hook"
```

## Performance Considerations

- **Latency**: Async design minimizes impact (resolution happens in parallel)
- **Timeout**: Default 500ms timeout prevents auction delays
- **Caching**: 1-hour cache significantly reduces API calls
- **Memory**: Default 50MB cache handles ~100k identities
- **Failsafe**: Auction continues even if identity resolution fails

## Privacy & Compliance

- Only sends identifiers already present in the bid request
- Respects existing privacy flags (GDPR consent, CCPA, etc.)
- Does not create new identifiers
- Cache keys are hashed to protect PII

## Supported Identity Providers

| Provider | Status | Notes |
|----------|--------|-------|
| LiveRamp | ✅ Supported | Default endpoint configured |
| UID2 | ✅ Supported | Default endpoint configured |
| ID5 | ✅ Supported | Default endpoint configured |
| Custom | ✅ Supported | Specify custom endpoint |

## Troubleshooting

### Identity enrichment not working

1. Check that `enabled: true` is set
2. Verify API key is correct
3. Check analytics tags for error details
4. Verify endpoint is reachable
5. Check timeout settings (increase if needed)

### High latency

1. Increase cache TTL to reduce API calls
2. Increase cache size if evictions are frequent
3. Reduce timeout if identity provider is slow
4. Consider using a faster identity provider

### Cache misses

1. Check cache size (may be too small)
2. Verify cache TTL is appropriate
3. Check if user IDs are stable across requests

## Future Enhancements

- [ ] Support for multiple providers in parallel
- [ ] Configurable cache eviction policies
- [ ] Metrics for cache hit/miss rates
- [ ] Support for batch identity resolution
- [ ] Webhook-based cache invalidation
