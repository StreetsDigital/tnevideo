package ortb

import (
	"testing"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

func TestValidateVideo_Duration(t *testing.T) {
	tests := []struct {
		name        string
		video       *openrtb2.Video
		wantError   bool
		errorString string
	}{
		{
			name: "valid_duration_constraints",
			video: &openrtb2.Video{
				MIMEs:       []string{"video/mp4"},
				MinDuration: 15,
				MaxDuration: 30,
			},
			wantError: false,
		},
		{
			name: "negative_minduration",
			video: &openrtb2.Video{
				MIMEs:       []string{"video/mp4"},
				MinDuration: -5,
			},
			wantError:   true,
			errorString: "minduration must be a non-negative number",
		},
		{
			name: "negative_maxduration",
			video: &openrtb2.Video{
				MIMEs:       []string{"video/mp4"},
				MaxDuration: -10,
			},
			wantError:   true,
			errorString: "maxduration must be a non-negative number",
		},
		{
			name: "minduration_exceeds_maxduration",
			video: &openrtb2.Video{
				MIMEs:       []string{"video/mp4"},
				MinDuration: 30,
				MaxDuration: 15,
			},
			wantError:   true,
			errorString: "minduration must not exceed maxduration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVideo(tt.video, 0)

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateVideo_Protocols(t *testing.T) {
	tests := []struct {
		name        string
		video       *openrtb2.Video
		wantError   bool
		errorString string
	}{
		{
			name: "valid_protocols",
			video: &openrtb2.Video{
				MIMEs:     []string{"video/mp4"},
				Protocols: []adcom1.MediaCreativeSubtype{2, 3, 7}, // VAST 2.0, 3.0, 4.0
			},
			wantError: false,
		},
		{
			name: "all_valid_protocols",
			video: &openrtb2.Video{
				MIMEs:     []string{"video/mp4"},
				Protocols: []adcom1.MediaCreativeSubtype{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
			},
			wantError: false,
		},
		{
			name: "invalid_protocol_zero",
			video: &openrtb2.Video{
				MIMEs:     []string{"video/mp4"},
				Protocols: []adcom1.MediaCreativeSubtype{0},
			},
			wantError:   true,
			errorString: "protocols contains invalid value 0 (must be 1-12)",
		},
		{
			name: "invalid_protocol_too_high",
			video: &openrtb2.Video{
				MIMEs:     []string{"video/mp4"},
				Protocols: []adcom1.MediaCreativeSubtype{13},
			},
			wantError:   true,
			errorString: "protocols contains invalid value 13 (must be 1-12)",
		},
		{
			name: "mixed_valid_invalid_protocols",
			video: &openrtb2.Video{
				MIMEs:     []string{"video/mp4"},
				Protocols: []adcom1.MediaCreativeSubtype{2, 3, 99},
			},
			wantError:   true,
			errorString: "protocols contains invalid value 99",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVideo(tt.video, 0)

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateVideo_SkipParameters(t *testing.T) {
	tests := []struct {
		name        string
		video       *openrtb2.Video
		wantError   bool
		errorString string
	}{
		{
			name: "valid_skip_enabled",
			video: &openrtb2.Video{
				MIMEs:     []string{"video/mp4"},
				Skip:      ptrutil.ToPtr[int8](1),
				SkipMin:   5,
				SkipAfter: 5,
			},
			wantError: false,
		},
		{
			name: "skip_disabled",
			video: &openrtb2.Video{
				MIMEs: []string{"video/mp4"},
				Skip:  ptrutil.ToPtr[int8](0),
			},
			wantError: false,
		},
		{
			name: "negative_skipmin",
			video: &openrtb2.Video{
				MIMEs:   []string{"video/mp4"},
				Skip:    ptrutil.ToPtr[int8](1),
				SkipMin: -5,
			},
			wantError:   true,
			errorString: "skipmin must be a non-negative number",
		},
		{
			name: "negative_skipafter",
			video: &openrtb2.Video{
				MIMEs:     []string{"video/mp4"},
				Skip:      ptrutil.ToPtr[int8](1),
				SkipAfter: -3,
			},
			wantError:   true,
			errorString: "skipafter must be a non-negative number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVideo(tt.video, 0)

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateVideo_Sequence(t *testing.T) {
	tests := []struct {
		name        string
		video       *openrtb2.Video
		wantError   bool
		errorString string
	}{
		{
			name: "valid_sequence_zero",
			video: &openrtb2.Video{
				MIMEs:    []string{"video/mp4"},
				Sequence: 0,
			},
			wantError: false,
		},
		{
			name: "valid_sequence_positive",
			video: &openrtb2.Video{
				MIMEs:    []string{"video/mp4"},
				Sequence: 5,
			},
			wantError: false,
		},
		{
			name: "negative_sequence",
			video: &openrtb2.Video{
				MIMEs:    []string{"video/mp4"},
				Sequence: -1,
			},
			wantError:   true,
			errorString: "sequence must be a non-negative number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVideo(tt.video, 0)

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateVideo_Placement(t *testing.T) {
	tests := []struct {
		name        string
		video       *openrtb2.Video
		wantError   bool
		errorString string
	}{
		{
			name: "valid_placement_in_stream",
			video: &openrtb2.Video{
				MIMEs:     []string{"video/mp4"},
				Placement: adcom1.VideoPlacementInStream,
			},
			wantError: false,
		},
		{
			name: "valid_placement_in_banner",
			video: &openrtb2.Video{
				MIMEs:     []string{"video/mp4"},
				Placement: adcom1.VideoPlacementInBanner,
			},
			wantError: false,
		},
		{
			name: "valid_placement_zero_unspecified",
			video: &openrtb2.Video{
				MIMEs:     []string{"video/mp4"},
				Placement: adcom1.VideoPlacementSubtype(0),
			},
			wantError: false, // 0 means "unspecified" per OpenRTB, which is valid
		},
		{
			name: "invalid_placement_too_high",
			video: &openrtb2.Video{
				MIMEs:     []string{"video/mp4"},
				Placement: adcom1.VideoPlacementSubtype(8),
			},
			wantError:   true,
			errorString: "placement must be between 1 and 7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVideo(tt.video, 0)

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateVideo_PlaybackMethod(t *testing.T) {
	tests := []struct {
		name        string
		video       *openrtb2.Video
		wantError   bool
		errorString string
	}{
		{
			name: "valid_playback_methods",
			video: &openrtb2.Video{
				MIMEs:          []string{"video/mp4"},
				PlaybackMethod: []adcom1.PlaybackMethod{1, 2, 3},
			},
			wantError: false,
		},
		{
			name: "invalid_playback_method_zero",
			video: &openrtb2.Video{
				MIMEs:          []string{"video/mp4"},
				PlaybackMethod: []adcom1.PlaybackMethod{0},
			},
			wantError:   true,
			errorString: "playbackmethod contains invalid value 0 (must be 1-7)",
		},
		{
			name: "invalid_playback_method_too_high",
			video: &openrtb2.Video{
				MIMEs:          []string{"video/mp4"},
				PlaybackMethod: []adcom1.PlaybackMethod{8},
			},
			wantError:   true,
			errorString: "playbackmethod contains invalid value 8 (must be 1-7)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVideo(tt.video, 0)

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateVideo_Delivery(t *testing.T) {
	tests := []struct {
		name        string
		video       *openrtb2.Video
		wantError   bool
		errorString string
	}{
		{
			name: "valid_delivery_methods",
			video: &openrtb2.Video{
				MIMEs:    []string{"video/mp4"},
				Delivery: []adcom1.DeliveryMethod{1, 2, 3},
			},
			wantError: false,
		},
		{
			name: "invalid_delivery_zero",
			video: &openrtb2.Video{
				MIMEs:    []string{"video/mp4"},
				Delivery: []adcom1.DeliveryMethod{0},
			},
			wantError:   true,
			errorString: "delivery contains invalid value 0 (must be 1-3)",
		},
		{
			name: "invalid_delivery_too_high",
			video: &openrtb2.Video{
				MIMEs:    []string{"video/mp4"},
				Delivery: []adcom1.DeliveryMethod{4},
			},
			wantError:   true,
			errorString: "delivery contains invalid value 4 (must be 1-3)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVideo(tt.video, 0)

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateVideo_APIs(t *testing.T) {
	tests := []struct {
		name        string
		video       *openrtb2.Video
		wantError   bool
		errorString string
	}{
		{
			name: "valid_apis",
			video: &openrtb2.Video{
				MIMEs: []string{"video/mp4"},
				API:   []adcom1.APIFramework{1, 2, 5, 7}, // VPAID 1.0, 2.0, MRAID-2, ORMMA
			},
			wantError: false,
		},
		{
			name: "invalid_api_zero",
			video: &openrtb2.Video{
				MIMEs: []string{"video/mp4"},
				API:   []adcom1.APIFramework{0},
			},
			wantError:   true,
			errorString: "api contains invalid value 0 (must be 1-8)",
		},
		{
			name: "invalid_api_too_high",
			video: &openrtb2.Video{
				MIMEs: []string{"video/mp4"},
				API:   []adcom1.APIFramework{9},
			},
			wantError:   true,
			errorString: "api contains invalid value 9 (must be 1-8)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVideo(tt.video, 0)

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateVideo_StartDelay(t *testing.T) {
	tests := []struct {
		name        string
		video       *openrtb2.Video
		wantError   bool
		errorString string
	}{
		{
			name: "valid_pre_roll",
			video: &openrtb2.Video{
				MIMEs:      []string{"video/mp4"},
				StartDelay: adcom1.StartDelay(-1).Ptr(),
			},
			wantError: false,
		},
		{
			name: "valid_post_roll",
			video: &openrtb2.Video{
				MIMEs:      []string{"video/mp4"},
				StartDelay: adcom1.StartDelay(-2).Ptr(),
			},
			wantError: false,
		},
		{
			name: "valid_generic_mid_roll",
			video: &openrtb2.Video{
				MIMEs:      []string{"video/mp4"},
				StartDelay: adcom1.StartDelay(0).Ptr(),
			},
			wantError: false,
		},
		{
			name: "valid_mid_roll_at_offset",
			video: &openrtb2.Video{
				MIMEs:      []string{"video/mp4"},
				StartDelay: adcom1.StartDelay(300).Ptr(),
			},
			wantError: false,
		},
		{
			name: "nil_start_delay",
			video: &openrtb2.Video{
				MIMEs:      []string{"video/mp4"},
				StartDelay: nil,
			},
			wantError: false,
		},
		{
			name: "invalid_start_delay_too_negative",
			video: &openrtb2.Video{
				MIMEs:      []string{"video/mp4"},
				StartDelay: adcom1.StartDelay(-3).Ptr(),
			},
			wantError:   true,
			errorString: "startdelay must be >= -2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVideo(tt.video, 0)

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateVideo_ComprehensiveCTVExample(t *testing.T) {
	// Comprehensive CTV video request
	video := &openrtb2.Video{
		MIMEs:          []string{"video/mp4", "video/webm"},
		MinDuration:    15,
		MaxDuration:    30,
		Protocols:      []adcom1.MediaCreativeSubtype{2, 3, 7}, // VAST 2.0, 3.0, 4.0
		W:              ptrutil.ToPtr[int64](1920),
		H:              ptrutil.ToPtr[int64](1080),
		StartDelay:     adcom1.StartDelay(-1).Ptr(), // Pre-roll
		Placement:      adcom1.VideoPlacementInStream,
		PlaybackMethod: []adcom1.PlaybackMethod{1}, // Auto-play sound on
		Skip:           ptrutil.ToPtr[int8](1),
		SkipMin:        5,
		SkipAfter:      5,
		Delivery:       []adcom1.DeliveryMethod{2}, // Progressive
		API:            []adcom1.APIFramework{2, 5}, // VPAID 2.0, MRAID-2
		Sequence:       0,
		MinBitRate:     300,
		MaxBitRate:     1500,
	}

	err := validateVideo(video, 0)
	assert.NoError(t, err)
}

func TestValidateVideo_ComprehensiveFieldParsing(t *testing.T) {
	tests := []struct {
		name        string
		video       *openrtb2.Video
		description string
		wantError   bool
	}{
		{
			name: "parse_all_video_fields",
			video: &openrtb2.Video{
				MIMEs:          []string{"video/mp4", "video/webm", "application/javascript"},
				MinDuration:    5,
				MaxDuration:    60,
				Protocols:      []adcom1.MediaCreativeSubtype{1, 2, 3, 7, 8}, // VAST 1.0, 2.0, 3.0, 4.0, DAAST 1.0
				StartDelay:     adcom1.StartDelay(-1).Ptr(),
				W:              ptrutil.ToPtr[int64](1920),
				H:              ptrutil.ToPtr[int64](1080),
				Skip:           ptrutil.ToPtr[int8](1),
				SkipMin:        5,
				SkipAfter:      5,
				Placement:      adcom1.VideoPlacementInStream,
				PlaybackMethod: []adcom1.PlaybackMethod{1, 2},
				Delivery:       []adcom1.DeliveryMethod{1, 2},
				API:            []adcom1.APIFramework{1, 2, 5},
				Sequence:       0,
				MinBitRate:     300,
				MaxBitRate:     1500,
			},
			description: "All video fields correctly parsed and validated",
			wantError:   false,
		},
		{
			name: "protocol_negotiation_subset",
			video: &openrtb2.Video{
				MIMEs:     []string{"video/mp4"},
				Protocols: []adcom1.MediaCreativeSubtype{2, 3}, // Only VAST 2.0 and 3.0
			},
			description: "Protocol negotiation with subset of protocols",
			wantError:   false,
		},
		{
			name: "duration_constraints_enforced",
			video: &openrtb2.Video{
				MIMEs:       []string{"video/mp4"},
				MinDuration: 10,
				MaxDuration: 5, // Invalid: min > max
			},
			description: "Duration constraints must be validated",
			wantError:   true,
		},
		{
			name: "skip_settings_complete",
			video: &openrtb2.Video{
				MIMEs:     []string{"video/mp4"},
				Skip:      ptrutil.ToPtr[int8](1),
				SkipMin:   0,
				SkipAfter: 10,
			},
			description: "Skip settings fully validated",
			wantError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVideo(tt.video, 0)

			if tt.wantError {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
			}
		})
	}
}
