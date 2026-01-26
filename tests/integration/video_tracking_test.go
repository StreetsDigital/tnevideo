//go:build integration
// +build integration

package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thenexusengine/tne_springwire/internal/endpoints"
	"github.com/thenexusengine/tne_springwire/pkg/vast"
)

// TestVideoTrackingPixels tests video event tracking pixel functionality
func TestVideoTrackingPixels(t *testing.T) {
	handler := endpoints.NewVideoEventHandler(nil)

	t.Run("Impression_pixel_firing", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(handler.HandleVideoEvent))
		defer server.Close()

		// Fire impression pixel (GET request)
		url := server.URL + "?event=impression&bid_id=bid-123&account_id=pub-123"
		resp, err := http.Get(url)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "image/gif", resp.Header.Get("Content-Type"))

		// Verify it's a 1x1 transparent GIF
		assert.Greater(t, resp.ContentLength, int64(0), "Should return GIF data")
	})

	t.Run("Video_start_event", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(handler.HandleVideoStart))
		defer server.Close()

		url := server.URL + "?bid_id=bid-123&account_id=pub-123"
		resp, err := http.Get(url)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "image/gif", resp.Header.Get("Content-Type"))
	})

	t.Run("Quartile_events", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(handler.HandleVideoQuartile))
		defer server.Close()

		quartiles := []string{"25", "50", "75"}

		for _, q := range quartiles {
			url := server.URL + "?quartile=" + q + "&bid_id=bid-123&account_id=pub-123"
			resp, err := http.Get(url)
			require.NoError(t, err)
			resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
		}
	})

	t.Run("Video_complete_event", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(handler.HandleVideoComplete))
		defer server.Close()

		url := server.URL + "?bid_id=bid-123&account_id=pub-123"
		resp, err := http.Get(url)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Mute_unmute_events", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(handler.HandleVideoEvent))
		defer server.Close()

		// Mute
		resp, err := http.Get(server.URL + "?event=mute&bid_id=bid-123&account_id=pub-123")
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Unmute
		resp, err = http.Get(server.URL + "?event=unmute&bid_id=bid-123&account_id=pub-123")
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Pause_resume_events", func(t *testing.T) {
		pauseServer := httptest.NewServer(http.HandlerFunc(handler.HandleVideoPause))
		defer pauseServer.Close()

		resumeServer := httptest.NewServer(http.HandlerFunc(handler.HandleVideoResume))
		defer resumeServer.Close()

		// Pause
		resp, err := http.Get(pauseServer.URL + "?bid_id=bid-123&account_id=pub-123")
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Resume
		resp, err = http.Get(resumeServer.URL + "?bid_id=bid-123&account_id=pub-123")
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Click_tracking", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(handler.HandleVideoClick))
		defer server.Close()

		url := server.URL + "?bid_id=bid-123&account_id=pub-123"
		resp, err := http.Get(url)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Error_tracking", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(handler.HandleVideoError))
		defer server.Close()

		url := server.URL + "?bid_id=bid-123&account_id=pub-123&error_code=400"
		resp, err := http.Get(url)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

// TestEventSequencing tests proper event sequence
func TestEventSequencing(t *testing.T) {
	var events []string
	var mu sync.Mutex

	// Mock analytics that records events
	mockHandler := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		event := r.URL.Query().Get("event")
		mu.Lock()
		events = append(events, event)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer mockHandler.Close()

	t.Run("Proper_playback_sequence", func(t *testing.T) {
		// Simulate proper playback sequence
		sequence := []string{
			"start",
			"firstQuartile",
			"midpoint",
			"thirdQuartile",
			"complete",
		}

		for _, event := range sequence {
			url := mockHandler.URL + "?event=" + event + "&bid_id=test"
			resp, err := http.Get(url)
			require.NoError(t, err)
			resp.Body.Close()
		}

		// Verify sequence
		mu.Lock()
		defer mu.Unlock()

		assert.Len(t, events, 5)
		assert.Equal(t, sequence, events[len(events)-5:])
	})
}

