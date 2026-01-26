# VAST Tag Generation Test Cases Specification

## Overview
This document defines comprehensive test cases for VAST XML generation functionality. These test cases ensure compliance with VAST 2.0, 3.0, and 4.x specifications.

## Test Categories

### 1. Basic VAST Structure Tests

#### TC-VG-001: Generate Empty VAST Response
**Objective**: Verify empty VAST generation for no-bid scenarios
**Input**: No auction winner
**Expected Output**:
```xml
<?xml version="1.0" encoding="UTF-8"?>
<VAST version="4.0">
</VAST>
```
**Validation**:
- Version attribute is "4.0"
- No Ad elements present
- Valid XML structure

#### TC-VG-002: Generate VAST 2.0 Document
**Objective**: Verify VAST 2.0 format compliance
**Input**: Builder with version="2.0", single inline ad
**Expected Output**: VAST with version="2.0" attribute
**Validation**:
- Version attribute matches requested version
- Only VAST 2.0 compatible elements present
- No VAST 3.0/4.0 specific fields

#### TC-VG-003: Generate VAST 3.0 Document
**Objective**: Verify VAST 3.0 format compliance
**Input**: Builder with version="3.0"
**Expected Output**: VAST with version="3.0" attribute
**Validation**:
- Supports UniversalAdId element
- Supports AdVerifications
- Backward compatible with 2.0 elements

#### TC-VG-004: Generate VAST 4.0 Document
**Objective**: Verify VAST 4.x format compliance (default)
**Input**: Builder with version="4.0" or no version
**Expected Output**: VAST with version="4.0" attribute
**Validation**:
- All VAST 4.0 elements supported
- ViewableImpression support
- Interactive creative extensions

### 2. Inline VAST Tests

#### TC-VG-010: Basic Inline Ad
**Objective**: Generate minimal valid inline VAST ad
**Input**:
- Ad ID: "test-ad-001"
- AdSystem: "TNEVideo"
- AdTitle: "Test Video Ad"
- Impression URL: "https://tracking.example.com/imp?id=123"
- Creative: Linear with 30s duration
- Media file: 1920x1080 MP4
**Expected Output**: Complete inline VAST with all required fields
**Validation**:
- `<InLine>` element present
- `<AdSystem>` = "TNEVideo"
- `<AdTitle>` = "Test Video Ad"
- `<Impression>` URL present
- `<Linear>` creative present
- Duration = "00:00:30"
- MediaFile attributes: width=1920, height=1080, type="video/mp4"

#### TC-VG-011: Multiple Impressions
**Objective**: Support multiple impression tracking URLs
**Input**: 3 different impression URLs with IDs
**Expected Output**: Multiple `<Impression>` elements
**Validation**:
- All impression URLs present in order
- Each has unique ID attribute
- CDATA wrapping for URLs

#### TC-VG-012: AdSystem with Version
**Objective**: Include AdSystem version attribute
**Input**: AdSystem="TNEVideo", version="1.0.0"
**Expected Output**: `<AdSystem version="1.0.0">TNEVideo</AdSystem>`
**Validation**:
- Version attribute present
- Value correct

#### TC-VG-013: Optional Fields (Description, Advertiser, Pricing)
**Objective**: Include optional InLine fields
**Input**:
- Description: "Premium video ad for brand awareness"
- Advertiser: "Example Brand Inc"
- Pricing: model="CPM", currency="USD", value="5.00"
**Expected Output**: All optional fields in VAST
**Validation**:
- Description element present
- Advertiser element present
- Pricing with all attributes

### 3. Wrapper VAST Tests

#### TC-VG-020: Basic Wrapper Ad
**Objective**: Generate VAST wrapper pointing to demand partner
**Input**:
- VASTAdTagURI: "https://demand-partner.com/vast?id=abc123"
- AdSystem: "TNEVideo"
- Impression URL for wrapper
**Expected Output**: Wrapper VAST with redirect
**Validation**:
- `<Wrapper>` element present (not InLine)
- `<VASTAdTagURI>` with CDATA wrapping
- Impression tracking at wrapper level

#### TC-VG-021: Wrapper with Additional Parameters
**Objective**: Test wrapper-specific attributes
**Input**:
- followAdditionalWrappers: true
- allowMultipleAds: false
- fallbackOnNoAd: true
**Expected Output**: Wrapper with boolean attributes
**Validation**:
- followAdditionalWrappers="true"
- allowMultipleAds="false"
- fallbackOnNoAd="true"

### 4. Linear Creative Tests

