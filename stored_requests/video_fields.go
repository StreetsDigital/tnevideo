package stored_requests

// VideoFields represents video-specific fields that can be stored in the database
type VideoFields struct {
	DurationMin  *int   `json:"video_duration_min,omitempty"`
	DurationMax  *int   `json:"video_duration_max,omitempty"`
	Protocols    []int  `json:"video_protocols,omitempty"`
	StartDelay   *int   `json:"video_start_delay,omitempty"`
	Mimes        []string `json:"video_mimes,omitempty"`
	Skippable    *bool  `json:"video_skippable,omitempty"`
	SkipDelay    *int   `json:"video_skip_delay,omitempty"`
}

// ValidateVideoFields validates video field constraints
func ValidateVideoFields(fields *VideoFields) []error {
	var errs []error

	if fields == nil {
		return nil
	}

	// Validate duration constraints
	if fields.DurationMin != nil && *fields.DurationMin < 0 {
		errs = append(errs, &ValidationError{
			Field:   "video_duration_min",
			Message: "must be non-negative",
		})
	}

	if fields.DurationMax != nil && *fields.DurationMax < 0 {
		errs = append(errs, &ValidationError{
			Field:   "video_duration_max",
			Message: "must be non-negative",
		})
	}

	if fields.DurationMin != nil && fields.DurationMax != nil {
		if *fields.DurationMin > *fields.DurationMax {
			errs = append(errs, &ValidationError{
				Field:   "video_duration_min",
				Message: "cannot be greater than video_duration_max",
			})
		}
	}

	// Validate protocols (OpenRTB 2.5 values: 1-12)
	for _, protocol := range fields.Protocols {
		if protocol < 1 || protocol > 12 {
			errs = append(errs, &ValidationError{
				Field:   "video_protocols",
				Message: "protocol values must be between 1 and 12 (OpenRTB 2.5 spec)",
			})
			break
		}
	}

	// Validate skip settings
	if fields.Skippable != nil && *fields.Skippable {
		if fields.SkipDelay != nil && *fields.SkipDelay < 0 {
			errs = append(errs, &ValidationError{
				Field:   "video_skip_delay",
				Message: "must be non-negative when video is skippable",
			})
		}
	}

	// Validate MIME types
	validMimes := map[string]bool{
		"video/mp4":                      true,
		"video/webm":                     true,
		"video/ogg":                      true,
		"video/3gpp":                     true,
		"video/x-flv":                    true,
		"application/javascript":         true,
		"application/x-shockwave-flash":  true,
	}

	for _, mime := range fields.Mimes {
		if !validMimes[mime] {
			errs = append(errs, &ValidationError{
				Field:   "video_mimes",
				Message: "contains unsupported MIME type: " + mime,
			})
		}
	}

	return errs
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
