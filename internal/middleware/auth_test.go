package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestAuthMiddlewareDisabled(t *testing.T) {
	auth := NewAuth(&AuthConfig{
		Enabled: false,
	})

	handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 when auth disabled, got %d", rec.Code)
	}
}

func TestAuthMiddlewareMissingKey(t *testing.T) {
	auth := NewAuth(&AuthConfig{
		Enabled:    true,
		APIKeys:    map[string]string{"valid-key": "pub1"},
		HeaderName: "X-API-Key",
	})

	handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for missing key, got %d", rec.Code)
	}
}

func TestAuthMiddlewareInvalidKey(t *testing.T) {
	auth := NewAuth(&AuthConfig{
		Enabled:    true,
		APIKeys:    map[string]string{"valid-key": "pub1"},
		HeaderName: "X-API-Key",
	})

	handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "invalid-key")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for invalid key, got %d", rec.Code)
	}
}

func TestAuthMiddlewareValidKey(t *testing.T) {
	auth := NewAuth(&AuthConfig{
		Enabled:    true,
		APIKeys:    map[string]string{"valid-key": "pub1"},
		HeaderName: "X-API-Key",
	})

	var gotPublisherID string
	handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPublisherID = PublisherIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "valid-key")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for valid key, got %d", rec.Code)
	}

	if gotPublisherID != "pub1" {
		t.Errorf("expected publisher ID from context 'pub1', got '%s'", gotPublisherID)
	}
}

func TestAuthMiddlewareBearerToken(t *testing.T) {
	auth := NewAuth(&AuthConfig{
		Enabled:    true,
		APIKeys:    map[string]string{"bearer-token": "pub2"},
		HeaderName: "X-API-Key",
	})

	handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer bearer-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for Bearer token, got %d", rec.Code)
	}
}

func TestAuthMiddlewareBypassPaths(t *testing.T) {
	auth := NewAuth(&AuthConfig{
		Enabled:     true,
		APIKeys:     map[string]string{"key": "pub"},
		HeaderName:  "X-API-Key",
		BypassPaths: []string{"/health", "/metrics"},
	})

	handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		path     string
		wantCode int
	}{
		{"/health", http.StatusOK},
		{"/health/live", http.StatusOK},
		{"/metrics", http.StatusOK},
		{"/api/test", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != tt.wantCode {
			t.Errorf("path %s: expected %d, got %d", tt.path, tt.wantCode, rec.Code)
		}
	}
}

func TestAuthAddRemoveAPIKey(t *testing.T) {
	auth := NewAuth(&AuthConfig{
		Enabled:    true,
		APIKeys:    map[string]string{},
		HeaderName: "X-API-Key",
	})

	ctx := context.Background()

	// Initially no keys
	_, valid := auth.GetPublisherID(ctx, "new-key")
	if valid {
		t.Error("expected key to be invalid before adding")
	}

	// Add key
	auth.AddAPIKey("new-key", "pub-new")

	pubID, valid := auth.GetPublisherID(ctx, "new-key")
	if !valid {
		t.Error("expected key to be valid after adding")
	}
	if pubID != "pub-new" {
		t.Errorf("expected pub-new, got %s", pubID)
	}

	// Remove key
	auth.RemoveAPIKey("new-key")

	_, valid = auth.GetPublisherID(ctx, "new-key")
	if valid {
		t.Error("expected key to be invalid after removing")
	}
}

func TestParseAPIKeys(t *testing.T) {
	tests := []struct {
		input    string
		expected map[string]string
	}{
		{"", map[string]string{}},
		{"key1:pub1", map[string]string{"key1": "pub1"}},
		{"key1:pub1,key2:pub2", map[string]string{"key1": "pub1", "key2": "pub2"}},
		{"key1", map[string]string{"key1": "default"}},
		{" key1 : pub1 , key2 : pub2 ", map[string]string{"key1": "pub1", "key2": "pub2"}},
	}

	for _, tt := range tests {
		result := parseAPIKeys(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("input %q: expected %d keys, got %d", tt.input, len(tt.expected), len(result))
			continue
		}
		for k, v := range tt.expected {
			if result[k] != v {
				t.Errorf("input %q: expected %s=%s, got %s", tt.input, k, v, result[k])
			}
		}
	}
}

