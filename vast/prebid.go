package vast

import (
	"fmt"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
)

// PrebidVASTConfig contains configuration for generating VAST from Prebid bids
type PrebidVASTConfig struct {
	// Bid information
	Bid *openrtb2.Bid

	// Tracking URLs
	ImpressionTrackers []string
	EventTrackers      map[string][]string // event type -> URLs
	ClickTrackers      []string
	ErrorTracker       string

	// Server information
	ServerName    string // Default: "Prebid Server"
	ServerVersion string // Default: "3.0"

	// VAST configuration
	VASTVersion string // Default: "4.2"
}

// MakeVASTFromBid generates a VAST document from an OpenRTB bid
// If bid.AdM contains VAST XML, it parses and optionally injects tracking
// If bid.AdM is empty or non-VAST, it generates a wrapper pointing to bid.NURL
func MakeVASTFromBid(config PrebidVASTConfig) (string, error) {
	// Set defaults
	if config.ServerName == "" {
		config.ServerName = "Prebid Server"
	}
	if config.ServerVersion == "" {
		config.ServerVersion = "3.0"
	}
	if config.VASTVersion == "" {
		config.VASTVersion = "4.2"
	}

	if config.Bid == nil {
		return "", fmt.Errorf("bid is required")
	}

	// Case 1: Bid contains VAST XML in AdM field
	if config.Bid.AdM != "" && isVASTXML(config.Bid.AdM) {
		return injectTrackingIntoVAST(config.Bid.AdM, config)
	}

	// Case 2: Generate wrapper from NURL
	if config.Bid.NURL != "" {
		return generateWrapperFromNURL(config)
	}

	// Case 3: Neither AdM nor NURL available
	return "", fmt.Errorf("bid must contain either AdM with VAST XML or NURL for wrapper generation")
}

// MakeVASTWrapper creates a VAST wrapper for a bid
// This is useful when you want to force wrapper generation even if AdM contains VAST
func MakeVASTWrapper(bid *openrtb2.Bid, vastTagURI string, impressionURLs []string) (string, error) {
	if bid == nil {
		return "", fmt.Errorf("bid is required")
	}

	if vastTagURI == "" && bid.NURL == "" {
		return "", fmt.Errorf("either vastTagURI or bid.NURL must be provided")
	}

	tagURI := vastTagURI
	if tagURI == "" {
		tagURI = bid.NURL
	}

	return NewDefaultWrapperXML("Prebid Server", tagURI, impressionURLs)
}

// InjectPrebidTracking injects Prebid tracking URLs into existing VAST XML
func InjectPrebidTracking(vastXML string, config PrebidVASTConfig) (string, error) {
	injector, err := NewTrackingInjectorFromXML(vastXML)
	if err != nil {
		return "", fmt.Errorf("failed to parse VAST for tracking injection: %w", err)
	}

	// Inject impressions
	if len(config.ImpressionTrackers) > 0 {
		injector.InjectImpressions(config.ImpressionTrackers)
	}

	// Inject video events
	if len(config.EventTrackers) > 0 {
		for event, urls := range config.EventTrackers {
			injector.InjectVideoEvent(event, urls)
		}
	}

	// Inject click tracking
	if len(config.ClickTrackers) > 0 {
		injector.InjectClickTracking(config.ClickTrackers)
	}

	// Inject error tracking
	if config.ErrorTracker != "" {
		injector.InjectError(config.ErrorTracker)
	}

	return injector.ToXML()
}

// Helper functions

func isVASTXML(content string) bool {
	// Simple check: does it contain <VAST tag
	return len(content) > 10 && containsVASTTag(content)
}

func containsVASTTag(content string) bool {
	// Check for VAST tag in various forms
	vast := []string{"<VAST", "<vast", "<?xml"}
	for _, tag := range vast {
		for i := 0; i < len(content)-len(tag); i++ {
			if content[i:i+len(tag)] == tag {
				return true
			}
		}
	}
	return false
}

