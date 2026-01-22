// Package pauseads handles pause ad detection and serving for CTV video ads
package pauseads

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/thenexusengine/tne_springwire/internal/openrtb"
	"github.com/thenexusengine/tne_springwire/pkg/vast"
)

// PauseAdConfig configures pause ad behavior
type PauseAdConfig struct {
	// Enabled indicates if pause ads are enabled
	Enabled bool `json:"enabled"`

	// MinPauseDuration is the minimum pause duration in seconds before showing an ad
	MinPauseDuration int `json:"min_pause_duration"`

	// MaxDisplayDuration is the maximum time a pause ad can be displayed in seconds
	MaxDisplayDuration int `json:"max_display_duration"`

	// Formats specifies allowed ad formats for pause ads
	Formats []string `json:"formats"`

	// MaxWidth is the maximum width for pause ad creatives
	MaxWidth int `json:"max_width"`

	// MaxHeight is the maximum height for pause ad creatives
	MaxHeight int `json:"max_height"`

	// FrequencyCap limits how often pause ads are shown per session
	FrequencyCap *FrequencyCap `json:"frequency_cap,omitempty"`
}

// FrequencyCap defines frequency capping rules for pause ads
type FrequencyCap struct {
	// MaxImpressions is the maximum number of impressions per time window
	MaxImpressions int `json:"max_impressions"`

	// TimeWindowSeconds is the time window in seconds
	TimeWindowSeconds int `json:"time_window_seconds"`
}

// DefaultConfig returns the default pause ad configuration
func DefaultConfig() PauseAdConfig {
	return PauseAdConfig{
		Enabled:            true,
		MinPauseDuration:   3,
		MaxDisplayDuration: 60,
		Formats:            []string{"image/jpeg", "image/png", "image/gif"},
		MaxWidth:           1920,
		MaxHeight:          1080,
		FrequencyCap: &FrequencyCap{
			MaxImpressions:    5,
			TimeWindowSeconds: 3600, // 1 hour
		},
	}
}

// PauseAdRequest represents a request for a pause ad
type PauseAdRequest struct {
	// SessionID identifies the viewing session
	SessionID string `json:"session_id"`

	// ContentID identifies the content being watched
	ContentID string `json:"content_id"`

	// PausedAt is the timestamp when pause occurred
	PausedAt time.Time `json:"paused_at"`

	// PlaybackPosition is the position in seconds when paused
	PlaybackPosition float64 `json:"playback_position"`

	// Device information
	Device *openrtb.Device `json:"device,omitempty"`

	// User information
	User *openrtb.User `json:"user,omitempty"`

	// Site or App context
	Site *openrtb.Site `json:"site,omitempty"`
	App  *openrtb.App  `json:"app,omitempty"`

	// Publisher ID
	PublisherID string `json:"publisher_id"`

	// Additional parameters
	Ext json.RawMessage `json:"ext,omitempty"`
}

// PauseAdResponse represents a pause ad response
type PauseAdResponse struct {
	// Ad contains the pause ad creative
	Ad *PauseAd `json:"ad,omitempty"`

	// Error contains error information if no ad available
	Error string `json:"error,omitempty"`

	// NoBid indicates no bid was received
	NoBid bool `json:"no_bid,omitempty"`
}

// PauseAd represents a pause ad creative
type PauseAd struct {
	// ID is the ad identifier
	ID string `json:"id"`

	// CreativeURL is the URL of the creative asset
	CreativeURL string `json:"creative_url"`

	// ClickURL is the click-through URL
	ClickURL string `json:"click_url,omitempty"`

	// Width of the creative
	Width int `json:"width"`

	// Height of the creative
	Height int `json:"height"`

	// Format is the MIME type of the creative
	Format string `json:"format"`

	// DisplayDuration is how long to display the ad in seconds
	DisplayDuration int `json:"display_duration"`

	// TrackingURLs contains tracking pixels
	TrackingURLs *PauseAdTracking `json:"tracking_urls,omitempty"`

	// Price information
	Price    float64 `json:"price,omitempty"`
	Currency string  `json:"currency,omitempty"`

	// Advertiser info
	Advertiser string `json:"advertiser,omitempty"`
}

// PauseAdTracking contains tracking URLs for pause ads
type PauseAdTracking struct {
	Impression []string `json:"impression,omitempty"`
	Click      []string `json:"click,omitempty"`
	ViewStart  []string `json:"view_start,omitempty"`
	ViewEnd    []string `json:"view_end,omitempty"`
}

// PauseAdService handles pause ad requests
type PauseAdService struct {
	config      PauseAdConfig
	adRequester AdRequester
	tracker     *PauseAdTracker
}

// AdRequester is an interface for requesting ads
type AdRequester interface {
	RequestPauseAd(ctx context.Context, req *PauseAdRequest) (*PauseAdResponse, error)
}

