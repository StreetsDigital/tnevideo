// Package ctv provides Connected TV device detection and targeting
package ctv

import (
	"regexp"
	"strings"

	"github.com/thenexusengine/tne_springwire/internal/openrtb"
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
	Type   DeviceType
	IsCTV  bool
	Model  string
	Make   string
	OS     string
	OSVer  string
}

// Pattern matching for CTV devices in user agent strings
var uaPatterns = map[DeviceType]*regexp.Regexp{
	DeviceRoku:        regexp.MustCompile(`(?i)roku`),
	DeviceFireTV:      regexp.MustCompile(`(?i)aft[a-z]|amazon.*fire|fire\s*tv`),
	DeviceAppleTV:     regexp.MustCompile(`(?i)apple.*tv|tvos`),
	DeviceChromecast:  regexp.MustCompile(`(?i)chromecast|crkey|google\s*tv`),
	DeviceAndroidTV:   regexp.MustCompile(`(?i)android.*tv|android\s+tv`),
	DeviceSamsung:     regexp.MustCompile(`(?i)samsung.*smart.*tv|tizen`),
	DeviceLG:          regexp.MustCompile(`(?i)lg.*smart.*tv|webos.*tv|web0s`),
	DeviceVizio:       regexp.MustCompile(`(?i)vizio`),
	DeviceXbox:        regexp.MustCompile(`(?i)xbox`),
	DevicePlayStation: regexp.MustCompile(`(?i)playstation|ps[345]`),
}

// Make patterns for matching device make field
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

// Model patterns for matching device model field
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

// DeviceTypeConnectedTV is the OpenRTB device type value for CTV
const DeviceTypeConnectedTV = 3

// DetectDevice analyzes an OpenRTB device object and extracts CTV device information
func DetectDevice(device *openrtb.Device) *DeviceInfo {
	if device == nil {
		return &DeviceInfo{
			Type:  DeviceUnknown,
			IsCTV: false,
		}
	}

	info := &DeviceInfo{
		Model: device.Model,
		Make:  device.Make,
		OS:    device.OS,
		OSVer: device.OSV,
	}

	// Check User Agent first
	if device.UA != "" {
		if deviceType := detectFromUA(device.UA); deviceType != DeviceUnknown {
			info.Type = deviceType
			info.IsCTV = true
			return info
		}
	}

	// Check Make and Model
	if device.Make != "" || device.Model != "" {
		if deviceType := detectFromMakeModel(device.Make, device.Model); deviceType != DeviceUnknown {
			info.Type = deviceType
			info.IsCTV = true
			return info
		}
	}

	// Check DeviceType field (OpenRTB 2.5 spec: 3 = Connected TV)
	if device.DeviceType == DeviceTypeConnectedTV {
		info.IsCTV = true
		if info.Type == DeviceUnknown {
			info.Type = DeviceGeneric
		}
		return info
	}

	return info
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
					// No model, return based on make for known CTV makers
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

// IsCTV returns true if the device is identified as a Connected TV device
func IsCTV(device *openrtb.Device) bool {
	info := DetectDevice(device)
	return info.IsCTV
}

// GetDeviceType returns the specific CTV device type, or empty string if not CTV
func GetDeviceType(device *openrtb.Device) DeviceType {
	info := DetectDevice(device)
	return info.Type
}

// SupportsDevice checks if a device matches any of the specified CTV device types
func SupportsDevice(device *openrtb.Device, supportedDevices []DeviceType) bool {
	if len(supportedDevices) == 0 {
		// No restrictions, all devices supported
		return true
	}

	info := DetectDevice(device)
	if !info.IsCTV {
		return false
	}

	// Check if device type is in supported list
	for _, supported := range supportedDevices {
		if info.Type == supported {
			return true
		}
		// Generic CTV matches if in supported list
		if info.Type == DeviceGeneric && supported == DeviceGeneric {
			return true
		}
	}

	return false
}

// GetScreenSize returns categorized screen size based on device resolution
func GetScreenSize(device *openrtb.Device) string {
	if device == nil {
		return "unknown"
	}

	w := device.W
	h := device.H

	// Ensure we're comparing the larger dimension
	if h > w {
		w, h = h, w
	}

	switch {
	case w >= 3840:
		return "4k"
	case w >= 1920:
		return "1080p"
	case w >= 1280:
		return "720p"
	case w >= 854:
		return "480p"
	default:
		return "sd"
	}
}

// DeviceCapabilities represents capabilities of a CTV device
type DeviceCapabilities struct {
	SupportsVPAID    bool
	SupportsMRAID   bool
	SupportsOMID    bool
	MaxBitrate      int
	PreferredFormat string
}

// GetCapabilities returns estimated device capabilities based on device type
func GetCapabilities(deviceType DeviceType) DeviceCapabilities {
	switch deviceType {
	case DeviceAppleTV:
		return DeviceCapabilities{
			SupportsVPAID:    false,
			SupportsMRAID:   false,
			SupportsOMID:    true,
			MaxBitrate:      15000,
			PreferredFormat: "video/mp4",
		}
	case DeviceRoku:
		return DeviceCapabilities{
			SupportsVPAID:    false,
			SupportsMRAID:   false,
			SupportsOMID:    true,
			MaxBitrate:      8000,
			PreferredFormat: "video/mp4",
		}
	case DeviceFireTV:
		return DeviceCapabilities{
			SupportsVPAID:    false,
			SupportsMRAID:   false,
			SupportsOMID:    true,
			MaxBitrate:      10000,
			PreferredFormat: "video/mp4",
		}
	case DeviceChromecast, DeviceAndroidTV:
		return DeviceCapabilities{
			SupportsVPAID:    true,
			SupportsMRAID:   false,
			SupportsOMID:    true,
			MaxBitrate:      10000,
			PreferredFormat: "video/webm",
		}
	case DeviceSamsung, DeviceLG, DeviceVizio:
		return DeviceCapabilities{
			SupportsVPAID:    true,
			SupportsMRAID:   false,
			SupportsOMID:    true,
			MaxBitrate:      8000,
			PreferredFormat: "video/mp4",
		}
	case DeviceXbox, DevicePlayStation:
		return DeviceCapabilities{
			SupportsVPAID:    false,
			SupportsMRAID:   false,
			SupportsOMID:    false,
			MaxBitrate:      15000,
			PreferredFormat: "video/mp4",
		}
	default:
		return DeviceCapabilities{
			SupportsVPAID:    false,
			SupportsMRAID:   false,
			SupportsOMID:    false,
			MaxBitrate:      5000,
			PreferredFormat: "video/mp4",
		}
	}
}
