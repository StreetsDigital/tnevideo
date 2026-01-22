package pauseads

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectPauseEvent(t *testing.T) {
	tests := []struct {
		name        string
		bidRequest  *openrtb2.BidRequest
		want        *PauseAdRequest
		wantErr     bool
		errContains string
	}{
		{
			name:        "nil_bid_request",
			bidRequest:  nil,
			want:        nil,
			wantErr:     true,
			errContains: "bid request cannot be nil",
		},
		{
			name: "no_pause_signal",
			bidRequest: &openrtb2.BidRequest{
				ID: "test-request-1",
			},
			want:        nil,
			wantErr:     true,
			errContains: "no pause state detected",
		},
		{
			name: "valid_pause_display",
			bidRequest: &openrtb2.BidRequest{
				ID: "test-request-2",
				Ext: json.RawMessage(`{
					"pause": {
						"state": "paused",
						"format": "display",
						"sessionId": "session-123",
						"timestamp": 1234567890000
					}
				}`),
			},
			want: &PauseAdRequest{
				State:     StatePaused,
				Format:    FormatDisplay,
				SessionID: "session-123",
				Timestamp: 1234567890000,
			},
			wantErr: false,
		},
		{
			name: "valid_pause_video",
			bidRequest: &openrtb2.BidRequest{
				ID: "test-request-3",
				Ext: json.RawMessage(`{
					"pause": {
						"state": "paused",
						"format": "video",
						"sessionId": "session-456"
					}
				}`),
			},
			want: &PauseAdRequest{
				State:     StatePaused,
				Format:    FormatVideo,
				SessionID: "session-456",
			},
			wantErr: false,
		},
		{
			name: "valid_resume",
			bidRequest: &openrtb2.BidRequest{
				ID: "test-request-4",
				Ext: json.RawMessage(`{
					"pause": {
						"state": "resumed",
						"sessionId": "session-789"
					}
				}`),
			},
			want: &PauseAdRequest{
				State:     StateResumed,
				Format:    FormatDisplay, // default
				SessionID: "session-789",
			},
			wantErr: false,
		},
		{
			name: "default_format_display",
			bidRequest: &openrtb2.BidRequest{
				ID: "test-request-5",
				Ext: json.RawMessage(`{
					"pause": {
						"state": "paused",
						"sessionId": "session-default"
					}
				}`),
			},
			want: &PauseAdRequest{
				State:     StatePaused,
				Format:    FormatDisplay,
				SessionID: "session-default",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DetectPauseEvent(tt.bidRequest)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want.State, got.State)
			assert.Equal(t, tt.want.Format, got.Format)
			assert.Equal(t, tt.want.SessionID, got.SessionID)
			if tt.want.Timestamp > 0 {
				assert.Equal(t, tt.want.Timestamp, got.Timestamp)
			}
			assert.NotNil(t, got.BidRequest)
		})
	}
}

func TestServePauseAd_DisplayFormat(t *testing.T) {
	tests := []struct {
		name        string
		pauseReq    *PauseAdRequest
		bidResponse *openrtb2.BidResponse
		want        *PauseAdResponse
		wantErr     bool
	}{
		{
			name:     "nil_pause_request",
			pauseReq: nil,
			want:     nil,
			wantErr:  true,
		},
		{
			name: "resumed_state",
			pauseReq: &PauseAdRequest{
				State:  StateResumed,
				Format: FormatDisplay,
			},
			bidResponse: nil,
			want: &PauseAdResponse{
				Format: FormatDisplay,
			},
			wantErr: false,
		},
		{
			name: "no_bid_response",
			pauseReq: &PauseAdRequest{
				State:  StatePaused,
				Format: FormatDisplay,
			},
			bidResponse: nil,
			want: &PauseAdResponse{
				Format: FormatDisplay,
				Error:  "no bids available for pause ad",
			},
			wantErr: false,
		},
		{
			name: "empty_seat_bids",
			pauseReq: &PauseAdRequest{
				State:  StatePaused,
				Format: FormatDisplay,
			},
			bidResponse: &openrtb2.BidResponse{
				ID:      "response-1",
				SeatBid: []openrtb2.SeatBid{},
			},
			want: &PauseAdResponse{
				Format: FormatDisplay,
				Error:  "no bids available for pause ad",
			},
			wantErr: false,
		},
		{
			name: "valid_display_ad",
			pauseReq: &PauseAdRequest{
				State:  StatePaused,
				Format: FormatDisplay,
			},
			bidResponse: &openrtb2.BidResponse{
				ID:  "response-2",
				Cur: "USD",
				SeatBid: []openrtb2.SeatBid{
					{
						Bid: []openrtb2.Bid{
							{
								ID:    "bid-1",
								ImpID: "imp-1",
								Price: 2.50,
								AdM:   "<div>Pause Ad Creative</div>",
								NURL:  "https://example.com/nurl",
							},
						},
					},
				},
			},
			want: &PauseAdResponse{
				Format:             FormatDisplay,
				BidID:              "bid-1",
				Price:              2.50,
				Currency:           "USD",
				AdMarkup:           "<div>Pause Ad Creative</div>",
				ImpressionTrackers: []string{"https://example.com/nurl"},
			},
			wantErr: false,
		},
		{
			name: "highest_price_wins",
			pauseReq: &PauseAdRequest{
				State:  StatePaused,
				Format: FormatDisplay,
			},
			bidResponse: &openrtb2.BidResponse{
				ID:  "response-3",
				Cur: "USD",
				SeatBid: []openrtb2.SeatBid{
					{
						Bid: []openrtb2.Bid{
							{
								ID:    "bid-low",
								ImpID: "imp-1",
								Price: 1.00,
								AdM:   "<div>Low Bid</div>",
							},
						},
					},
					{
						Bid: []openrtb2.Bid{
							{
								ID:    "bid-high",
								ImpID: "imp-1",
								Price: 5.00,
								AdM:   "<div>High Bid</div>",
								NURL:  "https://example.com/high-nurl",
							},
						},
					},
				},
			},
			want: &PauseAdResponse{
				Format:             FormatDisplay,
				BidID:              "bid-high",
				Price:              5.00,
				Currency:           "USD",
				AdMarkup:           "<div>High Bid</div>",
				ImpressionTrackers: []string{"https://example.com/high-nurl"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ServePauseAd(tt.pauseReq, tt.bidResponse)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want.Format, got.Format)
			assert.Equal(t, tt.want.BidID, got.BidID)
			assert.Equal(t, tt.want.Price, got.Price)
			assert.Equal(t, tt.want.Currency, got.Currency)
			assert.Equal(t, tt.want.AdMarkup, got.AdMarkup)
			assert.Equal(t, tt.want.Error, got.Error)

			if len(tt.want.ImpressionTrackers) > 0 {
				assert.ElementsMatch(t, tt.want.ImpressionTrackers, got.ImpressionTrackers)
			}
		})
	}
}

