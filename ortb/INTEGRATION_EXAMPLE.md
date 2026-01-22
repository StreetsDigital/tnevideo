# CTV Device Targeting - Integration Examples

## Overview
This document provides practical examples of integrating CTV device detection into prebid-server workflows.

## Example 1: Bidder Adapter Integration

### Filtering Bids by Supported CTV Devices

```go
package adapters

import (
    "github.com/prebid/openrtb/v20/openrtb2"
    "github.com/prebid/prebid-server/v3/ortb"
)

// ExampleAdapter demonstrates CTV device filtering
type ExampleAdapter struct {
    supportedCTVDevices []ortb.CTVDeviceType
}

// MakeBids filters responses based on CTV device support
func (a *ExampleAdapter) MakeBids(
    request *openrtb2.BidRequest,
    requestData *adapters.RequestData,
    responseData *adapters.ResponseData,
) (*adapters.BidderResponse, []error) {

    // Check if device is supported
    if request.Device != nil {
        if !ortb.SupportsCTVDevice(request.Device, a.supportedCTVDevices) {
            // Device not supported, return empty response
            return &adapters.BidderResponse{}, nil
        }
    }

    // Continue with normal bid processing
    // ...
}
```

## Example 2: Bid Request Preprocessing

### Enriching Requests with CTV Device Information

```go
package exchange

import (
    "github.com/prebid/openrtb/v20/openrtb2"
    "github.com/prebid/prebid-server/v3/openrtb_ext"
    "github.com/prebid/prebid-server/v3/ortb"
)

// EnrichBidRequestWithCTVInfo adds CTV device metadata to request extensions
func EnrichBidRequestWithCTVInfo(req *openrtb_ext.RequestWrapper) error {
    if req.Device == nil {
        return nil
    }

    ctvInfo := ortb.ParseCTVDevice(req.Device)
    if !ctvInfo.IsCTV {
        return nil
    }

    // Add CTV device type to request extension for downstream use
    reqExt, err := req.GetRequestExt()
    if err != nil {
        return err
    }

    prebidExt := reqExt.GetPrebid()
    if prebidExt == nil {
        prebidExt = &openrtb_ext.ExtRequestPrebid{}
    }

    // Store CTV device info in extension
    // This could be used by analytics or other modules
    // Example: prebidExt.Data["ctvDeviceType"] = string(ctvInfo.DeviceType)

    return nil
}
```

## Example 3: Price Floors Integration

### Applying Device-Specific Floor Prices

```go
package floors

import (
    "github.com/prebid/openrtb/v20/openrtb2"
    "github.com/prebid/prebid-server/v3/ortb"
)

// GetCTVDeviceFloorMultiplier returns a floor price multiplier based on CTV device type
func GetCTVDeviceFloorMultiplier(device *openrtb2.Device) float64 {
    if device == nil {
        return 1.0
    }

    ctvInfo := ortb.ParseCTVDevice(device)
    if !ctvInfo.IsCTV {
        return 1.0
    }

    // Apply different floor multipliers based on device type
    switch ctvInfo.DeviceType {
    case ortb.CTVDeviceRoku:
        return 1.2 // 20% higher floor for Roku
    case ortb.CTVDeviceFireTV:
        return 1.15 // 15% higher floor for Fire TV
    case ortb.CTVDeviceAppleTV:
        return 1.3 // 30% higher floor for Apple TV
    case ortb.CTVDeviceChromecast:
        return 1.1
    case ortb.CTVDeviceAndroidTV:
        return 1.1
    default:
        return 1.05 // 5% higher for generic CTV
    }
}

// Example usage in floor calculation
func CalculateFloorWithCTVMultiplier(baseFloor float64, device *openrtb2.Device) float64 {
    multiplier := GetCTVDeviceFloorMultiplier(device)
    return baseFloor * multiplier
}
```

## Example 4: Analytics Module Integration

### Tracking CTV Device Distribution

