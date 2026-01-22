package ctv

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"
)

func TestParseDevice_UserAgent(t *testing.T) {
	tests := []struct {
		name           string
		device         *openrtb2.Device
		expectedType   DeviceType
		expectedIsCTV  bool
	}{
		{
			name:           "Roku device from UA",
			device:         &openrtb2.Device{UA: "Roku/DVP-9.10 (519.10E04154A)"},
			expectedType:   DeviceRoku,
			expectedIsCTV:  true,
		},
		{
			name:           "Fire TV device from UA",
			device:         &openrtb2.Device{UA: "Mozilla/5.0 (Linux; Android 7.1.2; AFTMM) AppleWebKit/537.36"},
			expectedType:   DeviceFireTV,
			expectedIsCTV:  true,
		},
		{
			name:           "Amazon Fire TV explicit",
			device:         &openrtb2.Device{UA: "Amazon Fire TV"},
			expectedType:   DeviceFireTV,
			expectedIsCTV:  true,
		},
		{
			name:           "Apple TV from UA",
			device:         &openrtb2.Device{UA: "AppleTV11,1/tvOS 15.0"},
			expectedType:   DeviceAppleTV,
			expectedIsCTV:  true,
		},
		{
			name:           "tvOS device",
			device:         &openrtb2.Device{UA: "Mozilla/5.0 (tvOS; CPU iPhone OS 14_5 like Mac OS X)"},
			expectedType:   DeviceAppleTV,
			expectedIsCTV:  true,
		},
		{
			name:           "Chromecast from UA",
			device:         &openrtb2.Device{UA: "Mozilla/5.0 (X11; Linux armv7l) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.225 Safari/537.36 CrKey/1.56.500000"},
			expectedType:   DeviceChromecast,
			expectedIsCTV:  true,
		},
		{
			name:           "Google TV from UA",
			device:         &openrtb2.Device{UA: "Mozilla/5.0 (Linux; Android 10; Google TV) AppleWebKit/537.36"},
			expectedType:   DeviceChromecast,
			expectedIsCTV:  true,
		},
		{
			name:           "Android TV from UA",
			device:         &openrtb2.Device{UA: "Mozilla/5.0 (Linux; Android 9; Android TV) AppleWebKit/537.36"},
			expectedType:   DeviceAndroidTV,
			expectedIsCTV:  true,
		},
		{
			name:           "Samsung Smart TV from UA",
			device:         &openrtb2.Device{UA: "Mozilla/5.0 (SMART-TV; Linux; Tizen 5.0) AppleWebKit/537.36"},
			expectedType:   DeviceSamsung,
			expectedIsCTV:  true,
		},
		{
			name:           "Tizen TV from UA",
			device:         &openrtb2.Device{UA: "Mozilla/5.0 (Linux; Tizen 2.3) AppleWebKit/538.1"},
			expectedType:   DeviceSamsung,
			expectedIsCTV:  true,
		},
		{
			name:           "LG Smart TV from UA",
			device:         &openrtb2.Device{UA: "Mozilla/5.0 (Web0S; Linux/SmartTV) AppleWebKit/537.36"},
			expectedType:   DeviceLG,
			expectedIsCTV:  true,
		},
		{
			name:           "Vizio TV from UA",
			device:         &openrtb2.Device{UA: "Mozilla/5.0 (Linux; Vizio SmartCast) AppleWebKit/537.36"},
			expectedType:   DeviceVizio,
			expectedIsCTV:  true,
		},
		{
			name:           "Xbox from UA",
			device:         &openrtb2.Device{UA: "Mozilla/5.0 (Windows NT 10.0; Win64; x64; Xbox; Xbox One) AppleWebKit/537.36"},
			expectedType:   DeviceXbox,
			expectedIsCTV:  true,
		},
		{
			name:           "PlayStation from UA",
			device:         &openrtb2.Device{UA: "Mozilla/5.0 (PlayStation 4 5.55) AppleWebKit/601.2"},
			expectedType:   DevicePlayStation,
			expectedIsCTV:  true,
		},
		{
			name:           "PS5 from UA",
			device:         &openrtb2.Device{UA: "Mozilla/5.0 (PlayStation 5 3.00) AppleWebKit/605.1.15"},
			expectedType:   DevicePlayStation,
			expectedIsCTV:  true,
		},
		{
			name:           "Regular mobile device",
			device:         &openrtb2.Device{UA: "Mozilla/5.0 (iPhone; CPU iPhone OS 14_5 like Mac OS X)"},
			expectedType:   DeviceUnknown,
			expectedIsCTV:  false,
		},
		{
			name:           "Desktop browser",
			device:         &openrtb2.Device{UA: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"},
			expectedType:   DeviceUnknown,
			expectedIsCTV:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := ParseDevice(tt.device)
			assert.Equal(t, tt.expectedType, info.DeviceType, "Device type mismatch")
			assert.Equal(t, tt.expectedIsCTV, info.IsCTV, "IsCTV flag mismatch")
		})
	}
}

