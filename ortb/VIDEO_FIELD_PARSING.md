# OpenRTB Video Object Parsing Implementation

## Overview

This document describes the comprehensive video field parsing and validation implementation for Prebid Server's OpenRTB request handling.

## Implementation Location

- **Main Validator**: `/projects/prebid-server/ortb/request_validator_video.go`
- **Tests**: `/projects/prebid-server/ortb/request_validator_video_extended_test.go`
- **Helper Functions**: `/projects/prebid-server/ortb/video_helper.go`

## Validated Video Fields

### Core Fields

#### 1. MIMEs (Required)
- **Validation**: Must contain at least one MIME type
- **Error**: `request.imp[%d].video.mimes must contain at least one supported MIME type`
- **Common Values**:
  - `video/mp4`
  - `video/webm`
  - `application/javascript` (for VPAID)

#### 2. Duration Constraints
- **MinDuration**: Must be non-negative
- **MaxDuration**: Must be non-negative
- **Constraint**: MinDuration must not exceed MaxDuration
- **Errors**:
  - `request.imp[%d].video.minduration must be a non-negative number`
  - `request.imp[%d].video.maxduration must be a non-negative number`
  - `request.imp[%d].video.minduration must not exceed maxduration`

#### 3. Protocols
- **Validation**: Values must be between 1-12 (per OpenRTB 2.5 spec)
- **Protocol Values**:
  - 1 = VAST 1.0
  - 2 = VAST 2.0
  - 3 = VAST 3.0
  - 4 = VAST 1.0 Wrapper
  - 5 = VAST 2.0 Wrapper
  - 6 = VAST 3.0 Wrapper
  - 7 = VAST 4.0
  - 8 = DAAST 1.0
  - 9 = DAAST 1.0 Wrapper
  - 10 = VAST 4.1
  - 11 = VAST 4.2
  - 12 = VAST 4.2 Wrapper
- **Error**: `request.imp[%d].video.protocols contains invalid value %d (must be 1-12)`

#### 4. Start Delay
- **Validation**: Must be >= -2 (per OpenRTB spec)
- **Values**:
  - -2 = Post-roll
  - -1 = Pre-roll
  - 0 = Generic mid-roll
  - \>0 = Mid-roll at specific offset (seconds)
- **Error**: `request.imp[%d].video.startdelay must be >= -2 (per OpenRTB spec)`

#### 5. Skip Settings
- **Skip**: 0 = no, 1 = yes
- **SkipMin**: Minimum video duration (in seconds) before skip is enabled (only validated if Skip=1)
- **SkipAfter**: Number of seconds after which skip becomes enabled (only validated if Skip=1)
- **Validation**: Both must be non-negative when skip is enabled
- **Errors**:
  - `request.imp[%d].video.skipmin must be a non-negative number`
  - `request.imp[%d].video.skipafter must be a non-negative number`

### Additional Validated Fields

#### 6. Dimensions
- **W**: Width (must be non-negative)
- **H**: Height (must be non-negative)

#### 7. Bitrate
- **MinBitRate**: Minimum bitrate (must be non-negative)
- **MaxBitRate**: Maximum bitrate (must be non-negative)

#### 8. Placement
- **Validation**: Must be between 1-7
- **Values**:
  - 1 = In-Stream
  - 2 = In-Banner
  - 3 = In-Article
  - 4 = In-Feed
  - 5 = Interstitial/Slider/Floating
  - 6 = Standalone
  - 7 = Contextual Skin

#### 9. Playback Method
- **Validation**: Values must be between 1-7
- **Values**:
  - 1 = Auto-play sound on
  - 2 = Auto-play sound off
  - 3 = Click-to-play
  - 4 = Mouse-over
  - 5 = Entering viewport with sound on
  - 6 = Entering viewport with sound off
  - 7 = Continuous play

#### 10. Delivery Methods
- **Validation**: Values must be between 1-3
- **Values**:
  - 1 = Streaming
  - 2 = Progressive
  - 3 = Download

#### 11. API Frameworks
- **Validation**: Values must be between 1-8
- **Values**:
  - 1 = VPAID 1.0
  - 2 = VPAID 2.0
  - 3 = MRAID-1
  - 4 = ORMMA
  - 5 = MRAID-2
  - 6 = MRAID-3
  - 7 = OMID-1
  - 8 = SIMID-1

#### 12. Sequence
- **Validation**: Must be non-negative
- **Usage**: For ad pods (multiple ads in sequence)

## Test Coverage

### Test Cases Implemented

1. **Parse video object fields** - Comprehensive parsing of all video fields
2. **Validate duration constraints** - MinDuration/MaxDuration validation
3. **Handle protocol negotiation** - Multiple protocol support and validation
4. **Start delay validation** - Pre-roll, mid-roll, post-roll scenarios
5. **Skip settings validation** - Complete skip parameter validation
6. **Comprehensive CTV example** - Real-world Connected TV scenario

### Test Files

- `TestValidateVideo_Duration` - Duration constraint tests
- `TestValidateVideo_Protocols` - Protocol validation tests
- `TestValidateVideo_SkipParameters` - Skip settings tests
- `TestValidateVideo_StartDelay` - Start delay tests
- `TestValidateVideo_Placement` - Placement validation tests
- `TestValidateVideo_PlaybackMethod` - Playback method tests
- `TestValidateVideo_Delivery` - Delivery method tests
- `TestValidateVideo_APIs` - API framework tests
- `TestValidateVideo_ComprehensiveCTVExample` - Full integration test
- `TestValidateVideo_ComprehensiveFieldParsing` - All fields parsing test

## Usage Example

```go
import (
    "github.com/prebid/openrtb/v20/openrtb2"
    "github.com/prebid/prebid-server/v3/ortb"
)

// Create a video impression
video := &openrtb2.Video{
    MIMEs:          []string{"video/mp4", "video/webm"},
    MinDuration:    15,
    MaxDuration:    30,
    Protocols:      []openrtb2.Protocol{2, 3, 7}, // VAST 2.0, 3.0, 4.0
    StartDelay:     ptrutil.ToPtr[int64](-1),      // Pre-roll
    Skip:           ptrutil.ToPtr[int8](1),
    SkipMin:        ptrutil.ToPtr[int64](5),
    SkipAfter:      ptrutil.ToPtr[int64](5),
}

// Validate (called internally by request validator)
err := validateVideo(video, 0)
if err != nil {
    // Handle validation error
}
```

## Integration Points

The video validation is integrated into the main request validator and is called automatically when processing OpenRTB bid requests with video impressions.

### Related Components

- **VideoHelper** (`ortb/video_helper.go`): Helper functions for video field interpretation
- **VideoFields** (`stored_requests/video_fields.go`): Stored request video field handling
- **Event Tracking** (`endpoints/events/event.go`): Video event tracking for analytics

## OpenRTB Specification Compliance

This implementation follows the OpenRTB 2.5+ specification for video object validation, ensuring compatibility with industry-standard RTB protocols.

### Key Compliance Points

- Protocol values align with VAST/DAAST specifications
- Start delay values follow OpenRTB conventions
- API framework values match industry standards (VPAID, MRAID, OMID, SIMID)
- Placement types cover all standard video ad positions

## Future Enhancements

Potential areas for extension:

1. Additional MIME type validation (specific format checking)
2. Cross-field validation (e.g., certain APIs require specific protocols)
3. Enhanced protocol negotiation logic
4. Support for OpenRTB 3.0 video extensions
