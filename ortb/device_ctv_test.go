package ortb

import (
	"testing"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"
)

func TestParseCTVDevice_UserAgent(t *testing.T) {
	tests := []struct {
		name           string
		device         *openrtb2.Device
		expectedType   CTVDeviceType
		expectedIsCTV  bool
	}{
		{
			name:          "Roku device from UA",
			device:        &openrtb2.Device{UA: "Roku/DVP-9.10 (519.10E04154A)"},
			expectedType:  CTVDeviceRoku,
			expectedIsCTV: true,
		},
		{
			name:          "Fire TV device from UA",
			device:        &openrtb2.Device{UA: "Mozilla/5.0 (Linux; Android 7.1.2; AFTMM) AppleWebKit/537.36"},
			expectedType:  CTVDeviceFireTV,
			expectedIsCTV: true,
		},
		{
			name:          "Amazon Fire TV explicit",
			device:        &openrtb2.Device{UA: "Amazon Fire TV"},
			expectedType:  CTVDeviceFireTV,
			expectedIsCTV: true,
		},
		{
			name:          "Apple TV from UA",
			device:        &openrtb2.Device{UA: "AppleTV11,1/tvOS 15.0"},
			expectedType:  CTVDeviceAppleTV,
			expectedIsCTV: true,
		},
		{
			name:          "tvOS device",
			device:        &openrtb2.Device{UA: "Mozilla/5.0 (tvOS; CPU iPhone OS 14_5 like Mac OS X)"},
			expectedType:  CTVDeviceAppleTV,
			expectedIsCTV: true,
		},
		{
			name:          "Chromecast from UA",
			device:        &openrtb2.Device{UA: "Mozilla/5.0 (X11; Linux armv7l) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.225 Safari/537.36 CrKey/1.56.500000"},
			expectedType:  CTVDeviceChromecast,
			expectedIsCTV: true,
		},
		{
			name:          "Google TV from UA",
			device:        &openrtb2.Device{UA: "Mozilla/5.0 (Linux; Android 10; Google TV) AppleWebKit/537.36"},
			expectedType:  CTVDeviceChromecast,
			expectedIsCTV: true,
		},
		{
			name:          "Android TV from UA",
			device:        &openrtb2.Device{UA: "Mozilla/5.0 (Linux; Android 9; Android TV) AppleWebKit/537.36"},
			expectedType:  CTVDeviceAndroidTV,
			expectedIsCTV: true,
		},
		{
			name:          "Samsung Smart TV from UA",
			device:        &openrtb2.Device{UA: "Mozilla/5.0 (SMART-TV; Linux; Tizen 5.0) AppleWebKit/537.36"},
			expectedType:  CTVDeviceSamsung,
			expectedIsCTV: true,
		},
		{
			name:          "Tizen TV from UA",
			device:        &openrtb2.Device{UA: "Mozilla/5.0 (Linux; Tizen 2.3) AppleWebKit/538.1"},
			expectedType:  CTVDeviceSamsung,
			expectedIsCTV: true,
		},
		{
			name:          "LG Smart TV from UA",
			device:        &openrtb2.Device{UA: "Mozilla/5.0 (Web0S; Linux/SmartTV) AppleWebKit/537.36"},
			expectedType:  CTVDeviceLG,
			expectedIsCTV: true,
		},
		{
			name:          "Vizio TV from UA",
			device:        &openrtb2.Device{UA: "Mozilla/5.0 (Linux; Vizio SmartCast) AppleWebKit/537.36"},
			expectedType:  CTVDeviceVizio,
			expectedIsCTV: true,
		},
		{
			name:          "Xbox from UA",
			device:        &openrtb2.Device{UA: "Mozilla/5.0 (Windows NT 10.0; Win64; x64; Xbox; Xbox One) AppleWebKit/537.36"},
			expectedType:  CTVDeviceXbox,
			expectedIsCTV: true,
		},
		{
			name:          "PlayStation from UA",
			device:        &openrtb2.Device{UA: "Mozilla/5.0 (PlayStation 4 5.55) AppleWebKit/601.2"},
			expectedType:  CTVDevicePlayStation,
			expectedIsCTV: true,
		},
		{
			name:          "PS5 from UA",
			device:        &openrtb2.Device{UA: "Mozilla/5.0 (PlayStation 5 3.00) AppleWebKit/605.1.15"},
			expectedType:  CTVDevicePlayStation,
			expectedIsCTV: true,
		},
		{
			name:          "Regular mobile device",
			device:        &openrtb2.Device{UA: "Mozilla/5.0 (iPhone; CPU iPhone OS 14_5 like Mac OS X)"},
			expectedType:  CTVDeviceUnknown,
			expectedIsCTV: false,
		},
		{
			name:          "Desktop browser",
			device:        &openrtb2.Device{UA: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"},
			expectedType:  CTVDeviceUnknown,
			expectedIsCTV: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := ParseCTVDevice(tt.device)
			assert.Equal(t, tt.expectedType, info.DeviceType, "Device type mismatch")
			assert.Equal(t, tt.expectedIsCTV, info.IsCTV, "IsCTV flag mismatch")
		})
	}
}

