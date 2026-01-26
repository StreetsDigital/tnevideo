# Video Integration Guide

## Overview

TNEVideo provides comprehensive video advertising support with VAST 2.0, 3.0, and 4.x compliance. This guide covers integration for both supply-side (publishers) and demand-side (bidders) partners.

## Table of Contents

- [Quick Start](#quick-start)
- [Video Endpoints](#video-endpoints)
- [OpenRTB Video Integration](#openrtb-video-integration)
- [VAST Tag Generation](#vast-tag-generation)
- [VAST Tag Reception](#vast-tag-reception)
- [Event Tracking](#event-tracking)
- [CTV/OTT Support](#ctv-ott-support)
- [Testing](#testing)
- [Troubleshooting](#troubleshooting)

## Quick Start

### For Publishers

**Simple GET request for VAST:**
```bash
curl "https://your-server.com/video/vast?id=req-123&w=1920&h=1080&mindur=5&maxdur=30&mimes=video/mp4"
```

**Response:**
```xml
<?xml version="1.0" encoding="UTF-8"?>
<VAST version="4.0">
  <Ad id="ad-123">
    <InLine>
      <AdSystem>TNEVideo</AdSystem>
      <AdTitle>Sample Video Ad</AdTitle>
      <Impression><![CDATA[https://tracking.example.com/imp?id=123]]></Impression>
      <Creatives>
        <Creative>
          <Linear>
            <Duration>00:00:30</Duration>
            <MediaFiles>
              <MediaFile delivery="progressive" type="video/mp4" width="1920" height="1080">
                <![CDATA[https://cdn.example.com/video.mp4]]>
              </MediaFile>
            </MediaFiles>
            <TrackingEvents>
              <Tracking event="start"><![CDATA[https://tracking.example.com/start?id=123]]></Tracking>
              <Tracking event="complete"><![CDATA[https://tracking.example.com/complete?id=123]]></Tracking>
            </TrackingEvents>
          </Linear>
        </Creative>
      </Creatives>
    </InLine>
  </Ad>
</VAST>
```

### For Demand Partners

**Send OpenRTB video bid response:**
```json
{
  "id": "bid-response-123",
  "seatbid": [{
    "bid": [{
      "id": "bid-001",
      "impid": "1",
      "price": 5.50,
      "adm": "<?xml version=\"1.0\"?><VAST version=\"4.0\">...</VAST>",
      "crid": "creative-123",
      "w": 1920,
      "h": 1080
    }],
    "seat": "my-dsp"
  }],
  "cur": "USD"
}
```

## Video Endpoints

### GET /video/vast

Simple VAST tag generation via query parameters.

**Query Parameters:**

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| id | string | No | auto-generated | Request ID for tracking |
| w | int | No | 1920 | Video player width in pixels |
| h | int | No | 1080 | Video player height in pixels |
| mindur | int | No | 5 | Minimum video duration in seconds |
| maxdur | int | No | 30 | Maximum video duration in seconds |
| mimes | string | No | "video/mp4,video/webm" | Comma-separated MIME types |
| protocols | string | No | "2,3,5,6" | Comma-separated VAST protocol IDs |
| placement | int | No | 1 | Placement type (1=in-stream, 3=in-article, 4=in-feed, 5=interstitial) |
| skip | int | No | 0 | Skippable (0=no, 1=yes) |
| skipafter | int | No | 0 | Seconds before skip button appears |
| minbitrate | int | No | 300 | Minimum bitrate in kbps |
| maxbitrate | int | No | 5000 | Maximum bitrate in kbps |
| bidfloor | float | No | 0.0 | Minimum bid price (CPM) |
| site_id | string | No | - | Publisher site ID |
| domain | string | No | - | Publisher domain |
| page | string | No | - | Page URL |

**Example:**
```bash
curl "https://server.com/video/vast?w=1920&h=1080&skip=1&skipafter=5&bidfloor=3.0&site_id=pub-123"
```

### POST /video/openrtb

Full OpenRTB 2.x video bid request with JSON body.

**Request Body:**
```json
{
  "id": "video-request-001",
  "imp": [{
    "id": "1",
    "video": {
      "mimes": ["video/mp4", "video/webm"],
      "minduration": 5,
      "maxduration": 30,
      "protocols": [2, 3, 5, 6],
      "w": 1920,
      "h": 1080,
      "startdelay": 0,
      "placement": 1,
      "linearity": 1,
      "skip": 1,
      "skipafter": 5,
      "minbitrate": 1000,
      "maxbitrate": 5000,
      "api": [1, 2]
    },
    "bidfloor": 3.0,
    "bidfloorcur": "USD"
  }],
  "site": {
    "id": "site-123",
    "domain": "publisher.example.com",
    "page": "https://publisher.example.com/video-page"
  },
  "device": {
    "ua": "Mozilla/5.0...",
    "ip": "203.0.113.1"
  },
  "tmax": 1000,
  "cur": ["USD"]
}
```

**Response:** VAST 4.0 XML (same as GET /video/vast)

## OpenRTB Video Integration

### Video Object Fields

Required fields for video impressions:

```json
{
  "video": {
    "mimes": ["video/mp4"],           // Required: Supported MIME types
    "minduration": 15,                // Required: Min ad duration (seconds)
    "maxduration": 30,                // Required: Max ad duration (seconds)
    "protocols": [2, 3, 5, 6],        // Required: VAST protocols
    "w": 1920,                        // Required: Player width
    "h": 1080,                        // Required: Player height
    "startdelay": 0,                  // 0=pre-roll, -1=mid-roll, -2=post-roll
    "placement": 1,                   // 1=in-stream, 3=in-article, 4=in-feed, 5=interstitial
    "linearity": 1,                   // 1=linear/in-stream, 2=non-linear/overlay
    "skip": 1,                        // 0=no skip, 1=skippable
    "skipafter": 5,                   // Seconds before skip enabled
    "minbitrate": 1000,               // Min bitrate (kbps)
    "maxbitrate": 5000,               // Max bitrate (kbps)
    "api": [1, 2],                    // Supported APIs: 1=VPAID 1.0, 2=VPAID 2.0, 5=MRAID-1, 6=MRAID-2, 7=OMID-1
    "playbackmethod": [1],            // 1=auto-play sound on, 2=auto-play sound off, 3=click-to-play
    "delivery": [1]                   // 1=streaming, 2=progressive
  }
}
```

### VAST Protocol IDs

| ID | Protocol | Description |
|----|----------|-------------|
| 1 | VAST 1.0 | Legacy |
| 2 | VAST 2.0 | Widely supported |
| 3 | VAST 3.0 | Standard |
| 4 | VAST 1.0 Wrapper | Legacy wrapper |
| 5 | VAST 2.0 Wrapper | Common wrapper |
| 6 | VAST 3.0 Wrapper | Standard wrapper |
| 7 | VAST 4.0 | Latest inline |
| 8 | VAST 4.0 Wrapper | Latest wrapper |

## VAST Tag Generation

### Inline VAST

TNEVideo generates inline VAST tags with full creative details:

```xml
<VAST version="4.0">
  <Ad id="ad-12345">
    <InLine>
      <AdSystem version="1.0">TNEVideo</AdSystem>
      <AdTitle>Video Ad Title</AdTitle>
      <Description>Ad description</Description>
      <Impression id="imp-1"><![CDATA[https://tracking.example.com/impression]]></Impression>
      <Error><![CDATA[https://tracking.example.com/error?code=[ERRORCODE]]]></Error>
      <Creatives>
        <Creative id="creative-1">
          <Linear skipoffset="00:00:05">
            <Duration>00:00:30</Duration>
            <MediaFiles>
              <MediaFile delivery="progressive" type="video/mp4" width="1920" height="1080" bitrate="5000">
                <![CDATA[https://cdn.example.com/video.mp4]]>
              </MediaFile>
            </MediaFiles>
            <TrackingEvents>
              <Tracking event="start"><![CDATA[https://tracking.example.com/start]]></Tracking>
              <Tracking event="firstQuartile"><![CDATA[https://tracking.example.com/25]]></Tracking>
              <Tracking event="midpoint"><![CDATA[https://tracking.example.com/50]]></Tracking>
              <Tracking event="thirdQuartile"><![CDATA[https://tracking.example.com/75]]></Tracking>
              <Tracking event="complete"><![CDATA[https://tracking.example.com/complete]]></Tracking>
            </TrackingEvents>
            <VideoClicks>
              <ClickThrough><![CDATA[https://advertiser.example.com/landing]]></ClickThrough>
              <ClickTracking><![CDATA[https://tracking.example.com/click]]></ClickTracking>
            </VideoClicks>
          </Linear>
        </Creative>
      </Creatives>
    </InLine>
  </Ad>
</VAST>
```

### Wrapper VAST

For mediation scenarios, wrapper VAST redirects to demand partner:

```xml
<VAST version="4.0">
  <Ad id="wrapper-ad">
    <Wrapper>
      <AdSystem>TNEVideo SSP</AdSystem>
      <VASTAdTagURI><![CDATA[https://dsp.example.com/vast?auction=abc&price=${AUCTION_PRICE}]]></VASTAdTagURI>
      <Impression><![CDATA[https://ssp-tracking.example.com/impression]]></Impression>
      <Error><![CDATA[https://ssp-tracking.example.com/error]]></Error>
    </Wrapper>
  </Ad>
</VAST>
```

## Event Tracking

### Tracking Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| /video/event | POST/GET | Generic event tracking |
| /video/start | POST/GET | Video playback started |
| /video/complete | POST/GET | Video playback completed |
| /video/quartile | POST/GET | Quartile events (25%, 50%, 75%) |
| /video/click | POST/GET | User clicked video/overlay |
| /video/pause | POST/GET | Video paused |
| /video/resume | POST/GET | Video resumed |
| /video/error | POST/GET | Playback error occurred |

### Event Tracking Request (POST)

```json
{
  "event": "start",
  "bid_id": "bid-12345",
  "account_id": "pub-123",
  "bidder": "demand-partner-1",
  "timestamp": 1704067200000,
  "session_id": "session-abc",
  "content_id": "video-456"
}
```

### Event Tracking Request (GET)

```bash
GET /video/event?event=start&bid_id=bid-12345&account_id=pub-123&bidder=partner-1
```

**Response:** 1x1 transparent GIF for pixel tracking

### Supported Events

- **creativeView**: Ad creative loaded and visible
- **start**: Video playback started
- **firstQuartile**: 25% of video played
- **midpoint**: 50% of video played
- **thirdQuartile**: 75% of video played
- **complete**: 100% of video played
- **mute**: Audio muted
- **unmute**: Audio unmuted
- **pause**: Playback paused
- **resume**: Playback resumed from pause
- **rewind**: User rewound video
- **skip**: User skipped ad
- **fullscreen**: Entered fullscreen mode
- **exitFullscreen**: Exited fullscreen
- **click**: User clicked on ad

## CTV/OTT Support

### Detected Devices

TNEVideo automatically detects and optimizes for:

- **Roku**: Roku streaming devices
- **Fire TV**: Amazon Fire TV and Fire Stick
- **Apple TV**: Apple TV devices
- **Android TV**: Android TV and Google TV
- **Samsung Tizen**: Samsung Smart TVs
- **LG webOS**: LG Smart TVs
- **Vizio SmartCast**: Vizio Smart TVs
- **Chromecast**: Google Chromecast
- **Xbox**: Xbox gaming consoles
- **PlayStation**: PlayStation gaming consoles

### CTV Optimizations

When CTV device is detected:

1. **Bitrate Adjustment**: Limits max bitrate based on device capabilities
2. **VPAID Filtering**: Removes VPAID APIs if unsupported
3. **Format Selection**: Prioritizes compatible video formats
4. **Resolution Matching**: Optimizes for device screen resolution (1080p, 4K)

### 4K/UHD Support

For 4K content delivery:

```bash
curl "https://server.com/video/vast?w=3840&h=2160&minbitrate=10000&maxbitrate=25000&mimes=video/mp4"
```

Expected media files will include high-bitrate 4K options when available.

## Testing

### Test Fixtures

Located in `/tests/fixtures/`:
- `video_bid_requests.json` - Sample OpenRTB video requests
- `video_bid_responses.json` - Sample bid responses with VAST

### Integration Tests

Run video integration tests:

```bash
# All video tests
go test -tags=integration ./tests/integration/video_*

# Outbound VAST generation
go test -tags=integration ./tests/integration -run TestOutboundVAST

# Inbound VAST parsing
go test -tags=integration ./tests/integration -run TestInboundVAST

# With race detection
go test -tags=integration -race ./tests/integration/video_*
```

### Unit Tests

```bash
# VAST library tests
go test ./pkg/vast/...

# Video handler tests
go test ./internal/endpoints/video_*

# Coverage report
go test -cover ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Performance Benchmarks

```bash
# Run benchmarks
go test -bench=. ./tests/benchmark/video_*

# With memory profiling
go test -bench=. -benchmem ./tests/benchmark/video_*
```

## Troubleshooting

### Common Issues

#### No Ads Returned (Empty VAST)

**Symptoms:**
```xml
<VAST version="4.0"></VAST>
```

**Causes:**
- No demand partners bid
- Bid floor too high
- Unsupported video parameters
- Geographic targeting mismatch

**Solutions:**
1. Lower bid floor
2. Broaden video requirements (accept more formats, durations)
3. Check demand partner configuration
4. Review server logs for auction details

#### VAST Parse Errors

**Symptoms:** Player cannot parse VAST XML

**Causes:**
- Malformed XML from demand partner
- Invalid VAST structure
- Unsupported VAST version

**Solutions:**
1. Validate VAST against IAB XSD schema
2. Check VAST version compatibility (player vs. server)
3. Review demand partner VAST quality
4. Enable VAST debugging in player

#### Tracking Pixels Not Firing

**Symptoms:** Missing impression/event data

**Causes:**
- CORS restrictions
- Ad blocker interference
- Network issues
- Incorrect tracking URLs

**Solutions:**
1. Verify CORS headers: `Access-Control-Allow-Origin: *`
2. Test in private/incognito mode (no ad blockers)
3. Check tracking URL accessibility
4. Monitor server logs for incoming requests

#### Video Won't Play

**Symptoms:** Player error, blank screen

**Causes:**
- Unsupported video codec
- CORS on video file
- Invalid media file URL
- Bitrate too high for connection

**Solutions:**
1. Verify video codec compatibility (H.264 is safest)
2. Check CORS headers on CDN
3. Test media file URL directly in browser
4. Provide multiple bitrate options

### Debug Mode

Enable verbose logging:

```bash
LOG_LEVEL=debug ./server
```

Check logs for:
- Bid request/response details
- VAST generation steps
- Demand partner responses
- Auction winner selection

### VAST Validation

Use IAB VAST validator:
```bash
# Save VAST response
curl "https://server.com/video/vast?id=test" > vast_response.xml

# Validate against schema
xmllint --noout --schema vast4.xsd vast_response.xml
```

## Best Practices

### For Publishers

1. **Always specify video dimensions** (`w` and `h`) matching your player
2. **Set appropriate duration limits** based on content type
3. **Use realistic bid floors** - too high = no fill
4. **Enable skip for long ads** (>15s) to improve UX
5. **Test multiple devices** - desktop, mobile, CTV
6. **Monitor fill rates** and adjust parameters

### For Demand Partners

1. **Return complete VAST** with all required fields
2. **Include multiple bitrates** for adaptive streaming
3. **Use CDATA** for all URLs to avoid XML issues
4. **Test VAST in multiple players** (VideoJS, JW Player, etc.)
5. **Implement proper error tracking** with ErrorCode macro
6. **Support VAST 3.0 minimum**, 4.0 preferred
7. **Include companion ads** when available

## Support

For issues or questions:
- GitHub Issues: [thenexusengine/tne_springwire](https://github.com/thenexusengine/tne_springwire/issues)
- Documentation: `/docs`
- API Reference: `/docs/api`

## References

- [IAB VAST 4.0 Specification](https://www.iab.com/guidelines/vast/)
- [OpenRTB 2.5 Specification](https://www.iab.com/guidelines/openrtb/)
- [VAST Validator](https://vastvalidator.iabtechlab.com/)
- [Video Ad Serving Template Best Practices](https://www.iab.com/guidelines/vast/)
