package ortb

import (
	"regexp"
	"strings"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
)

// CTVDeviceType represents different Connected TV device types
type CTVDeviceType string

const (
	CTVDeviceRoku     CTVDeviceType = "roku"
	CTVDeviceFireTV   CTVDeviceType = "firetv"
	CTVDeviceAppleTV  CTVDeviceType = "appletv"
	CTVDeviceChromecast CTVDeviceType = "chromecast"
	CTVDeviceAndroidTV  CTVDeviceType = "androidtv"
	CTVDeviceSamsung    CTVDeviceType = "samsung"
	CTVDeviceLG         CTVDeviceType = "lg"
	CTVDeviceVizio      CTVDeviceType = "vizio"
	CTVDeviceXbox       CTVDeviceType = "xbox"
	CTVDevicePlayStation CTVDeviceType = "playstation"
	CTVDeviceGeneric    CTVDeviceType = "ctv"
	CTVDeviceUnknown    CTVDeviceType = ""
)

// CTVDeviceInfo contains parsed CTV device information
type CTVDeviceInfo struct {
	DeviceType CTVDeviceType
	IsCTV      bool
	Model      string
	Make       string
}

// Pattern matching for CTV devices in user agent strings
var ctvPatterns = map[CTVDeviceType]*regexp.Regexp{
	CTVDeviceRoku:        regexp.MustCompile(`(?i)roku`),
	CTVDeviceFireTV:      regexp.MustCompile(`(?i)aft[a-z]|amazon.*fire|fire tv`),
	CTVDeviceAppleTV:     regexp.MustCompile(`(?i)apple.*tv|tvos`),
	CTVDeviceChromecast:  regexp.MustCompile(`(?i)chromecast|crkey|google tv`),
	CTVDeviceAndroidTV:   regexp.MustCompile(`(?i)android.*tv`),
	CTVDeviceSamsung:     regexp.MustCompile(`(?i)samsung.*smart.*tv|tizen`),
	CTVDeviceLG:          regexp.MustCompile(`(?i)lg.*smart.*tv|webos.*tv`),
	CTVDeviceVizio:       regexp.MustCompile(`(?i)vizio`),
	CTVDeviceXbox:        regexp.MustCompile(`(?i)xbox`),
	CTVDevicePlayStation: regexp.MustCompile(`(?i)playstation|ps[345]`),
}

// Pattern matching for CTV devices in model/make strings
var ctvMakePatterns = map[CTVDeviceType][]string{
	CTVDeviceRoku:        {"roku"},
	CTVDeviceFireTV:      {"amazon", "aft"},
	CTVDeviceAppleTV:     {"apple"},
	CTVDeviceChromecast:  {"google", "chromecast"},
	CTVDeviceAndroidTV:   {"android"},
	CTVDeviceSamsung:     {"samsung"},
	CTVDeviceLG:          {"lg"},
	CTVDeviceVizio:       {"vizio"},
	CTVDeviceXbox:        {"microsoft", "xbox"},
	CTVDevicePlayStation: {"sony", "playstation"},
}

var ctvModelPatterns = map[CTVDeviceType][]string{
	CTVDeviceRoku:        {"roku"},
	CTVDeviceFireTV:      {"aft", "fire", "firetv"},
	CTVDeviceAppleTV:     {"appletv", "apple tv"},
	CTVDeviceChromecast:  {"chromecast"},
	CTVDeviceAndroidTV:   {"android tv", "androidtv"},
	CTVDeviceSamsung:     {"smart tv", "tizen"},
	CTVDeviceLG:          {"smart tv", "webos"},
	CTVDeviceVizio:       {"vizio"},
	CTVDeviceXbox:        {"xbox"},
	CTVDevicePlayStation: {"ps3", "ps4", "ps5", "playstation"},
}

// ParseCTVDevice analyzes an OpenRTB device object and extracts CTV device information
func ParseCTVDevice(device *openrtb2.Device) *CTVDeviceInfo {
	if device == nil {
		return &CTVDeviceInfo{
			DeviceType: CTVDeviceUnknown,
			IsCTV:      false,
		}
	}

	info := &CTVDeviceInfo{
		Model: device.Model,
		Make:  device.Make,
	}

	// Check User Agent first
	if device.UA != "" {
		if deviceType := detectCTVFromUA(device.UA); deviceType != CTVDeviceUnknown {
			info.DeviceType = deviceType
			info.IsCTV = true
			return info
		}
	}

	// Check Make and Model
	if device.Make != "" || device.Model != "" {
		if deviceType := detectCTVFromMakeModel(device.Make, device.Model); deviceType != CTVDeviceUnknown {
			info.DeviceType = deviceType
			info.IsCTV = true
			return info
		}
	}

	// Check DeviceType field (OpenRTB 2.5 spec: 3 = Connected TV)
	if device.DeviceType == adcom1.DeviceType(3) {
		info.IsCTV = true
		// If we couldn't determine specific type but DeviceType says it's CTV
		if info.DeviceType == CTVDeviceUnknown {
			info.DeviceType = CTVDeviceGeneric
		}
		return info
	}

	return info
}

// detectCTVFromUA attempts to identify CTV device type from user agent string
func detectCTVFromUA(ua string) CTVDeviceType {
	for deviceType, pattern := range ctvPatterns {
		if pattern.MatchString(ua) {
			return deviceType
		}
	}
	return CTVDeviceUnknown
}

// detectCTVFromMakeModel attempts to identify CTV device type from make and model strings
func detectCTVFromMakeModel(make, model string) CTVDeviceType {
	makeLower := strings.ToLower(make)
	modelLower := strings.ToLower(model)

	// Check make patterns
	for deviceType, patterns := range ctvMakePatterns {
		for _, pattern := range patterns {
			if strings.Contains(makeLower, pattern) {
				// Confirm with model if available
				if model != "" {
					if modelPatterns, ok := ctvModelPatterns[deviceType]; ok {
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
	for deviceType, patterns := range ctvModelPatterns {
		for _, pattern := range patterns {
			if strings.Contains(modelLower, pattern) {
				return deviceType
			}
		}
	}

	return CTVDeviceUnknown
}

// IsCTVDevice returns true if the device is identified as a Connected TV device
func IsCTVDevice(device *openrtb2.Device) bool {
	info := ParseCTVDevice(device)
	return info.IsCTV
}

// GetCTVDeviceType returns the specific CTV device type, or empty string if not CTV
func GetCTVDeviceType(device *openrtb2.Device) CTVDeviceType {
	info := ParseCTVDevice(device)
	return info.DeviceType
}

// SupportsCTVDevice checks if a device matches any of the specified CTV device types
func SupportsCTVDevice(device *openrtb2.Device, supportedDevices []CTVDeviceType) bool {
	if len(supportedDevices) == 0 {
		// No restrictions, all devices supported
		return true
	}

	info := ParseCTVDevice(device)
	if !info.IsCTV {
		return false
	}

	// Check if device type is in supported list
	for _, supported := range supportedDevices {
		if info.DeviceType == supported {
			return true
		}
		// Generic CTV matches if in supported list
		if info.DeviceType == CTVDeviceGeneric && supported == CTVDeviceGeneric {
			return true
		}
	}

	return false
}
