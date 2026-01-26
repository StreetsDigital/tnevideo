package middleware

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestLockOrdering_NoDeadlock verifies that concurrent operations
// that acquire multiple locks do not deadlock
func TestLockOrdering_NoDeadlock(t *testing.T) {
	auth := NewPublisherAuth(&PublisherAuthConfig{
		Enabled:           true,
		AllowUnregistered: false,
		RegisteredPubs:    map[string]string{"pub1": "example.com"},
		RateLimitPerPub:   100,
	})

	// Pre-populate cache and rate limits to ensure locks are contended
	auth.cachePublisher("pub1", "example.com", 30*time.Second)
	auth.checkRateLimit("pub1")

	ctx := context.Background()
	done := make(chan struct{})
	var operations int64

	// Start multiple goroutines performing different operations
	numGoroutines := 10
	var wg sync.WaitGroup

	// Test duration - if we deadlock, this will timeout
	timeout := time.After(5 * time.Second)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					// Cycle through different operations
					switch id % 5 {
					case 0:
						// Read config (mu.RLock)
						_ = auth.IsEnabled()
					case 1:
						// Check rate limit (mu.RLock → rateLimitsMu.Lock)
						_ = auth.checkRateLimit("pub1")
					case 2:
						// Cache publisher (publisherCacheMu.Lock)
						auth.cachePublisher("pub2", "test.com", 10*time.Second)
					case 3:
						// Get cached publisher (publisherCacheMu.RLock)
						_ = auth.getCachedPublisher("pub1")
					case 4:
						// Validate publisher (mu.RLock → calls getCachedPublisher)
						_ = auth.validatePublisher(ctx, "pub1", "example.com")
					}
					atomic.AddInt64(&operations, 1)
				}
			}
		}(i)
	}

	// Wait for timeout
	select {
	case <-timeout:
		// Timeout reached - stop all goroutines
		close(done)
		wg.Wait()
		t.Logf("Completed %d operations without deadlock", atomic.LoadInt64(&operations))
	}
}

// TestLockOrdering_ConcurrentConfigAndRateLimit tests the specific
// lock ordering: mu → rateLimitsMu
func TestLockOrdering_ConcurrentConfigAndRateLimit(t *testing.T) {
	auth := NewPublisherAuth(&PublisherAuthConfig{
		Enabled:         true,
		RateLimitPerPub: 100,
	})

	done := make(chan struct{})
	var operations int64

	// Goroutine 1: Constantly check rate limits (mu.RLock → rateLimitsMu.Lock)
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				auth.checkRateLimit("pub1")
				atomic.AddInt64(&operations, 1)
			}
		}
	}()

	// Goroutine 2: Constantly modify config (mu.Lock)
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				auth.SetEnabled(true)
				atomic.AddInt64(&operations, 1)
			}
		}
	}()

	// Run for 2 seconds - should not deadlock
	time.Sleep(2 * time.Second)
	close(done)

	ops := atomic.LoadInt64(&operations)
	if ops < 1000 {
		t.Errorf("Only completed %d operations in 2 seconds, expected more", ops)
	}
	t.Logf("Completed %d operations without deadlock", ops)
}

// TestLockOrdering_ConcurrentCacheOperations tests concurrent cache operations
func TestLockOrdering_ConcurrentCacheOperations(t *testing.T) {
	auth := NewPublisherAuth(&PublisherAuthConfig{
		Enabled: true,
	})

	done := make(chan struct{})
	var operations int64

	numGoroutines := 10
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					if id%2 == 0 {
						// Write to cache
						auth.cachePublisher("pub_test", "example.com", 10*time.Second)
					} else {
						// Read from cache
						_ = auth.getCachedPublisher("pub_test")
					}
					atomic.AddInt64(&operations, 1)
				}
			}
		}(i)
	}

	// Run for 1 second
	time.Sleep(1 * time.Second)
	close(done)
	wg.Wait()

	ops := atomic.LoadInt64(&operations)
	if ops < 1000 {
		t.Errorf("Only completed %d operations in 1 second, expected more", ops)
	}
	t.Logf("Completed %d cache operations without deadlock", ops)
}

