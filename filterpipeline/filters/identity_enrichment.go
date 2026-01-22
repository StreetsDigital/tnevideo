package filters

import (
	"context"
	"fmt"

	"github.com/prebid/prebid-server/v3/filterpipeline"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// IdentityEnrichmentFilter adds identity data to requests in the pre-filter stage.
type IdentityEnrichmentFilter struct {
	enabled  bool
	priority int
}

// NewIdentityEnrichmentFilter creates a new identity enrichment filter.
func NewIdentityEnrichmentFilter(enabled bool, priority int) *IdentityEnrichmentFilter {
	return &IdentityEnrichmentFilter{
		enabled:  enabled,
		priority: priority,
	}
}

// Name returns the filter identifier.
func (f *IdentityEnrichmentFilter) Name() string {
	return "identity_enrichment"
}

// Execute enriches the request with identity information.
func (f *IdentityEnrichmentFilter) Execute(ctx context.Context, req filterpipeline.PreFilterRequest) (filterpipeline.PreFilterResponse, error) {
	resp := filterpipeline.PreFilterResponse{
		Request:  req.Request,
		Metadata: make(map[string]interface{}),
	}

	// Check if request has a user object
	if req.Request == nil || req.Request.BidRequest == nil {
		resp.Errors = append(resp.Errors, "missing bid request")
		return resp, nil
	}

	bidReq := req.Request.BidRequest

	// Add identity enrichment metadata
	identityData := map[string]interface{}{
		"enriched": true,
		"source":   "identity_enrichment_filter",
	}

	// If user exists, mark as enriched
	if bidReq.User != nil {
		if bidReq.User.ID != "" {
			identityData["user_id"] = bidReq.User.ID
		}
		identityData["has_user"] = true
	} else {
		identityData["has_user"] = false
	}

	// If device exists, capture device info
	if bidReq.Device != nil {
		deviceInfo := make(map[string]interface{})
		if bidReq.Device.UA != "" {
			deviceInfo["has_ua"] = true
		}
		if bidReq.Device.IP != "" {
			deviceInfo["has_ip"] = true
		}
		if bidReq.Device.IFA != "" {
			deviceInfo["has_ifa"] = true
		}
		identityData["device"] = deviceInfo
	}

	resp.Metadata["identity"] = identityData
	resp.Warnings = append(resp.Warnings, fmt.Sprintf("Identity enrichment completed for account %s", req.Context.AccountID))

	return resp, nil
}

// Enabled returns whether the filter is enabled for the given account.
func (f *IdentityEnrichmentFilter) Enabled(accountID string) bool {
	return f.enabled
}

// Priority returns the execution priority.
func (f *IdentityEnrichmentFilter) Priority() int {
	return f.priority
}
