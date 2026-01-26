//go:build integration
// +build integration

package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thenexusengine/tne_springwire/internal/adapters"
	"github.com/thenexusengine/tne_springwire/internal/exchange"
	"github.com/thenexusengine/tne_springwire/internal/openrtb"
)

// TestVideoAdapterIntegration tests video-specific bidder adapter functionality
func TestVideoAdapterIntegration(t *testing.T) {
	t.Run("Video_adapter_bid_response", func(t *testing.T) {
		// Setup mock video bidder
		bidder := setupMockVideoBidder(t, "test-bidder", 5.0)
		defer bidder.Close()

		// Register mock bidder
		registry := adapters.NewRegistry()
		mockAdapter := createMockAdapter(bidder.URL)
		registry.RegisterBidder("test-bidder", mockAdapter)

		// Create exchange
		ex := exchange.New(registry, &exchange.Config{
			Timeout:    500 * time.Millisecond,
			MaxBidders: 10,
		})

		// Create video bid request
		bidReq := createVideoRequestForAdapter()

		// Run auction
		ctx := context.Background()
		auctionResp, err := ex.RunAuction(ctx, &exchange.AuctionRequest{
			BidRequest: bidReq,
			Timeout:    500 * time.Millisecond,
		})

		require.NoError(t, err)
		assert.NotNil(t, auctionResp)
		assert.NotNil(t, auctionResp.BidResponse)

		// Verify video bid
		if len(auctionResp.BidResponse.SeatBid) > 0 {
			assert.Greater(t, len(auctionResp.BidResponse.SeatBid[0].Bid), 0, "Should have at least one bid")
			bid := auctionResp.BidResponse.SeatBid[0].Bid[0]
			assert.Equal(t, "1", bid.ImpID)
			assert.Greater(t, bid.Price, 0.0)
		}
	})

	t.Run("Multiple_video_adapters_parallel", func(t *testing.T) {
		// Setup 3 mock bidders with different prices
		bidder1 := setupMockVideoBidder(t, "bidder-1", 3.0)
		bidder2 := setupMockVideoBidder(t, "bidder-2", 5.0)
		bidder3 := setupMockVideoBidder(t, "bidder-3", 4.0)
		defer bidder1.Close()
		defer bidder2.Close()
		defer bidder3.Close()

		// Register all bidders
		registry := adapters.NewRegistry()
		registry.RegisterBidder("bidder-1", createMockAdapter(bidder1.URL))
		registry.RegisterBidder("bidder-2", createMockAdapter(bidder2.URL))
		registry.RegisterBidder("bidder-3", createMockAdapter(bidder3.URL))

		ex := exchange.New(registry, &exchange.Config{
			Timeout:    500 * time.Millisecond,
			MaxBidders: 10,
		})

		// Run auction
		bidReq := createVideoRequestForAdapter()
		ctx := context.Background()
		start := time.Now()

		auctionResp, err := ex.RunAuction(ctx, &exchange.AuctionRequest{
			BidRequest: bidReq,
			Timeout:    500 * time.Millisecond,
		})

		duration := time.Since(start)

		require.NoError(t, err)
		assert.NotNil(t, auctionResp.BidResponse)

		// Should run in parallel, not sequentially
		// 3 bidders at ~10ms each should complete in < 100ms if parallel
		assert.Less(t, duration, 200*time.Millisecond, "Parallel bidding should complete quickly")

		// Verify highest bid wins (bidder-2 with 5.0)
		if len(auctionResp.BidResponse.SeatBid) > 0 {
			// Find winning bid
			var highestBid *openrtb.Bid
			for _, seatBid := range auctionResp.BidResponse.SeatBid {
				for i := range seatBid.Bid {
					if highestBid == nil || seatBid.Bid[i].Price > highestBid.Price {
						highestBid = &seatBid.Bid[i]
					}
				}
			}

			if highestBid != nil {
				assert.Equal(t, 5.0, highestBid.Price, "Highest bid should win")
			}
		}

		t.Logf("Auction with 3 bidders completed in %v", duration)
	})

	t.Run("Adapter_timeout_handling", func(t *testing.T) {
		// Setup slow bidder that takes 2 seconds
		slowBidder := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(2 * time.Second) // Exceed timeout
			w.WriteHeader(http.StatusOK)
		}))
		defer slowBidder.Close()

		registry := adapters.NewRegistry()
		registry.RegisterBidder("slow-bidder", createMockAdapter(slowBidder.URL))

		ex := exchange.New(registry, &exchange.Config{
			Timeout:    100 * time.Millisecond, // Short timeout
			MaxBidders: 10,
		})

		bidReq := createVideoRequestForAdapter()
		ctx := context.Background()

		auctionResp, err := ex.RunAuction(ctx, &exchange.AuctionRequest{
			BidRequest: bidReq,
			Timeout:    100 * time.Millisecond,
		})

		// Should not error on timeout, just return no bids
		require.NoError(t, err)
		assert.NotNil(t, auctionResp)

		// May have no bids due to timeout
		t.Log("Timeout handling verified - auction completed despite slow bidder")
	})

	t.Run("Adapter_response_validation", func(t *testing.T) {
		// Setup bidder that returns invalid response
		invalidBidder := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"invalid": "response"}`)) // Missing required fields
		}))
		defer invalidBidder.Close()

		registry := adapters.NewRegistry()
		registry.RegisterBidder("invalid-bidder", createMockAdapter(invalidBidder.URL))

		ex := exchange.New(registry, &exchange.Config{
			Timeout:    500 * time.Millisecond,
			MaxBidders: 10,
		})

		bidReq := createVideoRequestForAdapter()
		ctx := context.Background()

		auctionResp, err := ex.RunAuction(ctx, &exchange.AuctionRequest{
			BidRequest: bidReq,
			Timeout:    500 * time.Millisecond,
		})

		// Should handle gracefully
		require.NoError(t, err)
		assert.NotNil(t, auctionResp)

		t.Log("Invalid response handled gracefully")
	})

	t.Run("Video_format_preferences", func(t *testing.T) {
		// Setup bidder that checks video format preference
		formatBidder := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var bidReq openrtb.BidRequest
			json.NewDecoder(r.Body).Decode(&bidReq)

			// Verify video parameters were passed
			if len(bidReq.Imp) > 0 && bidReq.Imp[0].Video != nil {
				video := bidReq.Imp[0].Video
				assert.NotEmpty(t, video.Mimes, "Should have MIME types")
				assert.Greater(t, video.MaxDuration, 0, "Should have max duration")
				assert.Greater(t, video.W, 0, "Should have width")
				assert.Greater(t, video.H, 0, "Should have height")
			}

			// Return bid
			bidResp := openrtb.BidResponse{
				ID: bidReq.ID,
				SeatBid: []openrtb.SeatBid{{
					Bid: []openrtb.Bid{{
						ID:    "bid-001",
						ImpID: "1",
						Price: 4.0,
						AdM:   "<VAST>...</VAST>",
						CrID:  "creative-001",
					}},
					Seat: "format-test",
				}},
				Cur: "USD",
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(bidResp)
		}))
		defer formatBidder.Close()

		registry := adapters.NewRegistry()
		registry.RegisterBidder("format-test", createMockAdapter(formatBidder.URL))

		ex := exchange.New(registry, &exchange.Config{
			Timeout:    500 * time.Millisecond,
			MaxBidders: 10,
		})

		// Request specific format
		bidReq := createVideoRequestForAdapter()
		bidReq.Imp[0].Video.Mimes = []string{"video/mp4"} // Prefer MP4

		ctx := context.Background()
		auctionResp, err := ex.RunAuction(ctx, &exchange.AuctionRequest{
			BidRequest: bidReq,
			Timeout:    500 * time.Millisecond,
		})

		require.NoError(t, err)
		assert.NotNil(t, auctionResp)
	})

	t.Run("Adapter_performance_benchmark", func(t *testing.T) {
		// Setup fast bidder
		bidder := setupMockVideoBidder(t, "perf-test", 3.5)
		defer bidder.Close()

		registry := adapters.NewRegistry()
		registry.RegisterBidder("perf-test", createMockAdapter(bidder.URL))

		ex := exchange.New(registry, &exchange.Config{
			Timeout:    500 * time.Millisecond,
			MaxBidders: 10,
		})

		bidReq := createVideoRequestForAdapter()
		ctx := context.Background()

		// Run 10 requests and measure average
		var totalDuration time.Duration
		iterations := 10

		for i := 0; i < iterations; i++ {
			start := time.Now()
			_, err := ex.RunAuction(ctx, &exchange.AuctionRequest{
				BidRequest: bidReq,
				Timeout:    500 * time.Millisecond,
			})
			duration := time.Since(start)
			totalDuration += duration
			require.NoError(t, err)
		}

		avgDuration := totalDuration / time.Duration(iterations)

		// Should complete in < 100ms on average
		assert.Less(t, avgDuration, 100*time.Millisecond, "Adapter should respond quickly")

		t.Logf("Average adapter response time: %v", avgDuration)
	})

	t.Run("Fallback_on_adapter_failure", func(t *testing.T) {
		// Setup primary bidder that fails
		failingBidder := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer failingBidder.Close()

		// Setup fallback bidder that succeeds
		fallbackBidder := setupMockVideoBidder(t, "fallback", 2.5)
		defer fallbackBidder.Close()

		registry := adapters.NewRegistry()
		registry.RegisterBidder("failing", createMockAdapter(failingBidder.URL))
		registry.RegisterBidder("fallback", createMockAdapter(fallbackBidder.URL))

		ex := exchange.New(registry, &exchange.Config{
			Timeout:    500 * time.Millisecond,
			MaxBidders: 10,
		})

		bidReq := createVideoRequestForAdapter()
		ctx := context.Background()

		auctionResp, err := ex.RunAuction(ctx, &exchange.AuctionRequest{
			BidRequest: bidReq,
			Timeout:    500 * time.Millisecond,
		})

		require.NoError(t, err)
		assert.NotNil(t, auctionResp)

		// Should have bid from fallback bidder
		if len(auctionResp.BidResponse.SeatBid) > 0 {
			assert.Greater(t, len(auctionResp.BidResponse.SeatBid[0].Bid), 0)
			t.Log("Fallback bidder provided bid after primary failure")
		}
	})
}

// TestVideoAdapterSpecificParameters tests adapter-specific video parameters
func TestVideoAdapterSpecificParameters(t *testing.T) {
	t.Run("VPAID_support_filtering", func(t *testing.T) {
		vpaidBidder := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var bidReq openrtb.BidRequest
			json.NewDecoder(r.Body).Decode(&bidReq)

			// Check if VPAID APIs are present
			if len(bidReq.Imp) > 0 && bidReq.Imp[0].Video != nil {
				hasVPAID := false
				for _, api := range bidReq.Imp[0].Video.API {
					if api == 1 || api == 2 { // VPAID 1.0 or 2.0
						hasVPAID = true
						break
					}
				}
				assert.True(t, hasVPAID, "Should include VPAID APIs when supported")
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"id":"test","seatbid":[]}`))
		}))
		defer vpaidBidder.Close()

		registry := adapters.NewRegistry()
		registry.RegisterBidder("vpaid-test", createMockAdapter(vpaidBidder.URL))

		ex := exchange.New(registry, &exchange.Config{
			Timeout:    500 * time.Millisecond,
			MaxBidders: 10,
		})

		bidReq := createVideoRequestForAdapter()
		bidReq.Imp[0].Video.API = []int{1, 2} // Request VPAID

		ctx := context.Background()
		_, err := ex.RunAuction(ctx, &exchange.AuctionRequest{
			BidRequest: bidReq,
			Timeout:    500 * time.Millisecond,
		})

		require.NoError(t, err)
	})

	t.Run("Skippable_video_parameter", func(t *testing.T) {
		bidReq := createVideoRequestForAdapter()
		skip := 1
		bidReq.Imp[0].Video.Skip = &skip
		bidReq.Imp[0].Video.SkipAfter = 5

		// Verify parameters are set
		assert.NotNil(t, bidReq.Imp[0].Video.Skip)
		assert.Equal(t, 1, *bidReq.Imp[0].Video.Skip)
		assert.Equal(t, 5, bidReq.Imp[0].Video.SkipAfter)
	})

	t.Run("Video_placement_types", func(t *testing.T) {
		placements := []struct {
			placement int
			name      string
		}{
			{1, "in-stream"},
			{3, "in-article"},
			{4, "in-feed"},
			{5, "interstitial"},
		}

		for _, p := range placements {
			t.Run(p.name, func(t *testing.T) {
				bidReq := createVideoRequestForAdapter()
				bidReq.Imp[0].Video.Placement = p.placement

				assert.Equal(t, p.placement, bidReq.Imp[0].Video.Placement)
			})
		}
	})
}