#### TC-VG-030: Linear Creative with Duration Formats
**Objective**: Test various duration formats
**Input**: Durations of 15s, 30s, 60s, 90s, 2m30s, 1h
**Expected Output**: Properly formatted HH:MM:SS durations
**Validation**:
- 15s → "00:00:15"
- 30s → "00:00:30"
- 60s → "00:01:00"
- 90s → "00:01:30"
- 150s → "00:02:30"
- 3600s → "01:00:00"

#### TC-VG-031: Multiple Media Files
**Objective**: Support multiple video formats/bitrates
**Input**:
- MP4 1920x1080 @5000 kbps
- MP4 1280x720 @2500 kbps
- WebM 1920x1080 @4000 kbps
**Expected Output**: 3 MediaFile elements
**Validation**:
- Each with correct dimensions
- Bitrate attributes set
- Type attributes (video/mp4, video/webm)
- Delivery="progressive" (default)

#### TC-VG-032: Media File with All Attributes
**Objective**: Test complete MediaFile specification
**Input**:
- All MediaFile attributes: id, delivery, type, bitrate, width, height, codec, scalable, maintainAspectRatio, apiFramework
**Expected Output**: Fully specified MediaFile
**Validation**:
- delivery="progressive" or "streaming"
- codec="h264" or "vp9"
- scalable=true/false
- maintainAspectRatio=true/false

#### TC-VG-033: Skippable Ad with Skip Offset
**Objective**: Generate skippable video ad
**Input**:
- Skip offset: 5 seconds
- Duration: 30 seconds
**Expected Output**: Linear with skipoffset="00:00:05"
**Validation**:
- skipoffset attribute on `<Linear>`
- Value in HH:MM:SS format

### 5. Tracking Events Tests

#### TC-VG-040: Standard Quartile Tracking
**Objective**: Generate all standard tracking events
**Input**: Base tracking URL with event parameter
**Expected Output**: Tracking events for:
- start
- firstQuartile
- midpoint
- thirdQuartile
- complete
**Validation**:
- All 5 events present
- URLs properly formatted with event parameter
- CDATA wrapping

#### TC-VG-041: Extended Tracking Events
**Objective**: Support all VAST tracking events
**Input**: URLs for extended events
**Expected Output**: Tracking for:
- creativeView, start, firstQuartile, midpoint, thirdQuartile, complete
- mute, unmute, pause, resume, rewind
- fullscreen, exitFullscreen, expand, collapse
- acceptInvitation, close, skip
**Validation**:
- All event types supported
- Event attribute matches constant

#### TC-VG-042: Progress Tracking with Offset
**Objective**: Track at specific time offsets
**Input**:
- Event: "progress", offset: "00:00:10"
- Event: "progress", offset: "00:00:20"
**Expected Output**: Progress events with offset attributes
**Validation**:
- offset attribute present
- Proper time format

#### TC-VG-043: Click Tracking
**Objective**: Generate click tracking elements
**Input**:
- ClickThrough: "https://example.com/landing"
- ClickTracking: "https://tracking.example.com/click"
- CustomClick: "https://example.com/custom"
**Expected Output**: VideoClicks element with all children
**Validation**:
- ClickThrough present with URL
- ClickTracking array present
- CustomClick present

### 6. Error Handling Tests

#### TC-VG-050: Error Tracking URL
**Objective**: Include error tracking in VAST
**Input**: Error URL at VAST and Ad level
**Expected Output**: Error elements at both levels
**Validation**:
- Root VAST Error element
- InLine/Wrapper Error element
- CDATA wrapping

#### TC-VG-051: Error VAST Response
**Objective**: Generate error-only VAST (no ads)
**Input**: Error URL "https://tracking.example.com/error?code=204"
**Expected Output**: Empty VAST with error URL
**Validation**:
- No Ad elements
- Error element at root level
- Valid XML

### 7. Companion Ads Tests

#### TC-VG-060: Static Companion Ad
**Objective**: Include companion banner with video
**Input**:
- Companion: 300x250 static image
- Resource URL: "https://cdn.example.com/banner.jpg"
- ClickThrough URL
**Expected Output**: CompanionAds element with Companion
**Validation**:
- Width=300, height=250
- StaticResource with creativeType="image/jpeg"
- CompanionClickThrough present

#### TC-VG-061: HTML Companion Ad
**Objective**: HTML resource companion
**Input**: HTML snippet with xmlEncoded=true
**Expected Output**: Companion with HTMLResource
**Validation**:
- HTMLResource element
- xmlEncoded attribute
- CDATA wrapping for HTML

