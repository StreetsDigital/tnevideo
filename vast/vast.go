package vast

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// VAST represents the root VAST document supporting versions 2.0, 3.0, 4.0, 4.1, and 4.2
type VAST struct {
	XMLName xml.Name `xml:"VAST"`
	Version string   `xml:"version,attr"`
	Ads     []Ad     `xml:"Ad"`
}

// Ad represents a single ad in the VAST response
type Ad struct {
	ID       string    `xml:"id,attr,omitempty"`
	Sequence int       `xml:"sequence,attr,omitempty"`
	Wrapper  *Wrapper  `xml:"Wrapper,omitempty"`
	InLine   *InLine   `xml:"InLine,omitempty"`
}

// Wrapper represents a VAST wrapper ad that redirects to another VAST document
type Wrapper struct {
	AdSystem       AdSystem        `xml:"AdSystem"`
	VASTAdTagURI   *VASTAdTagURI   `xml:"VASTAdTagURI,omitempty"`
	AdTitle        string          `xml:"AdTitle,omitempty"`
	Impressions    []Impression    `xml:"Impression"`
	Error          *CDATAString    `xml:"Error,omitempty"`
	Creatives      []Creative      `xml:"Creatives>Creative,omitempty"`
	Extensions     *Extensions     `xml:"Extensions,omitempty"`
}

// InLine represents a direct VAST ad with creative assets
type InLine struct {
	AdSystem       AdSystem        `xml:"AdSystem"`
	AdTitle        string          `xml:"AdTitle"`
	Impressions    []Impression    `xml:"Impression"`
	Description    string          `xml:"Description,omitempty"`
	Advertiser     string          `xml:"Advertiser,omitempty"`
	Pricing        *Pricing        `xml:"Pricing,omitempty"`
	Survey         *CDATAString    `xml:"Survey,omitempty"`
	Error          *CDATAString    `xml:"Error,omitempty"`
	Creatives      []Creative      `xml:"Creatives>Creative"`
	Extensions     *Extensions     `xml:"Extensions,omitempty"`
}

// AdSystem identifies the ad server
type AdSystem struct {
	Version string `xml:"version,attr,omitempty"`
	Value   string `xml:",chardata"`
}

// VASTAdTagURI contains the URI to the next VAST document in a wrapper chain
type VASTAdTagURI struct {
	Value string `xml:",cdata"`
}

// Impression represents an impression tracking URL
type Impression struct {
	ID    string `xml:"id,attr,omitempty"`
	Value string `xml:",cdata"`
}

// Creative represents a creative asset within an ad
type Creative struct {
	ID              string           `xml:"id,attr,omitempty"`
	AdID            string           `xml:"adId,attr,omitempty"`
	Sequence        int              `xml:"sequence,attr,omitempty"`
	Linear          *Linear          `xml:"Linear,omitempty"`
	NonLinearAds    *NonLinearAds    `xml:"NonLinearAds,omitempty"`
	CompanionAds    *CompanionAds    `xml:"CompanionAds,omitempty"`
	UniversalAdId   *UniversalAdId   `xml:"UniversalAdId,omitempty"`
}

// Linear represents a linear video creative
type Linear struct {
	SkipOffset      string          `xml:"skipoffset,attr,omitempty"`
	Duration        string          `xml:"Duration"`
	MediaFiles      []MediaFile     `xml:"MediaFiles>MediaFile"`
	VideoClicks     *VideoClicks    `xml:"VideoClicks,omitempty"`
	TrackingEvents  []Tracking      `xml:"TrackingEvents>Tracking,omitempty"`
	AdParameters    *AdParameters   `xml:"AdParameters,omitempty"`
	Icons           []Icon          `xml:"Icons>Icon,omitempty"`
}

// MediaFile represents a video file for the creative
type MediaFile struct {
	ID                 string `xml:"id,attr,omitempty"`
	Delivery           string `xml:"delivery,attr"`
	Type               string `xml:"type,attr"`
	Width              int    `xml:"width,attr,omitempty"`
	Height             int    `xml:"height,attr,omitempty"`
	Codec              string `xml:"codec,attr,omitempty"`
	Bitrate            int    `xml:"bitrate,attr,omitempty"`
	MinBitrate         int    `xml:"minBitrate,attr,omitempty"`
	MaxBitrate         int    `xml:"maxBitrate,attr,omitempty"`
	Scalable           bool   `xml:"scalable,attr,omitempty"`
	MaintainAspectRatio bool   `xml:"maintainAspectRatio,attr,omitempty"`
	APIFramework       string `xml:"apiFramework,attr,omitempty"`
	Value              string `xml:",cdata"`
}

