package ortb

import (
	"testing"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

func TestVideoHelper_GetStartDelay(t *testing.T) {
	vh := NewVideoHelper()

	tests := []struct {
		name     string
		video    *openrtb2.Video
		expected StartDelayType
	}{
		{
			name:     "nil_video",
			video:    nil,
			expected: StartDelayUnknown,
		},
		{
			name:     "nil_start_delay",
			video:    &openrtb2.Video{},
			expected: StartDelayUnknown,
		},
		{
			name: "pre_roll",
			video: &openrtb2.Video{
				StartDelay: adcom1.StartDelay(-1).Ptr(),
			},
			expected: StartDelayPreRoll,
		},
		{
			name: "generic_mid_roll",
			video: &openrtb2.Video{
				StartDelay: adcom1.StartDelay(0).Ptr(),
			},
			expected: StartDelayGenericMidRoll,
		},
		{
			name: "mid_roll_at_300",
			video: &openrtb2.Video{
				StartDelay: adcom1.StartDelay(300).Ptr(),
			},
			expected: StartDelayMidRoll,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := vh.GetStartDelay(tt.video)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVideoHelper_IsSkippable(t *testing.T) {
	vh := NewVideoHelper()

	tests := []struct {
		name     string
		video    *openrtb2.Video
		expected bool
	}{
		{
			name:     "nil_video",
			video:    nil,
			expected: false,
		},
		{
			name:     "nil_skip",
			video:    &openrtb2.Video{},
			expected: false,
		},
		{
			name: "skip_enabled",
			video: &openrtb2.Video{
				Skip: ptrutil.ToPtr[int8](1),
			},
			expected: true,
		},
		{
			name: "skip_disabled",
			video: &openrtb2.Video{
				Skip: ptrutil.ToPtr[int8](0),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := vh.IsSkippable(tt.video)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVideoHelper_GetSkipDelay(t *testing.T) {
	vh := NewVideoHelper()

	tests := []struct {
		name     string
		video    *openrtb2.Video
		expected int64
	}{
		{
			name:     "nil_video",
			video:    nil,
			expected: 0,
		},
		{
			name: "skip_disabled",
			video: &openrtb2.Video{
				Skip: ptrutil.ToPtr[int8](0),
			},
			expected: 0,
		},
		{
			name: "skip_enabled_with_skipmin",
			video: &openrtb2.Video{
				Skip:    ptrutil.ToPtr[int8](1),
				SkipMin: 5,
			},
			expected: 5,
		},
		{
			name: "skip_enabled_with_skipafter",
			video: &openrtb2.Video{
				Skip:      ptrutil.ToPtr[int8](1),
				SkipAfter: 10,
			},
			expected: 10,
		},
		{
			name: "skip_enabled_with_both_prefer_skipmin",
			video: &openrtb2.Video{
				Skip:      ptrutil.ToPtr[int8](1),
				SkipMin:   5,
				SkipAfter: 10,
			},
			expected: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := vh.GetSkipDelay(tt.video)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVideoHelper_SupportsProtocol(t *testing.T) {
	vh := NewVideoHelper()

	tests := []struct {
		name     string
		video    *openrtb2.Video
		protocol adcom1.MediaCreativeSubtype
		expected bool
	}{
		{
			name:     "nil_video",
			video:    nil,
			protocol: 2,
			expected: false,
		},
		{
			name:     "empty_protocols",
			video:    &openrtb2.Video{},
			protocol: 2,
			expected: false,
		},
		{
			name: "protocol_supported",
			video: &openrtb2.Video{
				Protocols: []adcom1.MediaCreativeSubtype{2, 3, 7},
			},
			protocol: 3,
			expected: true,
		},
		{
			name: "protocol_not_supported",
			video: &openrtb2.Video{
				Protocols: []adcom1.MediaCreativeSubtype{2, 3, 7},
			},
			protocol: 5,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := vh.SupportsProtocol(tt.video, tt.protocol)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVideoHelper_SupportsMIME(t *testing.T) {
	vh := NewVideoHelper()

	tests := []struct {
		name     string
		video    *openrtb2.Video
		mime     string
		expected bool
	}{
		{
			name:     "nil_video",
			video:    nil,
			mime:     "video/mp4",
			expected: false,
		},
		{
			name:     "empty_mimes",
			video:    &openrtb2.Video{},
			mime:     "video/mp4",
			expected: false,
		},
		{
			name: "mime_supported",
			video: &openrtb2.Video{
				MIMEs: []string{"video/mp4", "video/webm"},
			},
			mime:     "video/mp4",
			expected: true,
		},
		{
			name: "mime_not_supported",
			video: &openrtb2.Video{
				MIMEs: []string{"video/mp4", "video/webm"},
			},
			mime:     "video/ogg",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := vh.SupportsMIME(tt.video, tt.mime)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVideoHelper_IsInDurationRange(t *testing.T) {
	vh := NewVideoHelper()

	tests := []struct {
		name     string
		video    *openrtb2.Video
		duration int64
		expected bool
	}{
		{
			name:     "nil_video",
			video:    nil,
			duration: 20,
			expected: true,
		},
		{
			name:     "no_duration_constraints",
			video:    &openrtb2.Video{},
			duration: 20,
			expected: true,
		},
		{
			name: "within_range",
			video: &openrtb2.Video{
				MinDuration: 15,
				MaxDuration: 30,
			},
			duration: 20,
			expected: true,
		},
		{
			name: "below_minimum",
			video: &openrtb2.Video{
				MinDuration: 15,
				MaxDuration: 30,
			},
			duration: 10,
			expected: false,
		},
		{
			name: "above_maximum",
			video: &openrtb2.Video{
				MinDuration: 15,
				MaxDuration: 30,
			},
			duration: 40,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := vh.IsInDurationRange(tt.video, tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVideoHelper_GetPlacementType(t *testing.T) {
	vh := NewVideoHelper()

	tests := []struct {
		name     string
		video    *openrtb2.Video
		expected string
	}{
		{
			name:     "nil_video",
			video:    nil,
			expected: "Unknown",
		},
		{
			name:     "nil_placement",
			video:    &openrtb2.Video{},
			expected: "Unknown",
		},
		{
			name: "in_stream",
			video: &openrtb2.Video{
				Placement: adcom1.VideoPlacementInStream,
			},
			expected: "In-Stream",
		},
		{
			name: "in_banner",
			video: &openrtb2.Video{
				Placement: adcom1.VideoPlacementInBanner,
			},
			expected: "In-Banner",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := vh.GetPlacementType(tt.video)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVideoHelper_GetProtocolName(t *testing.T) {
	vh := NewVideoHelper()

	tests := []struct {
		protocol adcom1.MediaCreativeSubtype
		expected string
	}{
		{1, "VAST 1.0"},
		{2, "VAST 2.0"},
		{3, "VAST 3.0"},
		{4, "VAST 1.0 Wrapper"},
		{7, "VAST 4.0"},
		{8, "VAST 4.0 Wrapper"},
		{99, "Unknown"},
	}

	for _, tt := range tests {
		result := vh.GetProtocolName(tt.protocol)
		assert.Equal(t, tt.expected, result)
	}
}

func TestVideoHelper_IsVASTWrapper(t *testing.T) {
	vh := NewVideoHelper()

	tests := []struct {
		protocol adcom1.MediaCreativeSubtype
		expected bool
	}{
		{2, false},  // VAST 2.0
		{3, false},  // VAST 3.0
		{4, true},   // VAST 1.0 Wrapper
		{5, true},   // VAST 2.0 Wrapper
		{6, true},   // VAST 3.0 Wrapper
		{7, false},  // VAST 4.0
		{8, true},   // VAST 4.0 Wrapper
		{10, true},  // DAAST 1.0 Wrapper
		{12, true},  // VAST 4.1 Wrapper
	}

	for _, tt := range tests {
		result := vh.IsVASTWrapper(tt.protocol)
		assert.Equal(t, tt.expected, result)
	}
}

func TestVideoHelper_IsAutoPlay(t *testing.T) {
	vh := NewVideoHelper()

	tests := []struct {
		name     string
		video    *openrtb2.Video
		expected bool
	}{
		{
			name:     "nil_video",
			video:    nil,
			expected: false,
		},
		{
			name:     "empty_playback_method",
			video:    &openrtb2.Video{},
			expected: false,
		},
		{
			name: "auto_play_sound_on",
			video: &openrtb2.Video{
				PlaybackMethod: []adcom1.PlaybackMethod{1},
			},
			expected: true,
		},
		{
			name: "auto_play_sound_off",
			video: &openrtb2.Video{
				PlaybackMethod: []adcom1.PlaybackMethod{3},
			},
			expected: true,
		},
		{
			name: "click_to_play",
			video: &openrtb2.Video{
				PlaybackMethod: []adcom1.PlaybackMethod{2},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := vh.IsAutoPlay(tt.video)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVideoHelper_IsCTV(t *testing.T) {
	vh := NewVideoHelper()

	tests := []struct {
		name     string
		video    *openrtb2.Video
		expected bool
	}{
		{
			name:     "nil_video",
			video:    nil,
			expected: false,
		},
		{
			name: "ctv_1920x1080",
			video: &openrtb2.Video{
				Placement: adcom1.VideoPlacementInStream,
				W:         ptrutil.ToPtr[int64](1920),
				H:         ptrutil.ToPtr[int64](1080),
			},
			expected: true,
		},
		{
			name: "ctv_1280x720",
			video: &openrtb2.Video{
				Placement: adcom1.VideoPlacementInStream,
				W:         ptrutil.ToPtr[int64](1280),
				H:         ptrutil.ToPtr[int64](720),
			},
			expected: true,
		},
		{
			name: "in_stream_no_size",
			video: &openrtb2.Video{
				Placement: adcom1.VideoPlacementInStream,
			},
			expected: true,
		},
		{
			name: "mobile_video",
			video: &openrtb2.Video{
				Placement: adcom1.VideoPlacementInStream,
				W:         ptrutil.ToPtr[int64](640),
				H:         ptrutil.ToPtr[int64](480),
			},
			expected: true, // Still in-stream
		},
		{
			name: "in_banner_video",
			video: &openrtb2.Video{
				Placement: adcom1.VideoPlacementInBanner,
				W:         ptrutil.ToPtr[int64](300),
				H:         ptrutil.ToPtr[int64](250),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := vh.IsCTV(tt.video)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStartDelayType_String(t *testing.T) {
	tests := []struct {
		startDelay StartDelayType
		expected   string
	}{
		{StartDelayPreRoll, "Pre-roll"},
		{StartDelayGenericMidRoll, "Generic Mid-roll"},
		{StartDelayMidRoll, "Mid-roll"},
		{StartDelayUnknown, "Unknown"},
	}

	for _, tt := range tests {
		result := tt.startDelay.String()
		assert.Equal(t, tt.expected, result)
	}
}