#### TC-VG-062: Multiple Companions
**Objective**: Support multiple companion sizes
**Input**: 300x250, 728x90, 160x600 companions
**Expected Output**: 3 Companion elements
**Validation**:
- required="all" or "any" or "none"
- Each with unique dimensions

### 8. Extensions Tests

#### TC-VG-070: Ad Extensions
**Objective**: Include custom extensions
**Input**: Custom extension with type attribute
**Expected Output**: Extensions element with Extension child
**Validation**:
- type attribute present
- Inner XML preserved

#### TC-VG-071: Creative Extensions
**Objective**: Creative-level extensions
**Input**: VPAID or SIMID extension data
**Expected Output**: CreativeExtensions element
**Validation**:
- type attribute
- Custom XML data preserved

### 9. Multiple Ads (Ad Pods) Tests

#### TC-VG-080: Sequential Ads with Sequence
**Objective**: Generate ad pod with sequence numbers
**Input**:
- Ad 1: sequence=1, duration=15s
- Ad 2: sequence=2, duration=30s
- Ad 3: sequence=3, duration=15s
**Expected Output**: 3 Ad elements with sequence attributes
**Validation**:
- sequence attributes: 1, 2, 3
- All ads present in order

#### TC-VG-081: Mixed Inline and Wrapper Ads
**Objective**: Support pod with mixed ad types
**Input**: Inline ad + Wrapper ad
**Expected Output**: 2 Ad elements with different types
**Validation**:
- First has InLine
- Second has Wrapper
- Valid VAST structure

### 10. XML Output Quality Tests

#### TC-VG-090: XML Declaration
**Objective**: Ensure proper XML header
**Expected Output**: `<?xml version="1.0" encoding="UTF-8"?>`
**Validation**:
- XML declaration present
- UTF-8 encoding specified

#### TC-VG-091: CDATA Wrapping for URLs
**Objective**: Properly escape URLs
**Input**: URLs with special characters (&, <, >, ", ')
**Expected Output**: CDATA sections for all URLs
**Validation**:
- All URLs wrapped in `<![CDATA[...]]>`
- Special characters preserved

#### TC-VG-092: Indented XML Output
**Objective**: Human-readable formatting
**Expected Output**: Properly indented XML (2 spaces)
**Validation**:
- Consistent indentation
- No extra blank lines
- Valid structure

#### TC-VG-093: Round-trip Parsing
**Objective**: Generated VAST can be parsed back
**Input**: Any generated VAST
**Expected Output**: Parse → Generate → Parse produces same structure
**Validation**:
- Parse(Marshal(vast)) == vast
- No data loss

### 11. Macro Replacement Tests

#### TC-VG-100: Auction Price Macro
**Objective**: Support ${AUCTION_PRICE} macro
**Input**: URL with macro: "https://tracking.example.com?price=${AUCTION_PRICE}"
**Expected Output**: Macro preserved in output
**Validation**:
- Macro not expanded in generation
- Properly escaped in CDATA

#### TC-VG-101: Standard VAST Macros
**Objective**: Support all standard macros
**Input**: URLs with [TIMESTAMP], [CACHEBUSTING], [CLICKURL], [ERRORCODE]
**Expected Output**: Macros preserved
**Validation**:
- All macros intact
- Proper CDATA wrapping

### 12. Performance Tests

#### TC-VG-110: Large VAST Generation
**Objective**: Handle complex VAST documents
**Input**:
- 5 ads in sequence
- Each with 3 media files
- 15 tracking events per ad
- 3 companions per ad
**Expected Output**: Complete VAST in < 5ms
**Validation**:
- Generation time < 5ms
- Valid output
- No memory leaks

#### TC-VG-111: Concurrent Generation
**Objective**: Thread-safe generation
**Input**: 100 concurrent VAST builds
**Expected Output**: All succeed without race conditions
**Validation**:
- No race conditions (go test -race)
- All outputs valid
- Consistent performance

## Test Implementation Notes

### Go Testing Framework
- Use `testing` package
- Use `testify/assert` for assertions
- Group tests with subtests (`t.Run()`)
- Test file: `tnevideo/pkg/vast/generation_test.go`

### Test Fixtures
- Sample bid requests in `tests/fixtures/video_bid_requests/`
- Expected VAST outputs in `tests/fixtures/vast_responses/`
- Use golden file pattern for snapshot testing

### Coverage Requirements
- Minimum 90% code coverage for generation logic
- All error paths tested
- All optional fields tested

### Validation Tools
- Use VAST XML Schema (XSD) validation
- IAB VAST validator (if available)
- Custom validation functions for business rules

## Success Criteria
- All test cases pass
- 90%+ code coverage
- Performance benchmarks met
- No race conditions
- XSD validation passes
