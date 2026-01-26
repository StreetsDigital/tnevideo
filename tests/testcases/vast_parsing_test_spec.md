# VAST Tag Reception/Parsing Test Cases Specification

## Overview
This document defines comprehensive test cases for parsing and processing incoming VAST XML from demand partners. These tests ensure robust handling of VAST 2.0, 3.0, and 4.x documents.

## Test Categories

### 1. Basic Parsing Tests

#### TC-VP-001: Parse Valid VAST 2.0
**Objective**: Successfully parse VAST 2.0 XML
**Input**: Valid VAST 2.0 XML document
**Expected Output**: Populated VAST struct
**Validation**:
- Version == "2.0"
- All fields correctly populated
- No parsing errors

#### TC-VP-002: Parse Valid VAST 3.0
**Objective**: Successfully parse VAST 3.0 XML
**Input**: Valid VAST 3.0 with UniversalAdId
**Expected Output**: VAST struct with 3.0 features
**Validation**:
- UniversalAdId parsed correctly
- AdVerifications if present
- Backward compatible elements

#### TC-VP-003: Parse Valid VAST 4.0
**Objective**: Successfully parse VAST 4.0 XML
**Input**: Valid VAST 4.0 with latest features
**Expected Output**: Complete VAST struct
**Validation**:
- All 4.0 features parsed
- ViewableImpression elements
- Extensions parsed correctly

#### TC-VP-004: Parse Empty VAST (No Bid)
**Objective**: Handle no-bid response
**Input**:
```xml
<?xml version="1.0"?>
<VAST version="4.0"></VAST>
```
**Expected Output**: Empty VAST with no ads
**Validation**:
- IsEmpty() returns true
- No errors
- Version parsed correctly

### 2. InLine Ad Parsing Tests

#### TC-VP-010: Parse Basic InLine Ad
**Objective**: Extract all required InLine fields
**Input**: Complete inline VAST with linear creative
**Expected Output**: Populated InLine struct
**Validation**:
- AdSystem.Value extracted
- AdTitle extracted
- Impressions array populated
- Creatives array populated
- Linear creative present

#### TC-VP-011: Parse Multiple Impressions
**Objective**: Handle multiple impression URLs
**Input**: InLine with 3 impression URLs
**Expected Output**: 3 Impression structs in array
**Validation**:
- All URLs extracted
- ID attributes preserved
- Order maintained

#### TC-VP-012: Parse Optional InLine Fields
**Objective**: Extract Description, Advertiser, Pricing
**Input**: InLine with all optional fields
**Expected Output**: All optional fields populated
**Validation**:
- Description != ""
- Advertiser != ""
- Pricing fields populated (model, currency, value)

#### TC-VP-013: Parse Error URL
**Objective**: Extract error tracking URL
**Input**: InLine with Error element
**Expected Output**: InLine.Error contains URL
**Validation**:
- Error URL extracted from CDATA
- Special characters preserved

### 3. Wrapper Ad Parsing Tests

#### TC-VP-020: Parse Basic Wrapper
**Objective**: Extract wrapper redirect URL
**Input**: Wrapper VAST pointing to demand partner
**Expected Output**: Wrapper struct with VASTAdTagURI
**Validation**:
- VASTAdTagURI extracted from CDATA
- AdSystem extracted
- Impressions extracted

#### TC-VP-021: Parse Wrapper Attributes
**Objective**: Extract wrapper-specific boolean attributes
**Input**: Wrapper with followAdditionalWrappers, allowMultipleAds, fallbackOnNoAd
**Expected Output**: Boolean attributes set correctly
**Validation**:
- followAdditionalWrappers == true/false
- allowMultipleAds == true/false
- fallbackOnNoAd == true/false

#### TC-VP-022: Unwrap Single-Level Wrapper
**Objective**: Follow wrapper to final VAST
**Input**: Wrapper → InLine
**Expected Output**: Final linear creative extracted
**Validation**:
- Wrapper detected
- VASTAdTagURI fetched (mock HTTP)
- Final creative extracted
- Impressions combined from both levels

#### TC-VP-023: Unwrap Multi-Level Wrapper
**Objective**: Handle wrapper chains
**Input**: Wrapper → Wrapper → InLine (2 levels deep)
**Expected Output**: Final creative after 2 unwraps
**Validation**:
- Maximum 5 wrapper depth enforced
- All impressions aggregated
- Final creative correct

#### TC-VP-024: Wrapper Chain Depth Limit
**Objective**: Prevent infinite wrapper loops
**Input**: 10 nested wrappers
**Expected Output**: Error after 5 levels
**Validation**:
- Error: "maximum wrapper depth exceeded"
- No infinite loop
- Partial data up to depth 5

### 4. Linear Creative Parsing Tests