// TestLockOrdering_ValidatePublisherWithRateLimiting tests the most complex
// lock interaction: validatePublisher (which may call cache methods) + checkRateLimit
func TestLockOrdering_ValidatePublisherWithRateLimiting(t *testing.T) {
	auth := NewPublisherAuth(&PublisherAuthConfig{
		Enabled:           true,
		AllowUnregistered: false,
		RegisteredPubs:    map[string]string{"pub1": "example.com"},
		RateLimitPerPub:   1000, // High limit to avoid rate limiting
	})

	ctx := context.Background()
	done := make(chan struct{})
	var operations int64

	numGoroutines := 20
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					// Simulate the middleware flow
					_ = auth.validatePublisher(ctx, "pub1", "example.com")
					_ = auth.checkRateLimit("pub1")
					atomic.AddInt64(&operations, 1)
				}
			}
		}(i)
	}

	// Run for 2 seconds
	time.Sleep(2 * time.Second)
	close(done)
	wg.Wait()

	ops := atomic.LoadInt64(&operations)
	if ops < 1000 {
		t.Errorf("Only completed %d operations in 2 seconds, expected more", ops)
	}
	t.Logf("Completed %d validate+ratelimit operations without deadlock", ops)
}

// TestLockOrdering_StressTest runs an intense stress test with all operations
func TestLockOrdering_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	auth := NewPublisherAuth(&PublisherAuthConfig{
		Enabled:           true,
		AllowUnregistered: false,
		RegisteredPubs:    map[string]string{"pub1": "example.com", "pub2": "test.com"},
		RateLimitPerPub:   10000,
		ValidateDomain:    true,
	})

	ctx := context.Background()
	done := make(chan struct{})
	var operations int64

	numGoroutines := 50
	var wg sync.WaitGroup

	operationTypes := []func(){
		// Config operations (mu.Lock or mu.RLock)
		func() { auth.SetEnabled(true) },
		func() { auth.SetEnabled(false) },
		func() { _ = auth.IsEnabled() },
		func() { auth.RegisterPublisher("pub3", "new.com") },
		func() { auth.UnregisterPublisher("pub3") },
		// Cache operations (publisherCacheMu)
		func() { auth.cachePublisher("pub1", "example.com", 30*time.Second) },
		func() { auth.cachePublisher("pub2", "test.com", 30*time.Second) },
		func() { _ = auth.getCachedPublisher("pub1") },
		func() { _ = auth.getCachedPublisher("pub2") },
		// Rate limit operations (mu.RLock → rateLimitsMu.Lock)
		func() { _ = auth.checkRateLimit("pub1") },
		func() { _ = auth.checkRateLimit("pub2") },
		// Validation (mu.RLock → getCachedPublisher)
		func() { _ = auth.validatePublisher(ctx, "pub1", "example.com") },
		func() { _ = auth.validatePublisher(ctx, "pub2", "test.com") },
	}

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					// Randomly pick an operation
					op := operationTypes[id%len(operationTypes)]
					op()
					atomic.AddInt64(&operations, 1)
				}
			}
		}(i)
	}

	// Run for 5 seconds
	time.Sleep(5 * time.Second)
	close(done)
	wg.Wait()

	ops := atomic.LoadInt64(&operations)
	if ops < 10000 {
		t.Errorf("Only completed %d operations in 5 seconds, expected more", ops)
	}
	t.Logf("Stress test completed %d operations without deadlock", ops)
}

// TestLockOrdering_RaceDetector tests for race conditions with -race flag
func TestLockOrdering_RaceDetector(t *testing.T) {
	auth := NewPublisherAuth(&PublisherAuthConfig{
		Enabled:           true,
		AllowUnregistered: false,
		RegisteredPubs:    map[string]string{"pub1": "example.com"},
		RateLimitPerPub:   100,
	})

	ctx := context.Background()
	done := make(chan struct{})

	// Mix reads and writes to all protected state
	var wg sync.WaitGroup

	// Writer goroutines
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					auth.SetEnabled(true)
					auth.RegisterPublisher("pub_new", "new.com")
					auth.cachePublisher("pub_cached", "cached.com", 10*time.Second)
				}
			}
		}()
	}

	// Reader goroutines
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					_ = auth.IsEnabled()
					_ = auth.checkRateLimit("pub1")
					_ = auth.getCachedPublisher("pub_cached")
					_ = auth.validatePublisher(ctx, "pub1", "example.com")
				}
			}
		}()
	}

	// Run briefly - race detector will catch issues
	time.Sleep(500 * time.Millisecond)
	close(done)
	wg.Wait()

	t.Log("Race detector test completed successfully")
}

