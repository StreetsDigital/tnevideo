package filters

import (
	"context"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/filterpipeline"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestPolicyEnforcerFilter_Name(t *testing.T) {
	config := PolicyEnforcerConfig{Enabled: true, Priority: 10}
	filter := NewPolicyEnforcerFilter(config)
	assert.Equal(t, "policy_enforcer", filter.Name())
}

func TestPolicyEnforcerFilter_Priority(t *testing.T) {
	config := PolicyEnforcerConfig{Enabled: true, Priority: 25}
	filter := NewPolicyEnforcerFilter(config)
	assert.Equal(t, 25, filter.Priority())
}

func TestPolicyEnforcerFilter_Enabled(t *testing.T) {
	tests := []struct {
		name      string
		enabled   bool
		accountID string
		expected  bool
	}{
		{
			name:      "enabled_filter",
			enabled:   true,
			accountID: "test-account",
			expected:  true,
		},
		{
			name:      "disabled_filter",
			enabled:   false,
			accountID: "test-account",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := PolicyEnforcerConfig{Enabled: tt.enabled, Priority: 10}
			filter := NewPolicyEnforcerFilter(config)
			result := filter.Enabled(tt.accountID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPolicyEnforcerFilter_Execute(t *testing.T) {
	tests := []struct {
		name               string
		config             PolicyEnforcerConfig
		response           *openrtb_ext.BidResponseWrapper
		expectedBidCount   int
		expectedSeatCount  int
		expectWarnings     bool
	}{
		{
			name: "no_policy_violations",
			config: PolicyEnforcerConfig{
				Enabled:     true,
				Priority:    10,
				MaxBidPrice: 10.0,
				MinBidPrice: 0.1,
			},
			response: &openrtb_ext.BidResponseWrapper{
				BidResponse: &openrtb2.BidResponse{
					ID: "test-resp",
					SeatBid: []openrtb2.SeatBid{
						{
							Seat: "bidder1",
							Bid: []*openrtb2.Bid{
								{ID: "bid1", Price: 1.5},
								{ID: "bid2", Price: 2.0},
							},
						},
					},
				},
			},
			expectedBidCount:  2,
			expectedSeatCount: 1,
			expectWarnings:    false,
		},
		{
			name: "max_price_violation",
			config: PolicyEnforcerConfig{
				Enabled:     true,
				Priority:    10,
				MaxBidPrice: 2.0,
				MinBidPrice: 0.0,
			},
			response: &openrtb_ext.BidResponseWrapper{
				BidResponse: &openrtb2.BidResponse{
					ID: "test-resp",
					SeatBid: []openrtb2.SeatBid{
						{
							Seat: "bidder1",
							Bid: []*openrtb2.Bid{
								{ID: "bid1", Price: 1.5},
								{ID: "bid2", Price: 3.0}, // Exceeds max
							},
						},
					},
				},
			},
			expectedBidCount:  1,
			expectedSeatCount: 1,
			expectWarnings:    true,
		},
		{
			name: "min_price_violation",
			config: PolicyEnforcerConfig{
				Enabled:     true,
				Priority:    10,
				MaxBidPrice: 0.0,
				MinBidPrice: 1.0,
			},
			response: &openrtb_ext.BidResponseWrapper{
				BidResponse: &openrtb2.BidResponse{
					ID: "test-resp",
					SeatBid: []openrtb2.SeatBid{
						{
							Seat: "bidder1",
							Bid: []*openrtb2.Bid{
								{ID: "bid1", Price: 1.5},
								{ID: "bid2", Price: 0.5}, // Below min
							},
						},
					},
				},
			},
			expectedBidCount:  1,
			expectedSeatCount: 1,
			expectWarnings:    true,
		},
		{
			name: "allowed_bidders_filter",
			config: PolicyEnforcerConfig{
				Enabled:        true,
				Priority:       10,
				AllowedBidders: []string{"bidder1"},
			},
			response: &openrtb_ext.BidResponseWrapper{
				BidResponse: &openrtb2.BidResponse{
					ID: "test-resp",
					SeatBid: []openrtb2.SeatBid{
						{
							Seat: "bidder1",
							Bid: []*openrtb2.Bid{
								{ID: "bid1", Price: 1.5},
							},
						},
						{
							Seat: "bidder2",
							Bid: []*openrtb2.Bid{
								{ID: "bid2", Price: 2.0},
							},
						},
					},
				},
			},
			expectedBidCount:  1,
			expectedSeatCount: 1,
			expectWarnings:    true,
		},
		{
			name: "all_bids_removed",
			config: PolicyEnforcerConfig{
				Enabled:     true,
				Priority:    10,
				MaxBidPrice: 1.0,
			},
			response: &openrtb_ext.BidResponseWrapper{
				BidResponse: &openrtb2.BidResponse{
					ID: "test-resp",
					SeatBid: []openrtb2.SeatBid{
						{
							Seat: "bidder1",
							Bid: []*openrtb2.Bid{
								{ID: "bid1", Price: 2.0},
								{ID: "bid2", Price: 3.0},
							},
						},
					},
				},
			},
			expectedBidCount:  0,
			expectedSeatCount: 0,
			expectWarnings:    true,
		},
		{
			name: "multiple_seatbids_mixed_violations",
			config: PolicyEnforcerConfig{
				Enabled:     true,
				Priority:    10,
				MaxBidPrice: 2.5,
				MinBidPrice: 0.5,
			},
			response: &openrtb_ext.BidResponseWrapper{
				BidResponse: &openrtb2.BidResponse{
					ID: "test-resp",
					SeatBid: []openrtb2.SeatBid{
						{
							Seat: "bidder1",
							Bid: []*openrtb2.Bid{
								{ID: "bid1", Price: 1.5},
								{ID: "bid2", Price: 3.0}, // Exceeds max
							},
						},
						{
							Seat: "bidder2",
							Bid: []*openrtb2.Bid{
								{ID: "bid3", Price: 2.0},
								{ID: "bid4", Price: 0.2}, // Below min
							},
						},
					},
				},
			},
			expectedBidCount:  2, // bid1 and bid3
			expectedSeatCount: 2,
			expectWarnings:    true,
		},
		{
			name: "nil_response",
			config: PolicyEnforcerConfig{
				Enabled:  true,
				Priority: 10,
			},
			response:           nil,
			expectedBidCount:   0,
			expectedSeatCount:  0,
			expectWarnings:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewPolicyEnforcerFilter(tt.config)

			req := filterpipeline.PostFilterRequest{
				Request: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID: "test-req",
					},
				},
				Response: tt.response,
				Context: filterpipeline.FilterContext{
					AccountID: "test-account",
					Endpoint:  "/openrtb2/auction",
				},
			}

			resp, err := filter.Execute(context.Background(), req)

			assert.NoError(t, err)
			assert.NotNil(t, resp)

			if tt.response != nil && tt.response.BidResponse != nil {
				// Count remaining bids
				totalBids := 0
				for _, seatBid := range resp.Response.BidResponse.SeatBid {
					totalBids += len(seatBid.Bid)
				}

				assert.Equal(t, tt.expectedBidCount, totalBids, "bid count mismatch")
				assert.Equal(t, tt.expectedSeatCount, len(resp.Response.BidResponse.SeatBid), "seat count mismatch")

				if tt.expectWarnings {
					assert.NotEmpty(t, resp.Warnings, "expected warnings")
				}

				// Verify policy metadata
				assert.NotNil(t, resp.Metadata)
				policyData, ok := resp.Metadata["policy"].(map[string]interface{})
				assert.True(t, ok)
				assert.NotNil(t, policyData["removed_bids"])
				assert.Equal(t, tt.expectedSeatCount, policyData["remaining_seatbids"])
			}
		})
	}
}

func TestPolicyEnforcerFilter_AllowedBidders(t *testing.T) {
	config := PolicyEnforcerConfig{
		Enabled:        true,
		Priority:       10,
		AllowedBidders: []string{"bidder1", "bidder3"},
	}
	filter := NewPolicyEnforcerFilter(config)

	response := &openrtb_ext.BidResponseWrapper{
		BidResponse: &openrtb2.BidResponse{
			ID: "test-resp",
			SeatBid: []openrtb2.SeatBid{
				{
					Seat: "bidder1",
					Bid:  []*openrtb2.Bid{{ID: "bid1", Price: 1.0}},
				},
				{
					Seat: "bidder2",
					Bid:  []*openrtb2.Bid{{ID: "bid2", Price: 1.0}},
				},
				{
					Seat: "bidder3",
					Bid:  []*openrtb2.Bid{{ID: "bid3", Price: 1.0}},
				},
			},
		},
	}

	req := filterpipeline.PostFilterRequest{
		Request: &openrtb_ext.RequestWrapper{
			BidRequest: &openrtb2.BidRequest{ID: "test-req"},
		},
		Response: response,
		Context: filterpipeline.FilterContext{
			AccountID: "test-account",
			Endpoint:  "/openrtb2/auction",
		},
	}

	resp, err := filter.Execute(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, 2, len(resp.Response.BidResponse.SeatBid))

	// Verify only allowed bidders remain
	seats := make(map[string]bool)
	for _, seatBid := range resp.Response.BidResponse.SeatBid {
		seats[seatBid.Seat] = true
	}
	assert.True(t, seats["bidder1"])
	assert.False(t, seats["bidder2"])
	assert.True(t, seats["bidder3"])
}