#### TC-VP-030: Parse Linear Creative
**Objective**: Extract linear video creative
**Input**: Linear element with all fields
**Expected Output**: Linear struct populated
**Validation**:
- Duration parsed (time.Duration)
- MediaFiles array populated
- TrackingEvents array populated
- VideoClicks extracted

#### TC-VP-031: Parse Duration
**Objective**: Convert HH:MM:SS to time.Duration
**Input**: Various duration formats
**Test Cases**:
- "00:00:15" → 15 * time.Second
- "00:01:30" → 90 * time.Second
- "01:00:00" → 1 * time.Hour
**Validation**:
- Correct conversion
- ParseDuration() function

#### TC-VP-032: Parse MediaFiles
**Objective**: Extract all media file variants
**Input**: 3 MediaFile elements (different bitrates/formats)
**Expected Output**: 3 MediaFile structs
**Validation**:
- All URLs extracted from CDATA
- Type (mime) extracted
- Width, height parsed as int
- Bitrate parsed as int
- delivery attribute
- codec attribute if present

#### TC-VP-033: Select Best MediaFile
**Objective**: Choose optimal media file based on criteria
**Input**: Multiple media files with different specs
**Selection Logic**:
- Prefer requested format (mp4 > webm)
- Match dimensions or closest
- Prefer higher bitrate within limits
**Expected Output**: Single optimal MediaFile
**Validation**:
- Selection algorithm correct
- Criteria applied in order

#### TC-VP-034: Parse Skip Offset
**Objective**: Extract skipoffset for skippable ads
**Input**: Linear with skipoffset="00:00:05"
**Expected Output**: SkipOffset = "00:00:05"
**Validation**:
- Attribute extracted
- Time format validated

### 5. Tracking Events Parsing Tests

#### TC-VP-040: Parse Standard Tracking Events
**Objective**: Extract quartile events
**Input**: Tracking events for start, firstQuartile, midpoint, thirdQuartile, complete
**Expected Output**: 5 Tracking structs
**Validation**:
- event attributes match constants
- URLs extracted from CDATA
- All events present

#### TC-VP-041: Parse Extended Tracking Events
**Objective**: Handle all event types
**Input**: 20+ different tracking events
**Expected Output**: All events parsed
**Validation**:
- mute, unmute, pause, resume events
- fullscreen, expand events
- skip, close events
- All URLs extracted

#### TC-VP-042: Parse Progress Events with Offset
**Objective**: Handle offset-based progress tracking
**Input**: progress events with offset="00:00:10"
**Expected Output**: Tracking with Offset field populated
**Validation**:
- Offset attribute extracted
- Time format validated

#### TC-VP-043: Extract Tracking URLs by Type
**Objective**: Provide helper to get URLs by event type
**Input**: Parsed VAST with multiple tracking events
**Method**: GetTrackingURLs(EventStart) []string
**Expected Output**: Array of start event URLs only
**Validation**:
- Filter by event type works
- Returns all matching URLs
- Empty array if none found

### 6. Video Clicks Parsing Tests

#### TC-VP-050: Parse ClickThrough
**Objective**: Extract landing page URL
**Input**: ClickThrough element with URL
**Expected Output**: ClickThrough.Value contains URL
**Validation**:
- URL extracted from CDATA
- ID attribute if present

#### TC-VP-051: Parse ClickTracking
**Objective**: Extract click tracking URLs
**Input**: Multiple ClickTracking elements
**Expected Output**: Array of ClickTracking structs
**Validation**:
- All URLs extracted
- IDs preserved
- Order maintained

#### TC-VP-052: Parse CustomClick
**Objective**: Handle custom click events
**Input**: CustomClick elements
**Expected Output**: Array of CustomClick structs
**Validation**:
- All extracted
- IDs and URLs correct

### 7. Companion Ads Parsing Tests

#### TC-VP-060: Parse Static Companion
**Objective**: Extract companion banner
**Input**: Companion with StaticResource
**Expected Output**: Companion struct with resource
**Validation**:
- Width, height parsed as int
- StaticResource.Value (URL) extracted
- creativeType attribute
- CompanionClickThrough if present

#### TC-VP-061: Parse HTML Companion
**Objective**: Extract HTML resource
**Input**: Companion with HTMLResource
**Expected Output**: HTMLResource with HTML content
**Validation**:
- xmlEncoded attribute
- HTML extracted from CDATA
- Special characters preserved

#### TC-VP-062: Parse IFrame Companion
**Objective**: Handle IFrame resource type
**Input**: Companion with IFrameResource URL
**Expected Output**: IFrameResource contains URL
**Validation**:
- URL extracted
- Width/height from Companion

#### TC-VP-063: Parse required Attribute
**Objective**: Handle companion display requirements
**Input**: CompanionAds with required="all|any|none"
**Expected Output**: Required field set correctly
**Validation**:
- required attribute extracted
- Valid values only

