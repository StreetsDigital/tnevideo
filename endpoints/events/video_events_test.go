package events

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/stretchr/testify/assert"
)

func TestVideoStartEndpoint(t *testing.T) {
	cfg := &config.Configuration{
		Event: config.Event{
			TimeoutMS: 1000,
		},
	}

	tests := []struct {
		name            string
		requestBody     VideoEventRequest
		accountID       string
		expectedStatus  int
		expectAnalytics bool
	}{
		{
			name: "Valid start event",
			requestBody: VideoEventRequest{
				BidID:       "bid-123",
				AccountID:   "events_enabled",
				Bidder:      "appnexus",
				Timestamp:   1706543210000,
				Integration: "prebid-video-1.0",
				Analytics:   "1",
			},
			accountID:       "events_enabled",
			expectedStatus:  http.StatusNoContent,
			expectAnalytics: true,
		},
		{
			name: "Missing bidId",
			requestBody: VideoEventRequest{
				AccountID: "events_enabled",
				Analytics: "1",
			},
			accountID:       "events_enabled",
			expectedStatus:  http.StatusBadRequest,
			expectAnalytics: false,
		},
		{
			name: "Missing accountId",
			requestBody: VideoEventRequest{
				BidID:     "bid-123",
				Analytics: "1",
			},
			accountID:       "",
			expectedStatus:  http.StatusUnauthorized,
			expectAnalytics: false,
		},
		{
			name: "Events disabled for account",
			requestBody: VideoEventRequest{
				BidID:     "bid-123",
				AccountID: "events_disabled",
				Analytics: "1",
			},
			accountID:       "events_disabled",
			expectedStatus:  http.StatusUnauthorized,
			expectAnalytics: false,
		},
		{
			name: "Analytics disabled in request",
			requestBody: VideoEventRequest{
				BidID:     "bid-123",
				AccountID: "events_enabled",
				Analytics: "0",
			},
			accountID:       "events_enabled",
			expectedStatus:  http.StatusNoContent,
			expectAnalytics: false,
		},
		{
			name: "Image format response",
			requestBody: VideoEventRequest{
				BidID:     "bid-123",
				AccountID: "events_enabled",
				Format:    "i",
				Analytics: "1",
			},
			accountID:       "events_enabled",
			expectedStatus:  http.StatusOK,
			expectAnalytics: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyticsModule := &eventsMockAnalyticsModule{}
			accountsFetcher := mockAccountsFetcher{}
			metricsEngine := &metrics.MetricsEngineMock{}

			endpoint := NewVideoStartEndpoint(cfg, accountsFetcher, analyticsModule, metricsEngine)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/v1/video/start", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			endpoint(rr, req, nil)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			assert.Equal(t, tt.expectAnalytics, analyticsModule.Invoked)

			if tt.requestBody.Format == "i" && tt.expectedStatus == http.StatusOK {
				assert.Equal(t, "image/png", rr.Header().Get("Content-Type"))
			}
		})
	}
}