func TestParseCTVDevice_MakeModel(t *testing.T) {
	tests := []struct {
		name           string
		device         *openrtb2.Device
		expectedType   CTVDeviceType
		expectedIsCTV  bool
	}{
		{
			name:          "Roku by make and model",
			device:        &openrtb2.Device{Make: "Roku", Model: "Roku Ultra"},
			expectedType:  CTVDeviceRoku,
			expectedIsCTV: true,
		},
		{
			name:          "Fire TV by make and model",
			device:        &openrtb2.Device{Make: "Amazon", Model: "AFTMM"},
			expectedType:  CTVDeviceFireTV,
			expectedIsCTV: true,
		},
		{
			name:          "Apple TV by make and model",
			device:        &openrtb2.Device{Make: "Apple", Model: "AppleTV"},
			expectedType:  CTVDeviceAppleTV,
			expectedIsCTV: true,
		},
		{
			name:          "Samsung Smart TV by model",
			device:        &openrtb2.Device{Make: "Samsung", Model: "Smart TV"},
			expectedType:  CTVDeviceSamsung,
			expectedIsCTV: true,
		},
		{
			name:          "LG Smart TV by model",
			device:        &openrtb2.Device{Make: "LG", Model: "webOS Smart TV"},
			expectedType:  CTVDeviceLG,
			expectedIsCTV: true,
		},
		{
			name:          "Android TV by model",
			device:        &openrtb2.Device{Make: "Sony", Model: "Android TV"},
			expectedType:  CTVDeviceAndroidTV,
			expectedIsCTV: true,
		},
		{
			name:          "Xbox by make",
			device:        &openrtb2.Device{Make: "Microsoft", Model: "Xbox One"},
			expectedType:  CTVDeviceXbox,
			expectedIsCTV: true,
		},
		{
			name:          "PlayStation by make",
			device:        &openrtb2.Device{Make: "Sony", Model: "PlayStation 5"},
			expectedType:  CTVDevicePlayStation,
			expectedIsCTV: true,
		},
		{
			name:          "Regular phone",
			device:        &openrtb2.Device{Make: "Apple", Model: "iPhone 12"},
			expectedType:  CTVDeviceUnknown,
			expectedIsCTV: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := ParseCTVDevice(tt.device)
			assert.Equal(t, tt.expectedType, info.DeviceType, "Device type mismatch")
			assert.Equal(t, tt.expectedIsCTV, info.IsCTV, "IsCTV flag mismatch")
		})
	}
}