```go
package analytics

import (
    "github.com/prebid/openrtb/v20/openrtb2"
    "github.com/prebid/prebid-server/v3/ortb"
)

// CTVMetrics tracks device distribution metrics
type CTVMetrics struct {
    deviceCounts map[ortb.CTVDeviceType]int
    totalCTV     int
    totalNonCTV  int
}

// RecordBidRequest tracks device types from bid requests
func (m *CTVMetrics) RecordBidRequest(req *openrtb2.BidRequest) {
    if req.Device == nil {
        return
    }

    ctvInfo := ortb.ParseCTVDevice(req.Device)

    if ctvInfo.IsCTV {
        m.totalCTV++
        if m.deviceCounts == nil {
            m.deviceCounts = make(map[ortb.CTVDeviceType]int)
        }
        m.deviceCounts[ctvInfo.DeviceType]++
    } else {
        m.totalNonCTV++
    }
}

// GetDeviceDistribution returns percentage distribution of CTV devices
func (m *CTVMetrics) GetDeviceDistribution() map[ortb.CTVDeviceType]float64 {
    if m.totalCTV == 0 {
        return nil
    }

    distribution := make(map[ortb.CTVDeviceType]float64)
    for deviceType, count := range m.deviceCounts {
        distribution[deviceType] = float64(count) / float64(m.totalCTV) * 100
    }
    return distribution
}
```

## Example 5: Stored Request Configuration

### CTV Device Targeting in Stored Requests

```json
{
  "id": "stored-request-ctv-targeting",
  "imp": [{
    "id": "1",
    "video": {
      "mimes": ["video/mp4"],
      "minduration": 15,
      "maxduration": 30,
      "protocols": [2, 3, 5, 6]
    },
    "ext": {
      "prebid": {
        "bidder": {
          "exampleBidder": {
            "placementId": "12345",
            "ctvTargeting": {
              "supportedDevices": ["roku", "firetv", "appletv"],
              "minFloor": 2.50
            }
          }
        }
      }
    }
  }]
}
```

### Processing CTV Targeting Parameters

```go
package storedrequest

import (
    "encoding/json"
    "github.com/prebid/openrtb/v20/openrtb2"
    "github.com/prebid/prebid-server/v3/ortb"
)

// CTVTargetingConfig represents CTV targeting parameters from stored request
type CTVTargetingConfig struct {
    SupportedDevices []string  `json:"supportedDevices"`
    MinFloor         float64   `json:"minFloor"`
}

// ValidateCTVTargeting checks if request matches CTV targeting criteria
func ValidateCTVTargeting(device *openrtb2.Device, config CTVTargetingConfig) bool {
    if len(config.SupportedDevices) == 0 {
        return true
    }

    // Convert string device types to CTVDeviceType
    supportedDevices := make([]ortb.CTVDeviceType, 0, len(config.SupportedDevices))
    for _, deviceStr := range config.SupportedDevices {
        supportedDevices = append(supportedDevices, ortb.CTVDeviceType(deviceStr))
    }

    return ortb.SupportsCTVDevice(device, supportedDevices)
}
```

## Example 6: Request Validation with CTV Check

### Validating CTV Requirements in Impressions

```go
package ortb

import (
    "fmt"
    "github.com/prebid/openrtb/v20/openrtb2"
    "github.com/prebid/prebid-server/v3/openrtb_ext"
)

// ValidateCTVImpression validates CTV-specific requirements for video impressions
func ValidateCTVImpression(imp *openrtb_ext.ImpWrapper, device *openrtb2.Device) error {
    // Check if this is a CTV device
    if !IsCTVDevice(device) {
        return nil // No CTV-specific validation needed
    }

    // CTV devices typically require video impressions
    if imp.Video == nil {
        return fmt.Errorf("CTV device detected but impression has no video object")
    }

    // Validate video parameters for CTV
    if imp.Video.W == nil || imp.Video.H == nil {
        return fmt.Errorf("CTV video impressions require width and height")
    }

    // Check for common CTV video sizes
    width := *imp.Video.W
    height := *imp.Video.H

    validCTVSizes := map[int64][]int64{
        1280: {720},   // 720p
        1920: {1080},  // 1080p
        3840: {2160},  // 4K
    }

    if heights, ok := validCTVSizes[width]; ok {
        for _, h := range heights {
            if height == h {
                return nil // Valid CTV size
            }
        }
    }

    // Not a standard CTV size, but don't fail - just log warning
    // In production, you might want to log this for analytics
    return nil
}
```

## Example 7: Dynamic Creative Optimization (DCO)

### Selecting Device-Appropriate Creatives