func TestServePauseAd_VideoFormat(t *testing.T) {
	vastXML := `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="4.0">
  <Ad id="test-ad">
    <InLine>
      <AdSystem>TestAdServer</AdSystem>
      <AdTitle>Pause Ad Video</AdTitle>
      <Impression><![CDATA[https://example.com/impression]]></Impression>
      <Creatives>
        <Creative>
          <Linear>
            <Duration>00:00:15</Duration>
            <MediaFiles>
              <MediaFile delivery="progressive" type="video/mp4" width="1920" height="1080">
                <![CDATA[https://example.com/video.mp4]]>
              </MediaFile>
            </MediaFiles>
            <VideoClicks>
              <ClickThrough><![CDATA[https://example.com/clickthrough]]></ClickThrough>
              <ClickTracking><![CDATA[https://example.com/clicktracking]]></ClickTracking>
            </VideoClicks>
          </Linear>
        </Creative>
      </Creatives>
    </InLine>
  </Ad>
</VAST>`

	tests := []struct {
		name        string
		pauseReq    *PauseAdRequest
		bidResponse *openrtb2.BidResponse
		wantFormat  PauseAdFormat
		wantErr     bool
		checkFunc   func(t *testing.T, resp *PauseAdResponse)
	}{
		{
			name: "valid_video_ad_with_vast",
			pauseReq: &PauseAdRequest{
				State:  StatePaused,
				Format: FormatVideo,
			},
			bidResponse: &openrtb2.BidResponse{
				ID:  "response-video-1",
				Cur: "USD",
				SeatBid: []openrtb2.SeatBid{
					{
						Bid: []openrtb2.Bid{
							{
								ID:    "bid-video-1",
								ImpID: "imp-1",
								Price: 3.75,
								AdM:   vastXML,
								NURL:  "https://example.com/video-nurl",
							},
						},
					},
				},
			},
			wantFormat: FormatVideo,
			wantErr:    false,
			checkFunc: func(t *testing.T, resp *PauseAdResponse) {
				assert.Equal(t, "bid-video-1", resp.BidID)
				assert.Equal(t, 3.75, resp.Price)
				assert.Equal(t, "USD", resp.Currency)
				assert.Contains(t, resp.AdMarkup, "VAST")
				assert.Contains(t, resp.ImpressionTrackers, "https://example.com/impression")
				assert.Contains(t, resp.ImpressionTrackers, "https://example.com/video-nurl")
				assert.Contains(t, resp.ClickTrackers, "https://example.com/clickthrough")
				assert.Contains(t, resp.ClickTrackers, "https://example.com/clicktracking")
			},
		},
		{
			name: "invalid_vast_xml",
			pauseReq: &PauseAdRequest{
				State:  StatePaused,
				Format: FormatVideo,
			},
			bidResponse: &openrtb2.BidResponse{
				ID:  "response-video-2",
				Cur: "USD",
				SeatBid: []openrtb2.SeatBid{
					{
						Bid: []openrtb2.Bid{
							{
								ID:    "bid-video-2",
								ImpID: "imp-1",
								Price: 2.00,
								AdM:   "invalid xml",
							},
						},
					},
				},
			},
			wantFormat: FormatVideo,
			wantErr:    false,
			checkFunc: func(t *testing.T, resp *PauseAdResponse) {
				assert.NotEmpty(t, resp.Error)
				assert.Contains(t, resp.Error, "failed to parse VAST")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ServePauseAd(tt.pauseReq, tt.bidResponse)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantFormat, got.Format)

			if tt.checkFunc != nil {
				tt.checkFunc(t, got)
			}
		})
	}
}

