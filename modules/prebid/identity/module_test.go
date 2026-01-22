package identity

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/moduledeps"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuilder(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid_liveramp_config",
			config: `{
				"provider": "liveramp",
				"api_key": "test-key",
				"enabled": true
			}`,
			expectError: false,
		},
		{
			name: "valid_uid2_config",
			config: `{
				"provider": "uid2",
				"api_key": "test-key",
				"enabled": true
			}`,
			expectError: false,
		},
		{
			name: "valid_custom_config",
			config: `{
				"provider": "custom",
				"endpoint": "https://custom-identity.com/resolve",
				"api_key": "test-key",
				"enabled": true
			}`,
			expectError: false,
		},
		{
			name: "missing_provider",
			config: `{
				"api_key": "test-key",
				"enabled": true
			}`,
			expectError: true,
			errorMsg:    "identity provider must be specified",
		},
		{
			name: "custom_provider_missing_endpoint",
			config: `{
				"provider": "custom",
				"api_key": "test-key",
				"enabled": true
			}`,
			expectError: true,
			errorMsg:    "endpoint is required for custom identity provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := moduledeps.ModuleDeps{
				HTTPClient: &http.Client{},
			}

			module, err := Builder(json.RawMessage(tt.config), deps)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, module)
			}
		})
	}
}

func TestHandleEntrypointHook(t *testing.T) {
	module := &Module{
		cfg: Config{Enabled: true},
	}

	req := httptest.NewRequest("POST", "/openrtb2/auction", nil)
	payload := hookstage.EntrypointPayload{Request: req}
	miCtx := hookstage.ModuleInvocationContext{}

	result, err := module.HandleEntrypointHook(context.Background(), miCtx, payload)

	assert.NoError(t, err)
	assert.NotNil(t, result.ModuleContext[asyncRequestKey])
}

