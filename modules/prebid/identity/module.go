// Package identity implements a Prebid Server module for Identity Resolution
package identity

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"net/http"
	"sync"
	"time"

	"github.com/coocood/freecache"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/hooks/hookanalytics"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/moduledeps"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

// Builder is the entry point for the module
func Builder(config json.RawMessage, deps moduledeps.ModuleDeps) (interface{}, error) {
	var cfg Config
	if err := jsonutil.Unmarshal(config, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Set defaults
	if cfg.Timeout == 0 {
		cfg.Timeout = 500 // 500ms default timeout for identity resolution
	}
	if cfg.CacheTTL == 0 {
		cfg.CacheTTL = 3600 // 1 hour default cache
	}
	if cfg.CacheSize == 0 {
		cfg.CacheSize = 50 * 1024 * 1024 // 50MB default cache size
	}

	// Validate provider configuration
	if cfg.Provider == "" {
		return nil, errors.New("identity provider must be specified (liveramp, uid2, id5, custom)")
	}

	// Validate endpoint for custom provider
	if cfg.Provider == "custom" && cfg.Endpoint == "" {
		return nil, errors.New("endpoint is required for custom identity provider")
	}

	// Set default endpoints for known providers
	switch cfg.Provider {
	case "liveramp":
		if cfg.Endpoint == "" {
			cfg.Endpoint = "https://api.liveramp.com/identity/v1/resolve"
		}
	case "uid2":
		if cfg.Endpoint == "" {
			cfg.Endpoint = "https://prod.uidapi.com/v2/identity/map"
		}
	case "id5":
		if cfg.Endpoint == "" {
			cfg.Endpoint = "https://id5-sync.com/api/v1/id"
		}
	}

	return &Module{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout:   time.Duration(cfg.Timeout) * time.Millisecond,
			Transport: deps.HTTPClient.Transport,
		},
		cache: freecache.NewCache(cfg.CacheSize),
		sha256Pool: &sync.Pool{
			New: func() any {
				return sha256.New()
			},
		},
	}, nil
}

const (
	asyncRequestKey = "identity.AsyncRequest"
)

var (
	// Declare hooks
	_ hookstage.Entrypoint        = (*Module)(nil)
	_ hookstage.RawAuctionRequest = (*Module)(nil)
	_ hookstage.BidderRequest     = (*Module)(nil)
)

// Config holds module configuration
type Config struct {
	Provider  string `json:"provider"`           // liveramp, uid2, id5, custom
	Endpoint  string `json:"endpoint"`           // API endpoint
	APIKey    string `json:"api_key"`            // API key for authentication
	Timeout   int    `json:"timeout_ms"`         // Request timeout
	CacheTTL  int    `json:"cache_ttl_seconds"`  // Cache TTL
	CacheSize int    `json:"cache_size"`         // Cache size in bytes
	Enabled   bool   `json:"enabled"`            // Enable/disable the module
}

// Module implements the Identity Resolution module
type Module struct {
	cfg        Config
	httpClient *http.Client
	cache      *freecache.Cache
	sha256Pool *sync.Pool
}

// IdentityResponse represents the response from an identity provider
type IdentityResponse struct {
	ResolvedIDs map[string]string `json:"resolved_ids"` // map of source -> ID
	EIDS        []openrtb2.EID    `json:"eids"`         // Extended IDs in OpenRTB format
}

// HandleEntrypointHook initializes the module context
func (m *Module) HandleEntrypointHook(
	ctx context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.EntrypointPayload,
) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	if !m.cfg.Enabled {
		return hookstage.HookResult[hookstage.EntrypointPayload]{}, nil
	}

	return hookstage.HookResult[hookstage.EntrypointPayload]{
		ModuleContext: hookstage.ModuleContext{
			asyncRequestKey: m.NewAsyncRequest(payload.Request),
		},
	}, nil
}