func TestVideoQuartileEndpoint(t *testing.T) {
	cfg := &config.Configuration{
		Event: config.Event{
			TimeoutMS: 1000,
		},
	}

	tests := []struct {
		name            string
		requestBody     VideoEventRequest
		quartile        string
		expectedStatus  int
		expectAnalytics bool
		expectedVType   analytics.VastType
	}{
		{
			name: "First quartile (25%)",
			requestBody: VideoEventRequest{
				BidID:     "bid-123",
				AccountID: "events_enabled",
				Analytics: "1",
			},
			quartile:        "25",
			expectedStatus:  http.StatusNoContent,
			expectAnalytics: true,
			expectedVType:   analytics.FirstQuartile,
		},
		{
			name: "First quartile (firstQuartile)",
			requestBody: VideoEventRequest{
				BidID:     "bid-123",
				AccountID: "events_enabled",
				Analytics: "1",
			},
			quartile:        "firstQuartile",
			expectedStatus:  http.StatusNoContent,
			expectAnalytics: true,
			expectedVType:   analytics.FirstQuartile,
		},
		{
			name: "Midpoint (50%)",
			requestBody: VideoEventRequest{
				BidID:     "bid-123",
				AccountID: "events_enabled",
				Analytics: "1",
			},
			quartile:        "50",
			expectedStatus:  http.StatusNoContent,
			expectAnalytics: true,
			expectedVType:   analytics.MidPoint,
		},
		{
			name: "Midpoint (midPoint)",
			requestBody: VideoEventRequest{
				BidID:     "bid-123",
				AccountID: "events_enabled",
				Analytics: "1",
			},
			quartile:        "midPoint",
			expectedStatus:  http.StatusNoContent,
			expectAnalytics: true,
			expectedVType:   analytics.MidPoint,
		},
		{
			name: "Third quartile (75%)",
			requestBody: VideoEventRequest{
				BidID:     "bid-123",
				AccountID: "events_enabled",
				Analytics: "1",
			},
			quartile:        "75",
			expectedStatus:  http.StatusNoContent,
			expectAnalytics: true,
			expectedVType:   analytics.ThirdQuartile,
		},
		{
			name: "Third quartile (thirdQuartile)",
			requestBody: VideoEventRequest{
				BidID:     "bid-123",
				AccountID: "events_enabled",
				Analytics: "1",
			},
			quartile:        "thirdQuartile",
			expectedStatus:  http.StatusNoContent,
			expectAnalytics: true,
			expectedVType:   analytics.ThirdQuartile,
		},
		{
			name: "Invalid quartile",
			requestBody: VideoEventRequest{
				BidID:     "bid-123",
				AccountID: "events_enabled",
				Analytics: "1",
			},
			quartile:        "100",
			expectedStatus:  http.StatusBadRequest,
			expectAnalytics: false,
		},
		{
			name: "Missing quartile parameter",
			requestBody: VideoEventRequest{
				BidID:     "bid-123",
				AccountID: "events_enabled",
				Analytics: "1",
			},
			quartile:        "",
			expectedStatus:  http.StatusBadRequest,
			expectAnalytics: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyticsModule := &eventsMockAnalyticsModule{}
			accountsFetcher := mockAccountsFetcher{}
			metricsEngine := &metrics.MetricsEngineMock{}

			endpoint := NewVideoQuartileEndpoint(cfg, accountsFetcher, analyticsModule, metricsEngine)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/v1/video/quartile?quartile="+tt.quartile, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			endpoint(rr, req, nil)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			assert.Equal(t, tt.expectAnalytics, analyticsModule.Invoked)
		})
	}
}

func TestVideoCompleteEndpoint(t *testing.T) {
	cfg := &config.Configuration{
		Event: config.Event{
			TimeoutMS: 1000,
		},
	}

	tests := []struct {
		name            string
		requestBody     VideoEventRequest
		expectedStatus  int
		expectAnalytics bool
	}{
		{
			name: "Valid complete event",
			requestBody: VideoEventRequest{
				BidID:       "bid-123",
				AccountID:   "events_enabled",
				Bidder:      "appnexus",
				Timestamp:   1706543210000,
				Integration: "prebid-video-1.0",
				Analytics:   "1",
			},
			expectedStatus:  http.StatusNoContent,
			expectAnalytics: true,
		},
		{
			name: "Complete with blank format",
			requestBody: VideoEventRequest{
				BidID:     "bid-123",
				AccountID: "events_enabled",
				Format:    "b",
				Analytics: "1",
			},
			expectedStatus:  http.StatusNoContent,
			expectAnalytics: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyticsModule := &eventsMockAnalyticsModule{}
			accountsFetcher := mockAccountsFetcher{}
			metricsEngine := &metrics.MetricsEngineMock{}

			endpoint := NewVideoCompleteEndpoint(cfg, accountsFetcher, analyticsModule, metricsEngine)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/v1/video/complete", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			endpoint(rr, req, nil)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			assert.Equal(t, tt.expectAnalytics, analyticsModule.Invoked)
		})
	}
}

