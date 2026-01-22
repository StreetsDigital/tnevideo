package stored_requests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateVideoFields(t *testing.T) {
	tests := []struct {
		name        string
		fields      *VideoFields
		expectError bool
		errorCount  int
	}{
		{
			name:        "nil_fields",
			fields:      nil,
			expectError: false,
		},
		{
			name: "valid_fields",
			fields: &VideoFields{
				DurationMin: intPtr(15),
				DurationMax: intPtr(30),
				Protocols:   []int{2, 3, 7},
				StartDelay:  intPtr(-1),
				Mimes:       []string{"video/mp4", "video/webm"},
				Skippable:   boolPtr(true),
				SkipDelay:   intPtr(5),
			},
			expectError: false,
		},
		{
			name: "negative_duration_min",
			fields: &VideoFields{
				DurationMin: intPtr(-5),
			},
			expectError: true,
			errorCount:  1,
		},
		{
			name: "negative_duration_max",
			fields: &VideoFields{
				DurationMax: intPtr(-10),
			},
			expectError: true,
			errorCount:  1,
		},
		{
			name: "duration_min_greater_than_max",
			fields: &VideoFields{
				DurationMin: intPtr(30),
				DurationMax: intPtr(15),
			},
			expectError: true,
			errorCount:  1,
		},
		{
			name: "invalid_protocol",
			fields: &VideoFields{
				Protocols: []int{1, 2, 99},
			},
			expectError: true,
			errorCount:  1,
		},
		{
			name: "negative_skip_delay_when_skippable",
			fields: &VideoFields{
				Skippable: boolPtr(true),
				SkipDelay: intPtr(-5),
			},
			expectError: true,
			errorCount:  1,
		},
		{
			name: "unsupported_mime_type",
			fields: &VideoFields{
				Mimes: []string{"video/mp4", "video/invalid"},
			},
			expectError: true,
			errorCount:  1,
		},
		{
			name: "multiple_errors",
			fields: &VideoFields{
				DurationMin: intPtr(-5),  // Error 1: negative
				DurationMax: intPtr(-10), // Error 2: negative, Error 3: min > max (-5 > -10)
				Protocols:   []int{99},   // Error 4: invalid protocol
				Skippable:   boolPtr(true),
				SkipDelay:   intPtr(-5),  // Error 5: negative skip delay
			},
			expectError: true,
			errorCount:  5, // 5 errors total
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateVideoFields(tt.fields)

			if tt.expectError {
				assert.NotEmpty(t, errs)
				assert.Equal(t, tt.errorCount, len(errs))
			} else {
				assert.Empty(t, errs)
			}
		})
	}
}

func TestValidateVideoFields_Protocols(t *testing.T) {
	tests := []struct {
		name      string
		protocols []int
		valid     bool
	}{
		{
			name:      "valid_vast_2_3_7",
			protocols: []int{2, 3, 7},
			valid:     true,
		},
		{
			name:      "valid_all_protocols",
			protocols: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
			valid:     true,
		},
		{
			name:      "invalid_protocol_0",
			protocols: []int{0},
			valid:     false,
		},
		{
			name:      "invalid_protocol_13",
			protocols: []int{13},
			valid:     false,
		},
		{
			name:      "invalid_negative_protocol",
			protocols: []int{-1},
			valid:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := &VideoFields{
				Protocols: tt.protocols,
			}

			errs := ValidateVideoFields(fields)

			if tt.valid {
				assert.Empty(t, errs)
			} else {
				assert.NotEmpty(t, errs)
			}
		})
	}
}

func TestValidateVideoFields_Mimes(t *testing.T) {
	tests := []struct {
		name  string
		mimes []string
		valid bool
	}{
		{
			name:  "valid_mp4_webm",
			mimes: []string{"video/mp4", "video/webm"},
			valid: true,
		},
		{
			name:  "valid_all_standard_types",
			mimes: []string{"video/mp4", "video/webm", "video/ogg"},
			valid: true,
		},
		{
			name:  "valid_vpaid",
			mimes: []string{"application/javascript"},
			valid: true,
		},
		{
			name:  "invalid_mime_type",
			mimes: []string{"video/unknown"},
			valid: false,
		},
		{
			name:  "mixed_valid_invalid",
			mimes: []string{"video/mp4", "video/invalid"},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := &VideoFields{
				Mimes: tt.mimes,
			}

			errs := ValidateVideoFields(fields)

			if tt.valid {
				assert.Empty(t, errs)
			} else {
				assert.NotEmpty(t, errs)
			}
		})
	}
}

func TestValidateVideoFields_StartDelay(t *testing.T) {
	tests := []struct {
		name       string
		startDelay int
		valid      bool
	}{
		{
			name:       "pre_roll",
			startDelay: -1,
			valid:      true,
		},
		{
			name:       "mid_roll",
			startDelay: 0,
			valid:      true,
		},
		{
			name:       "post_roll_300s",
			startDelay: 300,
			valid:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := &VideoFields{
				StartDelay: &tt.startDelay,
			}

			errs := ValidateVideoFields(fields)

			if tt.valid {
				assert.Empty(t, errs)
			} else {
				assert.NotEmpty(t, errs)
			}
		})
	}
}

// Helper functions
func intPtr(i int) *int {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}
