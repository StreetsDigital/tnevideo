package filters

import (
	"context"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/filterpipeline"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestIdentityEnrichmentFilter_Name(t *testing.T) {
	filter := NewIdentityEnrichmentFilter(true, 10)
	assert.Equal(t, "identity_enrichment", filter.Name())
}

func TestIdentityEnrichmentFilter_Priority(t *testing.T) {
	filter := NewIdentityEnrichmentFilter(true, 25)
	assert.Equal(t, 25, filter.Priority())
}

func TestIdentityEnrichmentFilter_Enabled(t *testing.T) {
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
			filter := NewIdentityEnrichmentFilter(tt.enabled, 10)
			result := filter.Enabled(tt.accountID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIdentityEnrichmentFilter_Execute(t *testing.T) {
	tests := []struct {
		name                 string
		request              *openrtb_ext.RequestWrapper
		expectError          bool
		expectUserData       bool
		expectDeviceData     bool
		expectWarnings       bool
	}{
		{
			name: "enrichment_with_user_and_device",
			request: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID: "test-req",
					User: &openrtb2.User{
						ID: "user123",
					},
					Device: &openrtb2.Device{
						UA: "Mozilla/5.0",
						IP: "192.168.1.1",
						IFA: "device-ifa-123",
					},
				},
			},
			expectError:      false,
			expectUserData:   true,
			expectDeviceData: true,
			expectWarnings:   true,
		},
		{
			name: "enrichment_with_user_only",
			request: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID: "test-req",
					User: &openrtb2.User{
						ID: "user123",
					},
				},
			},
			expectError:      false,
			expectUserData:   true,
			expectDeviceData: false,
			expectWarnings:   true,
		},
		{
			name: "enrichment_without_user_or_device",
			request: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID: "test-req",
				},
			},
			expectError:      false,
			expectUserData:   false,
			expectDeviceData: false,
			expectWarnings:   true,
		},
		{
			name:           "nil_request",
			request:        nil,
			expectError:    false,
			expectWarnings: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewIdentityEnrichmentFilter(true, 10)

			req := filterpipeline.PreFilterRequest{
				Request: tt.request,
				Context: filterpipeline.FilterContext{
					AccountID: "test-account",
					Endpoint:  "/openrtb2/auction",
				},
			}

			resp, err := filter.Execute(context.Background(), req)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)

				if tt.request != nil && tt.request.BidRequest != nil {
					// Check metadata
					assert.NotNil(t, resp.Metadata)
					identityData, ok := resp.Metadata["identity"].(map[string]interface{})
					assert.True(t, ok)
					assert.True(t, identityData["enriched"].(bool))

					if tt.expectUserData {
						assert.True(t, identityData["has_user"].(bool))
						if tt.request.BidRequest.User.ID != "" {
							assert.Equal(t, "user123", identityData["user_id"])
						}
					} else {
						assert.False(t, identityData["has_user"].(bool))
					}

					if tt.expectDeviceData {
						deviceInfo, ok := identityData["device"].(map[string]interface{})
						assert.True(t, ok)
						assert.True(t, deviceInfo["has_ua"].(bool))
						assert.True(t, deviceInfo["has_ip"].(bool))
						assert.True(t, deviceInfo["has_ifa"].(bool))
					}

					if tt.expectWarnings {
						assert.NotEmpty(t, resp.Warnings)
					}
				} else {
					assert.NotEmpty(t, resp.Errors)
				}
			}
		})
	}
}