func TestParseCTVDevice_DeviceTypeField(t *testing.T) {
	deviceTypeConnectedTV := adcom1.DeviceType(3)
	deviceTypeMobile := adcom1.DeviceType(1)

	tests := []struct {
		name           string
		device         *openrtb2.Device
		expectedType   CTVDeviceType
		expectedIsCTV  bool
	}{
		{
			name: "DeviceType 3 (Connected TV) with Roku UA",
			device: &openrtb2.Device{
				DeviceType: deviceTypeConnectedTV,
				UA:         "Roku/DVP-9.10",
			},
			expectedType:  CTVDeviceRoku,
			expectedIsCTV: true,
		},
		{
			name: "DeviceType 3 (Connected TV) without specific identification",
			device: &openrtb2.Device{
				DeviceType: deviceTypeConnectedTV,
				UA:         "Some unknown TV browser",
			},
			expectedType:  CTVDeviceGeneric,
			expectedIsCTV: true,
		},
		{
			name: "DeviceType 1 (Mobile) should not be CTV",
			device: &openrtb2.Device{
				DeviceType: deviceTypeMobile,
				UA:         "Mozilla/5.0 (iPhone)",
			},
			expectedType:  CTVDeviceUnknown,
			expectedIsCTV: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := ParseCTVDevice(tt.device)
			assert.Equal(t, tt.expectedType, info.DeviceType, "Device type mismatch")
			assert.Equal(t, tt.expectedIsCTV, info.IsCTV, "IsCTV flag mismatch")
		})
	}
}

func TestParseCTVDevice_NilDevice(t *testing.T) {
	info := ParseCTVDevice(nil)
	assert.Equal(t, CTVDeviceUnknown, info.DeviceType)
	assert.False(t, info.IsCTV)
}

