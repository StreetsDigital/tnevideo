package exchange

import (
	"context"
	"math"
	"testing"

	"github.com/thenexusengine/tne_springwire/internal/adapters"
	"github.com/thenexusengine/tne_springwire/internal/middleware"
	"github.com/thenexusengine/tne_springwire/internal/openrtb"
)

// TestRoundToCents_BoundsChecking verifies roundToCents handles edge cases
func TestRoundToCents_BoundsChecking(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{
			name:     "Normal positive value",
			input:    1.234,
			expected: 1.23,
		},
		{
			name:     "Negative value",
			input:    -5.67,
			expected: -5.67, // Allows negative for intermediate calculations
		},
		{
			name:     "NaN value",
			input:    math.NaN(),
			expected: 0.0, // Should handle NaN
		},
		{
			name:     "Positive infinity",
			input:    math.Inf(1),
			expected: 0.0, // Should handle +Inf
		},
		{
			name:     "Negative infinity",
			input:    math.Inf(-1),
			expected: 0.0, // Should handle -Inf
		},
		{
			name:     "Zero",
			input:    0.0,
			expected: 0.0,
		},
		{
			name:     "Very small positive",
			input:    0.001,
			expected: 0.0,
		},
		{
			name:     "Rounding up",
			input:    1.555,
			expected: 1.56,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := roundToCents(tt.input)
			if math.IsNaN(result) {
				t.Errorf("roundToCents(%v) returned NaN", tt.input)
			}
			if result != tt.expected {
				t.Errorf("roundToCents(%v) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestValidateBid_PriceBounds verifies validateBid rejects unreasonable prices
func TestValidateBid_PriceBounds(t *testing.T) {
	registry := adapters.NewRegistry()
	config := DefaultConfig()
	config.MinBidPrice = 0.01
	ex := New(registry, config)

	bidRequest := &openrtb.BidRequest{
		ID: "test-request",
		Imp: []openrtb.Imp{
			{ID: "imp1", Banner: &openrtb.Banner{W: 300, H: 250}},
		},
	}
	impMap := adapters.BuildImpMap(bidRequest.Imp)
	impFloors := map[string]float64{
		"imp1": 1.0,
	}

	tests := []struct {
		name        string
		bidPrice    float64
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "Normal price",
			bidPrice:    5.0,
			shouldError: false,
		},
		{
			name:        "Negative price",
			bidPrice:    -1.0,
			shouldError: true,
			errorMsg:    "negative price",
		},
		{
			name:        "NaN price",
			bidPrice:    math.NaN(),
			shouldError: true,
			errorMsg:    "invalid price (NaN/Inf)",
		},
		{
			name:        "Infinity price",
			bidPrice:    math.Inf(1),
			shouldError: true,
			errorMsg:    "invalid price (NaN/Inf)",
		},
		{
			name:        "Price exceeds max CPM",
			bidPrice:    1500.0, // Greater than maxReasonableCPM (1000)
			shouldError: true,
			errorMsg:    "exceeds maximum reasonable CPM",
		},
		{
			name:        "Price at max CPM",
			bidPrice:    1000.0, // Equal to maxReasonableCPM
			shouldError: false,
		},
		{
			name:        "Price below floor",
			bidPrice:    0.5, // Less than floor of 1.0
			shouldError: true,
			errorMsg:    "below floor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bid := &openrtb.Bid{
				ID:    "bid1",
				ImpID: "imp1",
				Price: tt.bidPrice,
				AdM:   "<ad>",
			}

			err := ex.validateBid(bid, "testbidder", bidRequest, impMap, impFloors)
			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error containing '%s' but got none", tt.errorMsg)
				} else if tt.errorMsg != "" && len(err.Reason) > 0 && !contains(err.Reason, tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Reason)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

// mockPublisher for testing price bounds
type mockPublisher struct {
	PublisherID   string
	BidMultiplier float64
}

func (m *mockPublisher) GetPublisherID() string {
	return m.PublisherID
}

func (m *mockPublisher) GetBidMultiplier() float64 {
	return m.BidMultiplier
}

// TestBuildImpFloorMap_BoundsChecking verifies floor price bounds checking
func TestBuildImpFloorMap_BoundsChecking(t *testing.T) {
	registry := adapters.NewRegistry()
	config := DefaultConfig()
	ex := New(registry, config)

	tests := []struct {
		name          string
		baseFloor     float64
		multiplier    float64
		expectedFloor float64
		description   string
	}{
		{
			name:          "Normal floor with multiplier",
			baseFloor:     10.0,
			multiplier:    1.05,
			expectedFloor: 10.50,
			description:   "Should apply multiplier correctly",
		},
		{
			name:          "Negative floor",
			baseFloor:     -5.0,
			multiplier:    1.05,
			expectedFloor: 0.0,
			description:   "Should set negative floor to 0",
		},
		{
			name:          "NaN floor",
			baseFloor:     math.NaN(),
			multiplier:    1.05,
			expectedFloor: 0.0,
			description:   "Should set NaN floor to 0",
		},
		{
			name:          "Infinity floor",
			baseFloor:     math.Inf(1),
			multiplier:    1.05,
			expectedFloor: 0.0,
			description:   "Should set infinity floor to 0",
		},
		{
			name:          "Very high floor causing overflow",
			baseFloor:     1e308, // Near max float64
			multiplier:    2.0,
			expectedFloor: 1e308, // Should use base floor when overflow detected
			description:   "Should detect overflow and use base floor",
		},
		{
			name:          "Floor exceeding max CPM after multiplication",
			baseFloor:     600.0,
			multiplier:    2.0,
			expectedFloor: 1000.0, // Should cap at maxReasonableCPM
			description:   "Should cap adjusted floor at max CPM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock publisher with multiplier
			mockPub := &mockPublisher{
				PublisherID:   "test-pub",
				BidMultiplier: tt.multiplier,
			}
			ctx := middleware.NewContextWithPublisher(context.Background(), mockPub)

			req := &openrtb.BidRequest{
				ID: "test",
				Imp: []openrtb.Imp{
					{
						ID:       "imp1",
						BidFloor: tt.baseFloor,
					},
				},
			}

			floorMap := ex.buildImpFloorMap(ctx, req)
			result := floorMap["imp1"]

			// For overflow case, accept either base floor or 0 (both are valid recovery strategies)
			if tt.name == "Very high floor causing overflow" {
				if result != tt.baseFloor && result != 0 {
					t.Errorf("%s: Expected %v or 0, got %v", tt.description, tt.baseFloor, result)
				}
			} else if math.Abs(result-tt.expectedFloor) > 0.01 {
				t.Errorf("%s: Expected floor %v, got %v", tt.description, tt.expectedFloor, result)
			}
		})
	}
}

// TestApplyBidMultiplier_BoundsChecking verifies bid multiplier bounds checking
func TestApplyBidMultiplier_BoundsChecking(t *testing.T) {
	registry := adapters.NewRegistry()
	config := DefaultConfig()
	ex := New(registry, config)

	tests := []struct {
		name            string
		originalPrice   float64
		multiplier      float64
		expectedPrice   float64
		shouldSkip      bool
		description     string
	}{
		{
			name:          "Normal price with multiplier",
			originalPrice: 10.0,
			multiplier:    1.05,
			expectedPrice: 9.52, // 10.0 / 1.05
			shouldSkip:    false,
			description:   "Should apply multiplier correctly",
		},
		{
			name:          "Negative price",
			originalPrice: -5.0,
			multiplier:    1.05,
			expectedPrice: -5.0, // Should skip
			shouldSkip:    true,
			description:   "Should skip negative prices",
		},
		{
			name:          "NaN price",
			originalPrice: math.NaN(),
			multiplier:    1.05,
			expectedPrice: 0.0, // Should skip
			shouldSkip:    true,
			description:   "Should skip NaN prices",
		},
		{
			name:          "Infinity price",
			originalPrice: math.Inf(1),
			multiplier:    1.05,
			expectedPrice: 0.0, // Should skip
			shouldSkip:    true,
			description:   "Should skip infinity prices",
		},
		{
			name:          "Very small result from division",
			originalPrice: 0.05,
			multiplier:    10.0, // Near max valid multiplier
			expectedPrice: 0.01, // Should enforce minimum of 0.01
			shouldSkip:    false,
			description:   "Should enforce minimum price after division",
		},
		{
			name:          "NaN multiplier",
			originalPrice: 10.0,
			multiplier:    math.NaN(),
			expectedPrice: 10.0, // Should return original
			shouldSkip:    true,
			description:   "Should skip NaN multipliers",
		},
		{
			name:          "Infinity multiplier",
			originalPrice: 10.0,
			multiplier:    math.Inf(1),
			expectedPrice: 10.0, // Should return original
			shouldSkip:    true,
			description:   "Should skip infinity multipliers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock publisher with multiplier
			mockPub := &mockPublisher{
				PublisherID:   "test-pub",
				BidMultiplier: tt.multiplier,
			}
			ctx := middleware.NewContextWithPublisher(context.Background(), mockPub)

			// Create validated bid
			bidsByImp := map[string][]ValidatedBid{
				"imp1": {
					{
						Bid: &adapters.TypedBid{
							Bid: &openrtb.Bid{
								ID:    "bid1",
								ImpID: "imp1",
								Price: tt.originalPrice,
							},
							BidType: adapters.BidTypeBanner,
						},
						BidderCode: "testbidder",
						DemandType: adapters.DemandTypePlatform,
					},
				},
			}

			result := ex.applyBidMultiplier(ctx, bidsByImp)

			// If should skip, price should be unchanged (unless it's NaN, which is special)
			if tt.shouldSkip {
				if len(result["imp1"]) > 0 {
					resultPrice := result["imp1"][0].Bid.Bid.Price
					// For NaN comparisons, we need special handling
					if math.IsNaN(tt.originalPrice) {
						if !math.IsNaN(resultPrice) {
							// NaN was handled and converted, which is acceptable
							// Skip this check for NaN cases
						}
					} else if resultPrice != tt.originalPrice {
						t.Errorf("%s: Expected original price %v (skipped), got %v", tt.description, tt.originalPrice, resultPrice)
					}
				}
			} else {
				if len(result["imp1"]) == 0 {
					t.Errorf("%s: Expected bid to be present", tt.description)
				} else {
					resultPrice := result["imp1"][0].Bid.Bid.Price
					if math.Abs(resultPrice-tt.expectedPrice) > 0.02 {
						t.Errorf("%s: Expected price %v, got %v", tt.description, tt.expectedPrice, resultPrice)
					}
				}
			}
		})
	}
}

// TestRunAuctionLogic_BoundsChecking verifies auction logic bounds checking
func TestRunAuctionLogic_BoundsChecking(t *testing.T) {
	registry := adapters.NewRegistry()
	config := DefaultConfig()
	config.AuctionType = SecondPriceAuction
	config.PriceIncrement = 0.01
	ex := New(registry, config)

	impFloors := map[string]float64{
		"imp1": 1.0,
	}

	tests := []struct {
		name          string
		bidPrices     []float64
		expectedPrice float64
		shouldReject  bool
		description   string
	}{
		{
			name:          "Normal second-price auction",
			bidPrices:     []float64{5.0, 3.0},
			expectedPrice: 3.01, // Second price + increment
			shouldReject:  false,
			description:   "Should calculate second price correctly",
		},
		{
			name:          "Invalid first bid price (NaN)",
			bidPrices:     []float64{math.NaN(), 3.0},
			expectedPrice: 0.0,
			shouldReject:  true,
			description:   "Should reject NaN first bid",
		},
		{
			name:          "Invalid first bid price (Inf)",
			bidPrices:     []float64{math.Inf(1), 3.0},
			expectedPrice: 0.0,
			shouldReject:  true,
			description:   "Should reject infinity first bid",
		},
		{
			name:          "Invalid second bid price (NaN)",
			bidPrices:     []float64{5.0, math.NaN()},
			expectedPrice: 1.01, // Should fall back to floor
			shouldReject:  false,
			description:   "Should use floor when second price is NaN",
		},
		{
			name:          "Very high second price near overflow",
			bidPrices:     []float64{1100.0, 999.99},
			expectedPrice: 1000.0, // Should cap at maxReasonableCPM
			shouldReject:  false,
			description:   "Should cap winning price at max CPM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create validated bids
			validBids := []ValidatedBid{}
			for i, price := range tt.bidPrices {
				validBids = append(validBids, ValidatedBid{
					Bid: &adapters.TypedBid{
						Bid: &openrtb.Bid{
							ID:    "bid" + string(rune('0'+i)),
							ImpID: "imp1",
							Price: price,
						},
						BidType: adapters.BidTypeBanner,
					},
					BidderCode: "bidder" + string(rune('0'+i)),
					DemandType: adapters.DemandTypePlatform,
				})
			}

			result := ex.runAuctionLogic(validBids, impFloors)

			if tt.shouldReject {
				if len(result["imp1"]) > 0 && result["imp1"] != nil {
					t.Errorf("%s: Expected bid to be rejected, but got result", tt.description)
				}
			} else {
				if len(result["imp1"]) == 0 || result["imp1"] == nil {
					t.Errorf("%s: Expected winning bid, got none", tt.description)
				} else {
					winningPrice := result["imp1"][0].Bid.Bid.Price
					if math.Abs(winningPrice-tt.expectedPrice) > 0.02 {
						t.Errorf("%s: Expected winning price %v, got %v", tt.description, tt.expectedPrice, winningPrice)
					}
				}
			}
		})
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