// VideoClicks represents click tracking and clickthrough URLs
type VideoClicks struct {
	ClickThrough    *CDATAString   `xml:"ClickThrough,omitempty"`
	ClickTracking   []ClickTracking `xml:"ClickTracking,omitempty"`
	CustomClick     []CDATAString   `xml:"CustomClick,omitempty"`
}

// ClickTracking represents a click tracking URL
type ClickTracking struct {
	ID    string `xml:"id,attr,omitempty"`
	Value string `xml:",cdata"`
}

// Tracking represents a tracking event
type Tracking struct {
	Event  string `xml:"event,attr"`
	Offset string `xml:"offset,attr,omitempty"`
	Value  string `xml:",cdata"`
}

// AdParameters contains data to be passed to the creative
type AdParameters struct {
	XMLEncoded bool   `xml:"xmlEncoded,attr,omitempty"`
	Value      string `xml:",cdata"`
}

// Icon represents an industry initiative icon (VAST 3.0+)
type Icon struct {
	Program        string          `xml:"program,attr"`
	Width          int             `xml:"width,attr"`
	Height         int             `xml:"height,attr"`
	XPosition      string          `xml:"xPosition,attr"`
	YPosition      string          `xml:"yPosition,attr"`
	Duration       string          `xml:"duration,attr,omitempty"`
	Offset         string          `xml:"offset,attr,omitempty"`
	APIFramework   string          `xml:"apiFramework,attr,omitempty"`
	StaticResource *StaticResource `xml:"StaticResource,omitempty"`
	IFrameResource *CDATAString    `xml:"IFrameResource,omitempty"`
	HTMLResource   *CDATAString    `xml:"HTMLResource,omitempty"`
	IconClicks     *IconClicks     `xml:"IconClicks,omitempty"`
	IconViewTracking []CDATAString `xml:"IconViewTracking,omitempty"`
}

// StaticResource represents a static resource like an image
type StaticResource struct {
	CreativeType string `xml:"creativeType,attr"`
	Value        string `xml:",cdata"`
}

// IconClicks represents click tracking for icons
type IconClicks struct {
	IconClickThrough  *CDATAString   `xml:"IconClickThrough,omitempty"`
	IconClickTracking []CDATAString  `xml:"IconClickTracking,omitempty"`
}

// NonLinearAds represents non-linear creatives (overlays)
type NonLinearAds struct {
	TrackingEvents []Tracking   `xml:"TrackingEvents>Tracking,omitempty"`
	NonLinear      []NonLinear  `xml:"NonLinear"`
}

// NonLinear represents a single non-linear creative
type NonLinear struct {
	ID                  string          `xml:"id,attr,omitempty"`
	Width               int             `xml:"width,attr"`
	Height              int             `xml:"height,attr"`
	ExpandedWidth       int             `xml:"expandedWidth,attr,omitempty"`
	ExpandedHeight      int             `xml:"expandedHeight,attr,omitempty"`
	Scalable            bool            `xml:"scalable,attr,omitempty"`
	MaintainAspectRatio bool            `xml:"maintainAspectRatio,attr,omitempty"`
	MinSuggestedDuration string          `xml:"minSuggestedDuration,attr,omitempty"`
	APIFramework        string          `xml:"apiFramework,attr,omitempty"`
	StaticResource      *StaticResource `xml:"StaticResource,omitempty"`
	IFrameResource      *CDATAString    `xml:"IFrameResource,omitempty"`
	HTMLResource        *CDATAString    `xml:"HTMLResource,omitempty"`
	NonLinearClickThrough *CDATAString  `xml:"NonLinearClickThrough,omitempty"`
	NonLinearClickTracking []CDATAString `xml:"NonLinearClickTracking,omitempty"`
	AdParameters        *AdParameters   `xml:"AdParameters,omitempty"`
}

// CompanionAds represents companion banner ads
type CompanionAds struct {
	Required   string      `xml:"required,attr,omitempty"`
	Companion  []Companion `xml:"Companion"`
}

