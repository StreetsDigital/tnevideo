// Package middleware provides HTTP middleware for PBS
package middleware

import (
	"context"
	"crypto/subtle"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	pbsconfig "github.com/thenexusengine/tne_springwire/internal/config"
)

// Context key for storing publisher ID (raw string for cross-package compatibility)
const publisherIDKey = "publisher_id"

// Redis key patterns (must match IDR's api_keys.py)
const (
	// #nosec G101 -- Redis key name, not a credential
	RedisAPIKeysHash = "tne_catalyst:api_keys" // hash: api_key -> publisher_id
)

// RedisClient interface for API key validation
type RedisClient interface {
	HGet(ctx context.Context, key, field string) (string, error)
	Ping(ctx context.Context) error
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Enabled     bool
	APIKeys     map[string]string // key -> publisher ID mapping (local fallback)
	HeaderName  string            // Header to check for API key (default: X-API-Key)
	BypassPaths []string          // Paths that don't require auth (e.g., /health, /status)
	RedisURL    string            // Redis URL for shared API keys
	UseRedis    bool              // Whether to use Redis for API key validation
}

// DefaultAuthConfig returns default auth configuration
func DefaultAuthConfig() *AuthConfig {
	redisURL := os.Getenv("REDIS_URL")
	return &AuthConfig{
		// SECURITY: Auth is ENABLED by default (secure by default)
		// Set AUTH_ENABLED=false to explicitly disable (not recommended for production)
		Enabled:     os.Getenv("AUTH_ENABLED") != "false",
		APIKeys:     parseAPIKeys(os.Getenv("API_KEYS")),
		HeaderName:  "X-API-Key",
		// SECURITY: /metrics and /admin/* endpoints now require authentication
		// Removed /metrics, /admin/dashboard, /admin/metrics from bypass list (CVE-2026-XXXX)
		BypassPaths: []string{"/health", "/status", "/info/bidders", "/cookie_sync", "/setuid", "/optout"},
		// Note: /openrtb2/auction is conditionally added to bypass list in cmd/server/main.go
		// based on whether PublisherAuth is enabled (primary auth) or disabled (fallback to API key)
		RedisURL: redisURL,
		UseRedis: redisURL != "" && os.Getenv("AUTH_USE_REDIS") != "false",
	}
}

// parseAPIKeys parses API keys from env var format: "key1:pub1,key2:pub2"
func parseAPIKeys(envValue string) map[string]string {
	keys := make(map[string]string)
	if envValue == "" {
		return keys
	}

	pairs := strings.Split(envValue, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(strings.TrimSpace(pair), ":", 2)
		if len(parts) == 2 {
			keys[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		} else if len(parts) == 1 && parts[0] != "" {
			// Key without publisher ID mapping
			keys[strings.TrimSpace(parts[0])] = "default"
		}
	}
	return keys
}

// AuthMetrics defines the metrics interface for auth middleware
type AuthMetrics interface {
	IncAuthFailures()
}

// Auth provides API key authentication middleware
type Auth struct {
	config      *AuthConfig
	redisClient RedisClient
	metrics     AuthMetrics // P0: Metrics for auth failures
	mu          sync.RWMutex
	// Cache for Redis lookups (reduces latency)
	keyCache     map[string]cachedKey
	cacheMu      sync.RWMutex
	cacheTimeout time.Duration
	// Cache size limit to prevent unbounded growth
	maxCacheSize int
	stopCleanup  chan struct{}
	cleanupDone  chan struct{}
	shutdown     bool
	shutdownMu   sync.Mutex
}

type cachedKey struct {
	publisherID string
	expiresAt   time.Time
}

// NewAuth creates a new Auth middleware
func NewAuth(config *AuthConfig) *Auth {
	if config == nil {
		config = DefaultAuthConfig()
	}
	a := &Auth{
		config:       config,
		keyCache:     make(map[string]cachedKey),
		cacheTimeout: pbsconfig.AuthCacheTimeout, // P2-6: use named constant
		maxCacheSize: 10000,                      // Limit cache to 10,000 entries
		stopCleanup:  make(chan struct{}),
		cleanupDone:  make(chan struct{}),
	}

	// Start periodic cleanup goroutine (every 10 minutes)
	go a.periodicCacheCleanup()

	return a
}

// periodicCacheCleanup runs a background task to clean up expired cache entries
func (a *Auth) periodicCacheCleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	defer close(a.cleanupDone)

	for {
		select {
		case <-ticker.C:
			a.cleanupExpiredCacheEntries()
		case <-a.stopCleanup:
			return
		}
	}
}

