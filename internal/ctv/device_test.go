package ctv

import (
	"testing"

	"github.com/thenexusengine/tne_springwire/internal/openrtb"
)

func TestDetectDevice_UserAgent(t *testing.T) {
	tests := []struct {
		name         string
		ua           string
		expectedType DeviceType
		expectedCTV  bool
	}{
		{
			name:         "Roku device",
			ua:           "Roku/DVP-9.10 (519.10E04154A)",
			expectedType: DeviceRoku,
			expectedCTV:  true,
		},
		{
			name:         "Fire TV AFTMM",
			ua:           "Mozilla/5.0 (Linux; Android 7.1.2; AFTMM) AppleWebKit/537.36",
			expectedType: DeviceFireTV,
			expectedCTV:  true,
		},
		{
			name:         "Apple TV",
			ua:           "AppleTV11,1/tvOS 15.0",
			expectedType: DeviceAppleTV,
			expectedCTV:  true,
		},
		{
			name:         "Chromecast CrKey",
			ua:           "Mozilla/5.0 (X11; Linux armv7l) CrKey/1.56.500000",
			expectedType: DeviceChromecast,
			expectedCTV:  true,
		},
		{
			name:         "Android TV",
			ua:           "Mozilla/5.0 (Linux; Android 9; Android TV) AppleWebKit/537.36",
			expectedType: DeviceAndroidTV,
			expectedCTV:  true,
		},
		{
			name:         "Samsung Tizen",
			ua:           "Mozilla/5.0 (SMART-TV; Linux; Tizen 5.0) AppleWebKit/537.36",
			expectedType: DeviceSamsung,
			expectedCTV:  true,
		},
		{
			name:         "LG webOS",
			ua:           "Mozilla/5.0 (Web0S; Linux/SmartTV) AppleWebKit/537.36",
			expectedType: DeviceLG,
			expectedCTV:  true,
		},
		{
			name:         "Xbox",
			ua:           "Mozilla/5.0 (Windows NT 10.0; Win64; x64; Xbox; Xbox One) AppleWebKit/537.36",
			expectedType: DeviceXbox,
			expectedCTV:  true,
		},
		{
			name:         "PlayStation",
			ua:           "Mozilla/5.0 (PlayStation 4 5.55) AppleWebKit/601.2",
			expectedType: DevicePlayStation,
			expectedCTV:  true,
		},
		{
			name:         "Regular mobile",
			ua:           "Mozilla/5.0 (iPhone; CPU iPhone OS 14_5 like Mac OS X)",
			expectedType: DeviceUnknown,
			expectedCTV:  false,
		},
		{
			name:         "Desktop browser",
			ua:           "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			expectedType: DeviceUnknown,
			expectedCTV:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			device := &openrtb.Device{UA: tt.ua}
			info := DetectDevice(device)

			if info.Type != tt.expectedType {
				t.Errorf("Expected type %s, got %s", tt.expectedType, info.Type)
			}
			if info.IsCTV != tt.expectedCTV {
				t.Errorf("Expected IsCTV %v, got %v", tt.expectedCTV, info.IsCTV)
			}
		})
	}
}

func TestDetectDevice_MakeModel(t *testing.T) {
	tests := []struct {
		name         string
		make         string
		model        string
		expectedType DeviceType
		expectedCTV  bool
	}{
		{
			name:         "Roku by make/model",
			make:         "Roku",
			model:        "Roku Ultra",
			expectedType: DeviceRoku,
			expectedCTV:  true,
		},
		{
			name:         "Amazon Fire TV",
			make:         "Amazon",
			model:        "AFTMM",
			expectedType: DeviceFireTV,
			expectedCTV:  true,
		},
		{
			name:         "Apple TV",
			make:         "Apple",
			model:        "AppleTV",
			expectedType: DeviceAppleTV,
			expectedCTV:  true,
		},
		{
			name:         "Samsung Smart TV",
			make:         "Samsung",
			model:        "Smart TV",
			expectedType: DeviceSamsung,
			expectedCTV:  true,
		},
		{
			name:         "iPhone (not CTV)",
			make:         "Apple",
			model:        "iPhone 12",
			expectedType: DeviceUnknown,
			expectedCTV:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			device := &openrtb.Device{Make: tt.make, Model: tt.model}
			info := DetectDevice(device)

			if info.Type != tt.expectedType {
				t.Errorf("Expected type %s, got %s", tt.expectedType, info.Type)
			}
			if info.IsCTV != tt.expectedCTV {
				t.Errorf("Expected IsCTV %v, got %v", tt.expectedCTV, info.IsCTV)
			}
		})
	}
}