// Additional tests for full coverage

func TestDefaultAuthConfig(t *testing.T) {
	// Clear environment
	os.Unsetenv("AUTH_ENABLED")
	os.Unsetenv("API_KEYS")
	os.Unsetenv("REDIS_URL")
	os.Unsetenv("AUTH_USE_REDIS")

	config := DefaultAuthConfig()

	if config == nil {
		t.Fatal("Expected config to be created")
	}

	// SECURITY: Auth should be ENABLED by default (secure by default)
	if !config.Enabled {
		t.Error("Expected auth to be ENABLED by default when AUTH_ENABLED not set (secure by default)")
	}

	if config.HeaderName != "X-API-Key" {
		t.Errorf("Expected default header name X-API-Key, got %s", config.HeaderName)
	}

	// Verify bypass paths - note: /openrtb2/auction is NOT in default list
	// It's conditionally added at runtime in cmd/server/main.go based on
	// whether PublisherAuth is enabled (see commit d61640d)
	// SECURITY: /metrics and /admin/* endpoints removed from bypass (CVE-2026-XXXX)
	expectedBypass := []string{"/health", "/status", "/info/bidders", "/cookie_sync", "/setuid", "/optout"}
	if len(config.BypassPaths) != len(expectedBypass) {
		t.Errorf("Expected %d bypass paths, got %d", len(expectedBypass), len(config.BypassPaths))
	}
}

func TestDefaultAuthConfig_Enabled(t *testing.T) {
	os.Setenv("AUTH_ENABLED", "true")
	defer os.Unsetenv("AUTH_ENABLED")

	config := DefaultAuthConfig()

	if !config.Enabled {
		t.Error("Expected auth to be enabled when AUTH_ENABLED=true")
	}
}

func TestDefaultAuthConfig_ExplicitlyDisabled(t *testing.T) {
	os.Setenv("AUTH_ENABLED", "false")
	defer os.Unsetenv("AUTH_ENABLED")

	config := DefaultAuthConfig()

	if config.Enabled {
		t.Error("Expected auth to be disabled when AUTH_ENABLED=false")
	}
}

func TestDefaultAuthConfig_SecureByDefault(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		setEnv   bool
		expected bool
	}{
		{
			name:     "not set - should be enabled (secure by default)",
			setEnv:   false,
			expected: true,
		},
		{
			name:     "explicitly true - should be enabled",
			envValue: "true",
			setEnv:   true,
			expected: true,
		},
		{
			name:     "explicitly false - should be disabled",
			envValue: "false",
			setEnv:   true,
			expected: false,
		},
		{
			name:     "invalid value - should be enabled (secure by default)",
			envValue: "invalid",
			setEnv:   true,
			expected: true,
		},
		{
			name:     "empty string - should be enabled (secure by default)",
			envValue: "",
			setEnv:   true,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				os.Setenv("AUTH_ENABLED", tt.envValue)
			} else {
				os.Unsetenv("AUTH_ENABLED")
			}
			defer os.Unsetenv("AUTH_ENABLED")

			config := DefaultAuthConfig()

			if config.Enabled != tt.expected {
				t.Errorf("Expected Enabled=%v, got %v", tt.expected, config.Enabled)
			}
		})
	}
}

func TestDefaultAuthConfig_WithAPIKeys(t *testing.T) {
	os.Setenv("API_KEYS", "key1:pub1,key2:pub2")
	defer os.Unsetenv("API_KEYS")

	config := DefaultAuthConfig()

	if len(config.APIKeys) != 2 {
		t.Errorf("Expected 2 API keys, got %d", len(config.APIKeys))
	}

	if config.APIKeys["key1"] != "pub1" {
		t.Errorf("Expected key1 -> pub1, got %s", config.APIKeys["key1"])
	}
}

