# Pause Ads Package

This package implements IAB Tech Lab CTV pause ad format detection and serving for Prebid Server.

## Overview

Pause ads are display or video advertisements shown when a user pauses CTV (Connected TV) content. This package provides:

1. **Pause Event Detection**: Detects pause/resume signals from OpenRTB bid requests
2. **Ad Serving**: Serves appropriate display or video ads during pause state
3. **Resume Handling**: Manages cleanup when content resumes
4. **VAST Integration**: Leverages the existing VAST package for video ad processing

## Features

- OpenRTB signal-based pause detection
- Support for both display and video pause ad formats
- VAST XML parsing for video ads
- Impression and click tracker extraction
- Bid price-based winner selection
- Session management for pause/resume flows

## Usage

### Detecting Pause Events

```go
import (
    "github.com/prebid/prebid-server/v3/pauseads"
    "github.com/prebid/openrtb/v20/openrtb2"
)

// OpenRTB bid request with pause signal in extensions
bidRequest := &openrtb2.BidRequest{
    ID: "request-123",
    Ext: json.RawMessage(`{
        "pause": {
            "state": "paused",
            "format": "display",
            "sessionId": "session-abc",
            "timestamp": 1234567890000
        }
    }`),
}

// Detect the pause event
pauseReq, err := pauseads.DetectPauseEvent(bidRequest)
if err != nil {
    // Handle error - no pause signal detected
}
```

### Serving Pause Ads

```go
// After running the bid auction, serve the pause ad
pauseResp, err := pauseads.ServePauseAd(pauseReq, bidResponse)
if err != nil {
    // Handle error
}

// Use the response
fmt.Println("Ad Markup:", pauseResp.AdMarkup)
fmt.Println("Impression Trackers:", pauseResp.ImpressionTrackers)
fmt.Println("Click Trackers:", pauseResp.ClickTrackers)
```

### Handling Resume Events

```go
// When content resumes
err := pauseads.HandleResume(sessionID)
if err != nil {
    // Handle error
}
```

### Validating Requests

```go
// Validate a pause ad request before processing
err := pauseads.IsValidPauseAdRequest(pauseReq)
if err != nil {
    // Handle validation error
}
```

## OpenRTB Request Format

Pause ad signals are passed in the bid request extensions:

```json
{
  "id": "request-id",
  "ext": {
    "pause": {
      "state": "paused",          // Required: "paused" or "resumed"
      "format": "display",         // Optional: "display" or "video" (default: "display")
      "sessionId": "session-123",  // Optional: unique session identifier
      "timestamp": 1234567890000   // Optional: Unix timestamp in milliseconds
    }
  }
}
```

## Response Format

### Display Ad Response

```json
{
  "format": "display",
  "bidId": "bid-123",
  "price": 2.50,
  "currency": "USD",
  "adMarkup": "<div>Ad Creative HTML</div>",
  "impressionTrackers": [
    "https://example.com/impression"
  ],
  "clickTrackers": []
}
```

### Video Ad Response

```json
{
  "format": "video",
  "bidId": "bid-456",
  "price": 3.75,
  "currency": "USD",
  "adMarkup": "<?xml version=\"1.0\"?><VAST>...</VAST>",
  "impressionTrackers": [
    "https://example.com/vast-impression",
    "https://example.com/nurl"
  ],
  "clickTrackers": [
    "https://example.com/clickthrough",
    "https://example.com/clicktracking"
  ]
}
```

## Data Structures

### PauseAdState

- `StatePaused`: Content is paused, serve pause ad
- `StateResumed`: Content resumed, clear pause ad

### PauseAdFormat

- `FormatDisplay`: Display/banner pause ad
- `FormatVideo`: Video pause ad (uses VAST)

### PauseAdRequest

Main request structure containing:
- `State`: Current pause state
- `Format`: Desired ad format
- `BidRequest`: OpenRTB bid request
- `SessionID`: Unique session identifier
- `Timestamp`: Event timestamp

### PauseAdResponse

Response structure containing:
- `AdMarkup`: Creative markup (HTML or VAST XML)
- `Format`: Ad format
- `BidID`: Winning bid identifier
- `Price`: Bid price
- `Currency`: Price currency
- `ImpressionTrackers`: URLs to fire on impression
- `ClickTrackers`: URLs to fire on click
- `Error`: Error message if request failed

## Integration with VAST Package

For video format pause ads, this package integrates with the existing `/vast` package:

1. Parses VAST XML from bid response using `vast.Parse()`
2. Extracts impression trackers from VAST document
3. Extracts click-through and click-tracking URLs
4. Validates VAST structure

## Testing

Run the test suite:

```bash
go test -v ./pauseads
```

Run with coverage:

```bash
go test -cover ./pauseads
```

## Examples

See `pauseads_test.go` for comprehensive examples including:

- Pause event detection with various signals
- Display ad serving with bid selection
- Video ad serving with VAST parsing
- Resume event handling
- Request validation
- Complete pause/resume flow

## IAB Tech Lab Compliance

This implementation follows IAB Tech Lab guidelines for CTV pause ads:

- OpenRTB extension-based signaling
- Support for both display and video formats
- Proper tracking pixel handling
- Session-based state management

## Future Enhancements

Potential improvements:

- State caching for pause sessions
- Advanced bid selection strategies
- Frequency capping for pause ads
- Ad pod support for longer pauses
- Integration with analytics module for pause event tracking
