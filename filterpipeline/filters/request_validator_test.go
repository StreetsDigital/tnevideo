package filters

import (
	"context"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/filterpipeline"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestRequestValidatorFilter_Name(t *testing.T) {
	config := RequestValidatorConfig{Enabled: true, Priority: 10}
	filter := NewRequestValidatorFilter(config)
	assert.Equal(t, "request_validator", filter.Name())
}

func TestRequestValidatorFilter_Priority(t *testing.T) {
	config := RequestValidatorConfig{Enabled: true, Priority: 25}
	filter := NewRequestValidatorFilter(config)
	assert.Equal(t, 25, filter.Priority())
}

func TestRequestValidatorFilter_Enabled(t *testing.T) {
	tests := []struct {
		name      string
		enabled   bool
		accountID string
		expected  bool
	}{
		{
			name:      "enabled_filter",
			enabled:   true,
			accountID: "test-account",
			expected:  true,
		},
		{
			name:      "disabled_filter",
			enabled:   false,
			accountID: "test-account",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := RequestValidatorConfig{Enabled: tt.enabled, Priority: 10}
			filter := NewRequestValidatorFilter(config)
			result := filter.Enabled(tt.accountID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRequestValidatorFilter_Execute(t *testing.T) {
	tests := []struct {
		name           string
		config         RequestValidatorConfig
		request        *openrtb_ext.RequestWrapper
		expectReject   bool
		rejectContains string
	}{
		{
			name: "valid_request",
			config: RequestValidatorConfig{
				Enabled:        true,
				Priority:       10,
				RequireUser:    false,
				RequireDevice:  false,
				MinImpressions: 1,
			},
			request: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID: "test-req",
					Imp: []openrtb2.Imp{
						{ID: "imp1"},
					},
				},
			},
			expectReject: false,
		},
		{
			name: "missing_user_required",
			config: RequestValidatorConfig{
				Enabled:        true,
				Priority:       10,
				RequireUser:    true,
				RequireDevice:  false,
				MinImpressions: 0,
			},
			request: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID: "test-req",
					Imp: []openrtb2.Imp{
						{ID: "imp1"},
					},
				},
			},
			expectReject:   true,
			rejectContains: "user",
		},
		{
			name: "missing_device_required",
			config: RequestValidatorConfig{
				Enabled:        true,
				Priority:       10,
				RequireUser:    false,
				RequireDevice:  true,
				MinImpressions: 0,
			},
			request: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID: "test-req",
					Imp: []openrtb2.Imp{
						{ID: "imp1"},
					},
				},
			},
			expectReject:   true,
			rejectContains: "device",
		},
		{
			name: "insufficient_impressions",
			config: RequestValidatorConfig{
				Enabled:        true,
				Priority:       10,
				RequireUser:    false,
				RequireDevice:  false,
				MinImpressions: 3,
			},
			request: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID: "test-req",
					Imp: []openrtb2.Imp{
						{ID: "imp1"},
					},
				},
			},
			expectReject:   true,
			rejectContains: "impression",
		},
		{
			name: "nil_request",
			config: RequestValidatorConfig{
				Enabled:  true,
				Priority: 10,
			},
			request:        nil,
			expectReject:   true,
			rejectContains: "missing",
		},
		{
			name: "valid_request_with_user_and_device",
			config: RequestValidatorConfig{
				Enabled:        true,
				Priority:       10,
				RequireUser:    true,
				RequireDevice:  true,
				MinImpressions: 2,
			},
			request: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID: "test-req",
					User: &openrtb2.User{
						ID: "user123",
					},
					Device: &openrtb2.Device{
						UA: "Mozilla/5.0",
					},
					Imp: []openrtb2.Imp{
						{ID: "imp1"},
						{ID: "imp2"},
					},
				},
			},
			expectReject: false,
		},
		{
			name: "impression_missing_id",
			config: RequestValidatorConfig{
				Enabled:        true,
				Priority:       10,
				RequireUser:    false,
				RequireDevice:  false,
				MinImpressions: 0,
			},
			request: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID: "test-req",
					Imp: []openrtb2.Imp{
						{ID: "imp1"},
						{ID: ""}, // Missing ID
					},
				},
			},
			expectReject: false, // Should not reject but add error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewRequestValidatorFilter(tt.config)

			req := filterpipeline.PreFilterRequest{
				Request: tt.request,
				Context: filterpipeline.FilterContext{
					AccountID: "test-account",
					Endpoint:  "/openrtb2/auction",
				},
			}

			resp, err := filter.Execute(context.Background(), req)

			assert.NoError(t, err)
			assert.NotNil(t, resp)

			if tt.expectReject {
				assert.True(t, resp.Reject)
				assert.NotEmpty(t, resp.RejectReason)
				if tt.rejectContains != "" {
					assert.Contains(t, resp.RejectReason, tt.rejectContains)
				}
			} else {
				assert.False(t, resp.Reject)
				assert.NotNil(t, resp.Metadata)

				// Verify validation metadata
				validationData, ok := resp.Metadata["validation"].(map[string]interface{})
				assert.True(t, ok)
				assert.True(t, validationData["passed"].(bool))
			}
		})
	}
}
