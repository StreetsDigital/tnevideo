package pauseads

import (
	"encoding/json"
	"fmt"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/vast"
)

// PauseAdState represents the current state of a pause ad session
type PauseAdState string

const (
	// StatePaused indicates the content is paused and a pause ad should be served
	StatePaused PauseAdState = "paused"
	// StateResumed indicates the content has resumed and the pause ad should be removed
	StateResumed PauseAdState = "resumed"
)

// PauseAdFormat represents the format of the pause ad
type PauseAdFormat string

const (
	// FormatDisplay represents a display/banner pause ad
	FormatDisplay PauseAdFormat = "display"
	// FormatVideo represents a video pause ad
	FormatVideo PauseAdFormat = "video"
)

// PauseAdRequest represents a request for a pause ad
type PauseAdRequest struct {
	// State indicates whether this is a pause or resume event
	State PauseAdState `json:"state"`

	// Format indicates the desired ad format (display or video)
	Format PauseAdFormat `json:"format"`

	// BidRequest is the OpenRTB bid request for the pause ad
	BidRequest *openrtb2.BidRequest `json:"bidRequest,omitempty"`

	// SessionID uniquely identifies this pause ad session
	SessionID string `json:"sessionId,omitempty"`

	// Timestamp when the pause event occurred (Unix timestamp in milliseconds)
	Timestamp int64 `json:"timestamp,omitempty"`
}

// PauseAdResponse represents the response for a pause ad request
type PauseAdResponse struct {
	// AdMarkup contains the ad creative markup (HTML for display, VAST for video)
	AdMarkup string `json:"adMarkup,omitempty"`

	// Format indicates the format of the returned ad
	Format PauseAdFormat `json:"format"`

	// BidID is the unique identifier for the winning bid
	BidID string `json:"bidId,omitempty"`

	// Price is the winning bid price
	Price float64 `json:"price,omitempty"`

	// Currency is the currency of the bid price
	Currency string `json:"currency,omitempty"`

	// ImpressionTrackers contains URLs to fire when the ad is displayed
	ImpressionTrackers []string `json:"impressionTrackers,omitempty"`

	// ClickTrackers contains URLs to fire when the ad is clicked
	ClickTrackers []string `json:"clickTrackers,omitempty"`

	// Error contains any error message if the request failed
	Error string `json:"error,omitempty"`
}

// DetectPauseEvent detects pause events from OpenRTB bid request signals
// It checks for pause-related signals in the request extensions and device/app context
func DetectPauseEvent(bidRequest *openrtb2.BidRequest) (*PauseAdRequest, error) {
	if bidRequest == nil {
		return nil, fmt.Errorf("bid request cannot be nil")
	}

	pauseReq := &PauseAdRequest{
		BidRequest: bidRequest,
	}

	// Check for pause signal in request extensions
	if bidRequest.Ext != nil {
		var reqExt map[string]interface{}
		if err := json.Unmarshal(bidRequest.Ext, &reqExt); err == nil {
			// Look for pause-specific signals in ext
			if pauseData, ok := reqExt["pause"]; ok {
				if pauseMap, ok := pauseData.(map[string]interface{}); ok {
					// Extract state
					if state, ok := pauseMap["state"].(string); ok {
						pauseReq.State = PauseAdState(state)
					}

					// Extract format
					if format, ok := pauseMap["format"].(string); ok {
						pauseReq.Format = PauseAdFormat(format)
					}

					// Extract session ID
					if sessionID, ok := pauseMap["sessionId"].(string); ok {
						pauseReq.SessionID = sessionID
					}

					// Extract timestamp
					if ts, ok := pauseMap["timestamp"].(float64); ok {
						pauseReq.Timestamp = int64(ts)
					}
				}
			}
		}
	}

	// Default to display format if not specified
	if pauseReq.Format == "" {
		pauseReq.Format = FormatDisplay
	}

	// Validate that we detected a pause state
	if pauseReq.State == "" {
		return nil, fmt.Errorf("no pause state detected in request")
	}

	return pauseReq, nil
}