### 8. Extensions Parsing Tests

#### TC-VP-070: Parse Ad Extensions
**Objective**: Extract custom extension data
**Input**: Extensions element with custom XML
**Expected Output**: Extension struct with innerxml
**Validation**:
- type attribute extracted
- Inner XML preserved as string
- Multiple extensions supported

#### TC-VP-071: Parse Creative Extensions
**Objective**: Handle creative-level extensions
**Input**: CreativeExtensions with VPAID data
**Expected Output**: CreativeExtension array
**Validation**:
- All extensions extracted
- type attributes
- XML data intact

### 9. Error Handling Tests

#### TC-VP-080: Invalid XML Format
**Objective**: Handle malformed XML gracefully
**Input**: XML with syntax errors
**Expected Output**: Parse error with descriptive message
**Validation**:
- Error returned (not panic)
- Error message indicates XML issue
- No partial data returned

#### TC-VP-081: Missing Required Fields
**Objective**: Validate required VAST elements
**Input**: InLine without AdTitle
**Expected Output**: Validation error
**Validation**:
- Error: "required field AdTitle missing"
- No silent failures

#### TC-VP-082: Invalid Duration Format
**Objective**: Handle malformed duration
**Input**: Duration = "invalid"
**Expected Output**: ParseDuration error
**Validation**:
- Error returned
- No panic
- Clear error message

#### TC-VP-083: Invalid URL in CDATA
**Objective**: Handle URLs with XML special characters
**Input**: URL with &, <, > without CDATA
**Expected Output**: URL correctly extracted or validation error
**Validation**:
- Either escapes handled or error raised
- No silent data corruption

#### TC-VP-084: Empty Ad Response
**Objective**: Handle VAST with no ads
**Input**: `<VAST version="4.0"></VAST>`
**Expected Output**: Empty VAST, no error
**Validation**:
- IsEmpty() == true
- No error
- Valid VAST struct

#### TC-VP-085: Network Error on Wrapper Fetch
**Objective**: Handle HTTP errors when unwrapping
**Input**: Wrapper with unreachable VASTAdTagURI
**Expected Output**: Error indicating network failure
**Validation**:
- Error message clear
- Timeout handling (5 second max)
- Partial wrapper data available

#### TC-VP-086: Circular Wrapper Reference
**Objective**: Detect wrapper loops
**Input**: Wrapper A → Wrapper B → Wrapper A
**Expected Output**: Error: "circular wrapper reference detected"
**Validation**:
- Loop detection works
- No infinite recursion
- Error after 2nd level

### 10. VAST Validation Tests

#### TC-VP-090: Validate Required InLine Elements
**Objective**: Check InLine has all required fields
**Input**: InLine VAST
**Required**: AdSystem, AdTitle, Impression, Creatives
**Expected Output**: Validation passes or specific field error
**Validation**:
- All required fields checked
- Error messages specific

#### TC-VP-091: Validate Required Linear Elements
**Objective**: Check Linear has required fields
**Input**: Linear creative
**Required**: Duration, MediaFiles (at least one)
**Expected Output**: Validation result
**Validation**:
- Duration present and valid
- At least one MediaFile
- MediaFile has required attrs (delivery, type, width, height, URL)

#### TC-VP-092: Validate MediaFile Attributes
**Objective**: Ensure MediaFile is usable
**Input**: MediaFile element
**Required**: delivery, type, width, height, URL
**Expected Output**: Validation passes
**Validation**:
- type is valid MIME type
- delivery is "progressive" or "streaming"
- width/height > 0
- URL is valid format

#### TC-VP-093: Validate Tracking URLs
**Objective**: Ensure tracking URLs are valid
**Input**: Tracking events with URLs
**Expected Output**: All URLs are valid HTTP(S)
**Validation**:
- URL scheme is http or https
- URL is not empty
- No malformed URLs

#### TC-VP-094: XSD Schema Validation
**Objective**: Validate against official VAST XSD
**Input**: Parsed VAST marshaled back to XML
**Expected Output**: Schema validation passes
**Validation**:
- Use official IAB VAST 4.0 XSD
- No schema violations
- Optional for 2.0/3.0 compatibility

### 11. Special Characters and Encoding Tests

#### TC-VP-100: Parse URLs with Special Characters
**Objective**: Handle &, <, >, ", ' in URLs
**Input**: URLs with query params containing special chars
**Expected Output**: URLs correctly extracted
**Validation**:
- & preserved (not &amp;)
- CDATA unwrapping works
- No data loss

#### TC-VP-101: Parse Unicode Characters
**Objective**: Handle international characters
**Input**: AdTitle with Chinese/Japanese/Arabic text
**Expected Output**: Characters preserved
**Validation**:
- UTF-8 encoding maintained
- Display correctly
- No mojibake

