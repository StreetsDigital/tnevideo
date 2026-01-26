package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestAuthCacheSizeLimit verifies that the auth cache respects the size limit
func TestAuthCacheSizeLimit(t *testing.T) {
	auth := NewAuth(&AuthConfig{
		Enabled: true,
		APIKeys: map[string]string{},
	})
	defer auth.Shutdown()

	// The cache should enforce the maxCacheSize limit
	if auth.maxCacheSize != 10000 {
		t.Errorf("expected maxCacheSize=10000, got %d", auth.maxCacheSize)
	}

	// Add entries up to the limit
	for i := 0; i < auth.maxCacheSize+100; i++ {
		key := string(rune('a' + (i % 26)))
		auth.updateCache(key, "pub")
	}

	// Cache should not exceed the limit
	auth.cacheMu.RLock()
	size := len(auth.keyCache)
	auth.cacheMu.RUnlock()

	if size > auth.maxCacheSize {
		t.Errorf("cache size %d exceeds limit %d", size, auth.maxCacheSize)
	}
}

// TestAuthPeriodicCleanup verifies that expired cache entries are cleaned up
func TestAuthPeriodicCleanup(t *testing.T) {
	auth := NewAuth(&AuthConfig{
		Enabled: true,
		APIKeys: map[string]string{},
	})
	defer auth.Shutdown()

	// Add an entry that will expire quickly
	auth.cacheTimeout = 100 * time.Millisecond
	auth.updateCache("test-key", "pub1")

	// Wait for expiration
	time.Sleep(200 * time.Millisecond)

	// Manually trigger cleanup
	auth.cleanupExpiredCacheEntries()

	// Entry should be removed
	auth.cacheMu.RLock()
	_, exists := auth.keyCache["test-key"]
	auth.cacheMu.RUnlock()

	if exists {
		t.Error("expected expired cache entry to be removed")
	}
}

// TestAuthShutdown verifies graceful shutdown
func TestAuthShutdown(t *testing.T) {
	auth := NewAuth(&AuthConfig{
		Enabled: true,
		APIKeys: map[string]string{},
	})

	// Should shut down cleanly
	auth.Shutdown()

	// Should be safe to call multiple times
	auth.Shutdown()
}

// TestPrivacyRequestBodySizeLimit verifies that large request bodies are rejected
func TestPrivacyRequestBodySizeLimit(t *testing.T) {
	config := DefaultPrivacyConfig()
	config.EnforceGDPR = false
	config.EnforceCOPPA = false
	config.EnforceCCPA = false

	middleware := NewPrivacyMiddleware(config)

	// Create a large request body (>1MB)
	largeBody := bytes.Repeat([]byte("x"), 2*1024*1024) // 2MB

	req := httptest.NewRequest("POST", "/openrtb2/auction", bytes.NewReader(largeBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	middleware(next).ServeHTTP(rec, req)

	// Should reject the large request
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413 for large request, got %d", rec.Code)
	}

	if nextCalled {
		t.Error("next handler should not have been called for oversized request")
	}
}

// TestPrivacyRequestBodySizeLimitWithinLimit verifies normal requests pass through
func TestPrivacyRequestBodySizeLimitWithinLimit(t *testing.T) {
	config := DefaultPrivacyConfig()
	config.EnforceGDPR = false
	config.EnforceCOPPA = false
	config.EnforceCCPA = false

	middleware := NewPrivacyMiddleware(config)

	// Create a small request body
	smallBody := []byte(`{"id":"test"}`)

	req := httptest.NewRequest("POST", "/openrtb2/auction", bytes.NewReader(smallBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		// Verify we can read the body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read body: %v", err)
		}
		if !bytes.Equal(body, smallBody) {
			t.Errorf("body mismatch: got %s, want %s", body, smallBody)
		}
		w.WriteHeader(http.StatusOK)
	})

	middleware(next).ServeHTTP(rec, req)

	if !nextCalled {
		t.Error("next handler should have been called for normal request")
	}
}