// NewPauseAdService creates a new pause ad service
func NewPauseAdService(config PauseAdConfig, requester AdRequester) *PauseAdService {
	return &PauseAdService{
		config:      config,
		adRequester: requester,
		tracker:     NewPauseAdTracker(),
	}
}

// HandlePauseAdRequest processes a pause ad request
func (s *PauseAdService) HandlePauseAdRequest(ctx context.Context, req *PauseAdRequest) (*PauseAdResponse, error) {
	if !s.config.Enabled {
		return &PauseAdResponse{
			NoBid: true,
			Error: "pause ads disabled",
		}, nil
	}

	// Check frequency cap
	if s.config.FrequencyCap != nil {
		if !s.tracker.CanShowAd(req.SessionID, s.config.FrequencyCap) {
			return &PauseAdResponse{
				NoBid: true,
				Error: "frequency cap reached",
			}, nil
		}
	}

	// Request ad from ad requester
	resp, err := s.adRequester.RequestPauseAd(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to request pause ad: %w", err)
	}

	// Track impression if ad was returned
	if resp.Ad != nil {
		s.tracker.RecordImpression(req.SessionID)
	}

	return resp, nil
}

// PauseAdTracker tracks pause ad impressions for frequency capping
type PauseAdTracker struct {
	impressions map[string][]time.Time
}

// NewPauseAdTracker creates a new pause ad tracker
func NewPauseAdTracker() *PauseAdTracker {
	return &PauseAdTracker{
		impressions: make(map[string][]time.Time),
	}
}

// CanShowAd checks if an ad can be shown based on frequency cap
func (t *PauseAdTracker) CanShowAd(sessionID string, cap *FrequencyCap) bool {
	if cap == nil {
		return true
	}

	now := time.Now()
	cutoff := now.Add(-time.Duration(cap.TimeWindowSeconds) * time.Second)

	// Get impressions for this session
	impressions, ok := t.impressions[sessionID]
	if !ok {
		return true
	}

	// Count recent impressions
	count := 0
	for _, imp := range impressions {
		if imp.After(cutoff) {
			count++
		}
	}

	return count < cap.MaxImpressions
}

// RecordImpression records a pause ad impression
func (t *PauseAdTracker) RecordImpression(sessionID string) {
	t.impressions[sessionID] = append(t.impressions[sessionID], time.Now())

	// Clean up old impressions
	t.cleanupOldImpressions(sessionID)
}

// cleanupOldImpressions removes impressions older than 24 hours
func (t *PauseAdTracker) cleanupOldImpressions(sessionID string) {
	cutoff := time.Now().Add(-24 * time.Hour)
	impressions := t.impressions[sessionID]

	var cleaned []time.Time
	for _, imp := range impressions {
		if imp.After(cutoff) {
			cleaned = append(cleaned, imp)
		}
	}

	if len(cleaned) > 0 {
		t.impressions[sessionID] = cleaned
	} else {
		delete(t.impressions, sessionID)
	}
}

// PauseAdHandler is an HTTP handler for pause ad requests
type PauseAdHandler struct {
	service *PauseAdService
}

// NewPauseAdHandler creates a new pause ad HTTP handler
func NewPauseAdHandler(service *PauseAdService) *PauseAdHandler {
	return &PauseAdHandler{service: service}
}

// ServeHTTP handles pause ad HTTP requests
func (h *PauseAdHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PauseAdRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request: %s", err), http.StatusBadRequest)
		return
	}

	resp, err := h.service.HandlePauseAdRequest(r.Context(), &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("error processing request: %s", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("error encoding response: %s", err), http.StatusInternalServerError)
		return
	}
}

// CreatePauseAdVAST creates a VAST response for a pause ad scenario
func CreatePauseAdVAST(ad *PauseAd, trackingBaseURL string) (*vast.VAST, error) {
	if ad == nil {
		return vast.CreateEmptyVAST(), nil
	}

	// Create non-linear VAST for pause ad (overlay style)
	builder := vast.NewBuilder("4.0")

	v, err := builder.
		AddAd(ad.ID).
		WithInLine("TNEVideo", "Pause Ad").
		WithImpression(fmt.Sprintf("%s/pause/impression?ad_id=%s", trackingBaseURL, ad.ID)).
		Done().
		Build()

	if err != nil {
		return nil, err
	}

	// Add non-linear creative for pause ad
	if len(v.Ads) > 0 && v.Ads[0].InLine != nil {
		v.Ads[0].InLine.Creatives.Creative = append(v.Ads[0].InLine.Creatives.Creative, vast.Creative{
			ID: ad.ID + "-creative",
			NonLinearAds: &vast.NonLinearAds{
				NonLinear: []vast.NonLinear{
					{
						ID:     ad.ID + "-nonlinear",
						Width:  ad.Width,
						Height: ad.Height,
						StaticResource: &vast.StaticResource{
							CreativeType: ad.Format,
							Value:        ad.CreativeURL,
						},
						NonLinearClickThrough: ad.ClickURL,
					},
				},
			},
		})
	}

	return v, nil
}