#### TC-VP-102: Parse HTML Entities
**Objective**: Handle &amp;, &lt;, &gt;, &quot;
**Input**: HTML entities in text fields
**Expected Output**: Entities decoded
**Validation**:
- &amp; → &
- &lt; → <
- Proper decoding

### 12. Performance Tests

#### TC-VP-110: Parse Large VAST Document
**Objective**: Handle complex VAST efficiently
**Input**: VAST with 10 ads, 50 media files, 200 tracking events
**Expected Output**: Parsed in < 10ms
**Validation**:
- Parse time < 10ms
- Memory usage reasonable (< 5MB)
- No memory leaks

#### TC-VP-111: Parse Invalid VAST Quickly
**Objective**: Fast-fail on invalid input
**Input**: Malformed XML
**Expected Output**: Error in < 1ms
**Validation**:
- Quick failure
- No expensive operations attempted
- Clear error message

#### TC-VP-112: Concurrent Parsing
**Objective**: Thread-safe parsing
**Input**: 100 concurrent Parse() calls
**Expected Output**: All succeed independently
**Validation**:
- No race conditions (go test -race)
- All results correct
- Consistent performance

### 13. Utility Function Tests

#### TC-VP-120: GetLinearCreative()
**Objective**: Extract first linear creative
**Input**: VAST with multiple ads/creatives
**Expected Output**: First Linear creative
**Validation**:
- Returns correct creative
- Handles InLine and Wrapper
- Returns nil if none found

#### TC-VP-121: GetMediaFiles()
**Objective**: Get all media files from VAST
**Input**: VAST with linear creative
**Expected Output**: Array of MediaFile structs
**Validation**:
- All media files returned
- Order preserved
- Empty array if none

#### TC-VP-122: GetImpressionURLs()
**Objective**: Collect all impression tracking URLs
**Input**: VAST with nested impressions (VAST root + Ad level)
**Expected Output**: Array of all unique impression URLs
**Validation**:
- All levels checked
- Deduplication if needed
- Order from outermost to innermost

#### TC-VP-123: GetDuration()
**Objective**: Extract video duration as time.Duration
**Input**: VAST with linear creative
**Expected Output**: Duration in time.Duration
**Validation**:
- HH:MM:SS converted correctly
- Returns 0 if not found
- Error handling for invalid format

#### TC-VP-124: HasAds()
**Objective**: Check if VAST has any ads
**Input**: Various VAST documents
**Expected Output**: Boolean
**Validation**:
- IsEmpty() inverse
- Correct for all cases

### 14. Real-World Compatibility Tests

#### TC-VP-130: Parse Google Ad Manager VAST
**Objective**: Handle real GAM VAST response
**Input**: Sample VAST from Google Ad Manager
**Expected Output**: Successful parse
**Validation**:
- All fields extracted
- No errors
- Tracking works

#### TC-VP-131: Parse SpotX VAST
**Objective**: Handle SpotX VAST format
**Input**: Sample SpotX VAST
**Expected Output**: Successful parse
**Validation**:
- SpotX-specific extensions handled
- No errors

#### TC-VP-132: Parse Beachfront VAST
**Objective**: Handle Beachfront VAST
**Input**: Sample Beachfront response
**Expected Output**: Successful parse
**Validation**:
- Beachfront format supported
- Extensions extracted

#### TC-VP-133: Parse Malformed Real-World VAST
**Objective**: Handle common VAST mistakes
**Input**: VAST with common errors (extra whitespace, wrong casing, missing CDATA)
**Expected Output**: Best-effort parse or clear error
**Validation**:
- Lenient where possible
- Clear errors when not
- No crashes

## Test Implementation Notes

### Go Testing Framework
- Use `testing` package
- Use `testify/assert` and `testify/require`
- Group tests with subtests
- Test file: `tnevideo/pkg/vast/parsing_test.go`

### Test Fixtures
- Sample VAST XMLs in `tests/fixtures/vast_samples/`
- Organized by version (vast_2.0/, vast_3.0/, vast_4.0/)
- Real-world samples in `vast_samples/real_world/`
- Invalid samples in `vast_samples/invalid/`

### Mock HTTP Server
- Use `httptest.Server` for wrapper unwrapping tests
- Mock demand partner responses
- Test timeouts and errors

### Validation Helpers
- Implement `ValidateVAST(v *VAST) error` function
- Use IAB VAST XSD for schema validation
- Custom business rule validators

### Performance Benchmarks
- Benchmark file: `vast/parsing_benchmark_test.go`
- Use `go test -bench=.`
- Track memory allocations with `-benchmem`

## Success Criteria
- All test cases pass
- 90%+ code coverage for parsing logic
- Performance benchmarks met
- No panics on invalid input
- Real-world VAST samples work
- XSD validation passes for generated output
