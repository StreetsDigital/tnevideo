package vast

import (
	"fmt"
)

// AddImpressionTracking adds impression tracking URLs to the VAST document
func (v *VAST) AddImpressionTracking(trackingURLs []string) error {
	if len(trackingURLs) == 0 {
		return nil
	}

	for i := range v.Ads {
		if v.Ads[i].Wrapper != nil {
			for _, url := range trackingURLs {
				v.Ads[i].Wrapper.Impressions = append(v.Ads[i].Wrapper.Impressions, Impression{
					Value: url,
				})
			}
		} else if v.Ads[i].InLine != nil {
			for _, url := range trackingURLs {
				v.Ads[i].InLine.Impressions = append(v.Ads[i].InLine.Impressions, Impression{
					Value: url,
				})
			}
		}
	}

	return nil
}

// AddVideoEventTracking adds video event tracking URLs to linear creatives
func (v *VAST) AddVideoEventTracking(event string, trackingURLs []string) error {
	if event == "" {
		return fmt.Errorf("event name is required")
	}

	if len(trackingURLs) == 0 {
		return nil
	}

	validEvents := map[string]bool{
		"start":          true,
		"firstQuartile":  true,
		"midpoint":       true,
		"thirdQuartile":  true,
		"complete":       true,
		"mute":           true,
		"unmute":         true,
		"pause":          true,
		"rewind":         true,
		"resume":         true,
		"fullscreen":     true,
		"exitFullscreen": true,
		"expand":         true,
		"collapse":       true,
		"acceptInvitation": true,
		"close":          true,
		"skip":           true,
		"progress":       true,
	}

	if !validEvents[event] {
		return fmt.Errorf("invalid tracking event: %s", event)
	}

	for i := range v.Ads {
		if v.Ads[i].Wrapper != nil {
			v.addTrackingToCreatives(v.Ads[i].Wrapper.Creatives, event, trackingURLs)
		} else if v.Ads[i].InLine != nil {
			v.addTrackingToCreatives(v.Ads[i].InLine.Creatives, event, trackingURLs)
		}
	}

	return nil
}

// addTrackingToCreatives adds tracking events to creative array
func (v *VAST) addTrackingToCreatives(creatives []Creative, event string, trackingURLs []string) {
	for i := range creatives {
		if creatives[i].Linear != nil {
			for _, url := range trackingURLs {
				creatives[i].Linear.TrackingEvents = append(
					creatives[i].Linear.TrackingEvents,
					Tracking{
						Event: event,
						Value: url,
					},
				)
			}
		}
	}
}

// AddErrorTracking adds error tracking URLs to the VAST document
func (v *VAST) AddErrorTracking(errorURL string) error {
	if errorURL == "" {
		return nil
	}

	for i := range v.Ads {
		if v.Ads[i].Wrapper != nil {
			v.Ads[i].Wrapper.Error = &CDATAString{Value: errorURL}
		} else if v.Ads[i].InLine != nil {
			v.Ads[i].InLine.Error = &CDATAString{Value: errorURL}
		}
	}

	return nil
}

// AddClickTracking adds click tracking URLs to linear creatives
func (v *VAST) AddClickTracking(trackingURLs []string) error {
	if len(trackingURLs) == 0 {
		return nil
	}

	for i := range v.Ads {
		var creatives []Creative

		if v.Ads[i].Wrapper != nil {
			creatives = v.Ads[i].Wrapper.Creatives
		} else if v.Ads[i].InLine != nil {
			creatives = v.Ads[i].InLine.Creatives
		}

		for j := range creatives {
			if creatives[j].Linear != nil {
				if creatives[j].Linear.VideoClicks == nil {
					creatives[j].Linear.VideoClicks = &VideoClicks{}
				}

				for _, url := range trackingURLs {
					creatives[j].Linear.VideoClicks.ClickTracking = append(
						creatives[j].Linear.VideoClicks.ClickTracking,
						ClickTracking{Value: url},
					)
				}
			}
		}
	}

	return nil
}

// GetImpressionURLs extracts all impression URLs from the VAST document
func (v *VAST) GetImpressionURLs() []string {
	var urls []string

	for _, ad := range v.Ads {
		if ad.Wrapper != nil {
			for _, imp := range ad.Wrapper.Impressions {
				if imp.Value != "" {
					urls = append(urls, imp.Value)
				}
			}
		} else if ad.InLine != nil {
			for _, imp := range ad.InLine.Impressions {
				if imp.Value != "" {
					urls = append(urls, imp.Value)
				}
			}
		}
	}

	return urls
}

// GetTrackingEvents extracts all tracking events by event type
func (v *VAST) GetTrackingEvents(event string) []string {
	var urls []string

	for _, ad := range v.Ads {
		var creatives []Creative

		if ad.Wrapper != nil {
			creatives = ad.Wrapper.Creatives
		} else if ad.InLine != nil {
			creatives = ad.InLine.Creatives
		}

		for _, creative := range creatives {
			if creative.Linear != nil {
				for _, tracking := range creative.Linear.TrackingEvents {
					if tracking.Event == event && tracking.Value != "" {
						urls = append(urls, tracking.Value)
					}
				}
			}
		}
	}

	return urls
}

// TrackingInjector provides methods for injecting tracking URLs into VAST XML
type TrackingInjector struct {
	vast *VAST
}

// NewTrackingInjector creates a new tracking injector from parsed VAST
func NewTrackingInjector(vast *VAST) *TrackingInjector {
	return &TrackingInjector{vast: vast}
}

// NewTrackingInjectorFromXML creates a new tracking injector from VAST XML string
func NewTrackingInjectorFromXML(xmlData string) (*TrackingInjector, error) {
	vast, err := Parse(xmlData)
	if err != nil {
		return nil, err
	}
	return &TrackingInjector{vast: vast}, nil
}

// InjectImpressions adds impression tracking URLs
func (ti *TrackingInjector) InjectImpressions(urls []string) *TrackingInjector {
	ti.vast.AddImpressionTracking(urls)
	return ti
}

// InjectVideoEvent adds video event tracking URLs
func (ti *TrackingInjector) InjectVideoEvent(event string, urls []string) *TrackingInjector {
	ti.vast.AddVideoEventTracking(event, urls)
	return ti
}

// InjectError adds error tracking URL
func (ti *TrackingInjector) InjectError(url string) *TrackingInjector {
	ti.vast.AddErrorTracking(url)
	return ti
}

// InjectClickTracking adds click tracking URLs
func (ti *TrackingInjector) InjectClickTracking(urls []string) *TrackingInjector {
	ti.vast.AddClickTracking(urls)
	return ti
}

// ToXML returns the modified VAST as XML string
func (ti *TrackingInjector) ToXML() (string, error) {
	return ti.vast.Marshal()
}

// GetVAST returns the underlying VAST structure
func (ti *TrackingInjector) GetVAST() *VAST {
	return ti.vast
}
