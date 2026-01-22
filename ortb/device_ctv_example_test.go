package ortb

import (
	"fmt"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
)

// Example demonstrates how to detect CTV devices in OpenRTB requests
func ExampleParseCTVDevice() {
	// Example OpenRTB device from a Roku request
	device := &openrtb2.Device{
		UA:    "Roku/DVP-9.10 (519.10E04154A)",
		Make:  "Roku",
		Model: "Roku Ultra",
	}

	info := ParseCTVDevice(device)

	fmt.Printf("Is CTV: %v\n", info.IsCTV)
	fmt.Printf("Device Type: %s\n", info.DeviceType)
	// Output:
	// Is CTV: true
	// Device Type: roku
}

// Example demonstrates checking if a device is CTV
func ExampleIsCTVDevice() {
	device := &openrtb2.Device{
		UA: "AppleTV11,1/tvOS 15.0",
	}

	isCTV := IsCTVDevice(device)
	fmt.Printf("Is CTV: %v\n", isCTV)
	// Output:
	// Is CTV: true
}

// Example demonstrates filtering bids by supported CTV devices
func ExampleSupportsCTVDevice() {
	// Bidder only supports Roku and Fire TV
	supportedDevices := []CTVDeviceType{
		CTVDeviceRoku,
		CTVDeviceFireTV,
	}

	// Test with Roku device
	rokuDevice := &openrtb2.Device{
		UA: "Roku/DVP-9.10",
	}

	// Test with Apple TV device
	appleTVDevice := &openrtb2.Device{
		UA: "AppleTV11,1/tvOS 15.0",
	}

	fmt.Printf("Roku supported: %v\n", SupportsCTVDevice(rokuDevice, supportedDevices))
	fmt.Printf("Apple TV supported: %v\n", SupportsCTVDevice(appleTVDevice, supportedDevices))
	// Output:
	// Roku supported: true
	// Apple TV supported: false
}

// Example demonstrates how to use CTV detection in bid request processing
func ExampleParseCTVDevice_bidRequestProcessing() {
	// Simulate processing a bid request
	bidRequest := &openrtb2.BidRequest{
		Device: &openrtb2.Device{
			UA:    "Mozilla/5.0 (Linux; Android 9; AFTMM) AppleWebKit/537.36",
			Make:  "Amazon",
			Model: "Fire TV Stick 4K",
		},
	}

	// Parse CTV device info
	ctvInfo := ParseCTVDevice(bidRequest.Device)

	if ctvInfo.IsCTV {
		switch ctvInfo.DeviceType {
		case CTVDeviceRoku:
			fmt.Println("Apply Roku-specific targeting")
		case CTVDeviceFireTV:
			fmt.Println("Apply Fire TV-specific targeting")
		case CTVDeviceAppleTV:
			fmt.Println("Apply Apple TV-specific targeting")
		default:
			fmt.Println("Apply generic CTV targeting")
		}
	} else {
		fmt.Println("Not a CTV device")
	}
	// Output:
	// Apply Fire TV-specific targeting
}

// Example demonstrates multi-device support checking
func ExampleSupportsCTVDevice_multipleDevices() {
	// Create a list of devices to check
	devices := []*openrtb2.Device{
		{UA: "Roku/DVP-9.10"},
		{UA: "AppleTV11,1/tvOS 15.0"},
		{UA: "Mozilla/5.0 (iPhone)"},
		{Make: "Amazon", Model: "Fire TV Stick"},
	}

	// Define supported devices
	supportedDevices := []CTVDeviceType{
		CTVDeviceRoku,
		CTVDeviceFireTV,
		CTVDeviceAppleTV,
	}

	// Check each device
	for i, device := range devices {
		supported := SupportsCTVDevice(device, supportedDevices)
		deviceType := GetCTVDeviceType(device)
		fmt.Printf("Device %d - Type: %s, Supported: %v\n", i+1, deviceType, supported)
	}
	// Output:
	// Device 1 - Type: roku, Supported: true
	// Device 2 - Type: appletv, Supported: true
	// Device 3 - Type: , Supported: false
	// Device 4 - Type: firetv, Supported: true
}

// Example demonstrates using OpenRTB devicetype field
func ExampleParseCTVDevice_deviceTypeField() {
	deviceTypeCTV := adcom1.DeviceType(3) // OpenRTB 2.5: 3 = Connected TV

	// Device with devicetype but no specific UA
	device := &openrtb2.Device{
		DeviceType: deviceTypeCTV,
		UA:         "Some unknown CTV browser",
	}

	info := ParseCTVDevice(device)

	fmt.Printf("Is CTV: %v\n", info.IsCTV)
	fmt.Printf("Device Type: %s\n", info.DeviceType)
	// Output:
	// Is CTV: true
	// Device Type: ctv
}

// Example demonstrates device detection priority
func ExampleParseCTVDevice_detectionPriority() {
	// Device with multiple signals
	device := &openrtb2.Device{
		UA:    "Roku/DVP-9.10 (519.10E04154A)",
		Make:  "Roku",
		Model: "Roku Ultra",
	}

	info := ParseCTVDevice(device)

	// UA is checked first, then Make/Model, then DeviceType field
	fmt.Printf("Detected from UA: %s\n", info.DeviceType)
	fmt.Printf("Make: %s\n", info.Make)
	fmt.Printf("Model: %s\n", info.Model)
	// Output:
	// Detected from UA: roku
	// Make: Roku
	// Model: Roku Ultra
}