func injectTrackingIntoVAST(vastXML string, config PrebidVASTConfig) (string, error) {
	// If no tracking to inject, return original
	if len(config.ImpressionTrackers) == 0 &&
		len(config.EventTrackers) == 0 &&
		len(config.ClickTrackers) == 0 &&
		config.ErrorTracker == "" {
		return vastXML, nil
	}

	return InjectPrebidTracking(vastXML, config)
}

func generateWrapperFromNURL(config PrebidVASTConfig) (string, error) {
	wrapperConfig := WrapperConfig{
		AdID:            config.Bid.ID,
		AdSystem:        config.ServerName,
		AdSystemVersion: config.ServerVersion,
		VASTAdTagURI:    config.Bid.NURL,
		ImpressionURLs:  config.ImpressionTrackers,
		ErrorURL:        config.ErrorTracker,
		TrackingEvents:  config.EventTrackers,
	}

	// Add bid price to AdTitle if available
	if config.Bid.Price > 0 {
		wrapperConfig.AdTitle = fmt.Sprintf("Prebid Bid %s ($%.2f)", config.Bid.ID, config.Bid.Price)
	}

	return NewWrapperBuilder(config.VASTVersion).AddWrapperAd(wrapperConfig).BuildXML()
}

// ExtractVideoMetadata extracts video metadata from VAST XML
type VideoMetadata struct {
	Duration       string
	Width          int
	Height         int
	MediaFileURL   string
	MediaType      string
	ImpressionURLs []string
	ClickThrough   string
}

// GetVideoMetadata extracts video metadata from VAST document
func GetVideoMetadata(vastXML string) (*VideoMetadata, error) {
	vast, err := Parse(vastXML)
	if err != nil {
		return nil, err
	}

	metadata := &VideoMetadata{
		ImpressionURLs: vast.GetImpressionURLs(),
	}

	// Extract from first linear creative
	for _, ad := range vast.Ads {
		var creatives []Creative
		if ad.InLine != nil {
			creatives = ad.InLine.Creatives
		} else if ad.Wrapper != nil {
			creatives = ad.Wrapper.Creatives
		}

		for _, creative := range creatives {
			if creative.Linear != nil {
				metadata.Duration = creative.Linear.Duration

				// Get first media file
				if len(creative.Linear.MediaFiles) > 0 {
					mf := creative.Linear.MediaFiles[0]
					metadata.MediaFileURL = strings.TrimSpace(mf.Value)
					metadata.MediaType = mf.Type
					metadata.Width = mf.Width
					metadata.Height = mf.Height
				}

				// Get click through
				if creative.Linear.VideoClicks != nil && creative.Linear.VideoClicks.ClickThrough != nil {
					metadata.ClickThrough = creative.Linear.VideoClicks.ClickThrough.Value
				}

				return metadata, nil
			}
		}
	}

	return metadata, nil
}

// DurationToSeconds converts VAST duration format (HH:MM:SS or HH:MM:SS.mmm) to seconds
func DurationToSeconds(duration string) (int, error) {
	if duration == "" {
		return 0, fmt.Errorf("duration is empty")
	}

	var hours, minutes, seconds int
	var milliseconds float64

	// Try HH:MM:SS.mmm format
	n, err := fmt.Sscanf(duration, "%d:%d:%f", &hours, &minutes, &milliseconds)
	if n == 3 && err == nil {
		seconds = int(milliseconds)
		return hours*3600 + minutes*60 + seconds, nil
	}

	// Try HH:MM:SS format
	n, err = fmt.Sscanf(duration, "%d:%d:%d", &hours, &minutes, &seconds)
	if n == 3 && err == nil {
		return hours*3600 + minutes*60 + seconds, nil
	}

	return 0, fmt.Errorf("invalid duration format: %s (expected HH:MM:SS or HH:MM:SS.mmm)", duration)
}

// SecondsToDuration converts seconds to VAST duration format (HH:MM:SS)
func SecondsToDuration(totalSeconds int) string {
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}
