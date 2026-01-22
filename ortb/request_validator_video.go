package ortb

import (
	"fmt"

	"github.com/prebid/openrtb/v20/openrtb2"
)

func validateVideo(video *openrtb2.Video, impIndex int) error {
	if video == nil {
		return nil
	}

	if len(video.MIMEs) < 1 {
		return fmt.Errorf("request.imp[%d].video.mimes must contain at least one supported MIME type", impIndex)
	}

	// The following fields were previously uints in the OpenRTB library we use, but have
	// since been changed to ints. We decided to maintain the non-negative check.
	if video.W != nil && *video.W < 0 {
		return fmt.Errorf("request.imp[%d].video.w must be a positive number", impIndex)
	}
	if video.H != nil && *video.H < 0 {
		return fmt.Errorf("request.imp[%d].video.h must be a positive number", impIndex)
	}
	if video.MinBitRate < 0 {
		return fmt.Errorf("request.imp[%d].video.minbitrate must be a positive number", impIndex)
	}
	if video.MaxBitRate < 0 {
		return fmt.Errorf("request.imp[%d].video.maxbitrate must be a positive number", impIndex)
	}

	// Validate duration constraints
	if video.MinDuration < 0 {
		return fmt.Errorf("request.imp[%d].video.minduration must be a non-negative number", impIndex)
	}
	if video.MaxDuration < 0 {
		return fmt.Errorf("request.imp[%d].video.maxduration must be a non-negative number", impIndex)
	}
	if video.MinDuration > 0 && video.MaxDuration > 0 && video.MinDuration > video.MaxDuration {
		return fmt.Errorf("request.imp[%d].video.minduration must not exceed maxduration", impIndex)
	}

	// Validate start delay
	// Per OpenRTB 2.5: -2 = post-roll, -1 = pre-roll, 0 = generic mid-roll, >0 = mid-roll at specific offset
	if video.StartDelay != nil && *video.StartDelay < -2 {
		return fmt.Errorf("request.imp[%d].video.startdelay must be >= -2 (per OpenRTB spec)", impIndex)
	}

	// Validate protocols
	if len(video.Protocols) > 0 {
		for _, protocol := range video.Protocols {
			if protocol < 1 || protocol > 12 {
				return fmt.Errorf("request.imp[%d].video.protocols contains invalid value %d (must be 1-12)", impIndex, protocol)
			}
		}
	}

	// Validate skip parameters
	if video.Skip != nil && *video.Skip == 1 {
		if video.SkipMin < 0 {
			return fmt.Errorf("request.imp[%d].video.skipmin must be a non-negative number", impIndex)
		}
		if video.SkipAfter < 0 {
			return fmt.Errorf("request.imp[%d].video.skipafter must be a non-negative number", impIndex)
		}
	}

	// Validate sequence (for ad pods)
	if video.Sequence < 0 {
		return fmt.Errorf("request.imp[%d].video.sequence must be a non-negative number", impIndex)
	}

	// Validate placement type
	if video.Placement != 0 && (video.Placement < 1 || video.Placement > 7) {
		return fmt.Errorf("request.imp[%d].video.placement must be between 1 and 7", impIndex)
	}

	// Validate playback method
	if len(video.PlaybackMethod) > 0 {
		for _, method := range video.PlaybackMethod {
			if method < 1 || method > 7 {
				return fmt.Errorf("request.imp[%d].video.playbackmethod contains invalid value %d (must be 1-7)", impIndex, method)
			}
		}
	}

	// Validate delivery methods
	if len(video.Delivery) > 0 {
		for _, delivery := range video.Delivery {
			if delivery < 1 || delivery > 3 {
				return fmt.Errorf("request.imp[%d].video.delivery contains invalid value %d (must be 1-3)", impIndex, delivery)
			}
		}
	}

	// Validate API frameworks
	if len(video.API) > 0 {
		for _, api := range video.API {
			if api < 1 || api > 8 {
				return fmt.Errorf("request.imp[%d].video.api contains invalid value %d (must be 1-8)", impIndex, api)
			}
		}
	}

	return nil
}
