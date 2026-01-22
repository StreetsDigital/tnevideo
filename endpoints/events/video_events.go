package events

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
	accountService "github.com/prebid/prebid-server/v3/account"
	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/privacy"
	"github.com/prebid/prebid-server/v3/stored_requests"
	"github.com/prebid/prebid-server/v3/util/httputil"
)

// VideoEventRequest represents the body of a video event POST request
type VideoEventRequest struct {
	BidID        string `json:"bidId"`
	AccountID    string `json:"accountId"`
	Bidder       string `json:"bidder,omitempty"`
	Timestamp    int64  `json:"timestamp,omitempty"`
	Integration  string `json:"integration,omitempty"`
	ErrorCode    string `json:"errorCode,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	ClickThrough string `json:"clickThrough,omitempty"`
	Format       string `json:"format,omitempty"`
	Analytics    string `json:"analytics,omitempty"`
}

type videoEventEndpoint struct {
	Accounts      stored_requests.AccountFetcher
	Analytics     analytics.Runner
	Cfg           *config.Configuration
	TrackingPixel *httputil.Pixel
	MetricsEngine metrics.MetricsEngine
	VType         analytics.VastType
}

// NewVideoStartEndpoint creates the POST /api/v1/video/start endpoint
func NewVideoStartEndpoint(cfg *config.Configuration, accounts stored_requests.AccountFetcher, analyticsRunner analytics.Runner, me metrics.MetricsEngine) httprouter.Handle {
	ee := &videoEventEndpoint{
		Accounts:      accounts,
		Analytics:     analyticsRunner,
		Cfg:           cfg,
		TrackingPixel: &httputil.Pixel1x1PNG,
		MetricsEngine: me,
		VType:         analytics.Start,
	}
	return ee.Handle
}

// NewVideoQuartileEndpoint creates the POST /api/v1/video/quartile endpoint
// This endpoint handles 25%, 50%, and 75% progress events
func NewVideoQuartileEndpoint(cfg *config.Configuration, accounts stored_requests.AccountFetcher, analyticsRunner analytics.Runner, me metrics.MetricsEngine) httprouter.Handle {
	ee := &videoEventEndpoint{
		Accounts:      accounts,
		Analytics:     analyticsRunner,
		Cfg:           cfg,
		TrackingPixel: &httputil.Pixel1x1PNG,
		MetricsEngine: me,
		VType:         analytics.FirstQuartile, // Default, will be determined from request
	}
	return ee.HandleQuartile
}

// NewVideoCompleteEndpoint creates the POST /api/v1/video/complete endpoint
func NewVideoCompleteEndpoint(cfg *config.Configuration, accounts stored_requests.AccountFetcher, analyticsRunner analytics.Runner, me metrics.MetricsEngine) httprouter.Handle {
	ee := &videoEventEndpoint{
		Accounts:      accounts,
		Analytics:     analyticsRunner,
		Cfg:           cfg,
		TrackingPixel: &httputil.Pixel1x1PNG,
		MetricsEngine: me,
		VType:         analytics.Complete,
	}
	return ee.Handle
}

// NewVideoClickEndpoint creates the POST /api/v1/video/click endpoint
func NewVideoClickEndpoint(cfg *config.Configuration, accounts stored_requests.AccountFetcher, analyticsRunner analytics.Runner, me metrics.MetricsEngine) httprouter.Handle {
	ee := &videoEventEndpoint{
		Accounts:      accounts,
		Analytics:     analyticsRunner,
		Cfg:           cfg,
		TrackingPixel: &httputil.Pixel1x1PNG,
		MetricsEngine: me,
		VType:         analytics.Click,
	}
	return ee.Handle
}

// NewVideoErrorEndpoint creates the POST /api/v1/video/error endpoint
func NewVideoErrorEndpoint(cfg *config.Configuration, accounts stored_requests.AccountFetcher, analyticsRunner analytics.Runner, me metrics.MetricsEngine) httprouter.Handle {
	ee := &videoEventEndpoint{
		Accounts:      accounts,
		Analytics:     analyticsRunner,
		Cfg:           cfg,
		TrackingPixel: &httputil.Pixel1x1PNG,
		MetricsEngine: me,
		VType:         analytics.Error,
	}
	return ee.Handle
}

// Handle processes video event POST requests
func (e *videoEventEndpoint) Handle(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Parse JSON request body
	var videoReq VideoEventRequest
	if err := json.NewDecoder(r.Body).Decode(&videoReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "invalid JSON request body: %s\n", err.Error())
		return
	}

	// Convert to analytics.EventRequest
	eventRequest, err := e.videoEventRequestToAnalyticsRequest(&videoReq)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "invalid request: %s\n", err.Error())
		return
	}

	// Process the event
	e.processEvent(w, r, eventRequest)
}

// HandleQuartile processes quartile events (25%, 50%, 75%)
func (e *videoEventEndpoint) HandleQuartile(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Parse JSON request body
	var videoReq VideoEventRequest
	if err := json.NewDecoder(r.Body).Decode(&videoReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "invalid JSON request body: %s\n", err.Error())
		return
	}

	// Determine quartile from query parameter
	quartile := r.URL.Query().Get("quartile")
	var vtype analytics.VastType

	switch quartile {
	case "25", "firstQuartile":
		vtype = analytics.FirstQuartile
	case "50", "midPoint":
		vtype = analytics.MidPoint
	case "75", "thirdQuartile":
		vtype = analytics.ThirdQuartile
	default:
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "invalid quartile parameter: must be 25, 50, 75, firstQuartile, midPoint, or thirdQuartile\n")
		return
	}

	// Override the VType with the specific quartile
	e.VType = vtype

	// Convert to analytics.EventRequest
	eventRequest, err := e.videoEventRequestToAnalyticsRequest(&videoReq)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "invalid request: %s\n", err.Error())
		return
	}

	// Process the event
	e.processEvent(w, r, eventRequest)
}

// processEvent handles the common event processing logic
func (e *videoEventEndpoint) processEvent(w http.ResponseWriter, r *http.Request, eventRequest *analytics.EventRequest) {
	// Validate account ID
	if eventRequest.AccountID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "accountId is required\n")
		return
	}

	// If analytics is disabled, return early
	if eventRequest.Analytics != analytics.Enabled {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	ctx := r.Context()

	// Get account details
	account, errs := accountService.GetAccount(ctx, e.Cfg, e.Accounts, eventRequest.AccountID, e.MetricsEngine)
	if len(errs) > 0 {
		status, messages := HandleAccountServiceErrors(errs)
		w.WriteHeader(status)
		for _, message := range messages {
			fmt.Fprintf(w, "invalid request: %s\n", message)
		}
		return
	}

	// Check if events are enabled for the account
	if !account.Events.Enabled {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "account '%s' doesn't support events\n", eventRequest.AccountID)
		return
	}

	activities := privacy.NewActivityControl(&account.Privacy)

	// Log the notification event
	e.Analytics.LogNotificationEventObject(&analytics.NotificationEvent{
		Request: eventRequest,
		Account: account,
	}, activities)

	// Return tracking pixel or blank response based on format
	if eventRequest.Format == analytics.Image {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", e.TrackingPixel.ContentType)
		w.Write(e.TrackingPixel.Content)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// videoEventRequestToAnalyticsRequest converts a VideoEventRequest to analytics.EventRequest
func (e *videoEventEndpoint) videoEventRequestToAnalyticsRequest(req *VideoEventRequest) (*analytics.EventRequest, error) {
	eventRequest := &analytics.EventRequest{
		Type:         analytics.Vast,
		VType:        e.VType,
		BidID:        req.BidID,
		AccountID:    req.AccountID,
		Timestamp:    req.Timestamp,
		Integration:  req.Integration,
		ErrorCode:    req.ErrorCode,
		ErrorMessage: req.ErrorMessage,
		ClickThrough: req.ClickThrough,
	}

	// Validate required fields
	if eventRequest.BidID == "" {
		return nil, fmt.Errorf("bidId is required")
	}

	if eventRequest.AccountID == "" {
		return nil, fmt.Errorf("accountId is required")
	}

	// Validate integration type
	if eventRequest.Integration != "" {
		if err := validateIntegrationType(eventRequest.Integration); err != nil {
			return nil, err
		}
	}

	// Normalize bidder name
	if req.Bidder != "" {
		if normalisedBidderName, ok := openrtb_ext.NormalizeBidderName(req.Bidder); ok {
			eventRequest.Bidder = normalisedBidderName.String()
		} else {
			eventRequest.Bidder = req.Bidder
		}
	}

	// Parse format
	switch req.Format {
	case "b", "blank", "":
		eventRequest.Format = analytics.Blank
	case "i", "image":
		eventRequest.Format = analytics.Image
	default:
		return nil, fmt.Errorf("unknown format: '%s'", req.Format)
	}

	// Parse analytics setting
	switch req.Analytics {
	case "1", "enabled", "":
		eventRequest.Analytics = analytics.Enabled
	case "0", "disabled":
		eventRequest.Analytics = analytics.Disabled
	default:
		return nil, fmt.Errorf("unknown analytics value: '%s'", req.Analytics)
	}

	return eventRequest, nil
}