func TestVideoClickEndpoint(t *testing.T) {
	cfg := &config.Configuration{
		Event: config.Event{
			TimeoutMS: 1000,
		},
	}

	tests := []struct {
		name            string
		requestBody     VideoEventRequest
		expectedStatus  int
		expectAnalytics bool
	}{
		{
			name: "Valid click event",
			requestBody: VideoEventRequest{
				BidID:        "bid-123",
				AccountID:    "events_enabled",
				ClickThrough: "https://example.com/landing",
				Analytics:    "1",
			},
			expectedStatus:  http.StatusNoContent,
			expectAnalytics: true,
		},
		{
			name: "Click without click-through URL",
			requestBody: VideoEventRequest{
				BidID:     "bid-123",
				AccountID: "events_enabled",
				Analytics: "1",
			},
			expectedStatus:  http.StatusNoContent,
			expectAnalytics: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyticsModule := &eventsMockAnalyticsModule{}
			accountsFetcher := mockAccountsFetcher{}
			metricsEngine := &metrics.MetricsEngineMock{}

			endpoint := NewVideoClickEndpoint(cfg, accountsFetcher, analyticsModule, metricsEngine)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/v1/video/click", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			endpoint(rr, req, nil)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			assert.Equal(t, tt.expectAnalytics, analyticsModule.Invoked)
		})
	}
}

func TestVideoErrorEndpoint(t *testing.T) {
	cfg := &config.Configuration{
		Event: config.Event{
			TimeoutMS: 1000,
		},
	}

	tests := []struct {
		name            string
		requestBody     VideoEventRequest
		expectedStatus  int
		expectAnalytics bool
	}{
		{
			name: "Valid error event",
			requestBody: VideoEventRequest{
				BidID:        "bid-123",
				AccountID:    "events_enabled",
				ErrorCode:    "400",
				ErrorMessage: "File not found",
				Analytics:    "1",
			},
			expectedStatus:  http.StatusNoContent,
			expectAnalytics: true,
		},
		{
			name: "Error event with VAST error code",
			requestBody: VideoEventRequest{
				BidID:        "bid-123",
				AccountID:    "events_enabled",
				ErrorCode:    "401",
				ErrorMessage: "File not found (broken link to asset)",
				Analytics:    "1",
			},
			expectedStatus:  http.StatusNoContent,
			expectAnalytics: true,
		},
		{
			name: "Error event without error details",
			requestBody: VideoEventRequest{
				BidID:     "bid-123",
				AccountID: "events_enabled",
				Analytics: "1",
			},
			expectedStatus:  http.StatusNoContent,
			expectAnalytics: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyticsModule := &eventsMockAnalyticsModule{}
			accountsFetcher := mockAccountsFetcher{}
			metricsEngine := &metrics.MetricsEngineMock{}

			endpoint := NewVideoErrorEndpoint(cfg, accountsFetcher, analyticsModule, metricsEngine)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/v1/video/error", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			endpoint(rr, req, nil)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			assert.Equal(t, tt.expectAnalytics, analyticsModule.Invoked)
		})
	}
}