func TestHandleResume(t *testing.T) {
	tests := []struct {
		name        string
		sessionID   string
		wantErr     bool
		errContains string
	}{
		{
			name:        "empty_session_id",
			sessionID:   "",
			wantErr:     true,
			errContains: "session ID is required",
		},
		{
			name:      "valid_session_id",
			sessionID: "session-123",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := HandleResume(tt.sessionID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestIsValidPauseAdRequest(t *testing.T) {
	tests := []struct {
		name        string
		req         *PauseAdRequest
		wantErr     bool
		errContains string
	}{
		{
			name:        "nil_request",
			req:         nil,
			wantErr:     true,
			errContains: "request cannot be nil",
		},
		{
			name: "empty_state",
			req: &PauseAdRequest{
				Format: FormatDisplay,
			},
			wantErr:     true,
			errContains: "state is required",
		},
		{
			name: "invalid_state",
			req: &PauseAdRequest{
				State: "invalid",
			},
			wantErr:     true,
			errContains: "invalid state",
		},
		{
			name: "invalid_format",
			req: &PauseAdRequest{
				State:  StatePaused,
				Format: "invalid",
			},
			wantErr:     true,
			errContains: "invalid format",
		},
		{
			name: "paused_without_bid_request",
			req: &PauseAdRequest{
				State:      StatePaused,
				Format:     FormatDisplay,
				BidRequest: nil,
			},
			wantErr:     true,
			errContains: "bid request is required",
		},
		{
			name: "valid_paused_display",
			req: &PauseAdRequest{
				State:  StatePaused,
				Format: FormatDisplay,
				BidRequest: &openrtb2.BidRequest{
					ID: "test-1",
				},
			},
			wantErr: false,
		},
		{
			name: "valid_paused_video",
			req: &PauseAdRequest{
				State:  StatePaused,
				Format: FormatVideo,
				BidRequest: &openrtb2.BidRequest{
					ID: "test-2",
				},
			},
			wantErr: false,
		},
		{
			name: "valid_resumed",
			req: &PauseAdRequest{
				State:     StateResumed,
				SessionID: "session-1",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := IsValidPauseAdRequest(tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestPauseAdRequest_RoundTrip(t *testing.T) {
	// Test that we can detect a pause event and serve an ad in a complete flow
	bidRequest := &openrtb2.BidRequest{
		ID: "test-request",
		Ext: json.RawMessage(`{
			"pause": {
				"state": "paused",
				"format": "display",
				"sessionId": "session-xyz",
				"timestamp": 1234567890000
			}
		}`),
		Imp: []openrtb2.Imp{
			{
				ID: "imp-1",
				Banner: &openrtb2.Banner{
					W: ptrInt64(300),
					H: ptrInt64(250),
				},
			},
		},
	}

	// Detect pause event
	pauseReq, err := DetectPauseEvent(bidRequest)
	require.NoError(t, err)
	assert.Equal(t, StatePaused, pauseReq.State)
	assert.Equal(t, FormatDisplay, pauseReq.Format)

	// Validate the request
	err = IsValidPauseAdRequest(pauseReq)
	require.NoError(t, err)

	// Serve the pause ad
	bidResponse := &openrtb2.BidResponse{
		ID:  "response-1",
		Cur: "USD",
		SeatBid: []openrtb2.SeatBid{
			{
				Bid: []openrtb2.Bid{
					{
						ID:    "bid-1",
						ImpID: "imp-1",
						Price: 2.50,
						AdM:   "<div>Test Pause Ad</div>",
						NURL:  "https://example.com/nurl",
					},
				},
			},
		},
	}

	pauseResp, err := ServePauseAd(pauseReq, bidResponse)
	require.NoError(t, err)
	assert.Equal(t, FormatDisplay, pauseResp.Format)
	assert.Equal(t, "bid-1", pauseResp.BidID)
	assert.Equal(t, 2.50, pauseResp.Price)
	assert.Contains(t, pauseResp.AdMarkup, "Test Pause Ad")
	assert.Contains(t, pauseResp.ImpressionTrackers, "https://example.com/nurl")

	// Handle resume
	err = HandleResume(pauseReq.SessionID)
	require.NoError(t, err)
}

func TestPauseAdConstants(t *testing.T) {
	// Test that constants are defined correctly
	assert.Equal(t, PauseAdState("paused"), StatePaused)
	assert.Equal(t, PauseAdState("resumed"), StateResumed)
	assert.Equal(t, PauseAdFormat("display"), FormatDisplay)
	assert.Equal(t, PauseAdFormat("video"), FormatVideo)
}

// Helper function to create int64 pointer
func ptrInt64(v int64) *int64 {
	return &v
}
