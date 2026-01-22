package ctv

import (
	"regexp"
	"strings"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
)

// DeviceType represents different Connected TV device types
type DeviceType string

const (
	DeviceRoku        DeviceType = "roku"
	DeviceFireTV      DeviceType = "firetv"
	DeviceAppleTV     DeviceType = "appletv"
	DeviceChromecast  DeviceType = "chromecast"
	DeviceAndroidTV   DeviceType = "androidtv"
	DeviceSamsung     DeviceType = "samsung"
	DeviceLG          DeviceType = "lg"
	DeviceVizio       DeviceType = "vizio"
	DeviceXbox        DeviceType = "xbox"
	DevicePlayStation DeviceType = "playstation"
	DeviceGeneric     DeviceType = "ctv"
	DeviceUnknown     DeviceType = ""
)

// DeviceInfo contains parsed CTV device information
type DeviceInfo struct {
	DeviceType DeviceType
	IsCTV      bool
	Model      string
	Make       string
}

// Pattern matching for CTV devices in user agent strings
var uaPatterns = map[DeviceType]*regexp.Regexp{
	DeviceRoku:        regexp.MustCompile(`(?i)roku`),
	DeviceFireTV:      regexp.MustCompile(`(?i)aft[a-z]|amazon.*fire|fire tv`),
	DeviceAppleTV:     regexp.MustCompile(`(?i)apple.*tv|tvos`),
	DeviceChromecast:  regexp.MustCompile(`(?i)chromecast|google tv`),
	DeviceAndroidTV:   regexp.MustCompile(`(?i)android.*tv`),
	DeviceSamsung:     regexp.MustCompile(`(?i)samsung.*smart.*tv|tizen`),
	DeviceLG:          regexp.MustCompile(`(?i)lg.*smart.*tv|webos.*tv`),
	DeviceVizio:       regexp.MustCompile(`(?i)vizio`),
	DeviceXbox:        regexp.MustCompile(`(?i)xbox`),
	DevicePlayStation: regexp.MustCompile(`(?i)playstation|ps[345]`),
}

// Pattern matching for CTV devices in make strings
var makePatterns = map[DeviceType][]string{
	DeviceRoku:        {"roku"},
	DeviceFireTV:      {"amazon", "aft"},
	DeviceAppleTV:     {"apple"},
	DeviceChromecast:  {"google", "chromecast"},
	DeviceAndroidTV:   {"android"},
	DeviceSamsung:     {"samsung"},
	DeviceLG:          {"lg"},
	DeviceVizio:       {"vizio"},
	DeviceXbox:        {"microsoft", "xbox"},
	DevicePlayStation: {"sony", "playstation"},
}

// Pattern matching for CTV devices in model strings
var modelPatterns = map[DeviceType][]string{
	DeviceRoku:        {"roku"},
	DeviceFireTV:      {"aft", "fire", "firetv"},
	DeviceAppleTV:     {"appletv", "apple tv"},
	DeviceChromecast:  {"chromecast"},
	DeviceAndroidTV:   {"android tv", "androidtv"},
	DeviceSamsung:     {"smart tv", "tizen"},
	DeviceLG:          {"smart tv", "webos"},
	DeviceVizio:       {"vizio"},
	DeviceXbox:        {"xbox"},
	DevicePlayStation: {"ps3", "ps4", "ps5", "playstation"},
}

// ParseDevice analyzes an OpenRTB device object and extracts CTV device information.
// It checks device.ua, device.make, device.model, and device.devicetype in that order.
func ParseDevice(device *openrtb2.Device) *DeviceInfo {
	if device == nil {
		return &DeviceInfo{
			DeviceType: DeviceUnknown,
			IsCTV:      false,
		}
	}

	info := &DeviceInfo{
		Model: device.Model,
		Make:  device.Make,
	}

	// Check User Agent first (most reliable)
	if device.UA != "" {
		if deviceType := detectFromUA(device.UA); deviceType != DeviceUnknown {
			info.DeviceType = deviceType
			info.IsCTV = true
			return info
		}
	}

	// Check Make and Model
	if device.Make != "" || device.Model != "" {
		if deviceType := detectFromMakeModel(device.Make, device.Model); deviceType != DeviceUnknown {
			info.DeviceType = deviceType
			info.IsCTV = true
			return info
		}
	}

	// Check DeviceType field (OpenRTB 2.5 spec: 3 = Connected TV)
	if device.DeviceType == adcom1.DeviceType(3) {
		info.IsCTV = true
		// If we couldn't determine specific type but DeviceType says it's CTV
		if info.DeviceType == DeviceUnknown {
			info.DeviceType = DeviceGeneric
		}
		return info
	}

	return info
}

// IsCTV returns true if the device is identified as a Connected TV device
func IsCTV(device *openrtb2.Device) bool {
	info := ParseDevice(device)
	return info.IsCTV
}

// GetDeviceType returns the specific CTV device type, or empty string if not CTV
func GetDeviceType(device *openrtb2.Device) DeviceType {
	info := ParseDevice(device)
	return info.DeviceType
}

// SupportsDevice checks if a device matches any of the specified CTV device types.
// If supportedDevices is empty, all devices are considered supported.
func SupportsDevice(device *openrtb2.Device, supportedDevices []DeviceType) bool {
	if len(supportedDevices) == 0 {
		// No restrictions, all devices supported
		return true
	}

	info := ParseDevice(device)
	if !info.IsCTV {
		return false
	}

	// Check if device type is in supported list
	for _, supported := range supportedDevices {
		if info.DeviceType == supported {
			return true
		}
		// Generic CTV matches if in supported list
		if info.DeviceType == DeviceGeneric && supported == DeviceGeneric {
			return true
		}
	}

	return false
}

// detectFromUA attempts to identify CTV device type from user agent string
func detectFromUA(ua string) DeviceType {
	for deviceType, pattern := range uaPatterns {
		if pattern.MatchString(ua) {
			return deviceType
		}
	}
	return DeviceUnknown
}

// detectFromMakeModel attempts to identify CTV device type from make and model strings
func detectFromMakeModel(make, model string) DeviceType {
	makeLower := strings.ToLower(make)
	modelLower := strings.ToLower(model)

	// Check make patterns
	for deviceType, patterns := range makePatterns {
		for _, pattern := range patterns {
			if strings.Contains(makeLower, pattern) {
				// Confirm with model if available
				if model != "" {
					if modelPatterns, ok := modelPatterns[deviceType]; ok {
						for _, modelPattern := range modelPatterns {
							if strings.Contains(modelLower, modelPattern) {
								return deviceType
							}
						}
					}
				} else {
					// No model, just return based on make
					return deviceType
				}
			}
		}
	}

	// Check model patterns independently
	for deviceType, patterns := range modelPatterns {
		for _, pattern := range patterns {
			if strings.Contains(modelLower, pattern) {
				return deviceType
			}
		}
	}

	return DeviceUnknown
}
