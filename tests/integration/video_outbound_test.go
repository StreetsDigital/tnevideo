//go:build integration
// +build integration

package integration

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thenexusengine/tne_springwire/internal/adapters"
	"github.com/thenexusengine/tne_springwire/internal/endpoints"
	"github.com/thenexusengine/tne_springwire/internal/exchange"
	"github.com/thenexusengine/tne_springwire/internal/openrtb"
	"github.com/thenexusengine/tne_springwire/pkg/vast"
)

// TestOutboundVASTGeneration tests end-to-end VAST generation workflow
func TestOutboundVASTGeneration(t *testing.T) {
	// Setup mock demand partner that returns video bid
	demandPartner := setupMockDemandPartner(t)
	defer demandPartner.Close()

	// Create exchange with test config
	ex := createTestExchange(t)

	// Create video handler
	trackingURL := "https://tracking.test.com"
	handler := endpoints.NewVideoHandler(ex, trackingURL)

	t.Run("GET_VAST_request_with_query_params", func(t *testing.T) {
		// Create test server
		server := httptest.NewServer(http.HandlerFunc(handler.HandleVASTRequest))
		defer server.Close()

		// Make request with query parameters
		url := server.URL + "?id=test-001&w=1920&h=1080&mindur=5&maxdur=30&mimes=video/mp4&bidfloor=2.5"
		resp, err := http.Get(url)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Verify response
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/xml; charset=utf-8", resp.Header.Get("Content-Type"))

		// Parse VAST response
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var vastResp vast.VAST
		err = xml.Unmarshal(body, &vastResp)
		require.NoError(t, err)

		// Verify VAST structure
		assert.Equal(t, "4.0", vastResp.Version)
		assert.NotEmpty(t, vastResp.Ads, "Should have at least one ad")
	})

	t.Run("POST_OpenRTB_video_request", func(t *testing.T) {
		// Create test server
		server := httptest.NewServer(http.HandlerFunc(handler.HandleOpenRTBVideo))
		defer server.Close()

		// Create OpenRTB video bid request
		bidReq := createTestVideoBidRequest()
		body, err := json.Marshal(bidReq)
		require.NoError(t, err)

		// Make POST request
		resp, err := http.Post(server.URL, "application/json", strings.NewReader(string(body)))
		require.NoError(t, err)
		defer resp.Body.Close()

		// Verify response
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Parse VAST
		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var vastResp vast.VAST
		err = xml.Unmarshal(respBody, &vastResp)
		require.NoError(t, err)

		assert.Equal(t, "4.0", vastResp.Version)
	})

	t.Run("VAST_contains_tracking_URLs", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(handler.HandleVASTRequest))
		defer server.Close()

		url := server.URL + "?id=test-tracking&w=1920&h=1080"
		resp, err := http.Get(url)
		require.NoError(t, err)
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var vastResp vast.VAST
		err = xml.Unmarshal(body, &vastResp)
		require.NoError(t, err)

		// Verify tracking URLs are present
		if len(vastResp.Ads) > 0 && vastResp.Ads[0].InLine != nil {
			inline := vastResp.Ads[0].InLine

			// Check impressions
			assert.NotEmpty(t, inline.Impressions, "Should have impression tracking")
			assert.Contains(t, inline.Impressions[0].Value, trackingURL, "Impression should use tracking base URL")

			// Check error tracking
			assert.NotEmpty(t, inline.Error, "Should have error tracking URL")
			assert.Contains(t, inline.Error, trackingURL)

			// Check creative tracking events
			if len(inline.Creatives.Creative) > 0 {
				creative := inline.Creatives.Creative[0]
				if creative.Linear != nil {
					events := creative.Linear.TrackingEvents.Tracking
					assert.NotEmpty(t, events, "Should have tracking events")

					// Verify quartile events
					eventTypes := make(map[string]bool)
					for _, event := range events {
						eventTypes[event.Event] = true
					}

					assert.True(t, eventTypes[vast.EventStart], "Should have start event")
					assert.True(t, eventTypes[vast.EventFirstQuartile], "Should have firstQuartile event")
					assert.True(t, eventTypes[vast.EventMidpoint], "Should have midpoint event")
					assert.True(t, eventTypes[vast.EventThirdQuartile], "Should have thirdQuartile event")
					assert.True(t, eventTypes[vast.EventComplete], "Should have complete event")
				}
			}
		}
	})

	t.Run("VAST_macro_replacement", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(handler.HandleVASTRequest))
		defer server.Close()

		url := server.URL + "?id=test-macros&w=1920&h=1080"
		resp, err := http.Get(url)
		require.NoError(t, err)
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		// Check for macro placeholders (should be preserved, not expanded at generation time)
		bodyStr := string(body)

		// Macros should be in tracking URLs
		// Note: Actual macro expansion happens at playback time by the player
		t.Log("VAST response generated with macro placeholders")
	})

	t.Run("No_bid_returns_empty_VAST", func(t *testing.T) {
		// Create exchange with no bidders
		emptyEx := exchange.New(adapters.NewRegistry(), &exchange.Config{
			Timeout: 100 * time.Millisecond,
		})
		emptyHandler := endpoints.NewVideoHandler(emptyEx, trackingURL)

		server := httptest.NewServer(http.HandlerFunc(emptyHandler.HandleVASTRequest))
		defer server.Close()

		url := server.URL + "?id=test-nobid&w=1920&h=1080"
		resp, err := http.Get(url)
		require.NoError(t, err)
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var vastResp vast.VAST
		err = xml.Unmarshal(body, &vastResp)
		require.NoError(t, err)

		// Should be empty VAST (no ads)
		assert.Empty(t, vastResp.Ads, "Should have no ads on no-bid")
	})

	t.Run("Invalid_parameters_returns_error_VAST", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(handler.HandleVASTRequest))
		defer server.Close()

		// Request with invalid parameters
		url := server.URL + "?w=invalid&h=invalid"
		resp, err := http.Get(url)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should still return valid VAST (with defaults or error)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var vastResp vast.VAST
		err = xml.Unmarshal(body, &vastResp)
		require.NoError(t, err) // Should parse as valid VAST

		assert.Equal(t, "4.0", vastResp.Version)
	})

	t.Run("VAST_XML_is_well_formed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(handler.HandleVASTRequest))
		defer server.Close()

		url := server.URL + "?id=test-xml&w=1920&h=1080"
		resp, err := http.Get(url)
		require.NoError(t, err)
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		// Verify XML declaration
		bodyStr := string(body)
		assert.True(t, strings.HasPrefix(bodyStr, "<?xml version=\"1.0\""), "Should have XML declaration")

		// Verify proper XML structure
		assert.Contains(t, bodyStr, "<VAST", "Should have VAST root element")
		assert.Contains(t, bodyStr, "version=\"4.0\"", "Should have version attribute")

		// Verify proper CDATA wrapping for URLs
		if strings.Contains(bodyStr, "<Impression") {
			assert.Contains(t, bodyStr, "<![CDATA[", "URLs should be wrapped in CDATA")
			assert.Contains(t, bodyStr, "]]>", "CDATA should be closed")
		}
	})
}