// ServePauseAd serves a pause ad based on the request
// It returns the appropriate ad markup and tracking information
func ServePauseAd(pauseReq *PauseAdRequest, bidResponse *openrtb2.BidResponse) (*PauseAdResponse, error) {
	if pauseReq == nil {
		return nil, fmt.Errorf("pause ad request cannot be nil")
	}

	// If state is resumed, return empty response
	if pauseReq.State == StateResumed {
		return &PauseAdResponse{
			Format: pauseReq.Format,
		}, nil
	}

	// If no bid response, return error
	if bidResponse == nil || len(bidResponse.SeatBid) == 0 {
		return &PauseAdResponse{
			Format: pauseReq.Format,
			Error:  "no bids available for pause ad",
		}, nil
	}

	// Find the winning bid
	var winningBid *openrtb2.Bid
	var winningPrice float64

	for _, seatBid := range bidResponse.SeatBid {
		for _, bid := range seatBid.Bid {
			if bid.Price > winningPrice {
				winningPrice = bid.Price
				winningBid = &bid
			}
		}
	}

	if winningBid == nil {
		return &PauseAdResponse{
			Format: pauseReq.Format,
			Error:  "no valid bids found",
		}, nil
	}

	response := &PauseAdResponse{
		Format:   pauseReq.Format,
		BidID:    winningBid.ID,
		Price:    winningBid.Price,
		Currency: bidResponse.Cur,
	}

	// Process based on format
	switch pauseReq.Format {
	case FormatDisplay:
		// For display ads, use the ad markup directly
		response.AdMarkup = winningBid.AdM

		// Extract impression trackers from bid extensions
		if winningBid.Ext != nil {
			var bidExt map[string]interface{}
			if err := json.Unmarshal(winningBid.Ext, &bidExt); err == nil {
				if burl, ok := bidExt["burl"].(string); ok && burl != "" {
					response.ImpressionTrackers = append(response.ImpressionTrackers, burl)
				}
			}
		}

		// Add NURL if present
		if winningBid.NURL != "" {
			response.ImpressionTrackers = append(response.ImpressionTrackers, winningBid.NURL)
		}

	case FormatVideo:
		// For video ads, parse VAST and extract necessary information
		if winningBid.AdM != "" {
			vastDoc, err := vast.Parse(winningBid.AdM)
			if err != nil {
				return &PauseAdResponse{
					Format: pauseReq.Format,
					Error:  fmt.Sprintf("failed to parse VAST: %v", err),
				}, nil
			}

			// Use the VAST markup
			response.AdMarkup = winningBid.AdM

			// Extract impression trackers from VAST
			for _, ad := range vastDoc.Ads {
				if ad.InLine != nil {
					for _, imp := range ad.InLine.Impressions {
						if imp.Value != "" {
							response.ImpressionTrackers = append(response.ImpressionTrackers, imp.Value)
						}
					}

					// Extract click trackers from linear creative
					for _, creative := range ad.InLine.Creatives {
						if creative.Linear != nil && creative.Linear.VideoClicks != nil {
							if creative.Linear.VideoClicks.ClickThrough != nil && creative.Linear.VideoClicks.ClickThrough.Value != "" {
								response.ClickTrackers = append(response.ClickTrackers, creative.Linear.VideoClicks.ClickThrough.Value)
							}
							for _, ct := range creative.Linear.VideoClicks.ClickTracking {
								if ct.Value != "" {
									response.ClickTrackers = append(response.ClickTrackers, ct.Value)
								}
							}
						}
					}
				}

				if ad.Wrapper != nil {
					for _, imp := range ad.Wrapper.Impressions {
						if imp.Value != "" {
							response.ImpressionTrackers = append(response.ImpressionTrackers, imp.Value)
						}
					}
				}
			}
		}

		// Add NURL if present
		if winningBid.NURL != "" {
			response.ImpressionTrackers = append(response.ImpressionTrackers, winningBid.NURL)
		}

	default:
		return &PauseAdResponse{
			Format: pauseReq.Format,
			Error:  fmt.Sprintf("unsupported ad format: %s", pauseReq.Format),
		}, nil
	}

	return response, nil
}

// HandleResume handles the resume event to clean up pause ad state
func HandleResume(sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("session ID is required for resume event")
	}

	// In a real implementation, this would:
	// 1. Clear any cached pause ad state for this session
	// 2. Send any necessary cleanup signals to the ad server
	// 3. Fire any required tracking pixels for ad removal

	// For now, this is a placeholder that validates the session ID
	return nil
}

// IsValidPauseAdRequest validates a pause ad request
func IsValidPauseAdRequest(req *PauseAdRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if req.State == "" {
		return fmt.Errorf("state is required")
	}

	if req.State != StatePaused && req.State != StateResumed {
		return fmt.Errorf("invalid state: %s (must be 'paused' or 'resumed')", req.State)
	}

	if req.Format != "" && req.Format != FormatDisplay && req.Format != FormatVideo {
		return fmt.Errorf("invalid format: %s (must be 'display' or 'video')", req.Format)
	}

	if req.State == StatePaused && req.BidRequest == nil {
		return fmt.Errorf("bid request is required for paused state")
	}

	return nil
}