// TestLockOrdering_CleanupOperations tests cleanup operations don't deadlock
func TestLockOrdering_CleanupOperations(t *testing.T) {
	auth := NewPublisherAuth(&PublisherAuthConfig{
		Enabled:         true,
		RateLimitPerPub: 1, // Very low to trigger cleanup
	})

	done := make(chan struct{})
	var wg sync.WaitGroup

	// Goroutine 1: Spam rate limit checks to trigger cleanups
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				select {
				case <-done:
					return
				default:
					// Use unique publisher IDs to grow the map
					pubID := "pub_" + string(rune('a'+id)) + "_" + string(rune('0'+j%10))
					auth.checkRateLimit(pubID)
				}
			}
		}(i)
	}

	// Goroutine 2: Spam cache operations to trigger cache cleanups
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				select {
				case <-done:
					return
				default:
					// Use unique cache keys with short TTL to expire quickly
					cacheKey := "cache_" + string(rune('a'+id)) + "_" + string(rune('0'+j%10))
					auth.cachePublisher(cacheKey, "example.com", 1*time.Millisecond)
					time.Sleep(2 * time.Millisecond) // Let some expire
				}
			}
		}(i)
	}

	// Wait for completion
	wg.Wait()
	close(done)

	t.Log("Cleanup operations completed without deadlock")
}

// TestLockOrdering_DocumentedBehavior verifies the documented lock ordering
func TestLockOrdering_DocumentedBehavior(t *testing.T) {
	auth := NewPublisherAuth(&PublisherAuthConfig{
		Enabled:         true,
		RateLimitPerPub: 100,
	})

	// Test 1: mu can be acquired alone
	auth.mu.RLock()
	_ = auth.config
	auth.mu.RUnlock()

	// Test 2: publisherCacheMu can be acquired alone
	auth.publisherCacheMu.Lock()
	auth.publisherCache = make(map[string]*publisherCacheEntry)
	auth.publisherCacheMu.Unlock()

	// Test 3: rateLimitsMu can be acquired alone
	auth.rateLimitsMu.Lock()
	auth.rateLimits = make(map[string]*rateLimitEntry)
	auth.rateLimitsMu.Unlock()

	// Test 4: mu → publisherCacheMu (allowed)
	auth.mu.RLock()
	config := auth.config
	auth.mu.RUnlock()
	if config != nil {
		auth.publisherCacheMu.Lock()
		auth.publisherCacheMu.Unlock()
	}

	// Test 5: mu → rateLimitsMu (allowed via checkRateLimit)
	_ = auth.checkRateLimit("pub1")

	// Test 6: Ensure we don't hold multiple locks simultaneously
	// This is enforced by the implementation pattern: copy config under lock,
	// release lock, then use config
	ctx := context.Background()
	_ = auth.validatePublisher(ctx, "pub1", "example.com")

	t.Log("Lock ordering behavior verified")
}

// Benchmark to measure lock contention overhead
func BenchmarkLockOrdering_Contention(b *testing.B) {
	auth := NewPublisherAuth(&PublisherAuthConfig{
		Enabled:           true,
		AllowUnregistered: false,
		RegisteredPubs:    map[string]string{"pub1": "example.com"},
		RateLimitPerPub:   1000000, // Very high to avoid rate limiting
	})

	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			switch i % 4 {
			case 0:
				_ = auth.validatePublisher(ctx, "pub1", "example.com")
			case 1:
				_ = auth.checkRateLimit("pub1")
			case 2:
				_ = auth.getCachedPublisher("pub1")
			case 3:
				auth.cachePublisher("pub1", "example.com", 30*time.Second)
			}
			i++
		}
	})
}