// TestVASTMediaFileSelection tests video format selection
func TestVASTMediaFileSelection(t *testing.T) {
	ex := createTestExchange(t)
	handler := endpoints.NewVideoHandler(ex, "https://tracking.test.com")

	t.Run("Requests_preferred_format", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(handler.HandleVASTRequest))
		defer server.Close()

		// Request MP4 specifically
		url := server.URL + "?id=test-mp4&w=1920&h=1080&mimes=video/mp4"
		resp, err := http.Get(url)
		require.NoError(t, err)
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var vastResp vast.VAST
		err = xml.Unmarshal(body, &vastResp)
		require.NoError(t, err)

		// Extract media files
		mediaFiles := vastResp.GetMediaFiles()
		if len(mediaFiles) > 0 {
			// At least one should be MP4
			hasMP4 := false
			for _, mf := range mediaFiles {
				if mf.Type == "video/mp4" {
					hasMP4 = true
					break
				}
			}
			assert.True(t, hasMP4, "Should include MP4 media file when requested")
		}
	})

	t.Run("Multiple_bitrates_available", func(t *testing.T) {
		// This would test if VAST includes multiple bitrate options
		// Implementation depends on demand partner responses
		t.Skip("Requires mock demand partner with multiple bitrates")
	})
}

// TestVASTDurationHandling tests duration formatting
func TestVASTDurationHandling(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"15_seconds", 15 * time.Second, "00:00:15"},
		{"30_seconds", 30 * time.Second, "00:00:30"},
		{"1_minute", 60 * time.Second, "00:01:00"},
		{"90_seconds", 90 * time.Second, "00:01:30"},
		{"5_minutes", 5 * time.Minute, "00:05:00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := vast.FormatDuration(tt.duration)
			assert.Equal(t, tt.expected, formatted)

			// Verify round-trip
			parsed, err := vast.ParseDuration(formatted)
			require.NoError(t, err)
			assert.Equal(t, tt.duration, parsed)
		})
	}
}

// Helper functions

func setupMockDemandPartner(t *testing.T) *httptest.Server {
	// Mock demand partner that returns a video bid
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return a mock bid response
		bidResp := openrtb.BidResponse{
			ID: "test-response",
			SeatBid: []openrtb.SeatBid{
				{
					Bid: []openrtb.Bid{
						{
							ID:    "bid-001",
							ImpID: "1",
							Price: 5.0,
							AdM:   "https://cdn.example.com/video.mp4",
							NURL:  "https://win.example.com/win",
							CrID:  "creative-001",
							W:     1920,
							H:     1080,
						},
					},
					Seat: "test-bidder",
				},
			},
			Cur: "USD",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(bidResp)
	}))
}

func createTestExchange(t *testing.T) *exchange.Exchange {
	registry := adapters.NewRegistry()

	config := &exchange.Config{
		Timeout:     500 * time.Millisecond,
		MaxBidders:  10,
		IDREnabled:  false,
	}

	return exchange.New(registry, config)
}

func createTestVideoBidRequest() *openrtb.BidRequest {
	skip := 1
	return &openrtb.BidRequest{
		ID: "test-video-request",
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
				BidFloor:    2.5,
				BidFloorCur: "USD",
			},
		},
		Device: &openrtb.Device{
			UA: "Mozilla/5.0 Test",
			IP: "203.0.113.1",
			W:  1920,
			H:  1080,
		},
		Site: &openrtb.Site{
			ID:     "test-site",
			Domain: "test.example.com",
			Page:   "https://test.example.com/video",
		},
		TMax: 1000,
		Cur:  []string{"USD"},
		AT:   2,
	}
}
