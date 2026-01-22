package filters

import (
	"context"
	"fmt"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/filterpipeline"
)

// PolicyEnforcerFilter enforces policies on bid responses in the post-filter stage.
type PolicyEnforcerFilter struct {
	enabled        bool
	priority       int
	maxBidPrice    float64
	minBidPrice    float64
	allowedBidders map[string]bool
}

// PolicyEnforcerConfig holds configuration for the policy enforcer.
type PolicyEnforcerConfig struct {
	Enabled        bool
	Priority       int
	MaxBidPrice    float64
	MinBidPrice    float64
	AllowedBidders []string
}

// NewPolicyEnforcerFilter creates a new policy enforcement filter.
func NewPolicyEnforcerFilter(config PolicyEnforcerConfig) *PolicyEnforcerFilter {
	allowedBidders := make(map[string]bool)
	for _, bidder := range config.AllowedBidders {
		allowedBidders[bidder] = true
	}

	return &PolicyEnforcerFilter{
		enabled:        config.Enabled,
		priority:       config.Priority,
		maxBidPrice:    config.MaxBidPrice,
		minBidPrice:    config.MinBidPrice,
		allowedBidders: allowedBidders,
	}
}

// Name returns the filter identifier.
func (f *PolicyEnforcerFilter) Name() string {
	return "policy_enforcer"
}

// Execute enforces policies on the bid response.
func (f *PolicyEnforcerFilter) Execute(ctx context.Context, req filterpipeline.PostFilterRequest) (filterpipeline.PostFilterResponse, error) {
	resp := filterpipeline.PostFilterResponse{
		Response: req.Response,
		Metadata: make(map[string]interface{}),
	}

	// Check if response exists
	if req.Response == nil {
		return resp, nil
	}

	bidResp := req.Response
	removedBids := 0
	policyViolations := make([]string, 0)

	// Iterate through seat bids and filter bids based on policies
	for seatIdx, seatBid := range bidResp.SeatBid {
		filteredBids := make([]openrtb2.Bid, 0, len(seatBid.Bid))

		for _, bid := range seatBid.Bid {
			removed := false
			reason := ""

			// Check bid price constraints
			if f.maxBidPrice > 0 && bid.Price > f.maxBidPrice {
				removed = true
				reason = fmt.Sprintf("bid price %.2f exceeds maximum %.2f", bid.Price, f.maxBidPrice)
			} else if f.minBidPrice > 0 && bid.Price < f.minBidPrice {
				removed = true
				reason = fmt.Sprintf("bid price %.2f below minimum %.2f", bid.Price, f.minBidPrice)
			}

			// Check allowed bidders (if configured)
			if len(f.allowedBidders) > 0 && seatBid.Seat != "" {
				if !f.allowedBidders[seatBid.Seat] {
					removed = true
					reason = fmt.Sprintf("bidder %s not in allowed list", seatBid.Seat)
				}
			}

			if removed {
				removedBids++
				policyViolations = append(policyViolations, fmt.Sprintf("Bid %s from %s: %s", bid.ID, seatBid.Seat, reason))
				resp.Warnings = append(resp.Warnings, reason)
			} else {
				filteredBids = append(filteredBids, bid)
			}
		}

		// Update seat bid with filtered bids
		bidResp.SeatBid[seatIdx].Bid = filteredBids
	}

	// Remove empty seat bids
	filteredSeatBids := make([]openrtb2.SeatBid, 0, len(bidResp.SeatBid))
	for _, seatBid := range bidResp.SeatBid {
		if len(seatBid.Bid) > 0 {
			filteredSeatBids = append(filteredSeatBids, seatBid)
		}
	}
	bidResp.SeatBid = filteredSeatBids

	// Add policy metadata
	resp.Metadata["policy"] = map[string]interface{}{
		"removed_bids":       removedBids,
		"policy_violations":  policyViolations,
		"remaining_seatbids": len(bidResp.SeatBid),
	}

	if removedBids > 0 {
		resp.Warnings = append(resp.Warnings, fmt.Sprintf("Policy enforcer removed %d bid(s)", removedBids))
	}

	return resp, nil
}

// Enabled returns whether the filter is enabled for the given account.
func (f *PolicyEnforcerFilter) Enabled(accountID string) bool {
	return f.enabled
}

// Priority returns the execution priority.
func (f *PolicyEnforcerFilter) Priority() int {
	return f.priority
}