func TestIsCTVDevice(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCTVDevice(tt.device)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetCTVDeviceType(t *testing.T) {
	tests := []struct {
		name     string
		device   *openrtb2.Device
		expected CTVDeviceType
	}{
		{
			name:     "Roku device",
			device:   &openrtb2.Device{UA: "Roku/DVP-9.10"},
			expected: CTVDeviceRoku,
		},
		{
			name:     "Apple TV device",
			device:   &openrtb2.Device{UA: "AppleTV11,1/tvOS 15.0"},
			expected: CTVDeviceAppleTV,
		},
		{
			name:     "Non-CTV device",
			device:   &openrtb2.Device{UA: "Mozilla/5.0 (Windows NT 10.0)"},
			expected: CTVDeviceUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetCTVDeviceType(tt.device)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSupportsCTVDevice(t *testing.T) {
	tests := []struct {
		name             string
		device           *openrtb2.Device
		supportedDevices []CTVDeviceType
		expected         bool
	}{
		{
			name:             "Roku device in supported list",
			device:           &openrtb2.Device{UA: "Roku/DVP-9.10"},
			supportedDevices: []CTVDeviceType{CTVDeviceRoku, CTVDeviceFireTV},
			expected:         true,
		},
		{
			name:             "Roku device not in supported list",
			device:           &openrtb2.Device{UA: "Roku/DVP-9.10"},
			supportedDevices: []CTVDeviceType{CTVDeviceFireTV, CTVDeviceAppleTV},
			expected:         false,
		},
		{
			name:             "Empty supported list (all devices allowed)",
			device:           &openrtb2.Device{UA: "Roku/DVP-9.10"},
			supportedDevices: []CTVDeviceType{},
			expected:         true,
		},
		{
			name:             "Non-CTV device with supported list",
			device:           &openrtb2.Device{UA: "Mozilla/5.0 (iPhone)"},
			supportedDevices: []CTVDeviceType{CTVDeviceRoku},
			expected:         false,
		},
		{
			name:             "Generic CTV in supported list",
			device:           &openrtb2.Device{DeviceType: adcom1.DeviceType(3)},
			supportedDevices: []CTVDeviceType{CTVDeviceGeneric},
			expected:         true,
		},
		{
			name:             "Fire TV device matches",
			device:           &openrtb2.Device{Make: "Amazon", Model: "AFTMM"},
			supportedDevices: []CTVDeviceType{CTVDeviceFireTV},
			expected:         true,
		},
		{
			name:             "Apple TV matches",
			device:           &openrtb2.Device{UA: "tvOS 15.0"},
			supportedDevices: []CTVDeviceType{CTVDeviceAppleTV, CTVDeviceRoku},
			expected:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SupportsCTVDevice(tt.device, tt.supportedDevices)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectCTVFromUA(t *testing.T) {
	tests := []struct {
		name     string
		ua       string
		expected CTVDeviceType
	}{
		{
			name:     "Roku UA",
			ua:       "Roku/DVP-9.10 (519.10E04154A)",
			expected: CTVDeviceRoku,
		},
		{
			name:     "Case insensitive Roku",
			ua:       "ROKU Device",
			expected: CTVDeviceRoku,
		},
		{
			name:     "Fire TV - AFTMM",
			ua:       "Linux; Android 7.1.2; AFTMM",
			expected: CTVDeviceFireTV,
		},
		{
			name:     "Fire TV - AFTB",
			ua:       "Linux; Android 9; AFTB",
			expected: CTVDeviceFireTV,
		},
		{
			name:     "Apple TV",
			ua:       "AppleTV/tvOS",
			expected: CTVDeviceAppleTV,
		},
		{
			name:     "Android TV",
			ua:       "Android 10; Android TV",
			expected: CTVDeviceAndroidTV,
		},
		{
			name:     "Samsung Tizen",
			ua:       "Tizen 5.0",
			expected: CTVDeviceSamsung,
		},
		{
			name:     "LG webOS",
			ua:       "webOS TV 6.0",
			expected: CTVDeviceLG,
		},
		{
			name:     "Chromecast",
			ua:       "CrKey/1.56.500000",
			expected: CTVDeviceChromecast,
		},
		{
			name:     "Xbox",
			ua:       "Xbox One",
			expected: CTVDeviceXbox,
		},
		{
			name:     "PlayStation 4",
			ua:       "PlayStation 4 5.55",
			expected: CTVDevicePlayStation,
		},
		{
			name:     "PS3",
			ua:       "PLAYSTATION 3",
			expected: CTVDevicePlayStation,
		},
		{
			name:     "Unknown device",
			ua:       "Mozilla/5.0 (Windows NT 10.0)",
			expected: CTVDeviceUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectCTVFromUA(tt.ua)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectCTVFromMakeModel(t *testing.T) {
	tests := []struct {
		name     string
		make     string
		model    string
		expected CTVDeviceType
	}{
		{
			name:     "Roku make and model",
			make:     "Roku",
			model:    "Roku Ultra",
			expected: CTVDeviceRoku,
		},
		{
			name:     "Amazon Fire TV",
			make:     "Amazon",
			model:    "Fire TV Stick",
			expected: CTVDeviceFireTV,
		},
		{
			name:     "Apple TV",
			make:     "Apple",
			model:    "Apple TV 4K",
			expected: CTVDeviceAppleTV,
		},
		{
			name:     "Samsung Smart TV",
			make:     "Samsung",
			model:    "Smart TV",
			expected: CTVDeviceSamsung,
		},
		{
			name:     "LG Smart TV",
			make:     "LG",
			model:    "webOS Smart TV",
			expected: CTVDeviceLG,
		},
		{
			name:     "Android TV",
			make:     "Sony",
			model:    "Android TV",
			expected: CTVDeviceAndroidTV,
		},
		{
			name:     "Chromecast",
			make:     "Google",
			model:    "Chromecast",
			expected: CTVDeviceChromecast,
		},
		{
			name:     "Xbox",
			make:     "Microsoft",
			model:    "Xbox Series X",
			expected: CTVDeviceXbox,
		},
		{
			name:     "PlayStation",
			make:     "Sony",
			model:    "PlayStation 5",
			expected: CTVDevicePlayStation,
		},
		{
			name:     "Case insensitive matching",
			make:     "ROKU",
			model:    "ROKU ULTRA",
			expected: CTVDeviceRoku,
		},
		{
			name:     "iPhone (not CTV)",
			make:     "Apple",
			model:    "iPhone 12",
			expected: CTVDeviceUnknown,
		},
		{
			name:     "Empty make and model",
			make:     "",
			model:    "",
			expected: CTVDeviceUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectCTVFromMakeModel(tt.make, tt.model)
			assert.Equal(t, tt.expected, result)
		})
	}
}
