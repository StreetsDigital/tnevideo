package pauseads

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler_HandlePauseAdRequest(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		auctionRunner  func(*openrtb2.BidRequest) (*openrtb2.BidResponse, error)
		expectedStatus int
		checkResponse  func(t *testing.T, resp *PauseAdResponse)
		checkError     func(t *testing.T, body string)
	}{
		{
			name:           "invalid_json_body",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
			checkError: func(t *testing.T, body string) {
				assert.Contains(t, body, "invalid request body")
			},
		},
		{
			name: "no_pause_signal",
			requestBody: &openrtb2.BidRequest{
				ID: "test-1",
			},
			expectedStatus: http.StatusBadRequest,
			checkError: func(t *testing.T, body string) {
				assert.Contains(t, body, "failed to detect pause event")
			},
		},
		{
			name: "valid_pause_display",
			requestBody: &openrtb2.BidRequest{
				ID: "test-2",
				Ext: json.RawMessage(`{
					"pause": {
						"state": "paused",
						"format": "display",
						"sessionId": "session-1"
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
			},
			auctionRunner: func(req *openrtb2.BidRequest) (*openrtb2.BidResponse, error) {
				return &openrtb2.BidResponse{
					ID:  "response-1",
					Cur: "USD",
					SeatBid: []openrtb2.SeatBid{
						{
							Bid: []openrtb2.Bid{
								{
									ID:    "bid-1",
									ImpID: "imp-1",
									Price: 2.50,
									AdM:   "<div>Test Ad</div>",
									NURL:  "https://example.com/nurl",
								},
							},
						},
					},
				}, nil
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *PauseAdResponse) {
				assert.Equal(t, FormatDisplay, resp.Format)
				assert.Equal(t, "bid-1", resp.BidID)
				assert.Equal(t, 2.50, resp.Price)
				assert.Equal(t, "USD", resp.Currency)
				assert.Contains(t, resp.AdMarkup, "Test Ad")
			},
		},
		{
			name: "valid_resume",
			requestBody: &openrtb2.BidRequest{
				ID: "test-3",
				Ext: json.RawMessage(`{
					"pause": {
						"state": "resumed",
						"sessionId": "session-2"
					}
				}`),
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *PauseAdResponse) {
				assert.Equal(t, FormatDisplay, resp.Format)
				assert.Empty(t, resp.BidID)
				assert.Empty(t, resp.AdMarkup)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create handler
			handler := NewHandler(tt.auctionRunner)

			// Create request body
			var body []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				require.NoError(t, err)
			}

			// Create HTTP request
			req := httptest.NewRequest(http.MethodPost, "/pausead", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			w := httptest.NewRecorder()

			// Handle request
			handler.HandlePauseAdRequest(w, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Check response or error
			if tt.expectedStatus == http.StatusOK {
				var resp PauseAdResponse
				err := json.NewDecoder(w.Body).Decode(&resp)
				require.NoError(t, err)

				if tt.checkResponse != nil {
					tt.checkResponse(t, &resp)
				}
			} else {
				if tt.checkError != nil {
					body, _ := io.ReadAll(w.Body)
					tt.checkError(t, string(body))
				}
			}
		})
	}
}

func TestHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		expectedStatus int
	}{
		{
			name:           "method_get_not_allowed",
			method:         http.MethodGet,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "method_put_not_allowed",
			method:         http.MethodPut,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "method_delete_not_allowed",
			method:         http.MethodDelete,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "method_post_allowed",
			method:         http.MethodPost,
			expectedStatus: http.StatusBadRequest, // Will fail on invalid body, but method is allowed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(nil)

			req := httptest.NewRequest(tt.method, "/pausead", bytes.NewReader([]byte("{}")))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_VideoFormat(t *testing.T) {
	vastXML := `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="4.0">
  <Ad id="test-ad">
    <InLine>
      <AdSystem>TestAdServer</AdSystem>
      <AdTitle>Test Pause Ad</AdTitle>
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
            </VideoClicks>
          </Linear>
        </Creative>
      </Creatives>
    </InLine>
  </Ad>
</VAST>`

	handler := NewHandler(func(req *openrtb2.BidRequest) (*openrtb2.BidResponse, error) {
		return &openrtb2.BidResponse{
			ID:  "response-video",
			Cur: "USD",
			SeatBid: []openrtb2.SeatBid{
				{
					Bid: []openrtb2.Bid{
						{
							ID:    "bid-video",
							ImpID: "imp-1",
							Price: 3.75,
							AdM:   vastXML,
						},
					},
				},
			},
		}, nil
	})

	bidRequest := &openrtb2.BidRequest{
		ID: "test-video",
		Ext: json.RawMessage(`{
			"pause": {
				"state": "paused",
				"format": "video",
				"sessionId": "session-video"
			}
		}`),
		Imp: []openrtb2.Imp{
			{
				ID: "imp-1",
				Video: &openrtb2.Video{
					W:     ptrInt64(1920),
					H:     ptrInt64(1080),
					MIMEs: []string{"video/mp4"},
				},
			},
		},
	}

	body, err := json.Marshal(bidRequest)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/pausead", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.HandlePauseAdRequest(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp PauseAdResponse
	err = json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, FormatVideo, resp.Format)
	assert.Equal(t, "bid-video", resp.BidID)
	assert.Equal(t, 3.75, resp.Price)
	assert.Contains(t, resp.AdMarkup, "VAST")
	assert.Contains(t, resp.ImpressionTrackers, "https://example.com/impression")
	assert.Contains(t, resp.ClickTrackers, "https://example.com/clickthrough")
}

func TestNewHandler(t *testing.T) {
	auctionRunner := func(req *openrtb2.BidRequest) (*openrtb2.BidResponse, error) {
		return nil, nil
	}

	handler := NewHandler(auctionRunner)
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.AuctionRunner)
}