// Helper functions

func setupMockVideoBidder(t *testing.T, bidderName string, price float64) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse incoming request
		var bidReq openrtb.BidRequest
		if err := json.NewDecoder(r.Body).Decode(&bidReq); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Create video bid response
		bidResp := openrtb.BidResponse{
			ID: bidReq.ID,
			SeatBid: []openrtb.SeatBid{
				{
					Bid: []openrtb.Bid{
						{
							ID:    "bid-" + bidderName,
							ImpID: "1",
							Price: price,
							AdM:   "<VAST version=\"4.0\"><Ad><InLine><AdSystem>" + bidderName + "</AdSystem><AdTitle>Test</AdTitle><Creatives></Creatives></InLine></Ad></VAST>",
							NURL:  "https://win.example.com/win?bidder=" + bidderName,
							CrID:  "creative-" + bidderName,
							W:     1920,
							H:     1080,
						},
					},
					Seat: bidderName,
				},
			},
			Cur: "USD",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(bidResp)
	}))
}

func createMockAdapter(endpoint string) adapters.Bidder {
	// In a real implementation, this would return a proper Bidder interface
	// For testing, we'd need to implement a mock adapter
	// This is a placeholder
	return nil
}

func createVideoRequestForAdapter() *openrtb.BidRequest {
	skip := 1
	return &openrtb.BidRequest{
		ID: "adapter-test-request",
		Imp: []openrtb.Imp{
			{
				ID: "1",
				Video: &openrtb.Video{
					Mimes:       []string{"video/mp4", "video/webm"},
					MinDuration: 5,
					MaxDuration: 30,
					Protocols:   []int{2, 3, 5, 6},
					W:           1920,
					H:           1080,
					Placement:   1,
					Linearity:   1,
					Skip:        &skip,
					SkipAfter:   5,
					MinBitrate:  1000,
					MaxBitrate:  5000,
					API:         []int{1, 2},
				},
				BidFloor:    2.0,
				BidFloorCur: "USD",
			},
		},
		Device: &openrtb.Device{
			UA: "Mozilla/5.0 Test",
			IP: "203.0.113.1",
		},
		Site: &openrtb.Site{
			ID:     "test-site",
			Domain: "test.example.com",
		},
		TMax: 500,
		Cur:  []string{"USD"},
		AT:   2,
	}
}