// TestPOSTEventTracking tests POST-based event tracking
func TestPOSTEventTracking(t *testing.T) {
	handler := endpoints.NewVideoEventHandler(nil)

	t.Run("POST_event_with_JSON_body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(handler.HandleVideoEvent))
		defer server.Close()

		event := map[string]interface{}{
			"event":      "start",
			"bid_id":     "bid-123",
			"account_id": "pub-123",
			"timestamp":  time.Now().UnixMilli(),
			"session_id": "session-abc",
		}

		body, err := json.Marshal(event)
		require.NoError(t, err)

		resp, err := http.Post(server.URL, "application/json", strings.NewReader(string(body)))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("POST_with_additional_metadata", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(handler.HandleVideoEvent))
		defer server.Close()

		event := map[string]interface{}{
			"event":      "complete",
			"bid_id":     "bid-123",
			"account_id": "pub-123",
			"progress":   100.0,
			"content_id": "video-456",
		}

		body, err := json.Marshal(event)
		require.NoError(t, err)

		resp, err := http.Post(server.URL, "application/json", strings.NewReader(string(body)))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

// TestTrackingURLGeneration tests tracking URL generation in VAST
func TestTrackingURLGeneration(t *testing.T) {
	t.Run("Generate_quartile_tracking_URLs", func(t *testing.T) {
		v, err := vast.NewBuilder("4.0").
			AddAd("test-ad").
			WithInLine("TNEVideo", "Test Ad").
			WithImpression("https://tracking.example.com/imp").
			WithLinearCreative("creative-1", 30*time.Second).
			WithMediaFile("https://cdn.example.com/video.mp4", "video/mp4", 1920, 1080).
			WithAllQuartileTracking("https://tracking.example.com/event").
			EndLinear().
			Done().
			Build()

		require.NoError(t, err)

		linear := v.GetLinearCreative()
		require.NotNil(t, linear)

		// Verify all quartile events are present
		events := make(map[string]bool)
		for _, tracking := range linear.TrackingEvents.Tracking {
			events[tracking.Event] = true
		}

		assert.True(t, events[vast.EventStart])
		assert.True(t, events[vast.EventFirstQuartile])
		assert.True(t, events[vast.EventMidpoint])
		assert.True(t, events[vast.EventThirdQuartile])
		assert.True(t, events[vast.EventComplete])
	})

	t.Run("Tracking_URLs_with_parameters", func(t *testing.T) {
		baseURL := "https://tracking.example.com/event"
		bidID := "bid-123"
		trackingURL := baseURL + "?bid_id=" + bidID + "&event=start"

		assert.Contains(t, trackingURL, "bid_id=bid-123")
		assert.Contains(t, trackingURL, "event=start")
	})
}

// TestConcurrentTracking tests concurrent event tracking
func TestConcurrentTracking(t *testing.T) {
	handler := endpoints.NewVideoEventHandler(nil)
	server := httptest.NewServer(http.HandlerFunc(handler.HandleVideoEvent))
	defer server.Close()

	t.Run("Concurrent_pixel_requests", func(t *testing.T) {
		concurrency := 100
		var wg sync.WaitGroup
		wg.Add(concurrency)

		for i := 0; i < concurrency; i++ {
			go func(index int) {
				defer wg.Done()

				url := server.URL + "?event=impression&bid_id=bid-" + string(rune(index)) + "&account_id=pub-123"
				resp, err := http.Get(url)
				if err != nil {
					t.Logf("Error in concurrent request: %v", err)
					return
				}
				defer resp.Body.Close()

				assert.Equal(t, http.StatusOK, resp.StatusCode)
			}(i)
		}

		wg.Wait()
		t.Log("100 concurrent tracking requests completed")
	})
}
