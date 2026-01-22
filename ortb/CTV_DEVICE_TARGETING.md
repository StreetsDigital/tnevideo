# CTV Device Targeting Implementation

## Overview
This implementation adds Connected TV (CTV) device detection and targeting capabilities to prebid-server. It parses OpenRTB device objects to identify specific CTV platforms and enables bid filtering based on device support.

## Files
- `/projects/prebid-server/ortb/device_ctv.go` - Core CTV device detection logic
- `/projects/prebid-server/ortb/device_ctv_test.go` - Comprehensive test suite

## Supported CTV Devices
The implementation detects the following CTV device types:

1. **Roku** - Roku streaming devices
2. **Fire TV** - Amazon Fire TV and Fire TV Stick
3. **Apple TV** - Apple TV devices running tvOS
4. **Chromecast** - Google Chromecast and Google TV
5. **Android TV** - Android TV devices
6. **Samsung** - Samsung Smart TVs (Tizen)
7. **LG** - LG Smart TVs (webOS)
8. **Vizio** - Vizio Smart TVs
9. **Xbox** - Microsoft Xbox gaming consoles
10. **PlayStation** - Sony PlayStation gaming consoles
11. **Generic CTV** - Any CTV device identified by OpenRTB devicetype field

## Detection Methods

### 1. User Agent Parsing
Analyzes the `device.ua` field for CTV-specific patterns:
- Roku: "roku"
- Fire TV: "aft*", "amazon.*fire", "fire tv"
- Apple TV: "apple.*tv", "tvos"
- Chromecast: "chromecast", "google tv"
- Android TV: "android.*tv"
- Samsung: "samsung.*smart.*tv", "tizen"
- LG: "lg.*smart.*tv", "webos.*tv"
- Xbox: "xbox"
- PlayStation: "playstation", "ps[345]"

### 2. Make/Model Detection
Examines the `device.make` and `device.model` fields:
- Cross-references make patterns (e.g., "Roku", "Amazon", "Apple")
- Validates with model patterns (e.g., "Fire TV Stick", "Apple TV 4K")

### 3. OpenRTB DeviceType Field
Uses the standard OpenRTB 2.5 `device.devicetype` field:
- Value of 3 indicates Connected TV
- If no specific device type is identified, returns generic CTV

## API Functions

### ParseCTVDevice
```go
func ParseCTVDevice(device *openrtb2.Device) *CTVDeviceInfo
```
Analyzes an OpenRTB device object and returns detailed CTV information.

**Returns:**
- `CTVDeviceInfo.DeviceType` - Specific CTV platform (roku, firetv, etc.)
- `CTVDeviceInfo.IsCTV` - Boolean indicating if device is CTV
- `CTVDeviceInfo.Model` - Device model string
- `CTVDeviceInfo.Make` - Device manufacturer string

### IsCTVDevice
```go
func IsCTVDevice(device *openrtb2.Device) bool
```
Simple check if a device is any type of CTV device.

### GetCTVDeviceType
```go
func GetCTVDeviceType(device *openrtb2.Device) CTVDeviceType
```
Returns the specific CTV device type, or empty string if not CTV.

### SupportsCTVDevice
```go
func SupportsCTVDevice(device *openrtb2.Device, supportedDevices []CTVDeviceType) bool
```
Checks if a device matches any of the specified supported CTV device types.

**Parameters:**
- `device` - OpenRTB device object
- `supportedDevices` - List of supported CTV device types

**Returns:**
- `true` if device is in supported list or list is empty (all devices allowed)
- `false` if device is not CTV or not in supported list

## Usage Examples

### Example 1: Detect CTV Device Type
```go
import (
    "github.com/prebid/openrtb/v20/openrtb2"
    "github.com/prebid/prebid-server/v3/ortb"
)

// Parse device from OpenRTB request
device := &openrtb2.Device{
    UA: "Roku/DVP-9.10 (519.10E04154A)",
}

info := ortb.ParseCTVDevice(device)
// info.DeviceType = "roku"
// info.IsCTV = true
```

### Example 2: Check if Device is CTV
```go
device := &openrtb2.Device{
    Make: "Amazon",
    Model: "Fire TV Stick",
}

if ortb.IsCTVDevice(device) {
    // Handle CTV-specific logic
}
```

### Example 3: Filter Bids by Supported Devices
```go
// Bidder only supports Roku and Fire TV
supportedDevices := []ortb.CTVDeviceType{
    ortb.CTVDeviceRoku,
    ortb.CTVDeviceFireTV,
}

device := &openrtb2.Device{
    UA: "Roku/DVP-9.10",
}

if ortb.SupportsCTVDevice(device, supportedDevices) {
    // Process bid for this device
} else {
    // Skip bid
}
```

### Example 4: OpenRTB Request Processing
```go
// In bid request processing
func processBidRequest(req *openrtb2.BidRequest) {
    if req.Device != nil {
        ctvInfo := ortb.ParseCTVDevice(req.Device)

        if ctvInfo.IsCTV {
            switch ctvInfo.DeviceType {
            case ortb.CTVDeviceRoku:
                // Roku-specific targeting
            case ortb.CTVDeviceFireTV:
                // Fire TV-specific targeting
            case ortb.CTVDeviceAppleTV:
                // Apple TV-specific targeting
            default:
                // Generic CTV targeting
            }
        }
    }
}
```

## Integration Points

### Bid Filtering
Bidders can use `SupportsCTVDevice()` to filter bid requests based on device compatibility.

### Analytics
Use `ParseCTVDevice()` to collect CTV device distribution metrics.

### Targeting Rules
Integrate with existing floors/rules system to apply device-specific pricing.

## Test Coverage

The implementation includes comprehensive tests covering:
- ✅ User agent pattern matching for all device types
- ✅ Make/model detection
- ✅ OpenRTB devicetype field handling
- ✅ Case-insensitive matching
- ✅ Edge cases (nil device, empty fields)
- ✅ Device filtering and support validation

Run tests with:
```bash
go test -v ./ortb -run TestParseCTVDevice
go test -v ./ortb -run TestIsCTVDevice
go test -v ./ortb -run TestSupportsCTVDevice
```

## Future Enhancements

1. **Additional Device Types**
   - Add support for more regional CTV platforms
   - Detect specific device models/versions

2. **Performance Optimization**
   - Cache compiled regex patterns
   - Implement device fingerprinting

3. **Extended Metadata**
   - OS version detection
   - Screen resolution mapping
   - HDR/4K capability detection

4. **Integration Examples**
   - Adapter-specific usage patterns
   - Stored request configuration examples
   - Price floor integration

## OpenRTB Specification Reference

Based on OpenRTB 2.5 specification:
- Section 3.2.18: Device Object
- Device Type values: 3 = Connected TV
- Fields used: `ua`, `make`, `model`, `devicetype`
