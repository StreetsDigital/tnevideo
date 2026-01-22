package vast

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// EventType represents a VAST tracking event type
type EventType string

const (
	// Standard VAST events
	EventTypeStart          EventType = "start"
	EventTypeFirstQuartile  EventType = "firstQuartile"
	EventTypeMidpoint       EventType = "midpoint"
	EventTypeThirdQuartile  EventType = "thirdQuartile"
	EventTypeComplete       EventType = "complete"
	EventTypePause          EventType = "pause"
	EventTypeResume         EventType = "resume"
	EventTypeMute           EventType = "mute"
	EventTypeUnmute         EventType = "unmute"
	EventTypeSkip           EventType = "skip"
	EventTypeClick          EventType = "click"
	EventTypeError          EventType = "error"
	EventTypeProgress       EventType = "progress"
	EventTypeFullscreen     EventType = "fullscreen"
	EventTypeExitFullscreen EventType = "exitFullscreen"
	EventTypeCreativeView   EventType = "creativeView"
)

// TrackingEvent represents a video tracking event
type TrackingEvent struct {
	Type         EventType
	BidID        string
	AccountID    string
	Bidder       string
	Timestamp    time.Time
	Progress     float64 // For progress events (0-100)
	ErrorCode    string  // For error events
	ErrorMessage string  // For error events
	Extra        map[string]string
}

// Tracker handles VAST event tracking
type Tracker struct {
	client      *http.Client
	baseURL     string
	accountID   string
	bidID       string
	bidder      string
	extraParams map[string]string
}

// TrackerOption configures a Tracker
type TrackerOption func(*Tracker)

// NewTracker creates a new VAST event tracker
func NewTracker(baseURL string, opts ...TrackerOption) *Tracker {
	t := &Tracker{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		baseURL:     strings.TrimSuffix(baseURL, "/"),
		extraParams: make(map[string]string),
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// WithAccountID sets the account ID for tracking
func WithAccountID(accountID string) TrackerOption {
	return func(t *Tracker) {
		t.accountID = accountID
	}
}

// WithBidID sets the bid ID for tracking
func WithBidID(bidID string) TrackerOption {
	return func(t *Tracker) {
		t.bidID = bidID
	}
}

// WithBidder sets the bidder name for tracking
func WithBidder(bidder string) TrackerOption {
	return func(t *Tracker) {
		t.bidder = bidder
	}
}

// WithExtraParam adds an extra parameter for tracking
func WithExtraParam(key, value string) TrackerOption {
	return func(t *Tracker) {
		t.extraParams[key] = value
	}
}

// WithHTTPClient sets a custom HTTP client for tracking
func WithHTTPClient(client *http.Client) TrackerOption {
	return func(t *Tracker) {
		t.client = client
	}
}

// Track sends a tracking event
func (t *Tracker) Track(ctx context.Context, event EventType) error {
	return t.TrackWithProgress(ctx, event, 0)
}

// TrackWithProgress sends a tracking event with progress information
func (t *Tracker) TrackWithProgress(ctx context.Context, event EventType, progress float64) error {
	trackingURL := t.BuildTrackingURL(event, progress)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, trackingURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create tracking request: %w", err)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send tracking request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("tracking request failed with status: %d", resp.StatusCode)
	}

	return nil
}

// TrackError sends an error tracking event
func (t *Tracker) TrackError(ctx context.Context, errorCode, errorMessage string) error {
	params := url.Values{}
	params.Set("event", string(EventTypeError))
	params.Set("error_code", errorCode)
	params.Set("error_message", errorMessage)

	if t.accountID != "" {
		params.Set("account_id", t.accountID)
	}
	if t.bidID != "" {
		params.Set("bid_id", t.bidID)
	}
	if t.bidder != "" {
		params.Set("bidder", t.bidder)
	}
	params.Set("timestamp", fmt.Sprintf("%d", time.Now().UnixMilli()))

	for k, v := range t.extraParams {
		params.Set(k, v)
	}

	trackingURL := fmt.Sprintf("%s/video/error?%s", t.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, trackingURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create error tracking request: %w", err)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send error tracking request: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

// BuildTrackingURL constructs a tracking URL for an event
func (t *Tracker) BuildTrackingURL(event EventType, progress float64) string {
	params := url.Values{}
	params.Set("event", string(event))

	if t.accountID != "" {
		params.Set("account_id", t.accountID)
	}
	if t.bidID != "" {
		params.Set("bid_id", t.bidID)
	}
	if t.bidder != "" {
		params.Set("bidder", t.bidder)
	}
	params.Set("timestamp", fmt.Sprintf("%d", time.Now().UnixMilli()))

	if progress > 0 {
		params.Set("progress", fmt.Sprintf("%.2f", progress))
	}

	for k, v := range t.extraParams {
		params.Set(k, v)
	}

	return fmt.Sprintf("%s/video/event?%s", t.baseURL, params.Encode())
}

// GenerateTrackingURLs generates all tracking URLs for a VAST response
func GenerateTrackingURLs(baseURL, bidID, accountID, bidder string) map[EventType]string {
	events := []EventType{
		EventTypeStart,
		EventTypeFirstQuartile,
		EventTypeMidpoint,
		EventTypeThirdQuartile,
		EventTypeComplete,
		EventTypePause,
		EventTypeResume,
		EventTypeMute,
		EventTypeUnmute,
		EventTypeSkip,
		EventTypeClick,
	}

	urls := make(map[EventType]string)
	for _, event := range events {
		params := url.Values{}
		params.Set("event", string(event))
		params.Set("bid_id", bidID)
		params.Set("account_id", accountID)
		params.Set("bidder", bidder)
		params.Set("t", "[TIMESTAMP]") // Placeholder for client-side timestamp

		urls[event] = fmt.Sprintf("%s/video/event?%s", strings.TrimSuffix(baseURL, "/"), params.Encode())
	}

	return urls
}

// InjectTracking injects tracking URLs into a VAST document
func InjectTracking(v *VAST, trackingURLs map[EventType]string) {
	for _, ad := range v.Ads {
		var creatives *Creatives
		if ad.InLine != nil {
			creatives = &ad.InLine.Creatives
		} else if ad.Wrapper != nil {
			creatives = &ad.Wrapper.Creatives
		}

		if creatives != nil {
			for i := range creatives.Creative {
				if creatives.Creative[i].Linear != nil {
					linear := creatives.Creative[i].Linear
					for event, trackingURL := range trackingURLs {
						linear.TrackingEvents.Tracking = append(linear.TrackingEvents.Tracking, Tracking{
							Event: string(event),
							Value: trackingURL,
						})
					}
				}
			}
		}
	}
}