func TestParseDevice_MakeModel(t *testing.T) {
	tests := []struct {
		name           string
		device         *openrtb2.Device
		expectedType   DeviceType
		expectedIsCTV  bool
	}{
		{
			name:           "Roku by make and model",
			device:         &openrtb2.Device{Make: "Roku", Model: "Roku Ultra"},
			expectedType:   DeviceRoku,
			expectedIsCTV:  true,
		},
		{
			name:           "Fire TV by make and model",
			device:         &openrtb2.Device{Make: "Amazon", Model: "AFTMM"},
			expectedType:   DeviceFireTV,
			expectedIsCTV:  true,
		},
		{
			name:           "Apple TV by make and model",
			device:         &openrtb2.Device{Make: "Apple", Model: "AppleTV"},
			expectedType:   DeviceAppleTV,
			expectedIsCTV:  true,
		},
		{
			name:           "Samsung Smart TV by model",
			device:         &openrtb2.Device{Make: "Samsung", Model: "Smart TV"},
			expectedType:   DeviceSamsung,
			expectedIsCTV:  true,
		},
		{
			name:           "LG Smart TV by model",
			device:         &openrtb2.Device{Make: "LG", Model: "webOS Smart TV"},
			expectedType:   DeviceLG,
			expectedIsCTV:  true,
		},
		{
			name:           "Android TV by model",
			device:         &openrtb2.Device{Make: "Sony", Model: "Android TV"},
			expectedType:   DeviceAndroidTV,
			expectedIsCTV:  true,
		},
		{
			name:           "Xbox by make",
			device:         &openrtb2.Device{Make: "Microsoft", Model: "Xbox One"},
			expectedType:   DeviceXbox,
			expectedIsCTV:  true,
		},
		{
			name:           "PlayStation by make",
			device:         &openrtb2.Device{Make: "Sony", Model: "PlayStation 5"},
			expectedType:   DevicePlayStation,
			expectedIsCTV:  true,
		},
		{
			name:           "Regular phone",
			device:         &openrtb2.Device{Make: "Apple", Model: "iPhone 12"},
			expectedType:   DeviceUnknown,
			expectedIsCTV:  false,
		},
		{
			name:           "Chromecast by make and model",
			device:         &openrtb2.Device{Make: "Google", Model: "Chromecast"},
			expectedType:   DeviceChromecast,
			expectedIsCTV:  true,
		},
		{
			name:           "Vizio by make",
			device:         &openrtb2.Device{Make: "Vizio", Model: "D-Series"},
			expectedType:   DeviceVizio,
			expectedIsCTV:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := ParseDevice(tt.device)
			assert.Equal(t, tt.expectedType, info.DeviceType, "Device type mismatch")
			assert.Equal(t, tt.expectedIsCTV, info.IsCTV, "IsCTV flag mismatch")
		})
	}
}

func TestParseDevice_DeviceTypeField(t *testing.T) {
	deviceTypeConnectedTV := int8(3)
	deviceTypeMobile := int8(1)

	tests := []struct {
		name           string
		device         *openrtb2.Device
		expectedType   DeviceType
		expectedIsCTV  bool
	}{
		{
			name: "DeviceType 3 (Connected TV) with Roku UA",
			device: &openrtb2.Device{
				DeviceType: &deviceTypeConnectedTV,
				UA:         "Roku/DVP-9.10",
			},
			expectedType:  DeviceRoku,
			expectedIsCTV: true,
		},
		{
			name: "DeviceType 3 (Connected TV) without specific identification",
			device: &openrtb2.Device{
				DeviceType: &deviceTypeConnectedTV,
				UA:         "Some unknown TV browser",
			},
			expectedType:  DeviceGeneric,
			expectedIsCTV: true,
		},
		{
			name: "DeviceType 1 (Mobile) should not be CTV",
			device: &openrtb2.Device{
				DeviceType: &deviceTypeMobile,
				UA:         "Mozilla/5.0 (iPhone)",
			},
			expectedType:  DeviceUnknown,
			expectedIsCTV: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := ParseDevice(tt.device)
			assert.Equal(t, tt.expectedType, info.DeviceType, "Device type mismatch")
			assert.Equal(t, tt.expectedIsCTV, info.IsCTV, "IsCTV flag mismatch")
		})
	}
}

