# Video Event Tracking Endpoints

## Overview

Prebid Server provides comprehensive video event tracking endpoints to monitor video ad playback, user interactions, and errors. These endpoints support the IAB VAST specification and provide detailed analytics for video advertising campaigns.

## Event Types

### Standard VAST Events

| Event | VType Parameter | Description |
|-------|----------------|-------------|
| Start | `start` | Video playback started |
| First Quartile | `firstQuartile` | 25% of video completed |
| Midpoint | `midPoint` | 50% of video completed |
| Third Quartile | `thirdQuartile` | 75% of video completed |
| Complete | `complete` | Video playback completed |

### User Interaction Events

| Event | VType Parameter | Description |
|-------|----------------|-------------|
| Click | `click` | User clicked on the video ad |
| Pause | `pause` | User paused video playback |
| Resume | `resume` | User resumed video playback |
| Mute | `mute` | User muted the video |
| Unmute | `unmute` | User unmuted the video |
| Fullscreen | `fullscreen` | User entered fullscreen mode |
| Exit Fullscreen | `exitFullscreen` | User exited fullscreen mode |
| Skip | `skip` | User skipped the video ad |

### Error Events

| Event | VType Parameter | Description |
|-------|----------------|-------------|
| Error | `error` | Video playback or load error |

## API Endpoint

### Base URL

```
GET /event
```

### Required Parameters

| Parameter | Description | Example |
|-----------|-------------|---------|
| `t` | Event type (must be "vast" for video events) | `vast` |
| `vtype` | VAST event type | `start`, `click`, `error`, etc. |
| `b` | Bid ID from the auction | `bid-abc-123` |
| `a` | Account ID | `publisher-123` |

### Optional Parameters

| Parameter | Description | Example |
|-----------|-------------|---------|
| `bidder` | Bidder name | `appnexus` |
| `ts` | Unix timestamp (milliseconds) | `1706543210000` |
| `f` | Response format (`b` = blank, `i` = image) | `i` |
| `x` | Analytics enabled (`1` = enabled, `0` = disabled) | `1` |
| `int` | Integration type | `prebid-video-1.0` |
| `ec` | Error code (for error events) | `400` |
| `em` | Error message (for error events) | `File not found` |
| `ct` | Click-through URL (for click events) | `https://example.com` |

## Usage Examples

### Track Video Start

```
GET /event?t=vast&vtype=start&b=bid-abc-123&a=publisher-123&bidder=appnexus&ts=1706543210000
```

### Track Video Quartiles

```
GET /event?t=vast&vtype=firstQuartile&b=bid-abc-123&a=publisher-123
GET /event?t=vast&vtype=midPoint&b=bid-abc-123&a=publisher-123
GET /event?t=vast&vtype=thirdQuartile&b=bid-abc-123&a=publisher-123
GET /event?t=vast&vtype=complete&b=bid-abc-123&a=publisher-123
```

### Track User Click

```
GET /event?t=vast&vtype=click&b=bid-abc-123&a=publisher-123&ct=https%3A%2F%2Fexample.com%2Flanding
```

### Track Video Error

```
GET /event?t=vast&vtype=error&b=bid-abc-123&a=publisher-123&ec=400&em=File+not+found
```

### Track User Interactions

```
# Pause
GET /event?t=vast&vtype=pause&b=bid-abc-123&a=publisher-123

# Resume
GET /event?t=vast&vtype=resume&b=bid-abc-123&a=publisher-123

# Mute
GET /event?t=vast&vtype=mute&b=bid-abc-123&a=publisher-123

# Fullscreen
GET /event?t=vast&vtype=fullscreen&b=bid-abc-123&a=publisher-123

# Skip
GET /event?t=vast&vtype=skip&b=bid-abc-123&a=publisher-123
```

## Response Formats

### Blank Response (Default)

```
HTTP/1.1 204 No Content
```

### Image Response (1x1 PNG Tracking Pixel)

```
GET /event?t=vast&vtype=start&b=bid-abc-123&a=publisher-123&f=i

HTTP/1.1 200 OK
Content-Type: image/png
[1x1 PNG binary data]
```

## VAST Error Codes

When tracking error events, use standard IAB VAST error codes:

### XML Error Codes (100-199)

