package pauseads_test

import (
	"encoding/json"
	"fmt"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/pauseads"
)

// ExampleDetectPauseEvent demonstrates how to detect pause events from OpenRTB requests
func ExampleDetectPauseEvent() {
	// Create an OpenRTB bid request with pause signal
	bidRequest := &openrtb2.BidRequest{
		ID: "example-request-1",
		Ext: json.RawMessage(`{
			"pause": {
				"state": "paused",
				"format": "display",
				"sessionId": "session-example-1"
			}
		}`),
		Imp: []openrtb2.Imp{
			{
				ID: "imp-1",
				Banner: &openrtb2.Banner{
					W: int64Ptr(300),
					H: int64Ptr(250),
				},
			},
		},
	}

	// Detect the pause event
	pauseReq, err := pauseads.DetectPauseEvent(bidRequest)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("State: %s\n", pauseReq.State)
	fmt.Printf("Format: %s\n", pauseReq.Format)
	fmt.Printf("Session ID: %s\n", pauseReq.SessionID)

	// Output:
	// State: paused
	// Format: display
	// Session ID: session-example-1
}

// ExampleServePauseAd demonstrates how to serve a display pause ad
func ExampleServePauseAd_display() {
	// Create a pause ad request
	pauseReq := &pauseads.PauseAdRequest{
		State:  pauseads.StatePaused,
		Format: pauseads.FormatDisplay,
		BidRequest: &openrtb2.BidRequest{
			ID: "example-request-2",
		},
	}

	// Create a bid response with a winning bid
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
						AdM:   "<div style='background-color: blue; color: white;'>Pause Ad</div>",
						NURL:  "https://example.com/impression",
					},
				},
			},
		},
	}

	// Serve the pause ad
	pauseResp, err := pauseads.ServePauseAd(pauseReq, bidResponse)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Format: %s\n", pauseResp.Format)
	fmt.Printf("Bid ID: %s\n", pauseResp.BidID)
	fmt.Printf("Price: %.2f %s\n", pauseResp.Price, pauseResp.Currency)
	fmt.Printf("Has Ad Markup: %v\n", len(pauseResp.AdMarkup) > 0)
	fmt.Printf("Impression Trackers: %d\n", len(pauseResp.ImpressionTrackers))

	// Output:
	// Format: display
	// Bid ID: bid-1
	// Price: 2.50 USD
	// Has Ad Markup: true
	// Impression Trackers: 1
}

// ExampleServePauseAd_video demonstrates how to serve a video pause ad
func ExampleServePauseAd_video() {
	vastXML := `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="4.0">
  <Ad id="pause-ad">
    <InLine>
      <AdSystem>ExampleAdServer</AdSystem>
      <AdTitle>Video Pause Ad</AdTitle>
      <Impression><![CDATA[https://example.com/vast-impression]]></Impression>
      <Creatives>
        <Creative>
          <Linear>
            <Duration>00:00:15</Duration>
            <MediaFiles>
              <MediaFile delivery="progressive" type="video/mp4" width="1920" height="1080">
                <![CDATA[https://example.com/pause-video.mp4]]>
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

	pauseReq := &pauseads.PauseAdRequest{
		State:  pauseads.StatePaused,
		Format: pauseads.FormatVideo,
		BidRequest: &openrtb2.BidRequest{
			ID: "example-request-3",
		},
	}

	bidResponse := &openrtb2.BidResponse{
		ID:  "response-2",
		Cur: "USD",
		SeatBid: []openrtb2.SeatBid{
			{
				Bid: []openrtb2.Bid{
					{
						ID:    "bid-video",
						ImpID: "imp-1",
						Price: 4.00,
						AdM:   vastXML,
					},
				},
			},
		},
	}

	pauseResp, err := pauseads.ServePauseAd(pauseReq, bidResponse)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Format: %s\n", pauseResp.Format)
	fmt.Printf("Bid ID: %s\n", pauseResp.BidID)
	fmt.Printf("Price: %.2f %s\n", pauseResp.Price, pauseResp.Currency)
	fmt.Printf("Has VAST Markup: %v\n", len(pauseResp.AdMarkup) > 0)
	fmt.Printf("Impression Trackers: %d\n", len(pauseResp.ImpressionTrackers))
	fmt.Printf("Click Trackers: %d\n", len(pauseResp.ClickTrackers))

	// Output:
	// Format: video
	// Bid ID: bid-video
	// Price: 4.00 USD
	// Has VAST Markup: true
	// Impression Trackers: 1
	// Click Trackers: 1
}

// ExampleHandleResume demonstrates how to handle content resume events
func ExampleHandleResume() {
	sessionID := "session-123"

	err := pauseads.HandleResume(sessionID)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("Resume handled successfully")

	// Output:
	// Resume handled successfully
}

// ExampleIsValidPauseAdRequest demonstrates request validation
func ExampleIsValidPauseAdRequest() {
	// Valid pause request
	validReq := &pauseads.PauseAdRequest{
		State:  pauseads.StatePaused,
		Format: pauseads.FormatDisplay,
		BidRequest: &openrtb2.BidRequest{
			ID: "test-request",
		},
		SessionID: "session-456",
	}

	err := pauseads.IsValidPauseAdRequest(validReq)
	fmt.Printf("Valid request error: %v\n", err)

	// Invalid pause request (missing bid request)
	invalidReq := &pauseads.PauseAdRequest{
		State:  pauseads.StatePaused,
		Format: pauseads.FormatDisplay,
	}

	err = pauseads.IsValidPauseAdRequest(invalidReq)
	fmt.Printf("Invalid request has error: %v\n", err != nil)

	// Output:
	// Valid request error: <nil>
	// Invalid request has error: true
}

// Helper function to create int64 pointer
func int64Ptr(v int64) *int64 {
	return &v
}