func TestHandleRawAuctionHook(t *testing.T) {
	// Create a test server that returns identity data
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := IdentityResponse{
			ResolvedIDs: map[string]string{
				"liveramp": "test-ramp-id",
			},
			EIDS: []openrtb2.EID{
				{
					Source: "liveramp.com",
					UIDs: []openrtb2.UID{
						{ID: "test-ramp-id"},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	deps := moduledeps.ModuleDeps{
		HTTPClient: &http.Client{},
	}

	config := Config{
		Provider: "custom",
		Endpoint: server.URL,
		APIKey:   "test-key",
		Enabled:  true,
		Timeout:  1000,
	}

	configJSON, _ := json.Marshal(config)
	moduleInterface, err := Builder(configJSON, deps)
	require.NoError(t, err)

	module := moduleInterface.(*Module)

	// Create test bid request
	bidRequest := openrtb2.BidRequest{
		ID: "test-auction",
		User: &openrtb2.User{
			ID: "test-user-123",
		},
	}
	payload, _ := json.Marshal(bidRequest)

	req := httptest.NewRequest("POST", "/openrtb2/auction", nil)
	asyncReq := module.NewAsyncRequest(req)

	miCtx := hookstage.ModuleInvocationContext{
		ModuleContext: hookstage.ModuleContext{
			asyncRequestKey: asyncReq,
		},
	}

	result, err := module.HandleRawAuctionHook(context.Background(), miCtx, payload)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestHandleBidderRequestHook(t *testing.T) {
	module := &Module{
		cfg: Config{Enabled: true},
	}

	// Create async request with identity response
	req := httptest.NewRequest("POST", "/openrtb2/auction", nil)
	asyncReq := module.NewAsyncRequest(req)
	asyncReq.Done = make(chan struct{})
	asyncReq.IdentityResponse = &IdentityResponse{
		EIDS: []openrtb2.EID{
			{
				Source: "liveramp.com",
				UIDs: []openrtb2.UID{
					{ID: "test-ramp-id"},
				},
			},
		},
	}
	close(asyncReq.Done)

	bidRequest := openrtb2.BidRequest{
		ID: "test-auction",
		User: &openrtb2.User{
			ID: "test-user",
		},
	}

	reqWrapper := &openrtb_ext.RequestWrapper{BidRequest: &bidRequest}

	payload := hookstage.BidderRequestPayload{
		Request: reqWrapper,
	}

	miCtx := hookstage.ModuleInvocationContext{
		ModuleContext: hookstage.ModuleContext{
			asyncRequestKey: asyncReq,
		},
	}

	result, err := module.HandleBidderRequestHook(context.Background(), miCtx, payload)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, len(result.AnalyticsTags.Activities))
	assert.Equal(t, "HandleBidderRequestHook.identity_enrichment", result.AnalyticsTags.Activities[0].Name)
}

func TestFetchIdentity(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

		response := IdentityResponse{
			ResolvedIDs: map[string]string{
				"liveramp": "resolved-ramp-id",
			},
			EIDS: []openrtb2.EID{
				{
					Source: "liveramp.com",
					UIDs: []openrtb2.UID{
						{ID: "resolved-ramp-id"},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	deps := moduledeps.ModuleDeps{
		HTTPClient: &http.Client{},
	}

	config := Config{
		Provider: "custom",
		Endpoint: server.URL,
		APIKey:   "test-api-key",
		Enabled:  true,
		Timeout:  1000,
	}

	configJSON, _ := json.Marshal(config)
	moduleInterface, err := Builder(configJSON, deps)
	require.NoError(t, err)

	module := moduleInterface.(*Module)

	bidRequest := &openrtb2.BidRequest{
		User: &openrtb2.User{
			ID: "test-user-123",
		},
	}

	identityResp, err := module.fetchIdentity(context.Background(), bidRequest)

	assert.NoError(t, err)
	assert.NotNil(t, identityResp)
	assert.Equal(t, "resolved-ramp-id", identityResp.ResolvedIDs["liveramp"])
	assert.Equal(t, 1, len(identityResp.EIDS))
	assert.Equal(t, "liveramp.com", identityResp.EIDS[0].Source)
}

func TestFetchIdentityWithCache(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		response := IdentityResponse{
			EIDS: []openrtb2.EID{
				{Source: "liveramp.com", UIDs: []openrtb2.UID{{ID: "test-id"}}},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	deps := moduledeps.ModuleDeps{
		HTTPClient: &http.Client{},
	}

	config := Config{
		Provider:  "custom",
		Endpoint:  server.URL,
		APIKey:    "test-key",
		Enabled:   true,
		Timeout:   1000,
		CacheTTL:  60,
		CacheSize: 1024 * 1024,
	}

	configJSON, _ := json.Marshal(config)
	moduleInterface, err := Builder(configJSON, deps)
	require.NoError(t, err)

	module := moduleInterface.(*Module)

	bidRequest := &openrtb2.BidRequest{
		User: &openrtb2.User{ID: "test-user"},
	}

	// First call - should hit the API
	_, err = module.fetchIdentity(context.Background(), bidRequest)
	assert.NoError(t, err)
	assert.Equal(t, 1, callCount)

	// Second call - should use cache
	_, err = module.fetchIdentity(context.Background(), bidRequest)
	assert.NoError(t, err)
	assert.Equal(t, 1, callCount, "Should not have made a second API call")
}

func TestFetchIdentityTimeout(t *testing.T) {
	// Create a server that never responds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {}
	}))
	defer server.Close()

	deps := moduledeps.ModuleDeps{
		HTTPClient: &http.Client{},
	}

	config := Config{
		Provider: "custom",
		Endpoint: server.URL,
		APIKey:   "test-key",
		Enabled:  true,
		Timeout:  10, // 10ms timeout
	}

	configJSON, _ := json.Marshal(config)
	moduleInterface, err := Builder(configJSON, deps)
	require.NoError(t, err)

	module := moduleInterface.(*Module)

	bidRequest := &openrtb2.BidRequest{
		User: &openrtb2.User{ID: "test-user"},
	}

	_, err = module.fetchIdentity(context.Background(), bidRequest)
	assert.Error(t, err)
}