func TestDefaultAuthConfig_WithRedis(t *testing.T) {
	os.Setenv("REDIS_URL", "redis://localhost:6379")
	defer os.Unsetenv("REDIS_URL")

	config := DefaultAuthConfig()

	if config.RedisURL != "redis://localhost:6379" {
		t.Errorf("Expected Redis URL to be set, got %s", config.RedisURL)
	}

	if !config.UseRedis {
		t.Error("Expected UseRedis to be true when REDIS_URL is set")
	}
}

func TestDefaultAuthConfig_RedisDisabled(t *testing.T) {
	os.Setenv("REDIS_URL", "redis://localhost:6379")
	os.Setenv("AUTH_USE_REDIS", "false")
	defer os.Unsetenv("REDIS_URL")
	defer os.Unsetenv("AUTH_USE_REDIS")

	config := DefaultAuthConfig()

	if config.UseRedis {
		t.Error("Expected UseRedis to be false when AUTH_USE_REDIS=false")
	}
}

func TestNewAuth_NilConfig(t *testing.T) {
	auth := NewAuth(nil)

	if auth == nil {
		t.Fatal("Expected auth to be created with default config")
	}

	if auth.config == nil {
		t.Error("Expected default config to be set")
	}

	if auth.keyCache == nil {
		t.Error("Expected cache to be initialized")
	}
}

func TestNewAuthWithRedis(t *testing.T) {
	mockRedis := &mockRedisClient{
		data: make(map[string]map[string]string),
	}

	config := &AuthConfig{
		Enabled: true,
		APIKeys: map[string]string{"key1": "pub1"},
	}

	auth := NewAuthWithRedis(config, mockRedis)

	if auth == nil {
		t.Fatal("Expected auth to be created")
	}

	if auth.redisClient == nil {
		t.Error("Expected Redis client to be set")
	}
}

func TestSetRedisClient_Auth(t *testing.T) {
	auth := NewAuth(&AuthConfig{Enabled: true})

	mockRedis := &mockRedisClient{
		data: make(map[string]map[string]string),
	}

	auth.SetRedisClient(mockRedis)

	if auth.redisClient == nil {
		t.Error("Expected Redis client to be set")
	}
}

func TestClearCache(t *testing.T) {
	auth := NewAuth(&AuthConfig{
		Enabled: true,
		APIKeys: map[string]string{"key1": "pub1"},
	})

	// Add something to cache
	auth.updateCache("key1", "pub1")

	// Verify cache has entry
	pubID, found := auth.checkCache("key1")
	if !found || pubID != "pub1" {
		t.Error("Expected cache to have entry")
	}

	// Clear cache
	auth.ClearCache()

	// Verify cache is empty
	_, found = auth.checkCache("key1")
	if found {
		t.Error("Expected cache to be cleared")
	}
}

func TestSetEnabled_Auth(t *testing.T) {
	auth := NewAuth(&AuthConfig{Enabled: true})

	auth.SetEnabled(false)

	if auth.config.Enabled {
		t.Error("Expected auth to be disabled")
	}

	auth.SetEnabled(true)

	if !auth.config.Enabled {
		t.Error("Expected auth to be enabled")
	}
}

func TestIsEnabled(t *testing.T) {
	auth := NewAuth(&AuthConfig{Enabled: true})

	if !auth.IsEnabled() {
		t.Error("Expected IsEnabled to return true")
	}

	auth.SetEnabled(false)

	if auth.IsEnabled() {
		t.Error("Expected IsEnabled to return false")
	}
}

func TestSetMetrics(t *testing.T) {
	auth := NewAuth(&AuthConfig{Enabled: true})

	mockMetrics := &mockAuthMetrics{}

	auth.SetMetrics(mockMetrics)

	if auth.metrics == nil {
		t.Error("Expected metrics to be set")
	}
}

type mockAuthMetrics struct {
	failureCount int
}

func (m *mockAuthMetrics) IncAuthFailures() {
	m.failureCount++
}

func TestRecordAuthFailure_WithMetrics(t *testing.T) {
	auth := NewAuth(&AuthConfig{Enabled: true})
	mockMetrics := &mockAuthMetrics{}
	auth.SetMetrics(mockMetrics)

	auth.recordAuthFailure()

	if mockMetrics.failureCount != 1 {
		t.Errorf("Expected 1 failure recorded, got %d", mockMetrics.failureCount)
	}
}

