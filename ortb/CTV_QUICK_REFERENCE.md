# CTV Device Targeting - Quick Reference Card

## Quick Start

```go
import "github.com/prebid/prebid-server/v3/ortb"

// Check if device is CTV
if ortb.IsCTVDevice(request.Device) {
    // CTV-specific logic
}

// Get specific device type
deviceType := ortb.GetCTVDeviceType(request.Device)
// Returns: "roku", "firetv", "appletv", etc.

// Filter by supported devices
supported := []ortb.CTVDeviceType{
    ortb.CTVDeviceRoku,
    ortb.CTVDeviceFireTV,
}
if ortb.SupportsCTVDevice(request.Device, supported) {
    // Process bid
}
```

## Device Types

| Constant | Value | Detects |
|----------|-------|---------|
| `CTVDeviceRoku` | "roku" | Roku devices |
| `CTVDeviceFireTV` | "firetv" | Amazon Fire TV |
| `CTVDeviceAppleTV` | "appletv" | Apple TV (tvOS) |
| `CTVDeviceChromecast` | "chromecast" | Chromecast, Google TV |
| `CTVDeviceAndroidTV` | "androidtv" | Android TV |
| `CTVDeviceSamsung` | "samsung" | Samsung Smart TV (Tizen) |
| `CTVDeviceLG` | "lg" | LG Smart TV (webOS) |
| `CTVDeviceVizio` | "vizio" | Vizio Smart TV |
| `CTVDeviceXbox` | "xbox" | Xbox consoles |
| `CTVDevicePlayStation` | "playstation" | PlayStation consoles |
| `CTVDeviceGeneric` | "ctv" | Generic CTV (OpenRTB devicetype=3) |
| `CTVDeviceUnknown` | "" | Not a CTV device |

## API Reference

### ParseCTVDevice
```go
func ParseCTVDevice(device *openrtb2.Device) *CTVDeviceInfo

// Returns:
type CTVDeviceInfo struct {
    DeviceType CTVDeviceType  // Specific device type
    IsCTV      bool           // true if CTV
    Model      string         // Device model
    Make       string         // Device manufacturer
}
```

### IsCTVDevice
```go
func IsCTVDevice(device *openrtb2.Device) bool
// Returns true if device is any CTV type
```

### GetCTVDeviceType
```go
func GetCTVDeviceType(device *openrtb2.Device) CTVDeviceType
// Returns specific device type or "" if not CTV
```

### SupportsCTVDevice
```go
func SupportsCTVDevice(device *openrtb2.Device, supportedDevices []CTVDeviceType) bool
// Returns true if device is in supported list
// Empty list = all devices allowed
```

## Common Patterns

### Pattern 1: Simple CTV Check
```go
if ortb.IsCTVDevice(req.Device) {
    // Apply CTV bidding strategy
}
```

### Pattern 2: Device-Specific Logic
```go
switch ortb.GetCTVDeviceType(req.Device) {
case ortb.CTVDeviceRoku:
    // Roku-specific handling
case ortb.CTVDeviceFireTV:
    // Fire TV-specific handling
case ortb.CTVDeviceAppleTV:
    // Apple TV-specific handling
default:
    // Generic or non-CTV
}
```

### Pattern 3: Filtering with Support List
```go
supportedDevices := []ortb.CTVDeviceType{
    ortb.CTVDeviceRoku,
    ortb.CTVDeviceFireTV,
    ortb.CTVDeviceAppleTV,
}

if !ortb.SupportsCTVDevice(req.Device, supportedDevices) {
    return nil, errors.New("device not supported")
}
```

### Pattern 4: Full Device Analysis
```go
info := ortb.ParseCTVDevice(req.Device)
if info.IsCTV {
    log.Printf("CTV Device: type=%s, make=%s, model=%s",
        info.DeviceType, info.Make, info.Model)
}
```

## Detection Priority

1. **User Agent** (checked first)
   - Fastest, most reliable
   - Regex patterns match device signatures

2. **Make + Model** (checked second)
   - String matching on device fields
   - Cross-validates make with model

3. **DeviceType Field** (checked last)
   - OpenRTB 2.5 standard field
   - devicetype=3 means Connected TV
   - Returns generic CTV if no specific type found

## Example User Agents

```go
// Roku
"Roku/DVP-9.10 (519.10E04154A)"

// Fire TV
"Mozilla/5.0 (Linux; Android 7.1.2; AFTMM) AppleWebKit/537.36"

// Apple TV
"AppleTV11,1/tvOS 15.0"

// Chromecast
"Mozilla/5.0 (X11; Linux armv7l) CrKey/1.56.500000"

// Android TV
"Mozilla/5.0 (Linux; Android 9; Android TV)"

// Samsung
"Mozilla/5.0 (SMART-TV; Linux; Tizen 5.0)"

// LG
"Mozilla/5.0 (Web0S; Linux/SmartTV)"
```

## Testing

```bash
# Run all CTV tests
go test -v ./ortb -run TestCTVDevice

# Run specific test
go test -v ./ortb -run TestParseCTVDevice_UserAgent

# Run examples
go test -v ./ortb -run Example
```

## Common Mistakes

❌ **Don't do this:**
```go
// Checking UA string directly
if strings.Contains(device.UA, "roku") {
    // Fragile, case-sensitive
}
```

✅ **Do this instead:**
```go
// Use the detection functions
if ortb.GetCTVDeviceType(device) == ortb.CTVDeviceRoku {
    // Robust, case-insensitive, multi-method
}
```

❌ **Don't do this:**
```go
// Assuming device is never nil
deviceType := ortb.GetCTVDeviceType(req.Device)
// Panics if req.Device is nil
```

✅ **Do this instead:**
```go
// Functions handle nil gracefully
deviceType := ortb.GetCTVDeviceType(req.Device)
// Returns CTVDeviceUnknown if device is nil
```

## Performance Notes

- Regex patterns compiled once at package init
- Detection returns on first match (early exit)
- No caching needed for single-request detection
- Consider caching if parsing same device repeatedly

## Files

- Implementation: `/projects/prebid-server/ortb/device_ctv.go`
- Tests: `/projects/prebid-server/ortb/device_ctv_test.go`
- Examples: `/projects/prebid-server/ortb/device_ctv_example_test.go`
- Documentation: `/projects/prebid-server/ortb/CTV_DEVICE_TARGETING.md`
- Integration: `/projects/prebid-server/ortb/INTEGRATION_EXAMPLE.md`

## Support

For questions or issues:
1. Check `CTV_DEVICE_TARGETING.md` for detailed documentation
2. Review `INTEGRATION_EXAMPLE.md` for integration patterns
3. Run example tests to see usage patterns
4. Check test suite for edge cases

## Version

Implementation: GT-FEAT-004
OpenRTB Version: 2.5
Prebid Server: v3
