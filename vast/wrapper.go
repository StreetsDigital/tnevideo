package vast

import (
	"fmt"
)

// WrapperBuilder helps construct VAST wrapper ads
type WrapperBuilder struct {
	vast *VAST
}

// NewWrapperBuilder creates a new wrapper builder with the specified VAST version
func NewWrapperBuilder(version string) *WrapperBuilder {
	return &WrapperBuilder{
		vast: &VAST{
			Version: version,
			Ads:     []Ad{},
		},
	}
}

// AddWrapperAd adds a wrapper ad to the VAST document
func (wb *WrapperBuilder) AddWrapperAd(config WrapperConfig) *WrapperBuilder {
	wrapper := &Wrapper{
		AdSystem: AdSystem{
			Version: config.AdSystemVersion,
			Value:   config.AdSystem,
		},
		VASTAdTagURI: &VASTAdTagURI{
			Value: config.VASTAdTagURI,
		},
		AdTitle:     config.AdTitle,
		Impressions: []Impression{},
		Creatives:   []Creative{},
	}

	// Add impressions
	for _, impURL := range config.ImpressionURLs {
		wrapper.Impressions = append(wrapper.Impressions, Impression{
			Value: impURL,
		})
	}

	// Add error tracking
	if config.ErrorURL != "" {
		wrapper.Error = &CDATAString{Value: config.ErrorURL}
	}

	// Add tracking events if provided
	if len(config.TrackingEvents) > 0 {
		creative := Creative{
			Linear: &Linear{
				TrackingEvents: []Tracking{},
			},
		}

		for event, urls := range config.TrackingEvents {
			for _, url := range urls {
				creative.Linear.TrackingEvents = append(creative.Linear.TrackingEvents, Tracking{
					Event: event,
					Value: url,
				})
			}
		}

		wrapper.Creatives = append(wrapper.Creatives, creative)
	}

	ad := Ad{
		ID:      config.AdID,
		Wrapper: wrapper,
	}

	wb.vast.Ads = append(wb.vast.Ads, ad)
	return wb
}

// Build returns the constructed VAST document
func (wb *WrapperBuilder) Build() (*VAST, error) {
	if err := wb.vast.Validate(); err != nil {
		return nil, fmt.Errorf("invalid VAST wrapper: %w", err)
	}
	return wb.vast, nil
}

// BuildXML returns the VAST document as XML string
func (wb *WrapperBuilder) BuildXML() (string, error) {
	vast, err := wb.Build()
	if err != nil {
		return "", err
	}
	return vast.Marshal()
}

// WrapperConfig contains configuration for building a wrapper ad
type WrapperConfig struct {
	// Required fields
	AdSystem       string
	VASTAdTagURI   string
	ImpressionURLs []string

	// Optional fields
	AdID             string
	AdSystemVersion  string
	AdTitle          string
	ErrorURL         string
	TrackingEvents   map[string][]string // event name -> URLs
}

// NewDefaultWrapper creates a simple VAST 4.2 wrapper with basic fields
func NewDefaultWrapper(adSystem, vastTagURI string, impressionURLs []string) (*VAST, error) {
	if adSystem == "" {
		return nil, fmt.Errorf("adSystem is required")
	}
	if vastTagURI == "" {
		return nil, fmt.Errorf("vastTagURI is required")
	}
	if len(impressionURLs) == 0 {
		return nil, fmt.Errorf("at least one impression URL is required")
	}

	config := WrapperConfig{
		AdSystem:       adSystem,
		VASTAdTagURI:   vastTagURI,
		ImpressionURLs: impressionURLs,
	}

	return NewWrapperBuilder("4.2").AddWrapperAd(config).Build()
}

// NewDefaultWrapperXML creates a simple VAST 4.2 wrapper and returns it as XML
func NewDefaultWrapperXML(adSystem, vastTagURI string, impressionURLs []string) (string, error) {
	vast, err := NewDefaultWrapper(adSystem, vastTagURI, impressionURLs)
	if err != nil {
		return "", err
	}
	return vast.Marshal()
}