| Code | Description |
|------|-------------|
| 100 | XML parsing error |
| 101 | VAST schema validation error |
| 102 | VAST version not supported |

### Trafficking Error Codes (200-299)

| Code | Description |
|------|-------------|
| 200 | Trafficking error (general) |
| 201 | Video player expecting different linearity |
| 202 | Video player expecting different duration |
| 203 | Video player expecting different size |

### Wrapper Error Codes (300-399)

| Code | Description |
|------|-------------|
| 300 | General wrapper error |
| 301 | Timeout of VAST URI provided in Wrapper |
| 302 | Wrapper limit reached |
| 303 | No VAST response after one or more Wrappers |

### Linear Error Codes (400-499)

| Code | Description |
|------|-------------|
| 400 | General linear error |
| 401 | File not found (broken link to asset) |
| 402 | Timeout when loading media file |
| 403 | Media file not supported (codec/format) |
| 405 | Problem displaying media file |

### Nonlinear Error Codes (500-599)

| Code | Description |
|------|-------------|
| 500 | General NonLinear error |
| 501 | Unable to display NonLinear ad |
| 502 | Unable to fetch NonLinear asset |
| 503 | NonLinear asset not supported |

### Companion Error Codes (600-699)

| Code | Description |
|------|-------------|
| 600 | General Companion ad error |
| 601 | Unable to display Companion |
| 602 | Unable to fetch Companion asset |
| 603 | Companion asset not supported |

### Undefined Error Codes (900-999)

| Code | Description |
|------|-------------|
| 900 | Undefined error |
| 901 | General VPAID error |

## Integration Example

### JavaScript Video Player

```javascript
// VAST event tracking integration
const trackVastEvent = (vtype, additionalParams = {}) => {
  const params = new URLSearchParams({
    t: 'vast',
    vtype: vtype,
    b: bidId,
    a: accountId,
    bidder: bidderName,
    ts: Date.now(),
    ...additionalParams
  });

  const url = `${prebidServerUrl}/event?${params.toString()}`;

  // Send tracking beacon
  if (navigator.sendBeacon) {
    navigator.sendBeacon(url);
  } else {
    new Image().src = url;
  }
};

// Video player event listeners
videoPlayer.on('start', () => {
  trackVastEvent('start');
});

videoPlayer.on('firstQuartile', () => {
  trackVastEvent('firstQuartile');
});

videoPlayer.on('midpoint', () => {
  trackVastEvent('midPoint');
});

videoPlayer.on('thirdQuartile', () => {
  trackVastEvent('thirdQuartile');
});

videoPlayer.on('complete', () => {
  trackVastEvent('complete');
});

videoPlayer.on('click', (clickThroughUrl) => {
  trackVastEvent('click', { ct: clickThroughUrl });
});

videoPlayer.on('error', (error) => {
  trackVastEvent('error', {
    ec: error.code || '900',
    em: error.message || 'Undefined error'
  });
});

videoPlayer.on('pause', () => {
  trackVastEvent('pause');
});

videoPlayer.on('resume', () => {
  trackVastEvent('resume');
});

videoPlayer.on('skip', () => {
  trackVastEvent('skip');
});
```

### VAST XML Integration