func TestParseDevice_NilDevice(t *testing.T) {
	info := ParseDevice(nil)
	assert.Equal(t, DeviceUnknown, info.DeviceType)
	assert.False(t, info.IsCTV)
}

func TestIsCTV(t *testing.T) {
	tests := []struct {
		name     string
		device   *openrtb2.Device
		expected bool
	}{
		{
			name:     "Roku device",
			device:   &openrtb2.Device{UA: "Roku/DVP-9.10"},
			expected: true,
		},
		{
			name:     "Fire TV device",
			device:   &openrtb2.Device{Make: "Amazon", Model: "Fire TV Stick"},
			expected: true,
		},
		{
			name:     "iPhone",
			device:   &openrtb2.Device{UA: "Mozilla/5.0 (iPhone)"},
			expected: false,
		},
		{
			name:     "Nil device",
			device:   nil,
			expected: false,
		},
		{
			name:     "Generic CTV by DeviceType field",
			device:   &openrtb2.Device{DeviceType: func() *int8 { v := int8(3); return &v }()},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCTV(tt.device)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetDeviceType(t *testing.T) {
	tests := []struct {
		name     string
		device   *openrtb2.Device
		expected DeviceType
	}{
		{
			name:     "Roku device",
			device:   &openrtb2.Device{UA: "Roku/DVP-9.10"},
			expected: DeviceRoku,
		},
		{
			name:     "Apple TV device",
			device:   &openrtb2.Device{UA: "AppleTV11,1/tvOS 15.0"},
			expected: DeviceAppleTV,
		},
		{
			name:     "Non-CTV device",
			device:   &openrtb2.Device{UA: "Mozilla/5.0 (Windows NT 10.0)"},
			expected: DeviceUnknown,
		},
		{
			name:     "Fire TV by make/model",
			device:   &openrtb2.Device{Make: "Amazon", Model: "Fire TV"},
			expected: DeviceFireTV,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetDeviceType(tt.device)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSupportsDevice(t *testing.T) {
	tests := []struct {
		name             string
		device           *openrtb2.Device
		supportedDevices []DeviceType
		expected         bool
	}{
		{
			name:             "Roku device in supported list",
			device:           &openrtb2.Device{UA: "Roku/DVP-9.10"},
			supportedDevices: []DeviceType{DeviceRoku, DeviceFireTV},
			expected:         true,
		},
		{
			name:             "Roku device not in supported list",
			device:           &openrtb2.Device{UA: "Roku/DVP-9.10"},
			supportedDevices: []DeviceType{DeviceFireTV, DeviceAppleTV},
			expected:         false,
		},
		{
			name:             "Empty supported list (all devices allowed)",
			device:           &openrtb2.Device{UA: "Roku/DVP-9.10"},
			supportedDevices: []DeviceType{},
			expected:         true,
		},
		{
			name:             "Non-CTV device with supported list",
			device:           &openrtb2.Device{UA: "Mozilla/5.0 (iPhone)"},
			supportedDevices: []DeviceType{DeviceRoku},
			expected:         false,
		},
		{
			name:             "Generic CTV in supported list",
			device:           &openrtb2.Device{DeviceType: func() *int8 { v := int8(3); return &v }()},
			supportedDevices: []DeviceType{DeviceGeneric},
			expected:         true,
		},
		{
			name:             "Fire TV device matches",
			device:           &openrtb2.Device{Make: "Amazon", Model: "AFTMM"},
			supportedDevices: []DeviceType{DeviceFireTV},
			expected:         true,
		},
		{
			name:             "Apple TV matches",
			device:           &openrtb2.Device{UA: "tvOS 15.0"},
			supportedDevices: []DeviceType{DeviceAppleTV, DeviceRoku},
			expected:         true,
		},
		{
			name:             "Samsung not in supported list",
			device:           &openrtb2.Device{UA: "Tizen 5.0"},
			supportedDevices: []DeviceType{DeviceRoku, DeviceFireTV, DeviceAppleTV},
			expected:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SupportsDevice(tt.device, tt.supportedDevices)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectFromUA(t *testing.T) {
	tests := []struct {
		name     string
		ua       string
		expected DeviceType
	}{
		{
			name:     "Roku UA",
			ua:       "Roku/DVP-9.10 (519.10E04154A)",
			expected: DeviceRoku,
		},
		{
			name:     "Case insensitive Roku",
			ua:       "ROKU Device",
			expected: DeviceRoku,
		},
		{
			name:     "Fire TV - AFTMM",
			ua:       "Linux; Android 7.1.2; AFTMM",
			expected: DeviceFireTV,
		},
		{
			name:     "Fire TV - AFTB",
			ua:       "Linux; Android 9; AFTB",
			expected: DeviceFireTV,
		},
		{
			name:     "Apple TV",
			ua:       "AppleTV/tvOS",
			expected: DeviceAppleTV,
		},
		{
			name:     "Android TV",
			ua:       "Android 10; Android TV",
			expected: DeviceAndroidTV,
		},
		{
			name:     "Samsung Tizen",
			ua:       "Tizen 5.0",
			expected: DeviceSamsung,
		},
		{
			name:     "LG webOS",
			ua:       "webOS TV 6.0",
			expected: DeviceLG,
		},
		{
			name:     "Chromecast",
			ua:       "CrKey/1.56.500000",
			expected: DeviceChromecast,
		},
		{
			name:     "Xbox",
			ua:       "Xbox One",
			expected: DeviceXbox,
		},
		{
			name:     "PlayStation 4",
			ua:       "PlayStation 4 5.55",
			expected: DevicePlayStation,
		},
		{
			name:     "PS3",
			ua:       "PLAYSTATION 3",
			expected: DevicePlayStation,
		},
		{
			name:     "Unknown device",
			ua:       "Mozilla/5.0 (Windows NT 10.0)",
			expected: DeviceUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectFromUA(tt.ua)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectFromMakeModel(t *testing.T) {
	tests := []struct {
		name     string
		make     string
		model    string
		expected DeviceType
	}{
		{
			name:     "Roku make and model",
			make:     "Roku",
			model:    "Roku Ultra",
			expected: DeviceRoku,
		},
		{
			name:     "Amazon Fire TV",
			make:     "Amazon",
			model:    "Fire TV Stick",
			expected: DeviceFireTV,
		},
		{
			name:     "Apple TV",
			make:     "Apple",
			model:    "Apple TV 4K",
			expected: DeviceAppleTV,
		},
		{
			name:     "Samsung Smart TV",
			make:     "Samsung",
			model:    "Smart TV",
			expected: DeviceSamsung,
		},
		{
			name:     "LG Smart TV",
			make:     "LG",
			model:    "webOS Smart TV",
			expected: DeviceLG,
		},
		{
			name:     "Android TV",
			make:     "Sony",
			model:    "Android TV",
			expected: DeviceAndroidTV,
		},
		{
			name:     "Chromecast",
			make:     "Google",
			model:    "Chromecast",
			expected: DeviceChromecast,
		},
		{
			name:     "Xbox",
			make:     "Microsoft",
			model:    "Xbox Series X",
			expected: DeviceXbox,
		},
		{
			name:     "PlayStation",
			make:     "Sony",
			model:    "PlayStation 5",
			expected: DevicePlayStation,
		},
		{
			name:     "Case insensitive matching",
			make:     "ROKU",
			model:    "ROKU ULTRA",
			expected: DeviceRoku,
		},
		{
			name:     "iPhone (not CTV)",
			make:     "Apple",
			model:    "iPhone 12",
			expected: DeviceUnknown,
		},
		{
			name:     "Empty make and model",
			make:     "",
			model:    "",
			expected: DeviceUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectFromMakeModel(tt.make, tt.model)
			assert.Equal(t, tt.expected, result)
		})
	}
}