func TestDetectDevice_DeviceType(t *testing.T) {
	// DeviceType = 3 is Connected TV in OpenRTB
	device := &openrtb.Device{
		DeviceType: DeviceTypeConnectedTV,
		UA:         "Unknown TV Browser",
	}
	info := DetectDevice(device)

	if !info.IsCTV {
		t.Error("Expected IsCTV true for DeviceType=3")
	}
	if info.Type != DeviceGeneric {
		t.Errorf("Expected type %s, got %s", DeviceGeneric, info.Type)
	}
}

func TestDetectDevice_Nil(t *testing.T) {
	info := DetectDevice(nil)

	if info.IsCTV {
		t.Error("Expected IsCTV false for nil device")
	}
	if info.Type != DeviceUnknown {
		t.Errorf("Expected type %s, got %s", DeviceUnknown, info.Type)
	}
}

func TestIsCTV(t *testing.T) {
	tests := []struct {
		name     string
		device   *openrtb.Device
		expected bool
	}{
		{
			name:     "Roku",
			device:   &openrtb.Device{UA: "Roku/DVP-9.10"},
			expected: true,
		},
		{
			name:     "iPhone",
			device:   &openrtb.Device{UA: "Mozilla/5.0 (iPhone)"},
			expected: false,
		},
		{
			name:     "Nil device",
			device:   nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCTV(tt.device)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestSupportsDevice(t *testing.T) {
	tests := []struct {
		name      string
		device    *openrtb.Device
		supported []DeviceType
		expected  bool
	}{
		{
			name:      "Roku in supported list",
			device:    &openrtb.Device{UA: "Roku/DVP-9.10"},
			supported: []DeviceType{DeviceRoku, DeviceFireTV},
			expected:  true,
		},
		{
			name:      "Roku not in supported list",
			device:    &openrtb.Device{UA: "Roku/DVP-9.10"},
			supported: []DeviceType{DeviceFireTV, DeviceAppleTV},
			expected:  false,
		},
		{
			name:      "Empty supported list (all allowed)",
			device:    &openrtb.Device{UA: "Roku/DVP-9.10"},
			supported: []DeviceType{},
			expected:  true,
		},
		{
			name:      "Non-CTV device",
			device:    &openrtb.Device{UA: "Mozilla/5.0 (iPhone)"},
			supported: []DeviceType{DeviceRoku},
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SupportsDevice(tt.device, tt.supported)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGetScreenSize(t *testing.T) {
	tests := []struct {
		name     string
		w, h     int
		expected string
	}{
		{"4K", 3840, 2160, "4k"},
		{"1080p", 1920, 1080, "1080p"},
		{"720p", 1280, 720, "720p"},
		{"480p", 854, 480, "480p"},
		{"SD", 640, 480, "sd"},
		{"Portrait 1080p", 1080, 1920, "1080p"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			device := &openrtb.Device{W: tt.w, H: tt.h}
			result := GetScreenSize(device)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGetCapabilities(t *testing.T) {
	tests := []struct {
		deviceType DeviceType
		vpaid      bool
	}{
		{DeviceAppleTV, false},
		{DeviceRoku, false},
		{DeviceChromecast, true},
		{DeviceAndroidTV, true},
		{DeviceSamsung, true},
		{DeviceXbox, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.deviceType), func(t *testing.T) {
			caps := GetCapabilities(tt.deviceType)
			if caps.SupportsVPAID != tt.vpaid {
				t.Errorf("Expected VPAID support %v, got %v", tt.vpaid, caps.SupportsVPAID)
			}
			if caps.MaxBitrate <= 0 {
				t.Error("Expected positive MaxBitrate")
			}
			if caps.PreferredFormat == "" {
				t.Error("Expected non-empty PreferredFormat")
			}
		})
	}
}