// HandleRawAuctionHook fetches identity resolution data
func (m *Module) HandleRawAuctionHook(
	ctx context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.RawAuctionRequestPayload,
) (hookstage.HookResult[hookstage.RawAuctionRequestPayload], error) {
	var ret hookstage.HookResult[hookstage.RawAuctionRequestPayload]

	if !m.cfg.Enabled {
		return ret, nil
	}

	analyticsNamePrefix := "HandleRawAuctionHook."

	asyncRequest, ok := miCtx.ModuleContext[asyncRequestKey].(*AsyncRequest)
	if !ok {
		ret.AnalyticsTags = hookanalytics.Analytics{
			Activities: []hookanalytics.Activity{{
				Name:   analyticsNamePrefix + asyncRequestKey,
				Status: hookanalytics.ActivityStatusError,
				Results: []hookanalytics.Result{{
					Status: hookanalytics.ResultStatusError,
					Values: map[string]interface{}{"error": "failed to get async request from module context"},
				}},
			}},
		}
		return ret, nil
	}

	// Parse OpenRTB request
	var bidRequest openrtb2.BidRequest
	if err := jsonutil.Unmarshal(payload, &bidRequest); err != nil {
		ret.AnalyticsTags = hookanalytics.Analytics{
			Activities: []hookanalytics.Activity{{
				Name:   analyticsNamePrefix + "bidRequest.unmarshal",
				Status: hookanalytics.ActivityStatusError,
				Results: []hookanalytics.Result{{
					Status: hookanalytics.ResultStatusError,
					Values: map[string]interface{}{"error": err.Error()},
				}},
			}},
		}
		return ret, nil
	}

	// Start async identity resolution
	asyncRequest.fetchIdentityAsync(&bidRequest)

	return ret, nil
}

// HandleBidderRequestHook enriches bidder requests with resolved identities
func (m *Module) HandleBidderRequestHook(
	ctx context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.BidderRequestPayload,
) (hookstage.HookResult[hookstage.BidderRequestPayload], error) {
	var ret hookstage.HookResult[hookstage.BidderRequestPayload]

	if !m.cfg.Enabled {
		return ret, nil
	}

	analyticsNamePrefix := "HandleBidderRequestHook."
	asyncRequest, ok := miCtx.ModuleContext[asyncRequestKey].(*AsyncRequest)
	if !ok {
		ret.AnalyticsTags = hookanalytics.Analytics{
			Activities: []hookanalytics.Activity{{
				Name:   analyticsNamePrefix + asyncRequestKey,
				Status: hookanalytics.ActivityStatusError,
				Results: []hookanalytics.Result{{
					Status: hookanalytics.ResultStatusError,
					Values: map[string]interface{}{"error": "failed to get async request from module context"},
				}},
			}},
		}
		return ret, nil
	}

	// Ensure we cancel the request context to free resources
	defer asyncRequest.Cancel()

	// Check if a request was made
	if asyncRequest.Done == nil {
		return ret, nil
	}

	// Wait for the async request to complete
	select {
	case <-asyncRequest.Done:
		// Continue with processing
	case <-ctx.Done():
		return ret, nil // Context cancelled, exit gracefully
	}

	// Get results
	identityResp, err := asyncRequest.IdentityResponse, asyncRequest.Err
	if err != nil {
		ret.AnalyticsTags = hookanalytics.Analytics{
			Activities: []hookanalytics.Activity{{
				Name:   analyticsNamePrefix + "identity_fetch",
				Status: hookanalytics.ActivityStatusError,
				Results: []hookanalytics.Result{{
					Status: hookanalytics.ResultStatusError,
					Values: map[string]interface{}{"error": err.Error()},
				}},
			}},
		}
		return ret, nil
	}

	if identityResp == nil || len(identityResp.EIDS) == 0 {
		return ret, nil
	}

	// Enrich the bidder request with resolved identities
	ret.ChangeSet.AddMutation(
		func(payload hookstage.BidderRequestPayload) (hookstage.BidderRequestPayload, error) {
			if payload.Request.User == nil {
				payload.Request.User = &openrtb2.User{}
			}

			// Add EIDs to user object
			if payload.Request.User.EIDs == nil {
				payload.Request.User.EIDs = identityResp.EIDS
			} else {
				// Merge with existing EIDs
				payload.Request.User.EIDs = append(payload.Request.User.EIDs, identityResp.EIDS...)
			}

			return payload, nil
		},
		hookstage.MutationUpdate,
		"user.eids",
	)

	ret.AnalyticsTags = hookanalytics.Analytics{
		Activities: []hookanalytics.Activity{{
			Name:   analyticsNamePrefix + "identity_enrichment",
			Status: hookanalytics.ActivityStatusSuccess,
			Results: []hookanalytics.Result{{
				Status: hookanalytics.ResultStatusModify,
				Values: map[string]interface{}{
					"eids_added": len(identityResp.EIDS),
				},
			}},
		}},
	}

	return ret, nil
}

