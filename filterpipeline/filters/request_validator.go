package filters

import (
	"context"
	"fmt"

	"github.com/prebid/prebid-server/v3/filterpipeline"
)

// RequestValidatorFilter validates incoming requests in the pre-filter stage.
type RequestValidatorFilter struct {
	enabled        bool
	priority       int
	requireUser    bool
	requireDevice  bool
	minImpressions int
}

// RequestValidatorConfig holds configuration for the request validator.
type RequestValidatorConfig struct {
	Enabled        bool
	Priority       int
	RequireUser    bool
	RequireDevice  bool
	MinImpressions int
}

// NewRequestValidatorFilter creates a new request validation filter.
func NewRequestValidatorFilter(config RequestValidatorConfig) *RequestValidatorFilter {
	return &RequestValidatorFilter{
		enabled:        config.Enabled,
		priority:       config.Priority,
		requireUser:    config.RequireUser,
		requireDevice:  config.RequireDevice,
		minImpressions: config.MinImpressions,
	}
}

// Name returns the filter identifier.
func (f *RequestValidatorFilter) Name() string {
	return "request_validator"
}

// Execute validates the request against configured rules.
func (f *RequestValidatorFilter) Execute(ctx context.Context, req filterpipeline.PreFilterRequest) (filterpipeline.PreFilterResponse, error) {
	resp := filterpipeline.PreFilterResponse{
		Request:  req.Request,
		Metadata: make(map[string]interface{}),
	}

	// Check if request exists
	if req.Request == nil || req.Request.BidRequest == nil {
		resp.Reject = true
		resp.RejectReason = "missing bid request"
		return resp, nil
	}

	bidReq := req.Request.BidRequest

	// Validate user requirement
	if f.requireUser && bidReq.User == nil {
		resp.Reject = true
		resp.RejectReason = "request missing required user object"
		return resp, nil
	}

	// Validate device requirement
	if f.requireDevice && bidReq.Device == nil {
		resp.Reject = true
		resp.RejectReason = "request missing required device object"
		return resp, nil
	}

	// Validate minimum impressions
	if f.minImpressions > 0 && len(bidReq.Imp) < f.minImpressions {
		resp.Reject = true
		resp.RejectReason = fmt.Sprintf("request has %d impressions, minimum required is %d", len(bidReq.Imp), f.minImpressions)
		return resp, nil
	}

	// Validate each impression has an ID
	for i, imp := range bidReq.Imp {
		if imp.ID == "" {
			resp.Errors = append(resp.Errors, fmt.Sprintf("impression at index %d missing ID", i))
		}
	}

	// Add validation metadata
	resp.Metadata["validation"] = map[string]interface{}{
		"passed":         true,
		"impression_count": len(bidReq.Imp),
	}

	return resp, nil
}

// Enabled returns whether the filter is enabled for the given account.
func (f *RequestValidatorFilter) Enabled(accountID string) bool {
	return f.enabled
}

// Priority returns the execution priority.
func (f *RequestValidatorFilter) Priority() int {
	return f.priority
}