// Companion represents a single companion ad
type Companion struct {
	ID                  string          `xml:"id,attr,omitempty"`
	Width               int             `xml:"width,attr"`
	Height              int             `xml:"height,attr"`
	ASRatio             string          `xml:"assetWidth,attr,omitempty"`
	ExpandedWidth       int             `xml:"expandedWidth,attr,omitempty"`
	ExpandedHeight      int             `xml:"expandedHeight,attr,omitempty"`
	APIFramework        string          `xml:"apiFramework,attr,omitempty"`
	AdSlotID            string          `xml:"adSlotId,attr,omitempty"`
	StaticResource      *StaticResource `xml:"StaticResource,omitempty"`
	IFrameResource      *CDATAString    `xml:"IFrameResource,omitempty"`
	HTMLResource        *CDATAString    `xml:"HTMLResource,omitempty"`
	TrackingEvents      []Tracking      `xml:"TrackingEvents>Tracking,omitempty"`
	CompanionClickThrough *CDATAString  `xml:"CompanionClickThrough,omitempty"`
	CompanionClickTracking []CDATAString `xml:"CompanionClickTracking,omitempty"`
	AltText             string          `xml:"AltText,omitempty"`
	AdParameters        *AdParameters   `xml:"AdParameters,omitempty"`
}

// UniversalAdId represents a universal ad identifier (VAST 3.0+)
type UniversalAdId struct {
	IDRegistry string `xml:"idRegistry,attr"`
	IDValue    string `xml:"idValue,attr"`
	Value      string `xml:",chardata"`
}

// Pricing represents pricing information (VAST 3.0+)
type Pricing struct {
	Model    string  `xml:"model,attr"`
	Currency string  `xml:"currency,attr"`
	Value    float64 `xml:",chardata"`
}

// Extensions represents custom extensions
type Extensions struct {
	Extensions []Extension `xml:"Extension"`
}

// Extension represents a single extension
type Extension struct {
	Type  string `xml:"type,attr,omitempty"`
	Value string `xml:",innerxml"`
}

// CDATAString represents a CDATA-wrapped string value
type CDATAString struct {
	Value string `xml:",cdata"`
}

// Parse parses a VAST XML string into a VAST struct
func Parse(xmlData string) (*VAST, error) {
	vast := &VAST{}

	// Trim whitespace
	xmlData = strings.TrimSpace(xmlData)

	if err := xml.Unmarshal([]byte(xmlData), vast); err != nil {
		return nil, fmt.Errorf("failed to parse VAST XML: %w", err)
	}

	// Validate version
	if vast.Version == "" {
		return nil, fmt.Errorf("VAST version attribute is required")
	}

	// Support versions 2.0, 3.0, 4.0, 4.1, 4.2
	supportedVersions := map[string]bool{
		"2.0": true, "3.0": true, "4.0": true, "4.1": true, "4.2": true,
	}

	if !supportedVersions[vast.Version] {
		return nil, fmt.Errorf("unsupported VAST version: %s (supported: 2.0, 3.0, 4.0, 4.1, 4.2)", vast.Version)
	}

	return vast, nil
}

// Marshal converts a VAST struct to XML string
func (v *VAST) Marshal() (string, error) {
	output, err := xml.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal VAST to XML: %w", err)
	}

	return xml.Header + string(output), nil
}

// Validate performs basic validation on the VAST document
func (v *VAST) Validate() error {
	if v.Version == "" {
		return fmt.Errorf("VAST version is required")
	}

	if len(v.Ads) == 0 {
		return fmt.Errorf("VAST must contain at least one Ad")
	}

	for i, ad := range v.Ads {
		if ad.Wrapper == nil && ad.InLine == nil {
			return fmt.Errorf("Ad at index %d must have either Wrapper or InLine", i)
		}

		if ad.Wrapper != nil && ad.InLine != nil {
			return fmt.Errorf("Ad at index %d cannot have both Wrapper and InLine", i)
		}

		// Validate wrapper
		if ad.Wrapper != nil {
			if len(ad.Wrapper.Impressions) == 0 {
				return fmt.Errorf("Wrapper ad at index %d must have at least one Impression", i)
			}
			if ad.Wrapper.VASTAdTagURI == nil || ad.Wrapper.VASTAdTagURI.Value == "" {
				return fmt.Errorf("Wrapper ad at index %d must have VASTAdTagURI", i)
			}
		}

		// Validate inline
		if ad.InLine != nil {
			if len(ad.InLine.Impressions) == 0 {
				return fmt.Errorf("InLine ad at index %d must have at least one Impression", i)
			}
			if ad.InLine.AdTitle == "" {
				return fmt.Errorf("InLine ad at index %d must have AdTitle", i)
			}
			if len(ad.InLine.Creatives) == 0 {
				return fmt.Errorf("InLine ad at index %d must have at least one Creative", i)
			}
		}
	}

	return nil
}