// fetchIdentity calls the identity provider API and retrieves resolved IDs
func (m *Module) fetchIdentity(ctx context.Context, bidRequest *openrtb2.BidRequest) (*IdentityResponse, error) {
	// Extract user identifiers for lookup
	if bidRequest.User == nil {
		return nil, errors.New("no user object in bid request")
	}

	// Create cache key based on user identifiers
	cacheKey := []byte(m.createCacheKey(bidRequest))

	// Check cache first
	if cachedData, err := m.cache.Get(cacheKey); err == nil {
		var identityResp IdentityResponse
		if err := json.Unmarshal(cachedData, &identityResp); err == nil {
			return &identityResp, nil
		}
	}

	// Prepare request to identity provider
	requestBody := m.prepareIdentityRequest(bidRequest)
	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", m.cfg.Endpoint, bytes.NewReader(requestJSON))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if m.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+m.cfg.APIKey)
	}

	// Make the request
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("identity provider returned status %d", resp.StatusCode)
	}

	// Parse response
	var identityResp IdentityResponse
	if err = json.NewDecoder(resp.Body).Decode(&identityResp); err != nil {
		return nil, err
	}

	// Cache the result
	cachedData, err := json.Marshal(identityResp)
	if err == nil {
		_ = m.cache.Set(cacheKey, cachedData, m.cfg.CacheTTL)
	}

	return &identityResp, nil
}

// prepareIdentityRequest prepares the request payload for the identity provider
func (m *Module) prepareIdentityRequest(bidRequest *openrtb2.BidRequest) map[string]interface{} {
	request := make(map[string]interface{})

	if bidRequest.User != nil {
		if bidRequest.User.ID != "" {
			request["user_id"] = bidRequest.User.ID
		}
		if bidRequest.User.BuyerUID != "" {
			request["buyer_uid"] = bidRequest.User.BuyerUID
		}

		// Include existing EIDs for identity graph resolution
		if len(bidRequest.User.EIDs) > 0 {
			request["eids"] = bidRequest.User.EIDs
		}
	}

	// Include device information for mobile ID resolution
	if bidRequest.Device != nil {
		if bidRequest.Device.IFA != "" {
			request["ifa"] = bidRequest.Device.IFA
		}
		if bidRequest.Device.DPIDMD5 != "" {
			request["dpid_md5"] = bidRequest.Device.DPIDMD5
		}
		if bidRequest.Device.DPIDSHA1 != "" {
			request["dpid_sha1"] = bidRequest.Device.DPIDSHA1
		}
	}

	return request
}

// createCacheKey generates a cache key based on user identifiers
func (m *Module) createCacheKey(bidRequest *openrtb2.BidRequest) string {
	hasher := m.sha256Pool.Get().(hash.Hash)
	hasher.Reset()
	defer m.sha256Pool.Put(hasher)

	hasher.Write([]byte("provider:" + m.cfg.Provider))

	if bidRequest.User != nil {
		if bidRequest.User.ID != "" {
			hasher.Write([]byte("user_id:" + bidRequest.User.ID))
		}
		if bidRequest.User.BuyerUID != "" {
			hasher.Write([]byte("buyer_uid:" + bidRequest.User.BuyerUID))
		}
	}

	if bidRequest.Device != nil && bidRequest.Device.IFA != "" {
		hasher.Write([]byte("ifa:" + bidRequest.Device.IFA))
	}

	return hex.EncodeToString(hasher.Sum(nil))
}