```xml
<VAST version="4.0">
  <Ad>
    <InLine>
      <AdSystem>Prebid Server</AdSystem>
      <Impression>https://prebid-server.com/event?t=vast&vtype=start&b=bid-123&a=acc-123</Impression>

      <Creatives>
        <Creative>
          <Linear>
            <Duration>00:00:30</Duration>

            <TrackingEvents>
              <Tracking event="start">https://prebid-server.com/event?t=vast&vtype=start&b=bid-123&a=acc-123</Tracking>
              <Tracking event="firstQuartile">https://prebid-server.com/event?t=vast&vtype=firstQuartile&b=bid-123&a=acc-123</Tracking>
              <Tracking event="midpoint">https://prebid-server.com/event?t=vast&vtype=midPoint&b=bid-123&a=acc-123</Tracking>
              <Tracking event="thirdQuartile">https://prebid-server.com/event?t=vast&vtype=thirdQuartile&b=bid-123&a=acc-123</Tracking>
              <Tracking event="complete">https://prebid-server.com/event?t=vast&vtype=complete&b=bid-123&a=acc-123</Tracking>
              <Tracking event="pause">https://prebid-server.com/event?t=vast&vtype=pause&b=bid-123&a=acc-123</Tracking>
              <Tracking event="resume">https://prebid-server.com/event?t=vast&vtype=resume&b=bid-123&a=acc-123</Tracking>
              <Tracking event="mute">https://prebid-server.com/event?t=vast&vtype=mute&b=bid-123&a=acc-123</Tracking>
              <Tracking event="unmute">https://prebid-server.com/event?t=vast&vtype=unmute&b=bid-123&a=acc-123</Tracking>
              <Tracking event="fullscreen">https://prebid-server.com/event?t=vast&vtype=fullscreen&b=bid-123&a=acc-123</Tracking>
              <Tracking event="exitFullscreen">https://prebid-server.com/event?t=vast&vtype=exitFullscreen&b=bid-123&a=acc-123</Tracking>
              <Tracking event="skip">https://prebid-server.com/event?t=vast&vtype=skip&b=bid-123&a=acc-123</Tracking>
            </TrackingEvents>

            <VideoClicks>
              <ClickTracking>https://prebid-server.com/event?t=vast&vtype=click&b=bid-123&a=acc-123</ClickTracking>
              <ClickThrough>https://advertiser.com/landing-page</ClickThrough>
            </VideoClicks>

            <MediaFiles>
              <MediaFile delivery="progressive" type="video/mp4" width="1280" height="720">
                https://cdn.example.com/video.mp4
              </MediaFile>
            </MediaFiles>
          </Linear>
        </Creative>
      </Creatives>

      <Error>https://prebid-server.com/event?t=vast&vtype=error&b=bid-123&a=acc-123&ec=[ERRORCODE]</Error>
    </InLine>
  </Ad>
</VAST>
```

## Analytics Integration

Video event data is automatically sent to configured analytics adapters when `x=1` (analytics enabled).

### Event Data Structure

```json
{
  "type": "vast",
  "vtype": "complete",
  "bidid": "bid-abc-123",
  "account_id": "publisher-123",
  "bidder": "appnexus",
  "timestamp": 1706543210000,
  "integration": "prebid-video-1.0",
  "error_code": null,
  "error_message": null,
  "click_through": null
}
```

## Account Configuration

Enable video event tracking for specific accounts:

```yaml
accounts:
  - id: "publisher-123"
    events:
      enabled: true
```

## Security & Privacy

- All events require valid account ID (`a` parameter)
- Account must have events enabled
- GDPR/CCPA privacy controls are respected
- No PII is tracked in event URLs
- Bid IDs are anonymized references

## Performance Considerations

- Event tracking uses HTTP GET requests for maximum compatibility
- Response format `f=b` (blank) returns 204 No Content for minimal bandwidth
- Response format `f=i` (image) returns 1x1 PNG for legacy compatibility
- Use `navigator.sendBeacon()` for reliable tracking without blocking page unload
- Batch events when possible to reduce network overhead

## Best Practices

1. **Always track core quartile events**: start, firstQuartile, midPoint, thirdQuartile, complete
2. **Track user interactions**: pause, resume, mute, fullscreen, skip provide engagement insights
3. **Include timestamps**: Add `ts` parameter for accurate time-series analysis
4. **Track errors with context**: Always include `ec` (error code) and `em` (error message)
5. **Use analytics flag**: Set `x=1` to enable analytics processing
6. **Include bidder name**: Add `bidder` parameter for bidder-level analytics

## Troubleshooting

### Events Not Tracking

1. Verify account has events enabled: Check account configuration
2. Check required parameters: Ensure `t`, `vtype`, `b`, and `a` are present
3. Verify account ID: Ensure account exists and is active
4. Check network requests: Use browser DevTools to inspect tracking calls

### Analytics Not Appearing

1. Ensure `x=1` parameter is set
2. Verify analytics adapter configuration
3. Check account analytics settings
4. Review analytics adapter logs

### Error Event Not Tracking

1. Verify `ec` (error code) is included
2. Ensure `vtype=error` is set
3. Check that error message (`em`) doesn't exceed URL length limits

## Related Documentation

- [VAST 4.2 Specification](https://iabtechlab.com/standards/vast/)
- [Prebid Video Overview](https://docs.prebid.org/prebid-video/video-overview.html)
- [Event Analytics](../../analytics/README.md)
