package ctv

import (
	"context"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/filterpipeline"
)

// DeviceFilter is a post-filter that filters bids based on CTV device support.
// It can be configured to only allow bids from specific CTV device types.
type DeviceFilter struct {
	enabled          bool
	priority         int
	supportedDevices []DeviceType
	rejectNonCTV     bool
}

// DeviceFilterConfig holds configuration for the CTV device filter
type DeviceFilterConfig struct {
	// Enabled controls whether the filter is active
	Enabled bool
	// Priority determines filter execution order (lower values execute first)
	Priority int
	// SupportedDevices lists allowed CTV device types. Empty list allows all CTV devices.
	SupportedDevices []DeviceType
	// RejectNonCTV when true, rejects all non-CTV traffic
	RejectNonCTV bool
}

// NewDeviceFilter creates a new CTV device filter with the given configuration
func NewDeviceFilter(config DeviceFilterConfig) *DeviceFilter {
	return &DeviceFilter{
		enabled:          config.Enabled,
		priority:         config.Priority,
		supportedDevices: config.SupportedDevices,
		rejectNonCTV:     config.RejectNonCTV,
	}
}

// Name returns the unique identifier for this filter
func (f *DeviceFilter) Name() string {
	return "ctv_device_filter"
}

// Priority returns the execution priority (lower values execute first)
func (f *DeviceFilter) Priority() int {
	return f.priority
}

// Enabled determines if this filter should be executed
func (f *DeviceFilter) Enabled(accountID string) bool {
	return f.enabled
}

// Execute processes the bid response and filters bids based on device support
func (f *DeviceFilter) Execute(ctx context.Context, req filterpipeline.PostFilterRequest) (filterpipeline.PostFilterResponse, error) {
	resp := filterpipeline.PostFilterResponse{
		Response: req.Response,
		Metadata: make(map[string]interface{}),
	}

	// Get device from request
	device := req.Request.BidRequest.Device

	// Check if device is CTV
	deviceInfo := ParseDevice(device)

	// If RejectNonCTV is enabled and device is not CTV, reject entire response
	if f.rejectNonCTV && !deviceInfo.IsCTV {
		resp.Reject = true
		resp.RejectReason = "non-ctv device not supported"
		resp.Metadata["device_type"] = string(deviceInfo.DeviceType)
		resp.Metadata["is_ctv"] = deviceInfo.IsCTV
		return resp, nil
	}

	// If we have device restrictions and device is CTV
	if len(f.supportedDevices) > 0 && deviceInfo.IsCTV {
		// Check if device is supported
		if !SupportsDevice(device, f.supportedDevices) {
			resp.Reject = true
			resp.RejectReason = "ctv device type not supported"
			resp.Metadata["device_type"] = string(deviceInfo.DeviceType)
			resp.Metadata["supported_devices"] = f.supportedDevices
			return resp, nil
		}
	}

	// Add device metadata
	resp.Metadata["device_type"] = string(deviceInfo.DeviceType)
	resp.Metadata["is_ctv"] = deviceInfo.IsCTV
	resp.Metadata["device_make"] = deviceInfo.Make
	resp.Metadata["device_model"] = deviceInfo.Model

	return resp, nil
}

// BidFilter provides bid-level filtering based on device support.
// This is a utility function that can be used outside the filter pipeline
// to filter individual bids based on device criteria.
type BidFilter struct {
	supportedDevices []DeviceType
	rejectNonCTV     bool
}

// NewBidFilter creates a new bid filter with device restrictions
func NewBidFilter(supportedDevices []DeviceType, rejectNonCTV bool) *BidFilter {
	return &BidFilter{
		supportedDevices: supportedDevices,
		rejectNonCTV:     rejectNonCTV,
	}
}

// ShouldIncludeBid determines if a bid should be included based on device criteria
func (bf *BidFilter) ShouldIncludeBid(device *openrtb2.Device) bool {
	deviceInfo := ParseDevice(device)

	// Reject if non-CTV and we require CTV
	if bf.rejectNonCTV && !deviceInfo.IsCTV {
		return false
	}

	// If we have device restrictions and device is CTV
	if len(bf.supportedDevices) > 0 && deviceInfo.IsCTV {
		return SupportsDevice(device, bf.supportedDevices)
	}

	// No restrictions, include bid
	return true
}

// FilterBidResponse filters bids in a response based on device criteria
func (bf *BidFilter) FilterBidResponse(response *openrtb2.BidResponse, device *openrtb2.Device) *openrtb2.BidResponse {
	if response == nil {
		return response
	}

	// Check if we should filter at all
	if !bf.ShouldIncludeBid(device) {
		// Clear all seat bids
		response.SeatBid = nil
		return response
	}

	// If device passes filter, return as-is
	return response
}
