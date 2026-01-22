# VAST Parser & Generator

A comprehensive Go package for parsing and generating Video Ad Serving Template (VAST) XML documents supporting versions 2.0, 3.0, 4.0, 4.1, and 4.2.

## Features

- **Parse VAST 2.0, 3.0, 4.0, 4.1, and 4.2** XML documents
- **Generate VAST wrapper ads** with builder pattern
- **Inject tracking URLs** for impressions, video events, errors, and clicks
- **Validate VAST documents** against specification requirements
- **Extract tracking URLs** from existing VAST documents
- **Type-safe Go structs** with full XML marshaling/unmarshaling support

## Installation

```go
import "github.com/prebid/prebid-server/v3/vast"
```

## Quick Start

### Parsing VAST XML

```go
package main

import (
    "fmt"
    "github.com/prebid/prebid-server/v3/vast"
)

func main() {
    xmlData := `<?xml version="1.0" encoding="UTF-8"?>
    <VAST version="4.2">
      <Ad id="12345">
        <InLine>
          <AdSystem>MyAdServer</AdSystem>
          <AdTitle>Sample Ad</AdTitle>
          <Impression><![CDATA[http://example.com/impression]]></Impression>
          <Creatives>
            <Creative>
              <Linear>
                <Duration>00:00:30</Duration>
                <MediaFiles>
                  <MediaFile delivery="progressive" type="video/mp4">
                    <![CDATA[http://example.com/video.mp4]]>
                  </MediaFile>
                </MediaFiles>
              </Linear>
            </Creative>
          </Creatives>
        </InLine>
      </Ad>
    </VAST>`

    // Parse the VAST XML
    vastDoc, err := vast.Parse(xmlData)
    if err != nil {
        panic(err)
    }

    fmt.Printf("VAST Version: %s\n", vastDoc.Version)
    fmt.Printf("Number of Ads: %d\n", len(vastDoc.Ads))
}
```

### Generating a VAST Wrapper

```go
package main

import (
    "fmt"
    "github.com/prebid/prebid-server/v3/vast"
)

func main() {
    // Simple wrapper creation
    vastDoc, err := vast.NewDefaultWrapper(
        "MyAdServer",                          // Ad system name
        "http://example.com/vast-tag",         // VAST tag URI
        []string{                              // Impression URLs
            "http://example.com/impression1",
            "http://example.com/impression2",
        },
    )
    if err != nil {
        panic(err)
    }

    // Convert to XML
    xmlString, err := vastDoc.Marshal()
    if err != nil {
        panic(err)
    }

    fmt.Println(xmlString)
}
```

### Advanced Wrapper Builder

```go
package main

import (
    "github.com/prebid/prebid-server/v3/vast"
)

func main() {
    config := vast.WrapperConfig{
        AdID:            "wrapper-123",
        AdSystem:        "MyAdServer",
        AdSystemVersion: "2.0",
        AdTitle:         "My Wrapper Ad",
        VASTAdTagURI:    "http://example.com/vast-tag",
        ImpressionURLs: []string{
            "http://example.com/impression",
        },
        ErrorURL: "http://example.com/error",
        TrackingEvents: map[string][]string{
            "start":    {"http://example.com/start"},
            "complete": {"http://example.com/complete"},
        },
    }

    vastDoc, err := vast.NewWrapperBuilder("4.2").
        AddWrapperAd(config).
        Build()

    if err != nil {
        panic(err)
    }

    // Use vastDoc...
}
```

### Injecting Tracking URLs

```go
package main

import (
    "github.com/prebid/prebid-server/v3/vast"
)

func main() {
    // Parse existing VAST
    xmlData := `<VAST version="4.2">...</VAST>`

    // Create tracking injector
    injector, err := vast.NewTrackingInjectorFromXML(xmlData)
    if err != nil {
        panic(err)
    }

    // Chain multiple tracking injections
    modifiedXML, err := injector.
        InjectImpressions([]string{"http://example.com/extra-impression"}).
        InjectVideoEvent("start", []string{"http://example.com/start-tracking"}).
        InjectVideoEvent("complete", []string{"http://example.com/complete-tracking"}).
        InjectError("http://example.com/error-tracking").
        InjectClickTracking([]string{"http://example.com/click-tracking"}).
        ToXML()

    if err != nil {
        panic(err)
    }

    // Use modifiedXML...
}
```

### Extracting Tracking URLs

```go
package main

import (
    "fmt"
    "github.com/prebid/prebid-server/v3/vast"
)

func main() {
    vastDoc, _ := vast.Parse(xmlData)

    // Get all impression URLs
    impressions := vastDoc.GetImpressionURLs()
    fmt.Printf("Impressions: %v\n", impressions)

    // Get specific tracking events
    startEvents := vastDoc.GetTrackingEvents("start")
    fmt.Printf("Start events: %v\n", startEvents)

    completeEvents := vastDoc.GetTrackingEvents("complete")
    fmt.Printf("Complete events: %v\n", completeEvents)
}
```

### Validating VAST Documents

```go
package main

import (
    "fmt"
    "github.com/prebid/prebid-server/v3/vast"
)

func main() {
    vastDoc, err := vast.Parse(xmlData)
    if err != nil {
        panic(err)
    }

    // Validate the VAST document
    if err := vastDoc.Validate(); err != nil {
        fmt.Printf("VAST validation failed: %v\n", err)
        return
    }

    fmt.Println("VAST document is valid!")
}
```

## API Reference

### Core Functions

#### `Parse(xmlData string) (*VAST, error)`
Parses a VAST XML string into a structured VAST object. Supports versions 2.0, 3.0, 4.0, 4.1, and 4.2.

#### `(*VAST) Marshal() (string, error)`
Converts a VAST object back to XML string with proper formatting.

#### `(*VAST) Validate() error`
Validates the VAST document against specification requirements.

### Wrapper Generation

#### `NewDefaultWrapper(adSystem, vastTagURI string, impressionURLs []string) (*VAST, error)`
Creates a simple VAST 4.2 wrapper with basic fields.

#### `NewDefaultWrapperXML(adSystem, vastTagURI string, impressionURLs []string) (string, error)`
Creates a VAST 4.2 wrapper and returns it as XML string.

#### `NewWrapperBuilder(version string) *WrapperBuilder`
Creates a new wrapper builder for constructing complex wrapper ads.

### Tracking Injection

#### `(*VAST) AddImpressionTracking(trackingURLs []string) error`
Adds impression tracking URLs to all ads in the VAST document.

#### `(*VAST) AddVideoEventTracking(event string, trackingURLs []string) error`
Adds video event tracking URLs to linear creatives.

Supported events:
- `start`, `firstQuartile`, `midpoint`, `thirdQuartile`, `complete`
- `mute`, `unmute`, `pause`, `resume`, `rewind`
- `fullscreen`, `exitFullscreen`, `expand`, `collapse`
- `skip`, `progress`, `close`, `acceptInvitation`

#### `(*VAST) AddErrorTracking(errorURL string) error`
Adds error tracking URL to all ads.

#### `(*VAST) AddClickTracking(trackingURLs []string) error`
Adds click tracking URLs to linear creatives.

### Tracking Extraction

#### `(*VAST) GetImpressionURLs() []string`
Extracts all impression URLs from the VAST document.

#### `(*VAST) GetTrackingEvents(event string) []string`
Extracts all tracking URLs for a specific event type.

### TrackingInjector

#### `NewTrackingInjector(vast *VAST) *TrackingInjector`
Creates a tracking injector from a VAST object.

#### `NewTrackingInjectorFromXML(xmlData string) (*TrackingInjector, error)`
Creates a tracking injector from VAST XML string.

#### `(*TrackingInjector) InjectImpressions(urls []string) *TrackingInjector`
Injects impression tracking URLs (chainable).

#### `(*TrackingInjector) InjectVideoEvent(event string, urls []string) *TrackingInjector`
Injects video event tracking URLs (chainable).

#### `(*TrackingInjector) InjectError(url string) *TrackingInjector`
Injects error tracking URL (chainable).

#### `(*TrackingInjector) InjectClickTracking(urls []string) *TrackingInjector`
Injects click tracking URLs (chainable).

#### `(*TrackingInjector) ToXML() (string, error)`
Returns the modified VAST as XML string.

## Data Structures

### Main Types

- `VAST` - Root VAST document
- `Ad` - Individual ad (contains Wrapper or InLine)
- `Wrapper` - Wrapper ad that redirects to another VAST
- `InLine` - Direct ad with creative assets
- `Creative` - Creative element (Linear, NonLinear, or Companion)
- `Linear` - Linear video creative
- `MediaFile` - Video file specification
- `Tracking` - Tracking event URL
- `Impression` - Impression tracking URL

See [vast.go](./vast.go) for complete type definitions.

## Testing

Run the test suite:

```bash
go test -v ./vast/
```

Test coverage:

```bash
go test -cover ./vast/
```

## Standards Compliance

This package implements the IAB Tech Lab VAST specifications:

- **VAST 2.0** - Initial video ad serving standard
- **VAST 3.0** - Added support for verification, icons, and universal ad IDs
- **VAST 4.0** - Enhanced programmatic support
- **VAST 4.1** - Improved verification and measurement
- **VAST 4.2** - Latest stable version with additional tracking events

## References

- [VAST 4.2 Specification (PDF)](https://iabtechlab.com/wp-content/uploads/2019/06/VAST_4.2_final_june26.pdf)
- [IAB Tech Lab VAST Documentation](https://iabtechlab.com/standards/vast/)
- [GitHub - VAST Specification](https://github.com/InteractiveAdvertisingBureau/vast)

## Integration with Prebid Server

This package is designed to replace the existing string-based VAST handling in Prebid Server:

- **Current**: `/prebid-server/exchange/auction.go` - `makeVAST()` function
- **Current**: `/prebid-server/endpoints/events/vtrack.go` - `ModifyVastXmlString()` function

### Migration Example

**Before** (string-based):
```go
func makeVAST(bid *openrtb2.Bid) string {
    if bid.AdM == "" {
        return `<VAST version="3.0"><Ad><Wrapper>...`
    }
    return bid.AdM
}
```

**After** (structured):
```go
import "github.com/prebid/prebid-server/v3/vast"

func makeVAST(bid *openrtb2.Bid) (string, error) {
    if bid.AdM == "" {
        return vast.NewDefaultWrapperXML(
            "PrebidServer",
            bid.NURL,
            []string{/* impression URLs */},
        )
    }
    return bid.AdM, nil
}
```

## License

This package is part of Prebid Server and follows the same license.