func TestRecordAuthFailure_NoMetrics(t *testing.T) {
	auth := NewAuth(&AuthConfig{Enabled: true})

	// Should not panic when metrics is nil
	auth.recordAuthFailure()
}

func TestCheckCache_Expired(t *testing.T) {
	auth := NewAuth(&AuthConfig{Enabled: true})

	// Add entry to cache with past expiration
	auth.cacheMu.Lock()
	auth.keyCache["key1"] = cachedKey{
		publisherID: "pub1",
		expiresAt:   time.Now().Add(-1 * time.Hour), // Expired
	}
	auth.cacheMu.Unlock()

	// Should not find expired entry
	_, found := auth.checkCache("key1")
	if found {
		t.Error("Expected expired cache entry to not be found")
	}
}

func TestValidateKey_NegativeCaching(t *testing.T) {
	auth := NewAuth(&AuthConfig{
		Enabled: true,
		APIKeys: map[string]string{},
	})

	// Validate invalid key
	pubID, valid := auth.validateKey(context.Background(), "invalid_key")
	if valid || pubID != "" {
		t.Error("Expected invalid key to fail")
	}

	// Check that negative result is cached
	auth.cacheMu.RLock()
	cached, exists := auth.keyCache["invalid_key"]
	auth.cacheMu.RUnlock()

	if !exists {
		t.Error("Expected negative result to be cached")
	}

	if cached.publisherID != "" {
		t.Error("Expected cached publisher ID to be empty for negative result")
	}
}

func TestValidateKey_WithRedis(t *testing.T) {
	mockRedis := &mockRedisClient{
		data: map[string]map[string]string{
			RedisAPIKeysHash: {
				"redis_key": "redis_pub",
			},
		},
	}

	auth := NewAuthWithRedis(&AuthConfig{
		Enabled:  true,
		UseRedis: true,
	}, mockRedis)

	pubID, valid := auth.validateKey(context.Background(), "redis_key")

	if !valid {
		t.Error("Expected Redis key to be valid")
	}

	if pubID != "redis_pub" {
		t.Errorf("Expected publisher ID redis_pub, got %s", pubID)
	}

	// Verify it's now cached
	cachedPubID, found := auth.checkCache("redis_key")
	if !found || cachedPubID != "redis_pub" {
		t.Error("Expected Redis result to be cached")
	}
}

func TestAddAPIKey_NilMap(t *testing.T) {
	auth := NewAuth(&AuthConfig{
		Enabled: true,
		APIKeys: nil, // nil map
	})

	// Should create map and add key
	auth.AddAPIKey("new_key", "new_pub")

	if auth.config.APIKeys == nil {
		t.Error("Expected APIKeys map to be created")
	}

	if auth.config.APIKeys["new_key"] != "new_pub" {
		t.Error("Expected key to be added")
	}
}

func TestGetPublisherID(t *testing.T) {
	auth := NewAuth(&AuthConfig{
		Enabled: true,
		APIKeys: map[string]string{"key1": "pub1"},
	})

	pubID, valid := auth.GetPublisherID(context.Background(), "key1")

	if !valid {
		t.Error("Expected key to be valid")
	}

	if pubID != "pub1" {
		t.Errorf("Expected publisher ID pub1, got %s", pubID)
	}
}

func TestMiddleware_MetricsOnFailure(t *testing.T) {
	mockMetrics := &mockAuthMetrics{}
	auth := NewAuth(&AuthConfig{
		Enabled: true,
		APIKeys: map[string]string{},
	})
	auth.SetMetrics(mockMetrics)

	handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Request without API key
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if mockMetrics.failureCount != 1 {
		t.Errorf("Expected 1 auth failure metric, got %d", mockMetrics.failureCount)
	}

	// Request with invalid API key
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "invalid")
	w = httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if mockMetrics.failureCount != 2 {
		t.Errorf("Expected 2 auth failure metrics, got %d", mockMetrics.failureCount)
	}
}
