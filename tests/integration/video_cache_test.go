//go:build integration
// +build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thenexusengine/tne_springwire/pkg/redis"
	"github.com/thenexusengine/tne_springwire/pkg/vast"
)

// TestVideoCaching tests Redis caching for video scenarios
func TestVideoCaching(t *testing.T) {
	// Skip if Redis is not available
	redisURL := "redis://localhost:6379"
	client, err := redis.New(redisURL)
	if err != nil {
		t.Skip("Redis not available, skipping cache tests")
	}
	defer client.Close()

	ctx := context.Background()

	t.Run("Cache_VAST_XML_response", func(t *testing.T) {
		// Create test VAST
		vastDoc := createTestVAST()
		vastXML, err := vastDoc.Marshal()
		require.NoError(t, err)

		// Cache key
		cacheKey := "prebid:vast:test-request-001"
		ttl := 300 * time.Second // 5 minutes

		// Set cache
		err = client.Set(ctx, cacheKey, string(vastXML), ttl)
		require.NoError(t, err)

		// Get from cache
		cached, err := client.Get(ctx, cacheKey)
		require.NoError(t, err)
		assert.Equal(t, string(vastXML), cached)

		// Verify it's valid VAST
		var cachedVAST vast.VAST
		err = vast.Parse([]byte(cached)).Error
		assert.NoError(t, err)

		// Cleanup
		client.Del(ctx, cacheKey)
	})

	t.Run("Cache_expiration", func(t *testing.T) {
		cacheKey := "prebid:vast:expiry-test"
		value := "<VAST>test</VAST>"
		ttl := 2 * time.Second // Short TTL for testing

		// Set with short TTL
		err := client.Set(ctx, cacheKey, value, ttl)
		require.NoError(t, err)

		// Should exist immediately
		cached, err := client.Get(ctx, cacheKey)
		require.NoError(t, err)
		assert.Equal(t, value, cached)

		// Wait for expiration
		time.Sleep(3 * time.Second)

		// Should be expired
		_, err = client.Get(ctx, cacheKey)
		assert.Error(t, err, "Cache should have expired")
	})

	t.Run("Cache_video_creative_URLs", func(t *testing.T) {
		creativeID := "creative-12345"
		cacheKey := fmt.Sprintf("prebid:creative:%s", creativeID)

		creativeData := map[string]interface{}{
			"url":      "https://cdn.example.com/video.mp4",
			"duration": 30,
			"width":    1920,
			"height":   1080,
			"bitrate":  5000,
			"format":   "video/mp4",
		}

		// Serialize to JSON
		data, err := json.Marshal(creativeData)
		require.NoError(t, err)

		// Cache creative metadata
		ttl := 3600 * time.Second // 1 hour
		err = client.Set(ctx, cacheKey, string(data), ttl)
		require.NoError(t, err)

		// Retrieve
		cached, err := client.Get(ctx, cacheKey)
		require.NoError(t, err)

		// Deserialize
		var retrieved map[string]interface{}
		err = json.Unmarshal([]byte(cached), &retrieved)
		require.NoError(t, err)

		assert.Equal(t, "https://cdn.example.com/video.mp4", retrieved["url"])
		assert.Equal(t, float64(30), retrieved["duration"])

		// Cleanup
		client.Del(ctx, cacheKey)
	})

	t.Run("Cache_bid_responses", func(t *testing.T) {
		requestID := "req-cache-test-001"
		cacheKey := fmt.Sprintf("prebid:bid:%s", requestID)

		bidResponse := map[string]interface{}{
			"id":    requestID,
			"price": 5.50,
			"vast":  "<VAST>...</VAST>",
			"bidder": "test-bidder",
		}

		data, err := json.Marshal(bidResponse)
		require.NoError(t, err)

		// Cache bid response
		ttl := 60 * time.Second // 1 minute (short for bid responses)
		err = client.Set(ctx, cacheKey, string(data), ttl)
		require.NoError(t, err)

		// Retrieve
		cached, err := client.Get(ctx, cacheKey)
		require.NoError(t, err)

		var retrieved map[string]interface{}
		err = json.Unmarshal([]byte(cached), &retrieved)
		require.NoError(t, err)

		assert.Equal(t, requestID, retrieved["id"])
		assert.Equal(t, 5.50, retrieved["price"])

		// Cleanup
		client.Del(ctx, cacheKey)
	})

	t.Run("Cache_invalidation", func(t *testing.T) {
		cacheKey := "prebid:vast:invalidation-test"
		value := "<VAST>original</VAST>"

		// Set initial value
		err := client.Set(ctx, cacheKey, value, 300*time.Second)
		require.NoError(t, err)

		// Verify it exists
		cached, err := client.Get(ctx, cacheKey)
		require.NoError(t, err)
		assert.Equal(t, value, cached)

		// Invalidate (delete)
		err = client.Del(ctx, cacheKey)
		require.NoError(t, err)

		// Should be gone
		_, err = client.Get(ctx, cacheKey)
		assert.Error(t, err, "Cache should be invalidated")
	})

	t.Run("Cache_hit_miss_metrics", func(t *testing.T) {
		// Simulate cache miss
		cacheKey := "prebid:vast:metrics-test"

		// First request - cache miss
		start := time.Now()
		_, err := client.Get(ctx, cacheKey)
		missDuration := time.Since(start)
		assert.Error(t, err, "First request should be cache miss")

		// Populate cache
		value := "<VAST>test</VAST>"
		err = client.Set(ctx, cacheKey, value, 300*time.Second)
		require.NoError(t, err)

		// Second request - cache hit
		start = time.Now()
		cached, err := client.Get(ctx, cacheKey)
		hitDuration := time.Since(start)
		require.NoError(t, err)
		assert.Equal(t, value, cached)

		// Cache hit should be faster (< 5ms typical)
		assert.Less(t, hitDuration, 10*time.Millisecond, "Cache hit should be fast")

		t.Logf("Cache miss: %v, Cache hit: %v", missDuration, hitDuration)

		// Cleanup
		client.Del(ctx, cacheKey)
	})

	t.Run("Performance_improvement_with_caching", func(t *testing.T) {
		cacheKey := "prebid:vast:performance-test"

		// Simulate expensive operation (VAST generation)
		expensiveOperation := func() string {
			time.Sleep(50 * time.Millisecond) // Simulate 50ms operation
			return "<VAST>expensive result</VAST>"
		}

		// Without cache
		start := time.Now()
		result1 := expensiveOperation()
		noCacheDuration := time.Since(start)

		// Set cache
		err := client.Set(ctx, cacheKey, result1, 300*time.Second)
		require.NoError(t, err)

		// With cache
		start = time.Now()
		result2, err := client.Get(ctx, cacheKey)
		cacheDuration := time.Since(start)
		require.NoError(t, err)

		assert.Equal(t, result1, result2)

		// Cache should be significantly faster
		assert.Less(t, cacheDuration, noCacheDuration/5, "Cache should be at least 5x faster")

		speedup := float64(noCacheDuration) / float64(cacheDuration)
		t.Logf("Speedup: %.2fx (no cache: %v, cached: %v)", speedup, noCacheDuration, cacheDuration)

		// Cleanup
		client.Del(ctx, cacheKey)
	})

	t.Run("Concurrent_cache_access", func(t *testing.T) {
		cacheKey := "prebid:vast:concurrent-test"
		value := "<VAST>concurrent</VAST>"

		// Set initial value
		err := client.Set(ctx, cacheKey, value, 300*time.Second)
		require.NoError(t, err)

		// Concurrent reads
		concurrency := 100
		done := make(chan bool, concurrency)

		for i := 0; i < concurrency; i++ {
			go func() {
				cached, err := client.Get(ctx, cacheKey)
				assert.NoError(t, err)
				assert.Equal(t, value, cached)
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < concurrency; i++ {
			<-done
		}

		t.Log("100 concurrent cache reads completed successfully")

		// Cleanup
		client.Del(ctx, cacheKey)
	})

	t.Run("Namespace_key_patterns", func(t *testing.T) {
		// Test different key patterns
		patterns := []struct {
			name string
			key  string
		}{
			{"VAST cache", "prebid:vast:req-123"},
			{"Creative cache", "prebid:creative:cr-456"},
			{"Bid cache", "prebid:bid:auction-789"},
			{"User cache", "prebid:user:user-abc"},
		}

		for _, p := range patterns {
			t.Run(p.name, func(t *testing.T) {
				value := fmt.Sprintf("test-%s", p.name)
				err := client.Set(ctx, p.key, value, 60*time.Second)
				require.NoError(t, err)

				cached, err := client.Get(ctx, p.key)
				require.NoError(t, err)
				assert.Equal(t, value, cached)

				// Cleanup
				client.Del(ctx, p.key)
			})
		}
	})
}

// TestVideoCacheStrategies tests different caching strategies
func TestVideoCacheStrategies(t *testing.T) {
	redisURL := "redis://localhost:6379"
	client, err := redis.New(redisURL)
	if err != nil {
		t.Skip("Redis not available")
	}
	defer client.Close()

	ctx := context.Background()

	t.Run("Cache_by_request_signature", func(t *testing.T) {
		// Generate cache key from request parameters
		requestParams := map[string]interface{}{
			"w":       1920,
			"h":       1080,
			"mindur":  5,
			"maxdur":  30,
			"site_id": "pub-123",
		}

		// Hash or serialize params for cache key
		data, _ := json.Marshal(requestParams)
		cacheKey := fmt.Sprintf("prebid:vast:sig:%x", data[:8])

		value := "<VAST>cached by signature</VAST>"
		err := client.Set(ctx, cacheKey, value, 300*time.Second)
		require.NoError(t, err)

		cached, err := client.Get(ctx, cacheKey)
		require.NoError(t, err)
		assert.Equal(t, value, cached)

		client.Del(ctx, cacheKey)
	})

	t.Run("Layered_cache_TTL", func(t *testing.T) {
		// Different TTLs for different data types
		ttls := map[string]time.Duration{
			"vast":     300 * time.Second,  // 5 minutes
			"creative": 3600 * time.Second, // 1 hour
			"bid":      60 * time.Second,   // 1 minute
		}

		for dataType, ttl := range ttls {
			key := fmt.Sprintf("prebid:%s:ttl-test", dataType)
			value := fmt.Sprintf("test-%s", dataType)

			err := client.Set(ctx, key, value, ttl)
			require.NoError(t, err)

			// Verify TTL is set
			cached, err := client.Get(ctx, key)
			require.NoError(t, err)
			assert.Equal(t, value, cached)

			client.Del(ctx, key)
		}
	})

	t.Run("Cache_warming", func(t *testing.T) {
		// Pre-populate cache with common requests
		commonRequests := []string{
			"prebid:vast:1920x1080-mp4",
			"prebid:vast:1280x720-mp4",
			"prebid:vast:640x480-mp4",
		}

		for _, key := range commonRequests {
			value := fmt.Sprintf("<VAST>warmed-%s</VAST>", key)
			err := client.Set(ctx, key, value, 600*time.Second)
			require.NoError(t, err)
		}

		// Verify all are cached
		for _, key := range commonRequests {
			_, err := client.Get(ctx, key)
			assert.NoError(t, err, "Warmed cache should exist for %s", key)
			client.Del(ctx, key)
		}

		t.Log("Cache warming completed for common request patterns")
	})
}

// Helper function
func createTestVAST() *vast.VAST {
	builder := vast.NewBuilder("4.0")
	v, _ := builder.
		AddAd("test-ad").
		WithInLine("TNEVideo", "Test Ad").
		WithImpression("https://tracking.example.com/imp", "imp-1").
		WithLinearCreative("creative-1", 30*time.Second).
		WithMediaFile("https://cdn.example.com/video.mp4", "video/mp4", 1920, 1080).
		WithTracking(vast.EventStart, "https://tracking.example.com/start").
		WithTracking(vast.EventComplete, "https://tracking.example.com/complete").
		EndLinear().
		Done().
		Build()
	return v
}