```go
package creative

import (
    "github.com/prebid/openrtb/v20/openrtb2"
    "github.com/prebid/prebid-server/v3/ortb"
)

// CreativeSelector chooses appropriate creative based on device
type CreativeSelector struct {
    creatives map[ortb.CTVDeviceType][]string
}

// SelectCreative returns the best creative ID for the device
func (cs *CreativeSelector) SelectCreative(device *openrtb2.Device) string {
    ctvInfo := ortb.ParseCTVDevice(device)

    if !ctvInfo.IsCTV {
        return cs.getDefaultCreative()
    }

    // Try to get device-specific creative
    if creatives, ok := cs.creatives[ctvInfo.DeviceType]; ok && len(creatives) > 0 {
        return creatives[0]
    }

    // Fallback to generic CTV creative
    if creatives, ok := cs.creatives[ortb.CTVDeviceGeneric]; ok && len(creatives) > 0 {
        return creatives[0]
    }

    return cs.getDefaultCreative()
}

func (cs *CreativeSelector) getDefaultCreative() string {
    return "default-creative-id"
}
```

## Example 8: Testing Helper Functions

### Unit Test Utilities for CTV Device Testing

```go
package testutils

import (
    "github.com/prebid/openrtb/v20/openrtb2"
    "github.com/prebid/prebid-server/v3/ortb"
)

// CreateRokuDevice creates a test Roku device object
func CreateRokuDevice() *openrtb2.Device {
    return &openrtb2.Device{
        UA:    "Roku/DVP-9.10 (519.10E04154A)",
        Make:  "Roku",
        Model: "Roku Ultra",
    }
}

// CreateFireTVDevice creates a test Fire TV device object
func CreateFireTVDevice() *openrtb2.Device {
    return &openrtb2.Device{
        UA:    "Mozilla/5.0 (Linux; Android 7.1.2; AFTMM) AppleWebKit/537.36",
        Make:  "Amazon",
        Model: "Fire TV Stick 4K",
    }
}

// CreateAppleTVDevice creates a test Apple TV device object
func CreateAppleTVDevice() *openrtb2.Device {
    return &openrtb2.Device{
        UA:    "AppleTV11,1/tvOS 15.0",
        Make:  "Apple",
        Model: "Apple TV 4K",
    }
}

// CreateGenericCTVDevice creates a test generic CTV device
func CreateGenericCTVDevice() *openrtb2.Device {
    deviceType := int8(3) // OpenRTB 2.5: 3 = Connected TV
    return &openrtb2.Device{
        DeviceType: &deviceType,
        UA:         "Unknown CTV Device",
    }
}

// CreateNonCTVDevice creates a test mobile device
func CreateNonCTVDevice() *openrtb2.Device {
    return &openrtb2.Device{
        UA:    "Mozilla/5.0 (iPhone; CPU iPhone OS 14_5 like Mac OS X)",
        Make:  "Apple",
        Model: "iPhone 12",
    }
}
```

## Testing the Implementation

To test these integrations, you can create test cases:

```go
package integration_test

import (
    "testing"
    "github.com/prebid/openrtb/v20/openrtb2"
    "github.com/prebid/prebid-server/v3/ortb"
    "github.com/stretchr/testify/assert"
)

func TestCTVDeviceFiltering(t *testing.T) {
    supportedDevices := []ortb.CTVDeviceType{
        ortb.CTVDeviceRoku,
        ortb.CTVDeviceFireTV,
    }

    tests := []struct {
        name     string
        device   *openrtb2.Device
        expected bool
    }{
        {
            name:     "Supported Roku device",
            device:   CreateRokuDevice(),
            expected: true,
        },
        {
            name:     "Unsupported Apple TV",
            device:   CreateAppleTVDevice(),
            expected: false,
        },
        {
            name:     "Non-CTV device",
            device:   CreateNonCTVDevice(),
            expected: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := ortb.SupportsCTVDevice(tt.device, supportedDevices)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

## Performance Considerations

1. **Regex Compilation**: Regex patterns are compiled once at package initialization
2. **Early Returns**: Detection functions return as soon as a match is found
3. **Caching**: Consider caching device detection results for repeated requests
4. **String Operations**: Case-insensitive matching uses regex for UA, simple strings.ToLower for make/model

## Best Practices

1. **Always check for nil device**: Use the provided helper functions that handle nil gracefully
2. **Use specific device types when possible**: More specific targeting improves performance
3. **Fallback to generic CTV**: Support `CTVDeviceGeneric` for unknown CTV devices
4. **Log unrecognized patterns**: Help improve detection by logging unknown CTV user agents
5. **Test with real user agents**: Use actual device UAs from your traffic for testing

## Next Steps

1. Integrate into your bidder adapters
2. Add CTV-specific floor pricing
3. Implement device-based analytics
4. Create stored request templates for common CTV scenarios
5. Monitor and optimize detection patterns based on real traffic