// cleanupExpiredCacheEntries removes expired entries from the cache
func (a *Auth) cleanupExpiredCacheEntries() {
	a.cacheMu.Lock()
	defer a.cacheMu.Unlock()

	now := time.Now()
	for key, cached := range a.keyCache {
		if now.After(cached.expiresAt) {
			delete(a.keyCache, key)
		}
	}
}

// Shutdown stops the periodic cleanup goroutine and waits for it to finish
func (a *Auth) Shutdown() {
	a.shutdownMu.Lock()
	defer a.shutdownMu.Unlock()

	if a.shutdown {
		return // Already shut down
	}

	if a.stopCleanup != nil {
		close(a.stopCleanup)
		<-a.cleanupDone
	}

	a.shutdown = true
}

// NewAuthWithRedis creates Auth middleware with a Redis client
func NewAuthWithRedis(config *AuthConfig, redisClient RedisClient) *Auth {
	auth := NewAuth(config)
	auth.redisClient = redisClient
	return auth
}

// SetRedisClient sets the Redis client for API key validation
func (a *Auth) SetRedisClient(client RedisClient) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.redisClient = client
}

// Middleware returns the authentication middleware handler
func (a *Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Copy all needed config fields while holding the lock to prevent data race
		a.mu.RLock()
		enabled := a.config.Enabled
		bypassPaths := a.config.BypassPaths
		headerName := a.config.HeaderName
		a.mu.RUnlock()

		// Skip auth if disabled
		if !enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Check bypass paths with exact matching to prevent bypass attacks
		// SECURITY: Use exact path matching instead of HasPrefix (CVE-2026-XXXX)
		// This prevents /statusanything from matching /status
		for _, path := range bypassPaths {
			// Exact match or prefix followed by / or ?
			if r.URL.Path == path ||
				strings.HasPrefix(r.URL.Path, path+"/") ||
				strings.HasPrefix(r.URL.Path, path+"?") {
				next.ServeHTTP(w, r)
				return
			}
		}

		// Get API key from header
		apiKey := r.Header.Get(headerName)
		if apiKey == "" {
			// Also check Authorization header with Bearer scheme
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				apiKey = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if apiKey == "" {
			a.recordAuthFailure()
			http.Error(w, `{"error":"missing API key"}`, http.StatusUnauthorized)
			return
		}

		// Validate API key
		publisherID, valid := a.validateKey(r.Context(), apiKey)
		if !valid {
			a.recordAuthFailure()
			http.Error(w, `{"error":"invalid API key"}`, http.StatusForbidden)
			return
		}

		// Add publisher ID to request context (secure - can't be spoofed by client)
		ctx := context.WithValue(r.Context(), publisherIDKey, publisherID)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// validateKey checks if an API key is valid and returns the associated publisher ID
func (a *Auth) validateKey(ctx context.Context, key string) (string, bool) {
	// Check local cache first
	if pubID, found := a.checkCache(key); found {
		return pubID, pubID != ""
	}

	// Try Redis if available
	a.mu.RLock()
	redisClient := a.redisClient
	useRedis := a.config.UseRedis
	a.mu.RUnlock()

	if useRedis && redisClient != nil {
		pubID, err := redisClient.HGet(ctx, RedisAPIKeysHash, key)
		if err == nil && pubID != "" {
			a.updateCache(key, pubID)
			return pubID, true
		}
		if err != nil {
			log.Debug().Err(err).Msg("Redis API key lookup failed, falling back to local")
		}
	}

	// Fall back to local config
	// Note: We release the lock before calling updateCache to maintain
	// consistent lock ordering (mu before cacheMu) and prevent potential deadlocks
	var foundPubID string
	var found bool

	a.mu.RLock()
	for validKey, pubID := range a.config.APIKeys {
		// Use constant-time comparison to prevent timing attacks
		if subtle.ConstantTimeCompare([]byte(key), []byte(validKey)) == 1 {
			foundPubID = pubID
			found = true
			break
		}
	}
	a.mu.RUnlock()

	if found {
		a.updateCache(key, foundPubID)
		return foundPubID, true
	}

	// Cache negative result briefly to avoid hammering Redis
	a.updateCache(key, "")
	return "", false
}

// checkCache checks if a key is in the cache and still valid
func (a *Auth) checkCache(key string) (string, bool) {
	a.cacheMu.RLock()
	defer a.cacheMu.RUnlock()

	cached, exists := a.keyCache[key]
	if !exists {
		return "", false
	}

	if time.Now().After(cached.expiresAt) {
		return "", false
	}

	return cached.publisherID, true
}

// updateCache adds or updates a key in the cache
func (a *Auth) updateCache(key, publisherID string) {
	a.cacheMu.Lock()
	defer a.cacheMu.Unlock()

	// Enforce cache size limit to prevent unbounded growth
	if len(a.keyCache) >= a.maxCacheSize {
		// Cache is full - evict oldest entry (simple LRU-like behavior)
		// In production, consider using a proper LRU cache implementation
		var oldestKey string
		var oldestTime time.Time
		for k, v := range a.keyCache {
			if oldestKey == "" || v.expiresAt.Before(oldestTime) {
				oldestKey = k
				oldestTime = v.expiresAt
			}
		}
		if oldestKey != "" {
			delete(a.keyCache, oldestKey)
		}
	}

	// Use shorter timeout for negative results (P2-6: use named constant)
	timeout := a.cacheTimeout
	if publisherID == "" {
		timeout = pbsconfig.AuthNegativeCacheTimeout
	}

	a.keyCache[key] = cachedKey{
		publisherID: publisherID,
		expiresAt:   time.Now().Add(timeout),
	}
}

// ClearCache clears the API key cache
func (a *Auth) ClearCache() {
	a.cacheMu.Lock()
	defer a.cacheMu.Unlock()
	a.keyCache = make(map[string]cachedKey)
}

// AddAPIKey adds a new API key at runtime
func (a *Auth) AddAPIKey(key, publisherID string) {
	// Update config first
	a.mu.Lock()
	if a.config.APIKeys == nil {
		a.config.APIKeys = make(map[string]string)
	}
	a.config.APIKeys[key] = publisherID
	a.mu.Unlock()

	// Update cache after releasing mu to maintain consistent lock ordering
	a.updateCache(key, publisherID)
}

// RemoveAPIKey removes an API key at runtime
func (a *Auth) RemoveAPIKey(key string) {
	// Remove from config first
	a.mu.Lock()
	delete(a.config.APIKeys, key)
	a.mu.Unlock()

	// Clear from cache after releasing mu to maintain consistent lock ordering
	a.cacheMu.Lock()
	delete(a.keyCache, key)
	a.cacheMu.Unlock()
}

// SetEnabled enables or disables authentication
func (a *Auth) SetEnabled(enabled bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.config.Enabled = enabled
}

// IsEnabled returns whether authentication is enabled
func (a *Auth) IsEnabled() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.config.Enabled
}

// GetPublisherID returns the publisher ID for a given API key
func (a *Auth) GetPublisherID(ctx context.Context, key string) (string, bool) {
	return a.validateKey(ctx, key)
}

// SetMetrics sets the metrics interface for auth middleware
func (a *Auth) SetMetrics(m AuthMetrics) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.metrics = m
}

// recordAuthFailure increments the auth failures metric if available
func (a *Auth) recordAuthFailure() {
	a.mu.RLock()
	m := a.metrics
	a.mu.RUnlock()
	if m != nil {
		m.IncAuthFailures()
	}
}
