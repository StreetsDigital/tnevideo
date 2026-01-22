package ortb

import (
	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
)

// Type aliases to avoid non-existent openrtb2 types
type (
	// PlaybackMethod from adcom1
	PlaybackMethod = adcom1.PlaybackMethod
	// VideoPlacement from adcom1
	VideoPlacement = adcom1.VideoPlacementSubtype
)

// VideoHelper provides utilities for working with OpenRTB Video objects
type VideoHelper struct{}

// NewVideoHelper creates a new VideoHelper instance
func NewVideoHelper() *VideoHelper {
	return &VideoHelper{}
}

// GetStartDelay returns the start delay type for the video
func (vh *VideoHelper) GetStartDelay(video *openrtb2.Video) StartDelayType {
	if video == nil || video.StartDelay == nil {
		return StartDelayUnknown
	}

	delay := *video.StartDelay
	switch {
	case delay == -1:
		return StartDelayPreRoll
	case delay == 0:
		return StartDelayGenericMidRoll
	case delay > 0:
		return StartDelayMidRoll
	default:
		return StartDelayUnknown
	}
}

// IsSkippable returns whether the video ad is skippable
func (vh *VideoHelper) IsSkippable(video *openrtb2.Video) bool {
	if video == nil || video.Skip == nil {
		return false
	}
	return *video.Skip == 1
}

// GetSkipDelay returns the skip delay in seconds
func (vh *VideoHelper) GetSkipDelay(video *openrtb2.Video) int64 {
	if video == nil || !vh.IsSkippable(video) {
		return 0
	}

	if video.SkipMin > 0 {
		return video.SkipMin
	}

	if video.SkipAfter > 0 {
		return video.SkipAfter
	}

	return 0
}

// SupportsProtocol checks if the video supports a specific protocol
func (vh *VideoHelper) SupportsProtocol(video *openrtb2.Video, protocol adcom1.MediaCreativeSubtype) bool {
	if video == nil || len(video.Protocols) == 0 {
		return false
	}

	for _, p := range video.Protocols {
		if p == protocol {
			return true
		}
	}
	return false
}

// SupportsMIME checks if the video supports a specific MIME type
func (vh *VideoHelper) SupportsMIME(video *openrtb2.Video, mime string) bool {
	if video == nil || len(video.MIMEs) == 0 {
		return false
	}

	for _, m := range video.MIMEs {
		if m == mime {
			return true
		}
	}
	return false
}

// IsInDurationRange checks if a duration is within the video's acceptable range
func (vh *VideoHelper) IsInDurationRange(video *openrtb2.Video, duration int64) bool {
	if video == nil {
		return true
	}

	if video.MinDuration > 0 && duration < video.MinDuration {
		return false
	}

	if video.MaxDuration > 0 && duration > video.MaxDuration {
		return false
	}

	return true
}

// GetPlacementType returns a human-readable placement type
func (vh *VideoHelper) GetPlacementType(video *openrtb2.Video) string {
	if video == nil || video.Placement == 0 {
		return "Unknown"
	}

	switch video.Placement {
	case adcom1.VideoPlacementInStream:
		return "In-Stream"
	case adcom1.VideoPlacementInBanner:
		return "In-Banner"
	case adcom1.VideoPlacementInArticle:
		return "In-Article"
	case adcom1.VideoPlacementInFeed:
		return "In-Feed"
	case adcom1.VideoPlacementAlwaysVisible:
		return "Interstitial/Slider/Floating"
	default:
		return "Unknown"
	}
}

// GetProtocolName returns the protocol name
func (vh *VideoHelper) GetProtocolName(protocol adcom1.MediaCreativeSubtype) string {
	protocolNames := map[adcom1.MediaCreativeSubtype]string{
		1:  "VAST 1.0",
		2:  "VAST 2.0",
		3:  "VAST 3.0",
		4:  "VAST 1.0 Wrapper",
		5:  "VAST 2.0 Wrapper",
		6:  "VAST 3.0 Wrapper",
		7:  "VAST 4.0",
		8:  "VAST 4.0 Wrapper",
		9:  "DAAST 1.0",
		10: "DAAST 1.0 Wrapper",
		11: "VAST 4.1",
		12: "VAST 4.1 Wrapper",
	}

	if name, ok := protocolNames[protocol]; ok {
		return name
	}
	return "Unknown"
}

// IsVASTWrapper checks if the protocol is a VAST wrapper
func (vh *VideoHelper) IsVASTWrapper(protocol adcom1.MediaCreativeSubtype) bool {
	wrapperProtocols := []adcom1.MediaCreativeSubtype{4, 5, 6, 8, 10, 12}
	for _, wp := range wrapperProtocols {
		if protocol == wp {
			return true
		}
	}
	return false
}

// GetAPIFrameworkName returns the API framework name
func (vh *VideoHelper) GetAPIFrameworkName(api adcom1.APIFramework) string {
	apiNames := map[adcom1.APIFramework]string{
		1: "VPAID 1.0",
		2: "VPAID 2.0",
		3: "MRAID-1",
		4: "ORMMA",
		5: "MRAID-2",
		6: "MRAID-3",
		7: "OMID-1",
		8: "SIMID-1",
	}

	if name, ok := apiNames[api]; ok {
		return name
	}
	return "Unknown"
}

// IsAutoPlay checks if video will auto-play
func (vh *VideoHelper) IsAutoPlay(video *openrtb2.Video) bool {
	if video == nil || len(video.PlaybackMethod) == 0 {
		return false
	}

	// PlaybackMethod 1 = Auto-play sound on, 3 = Auto-play sound off, 5 = Continuous auto-play sound on/off
	autoPlayMethods := []adcom1.PlaybackMethod{1, 3, 5}
	for _, method := range video.PlaybackMethod {
		for _, autoPlayMethod := range autoPlayMethods {
			if method == autoPlayMethod {
				return true
			}
		}
	}
	return false
}

// IsCTV checks if the video impression is for CTV based on placement and size
func (vh *VideoHelper) IsCTV(video *openrtb2.Video) bool {
	if video == nil {
		return false
	}

	// CTV typically has:
	// - In-stream placement
	// - Larger dimensions (typically 1280x720 or 1920x1080)
	// - Pre-roll or mid-roll placement

	if video.Placement == adcom1.VideoPlacementInStream {
		if video.W != nil && video.H != nil {
			w, h := *video.W, *video.H
			// Check for common CTV resolutions
			if (w >= 1280 && h >= 720) || (w >= 1920 && h >= 1080) {
				return true
			}
		}
		return true // In-stream without specific size is likely CTV
	}

	return false
}

// StartDelayType represents the type of video ad start delay
type StartDelayType int

const (
	StartDelayUnknown StartDelayType = iota
	StartDelayPreRoll
	StartDelayGenericMidRoll
	StartDelayMidRoll
)

func (s StartDelayType) String() string {
	switch s {
	case StartDelayPreRoll:
		return "Pre-roll"
	case StartDelayGenericMidRoll:
		return "Generic Mid-roll"
	case StartDelayMidRoll:
		return "Mid-roll"
	default:
		return "Unknown"
	}
}