func TestVideoEventRequestToAnalyticsRequest(t *testing.T) {
	tests := []struct {
		name        string
		endpoint    *videoEventEndpoint
		request     *VideoEventRequest
		expectedErr bool
		validate    func(*testing.T, *analytics.EventRequest)
	}{
		{
			name: "Complete valid request",
			endpoint: &videoEventEndpoint{
				VType: analytics.Start,
			},
			request: &VideoEventRequest{
				BidID:        "bid-123",
				AccountID:    "acc-456",
				Bidder:       "appnexus",
				Timestamp:    1706543210000,
				Integration:  "prebid-video-1.0",
				Format:       "i",
				Analytics:    "1",
				ClickThrough: "https://example.com",
				ErrorCode:    "400",
				ErrorMessage: "Error message",
			},
			expectedErr: false,
			validate: func(t *testing.T, er *analytics.EventRequest) {
				assert.Equal(t, analytics.Vast, er.Type)
				assert.Equal(t, analytics.Start, er.VType)
				assert.Equal(t, "bid-123", er.BidID)
				assert.Equal(t, "acc-456", er.AccountID)
				assert.Equal(t, "appnexus", er.Bidder)
				assert.Equal(t, int64(1706543210000), er.Timestamp)
				assert.Equal(t, "prebid-video-1.0", er.Integration)
				assert.Equal(t, analytics.Image, er.Format)
				assert.Equal(t, analytics.Enabled, er.Analytics)
				assert.Equal(t, "https://example.com", er.ClickThrough)
				assert.Equal(t, "400", er.ErrorCode)
				assert.Equal(t, "Error message", er.ErrorMessage)
			},
		},
		{
			name: "Missing bidId",
			endpoint: &videoEventEndpoint{
				VType: analytics.Start,
			},
			request: &VideoEventRequest{
				AccountID: "acc-456",
			},
			expectedErr: true,
		},
		{
			name: "Missing accountId",
			endpoint: &videoEventEndpoint{
				VType: analytics.Start,
			},
			request: &VideoEventRequest{
				BidID: "bid-123",
			},
			expectedErr: true,
		},
		{
			name: "Invalid format",
			endpoint: &videoEventEndpoint{
				VType: analytics.Start,
			},
			request: &VideoEventRequest{
				BidID:     "bid-123",
				AccountID: "acc-456",
				Format:    "invalid",
			},
			expectedErr: true,
		},
		{
			name: "Invalid analytics value",
			endpoint: &videoEventEndpoint{
				VType: analytics.Start,
			},
			request: &VideoEventRequest{
				BidID:     "bid-123",
				AccountID: "acc-456",
				Analytics: "invalid",
			},
			expectedErr: true,
		},
		{
			name: "Default analytics enabled",
			endpoint: &videoEventEndpoint{
				VType: analytics.Start,
			},
			request: &VideoEventRequest{
				BidID:     "bid-123",
				AccountID: "acc-456",
			},
			expectedErr: false,
			validate: func(t *testing.T, er *analytics.EventRequest) {
				assert.Equal(t, analytics.Enabled, er.Analytics)
			},
		},
		{
			name: "Default format blank",
			endpoint: &videoEventEndpoint{
				VType: analytics.Start,
			},
			request: &VideoEventRequest{
				BidID:     "bid-123",
				AccountID: "acc-456",
			},
			expectedErr: false,
			validate: func(t *testing.T, er *analytics.EventRequest) {
				assert.Equal(t, analytics.Blank, er.Format)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.endpoint.videoEventRequestToAnalyticsRequest(tt.request)

			if tt.expectedErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

func TestInvalidJSONBody(t *testing.T) {
	cfg := &config.Configuration{
		Event: config.Event{
			TimeoutMS: 1000,
		},
	}

	analyticsModule := &eventsMockAnalyticsModule{}
	accountsFetcher := mockAccountsFetcher{}
	metricsEngine := &metrics.MetricsEngineMock{}

	endpoint := NewVideoStartEndpoint(cfg, accountsFetcher, analyticsModule, metricsEngine)

	req := httptest.NewRequest("POST", "/api/v1/video/start", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	endpoint(rr, req, nil)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid JSON request body")
}

func TestAccountNotFound(t *testing.T) {
	cfg := &config.Configuration{
		Event: config.Event{
			TimeoutMS: 1000,
		},
	}

	analyticsModule := &eventsMockAnalyticsModule{}
	accountsFetcher := mockAccountsFetcher{}
	metricsEngine := &metrics.MetricsEngineMock{}

	endpoint := NewVideoStartEndpoint(cfg, accountsFetcher, analyticsModule, metricsEngine)

	requestBody := VideoEventRequest{
		BidID:     "bid-123",
		AccountID: "nonexistent_account",
		Analytics: "1",
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/api/v1/video/start", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	endpoint(rr, req, nil)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.False(t, analyticsModule.Invoked)
}

func TestIntegrationTypeValidation(t *testing.T) {
	cfg := &config.Configuration{
		Event: config.Event{
			TimeoutMS: 1000,
		},
	}

	tests := []struct {
		name           string
		integration    string
		expectedStatus int
	}{
		{
			name:           "Valid integration",
			integration:    "prebid-video-1.0",
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "Integration with underscore",
			integration:    "prebid_video_1_0",
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "Integration with dash",
			integration:    "prebid-video-1-0",
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "Too long integration",
			integration:    "this-is-a-very-long-integration-type-that-exceeds-the-maximum-allowed-length-of-64-characters",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyticsModule := &eventsMockAnalyticsModule{}
			accountsFetcher := mockAccountsFetcher{}
			metricsEngine := &metrics.MetricsEngineMock{}

			endpoint := NewVideoStartEndpoint(cfg, accountsFetcher, analyticsModule, metricsEngine)

			requestBody := VideoEventRequest{
				BidID:       "bid-123",
				AccountID:   "events_enabled",
				Integration: tt.integration,
				Analytics:   "1",
			}

			body, _ := json.Marshal(requestBody)
			req := httptest.NewRequest("POST", "/api/v1/video/start", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			endpoint(rr, req, nil)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestCompleteVideoPlaybackFlow(t *testing.T) {
	cfg := &config.Configuration{
		Event: config.Event{
			TimeoutMS: 1000,
		},
	}

	// Test a complete video playback sequence
	sequence := []struct {
		endpoint    func(*config.Configuration, mockAccountsFetcher, *eventsMockAnalyticsModule, *metrics.MetricsEngineMock) httprouter.Handle
		path        string
		queryString string
	}{
		{
			endpoint: func(cfg *config.Configuration, af mockAccountsFetcher, am *eventsMockAnalyticsModule, me *metrics.MetricsEngineMock) httprouter.Handle {
				return NewVideoStartEndpoint(cfg, af, am, me)
			},
			path: "/api/v1/video/start",
		},
		{
			endpoint: func(cfg *config.Configuration, af mockAccountsFetcher, am *eventsMockAnalyticsModule, me *metrics.MetricsEngineMock) httprouter.Handle {
				return NewVideoQuartileEndpoint(cfg, af, am, me)
			},
			path:        "/api/v1/video/quartile",
			queryString: "?quartile=25",
		},
		{
			endpoint: func(cfg *config.Configuration, af mockAccountsFetcher, am *eventsMockAnalyticsModule, me *metrics.MetricsEngineMock) httprouter.Handle {
				return NewVideoQuartileEndpoint(cfg, af, am, me)
			},
			path:        "/api/v1/video/quartile",
			queryString: "?quartile=50",
		},
		{
			endpoint: func(cfg *config.Configuration, af mockAccountsFetcher, am *eventsMockAnalyticsModule, me *metrics.MetricsEngineMock) httprouter.Handle {
				return NewVideoQuartileEndpoint(cfg, af, am, me)
			},
			path:        "/api/v1/video/quartile",
			queryString: "?quartile=75",
		},
		{
			endpoint: func(cfg *config.Configuration, af mockAccountsFetcher, am *eventsMockAnalyticsModule, me *metrics.MetricsEngineMock) httprouter.Handle {
				return NewVideoCompleteEndpoint(cfg, af, am, me)
			},
			path: "/api/v1/video/complete",
		},
	}

	for i, step := range sequence {
		analyticsModule := &eventsMockAnalyticsModule{}
		accountsFetcher := mockAccountsFetcher{}
		metricsEngine := &metrics.MetricsEngineMock{}

		handler := step.endpoint(cfg, accountsFetcher, analyticsModule, metricsEngine)

		requestBody := VideoEventRequest{
			BidID:     "bid-123",
			AccountID: "events_enabled",
			Analytics: "1",
		}

		body, _ := json.Marshal(requestBody)
		req := httptest.NewRequest("POST", step.path+step.queryString, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		handler(rr, req, httprouter.Params{})

		assert.Equal(t, http.StatusNoContent, rr.Code, "Step %d failed", i)
		assert.True(t, analyticsModule.Invoked, "Analytics not invoked for step %d", i)
	}
}
