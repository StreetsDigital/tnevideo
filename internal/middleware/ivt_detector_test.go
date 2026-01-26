package middleware

import (
	"context"
	"net/http/httptest"
	"testing"
)

func TestIVTDetector_SuspiciousUA(t *testing.T) {
	detector := NewIVTDetector(nil) // Use defaults

	tests := []struct {
		name               string
		userAgent          string
		shouldBeSuspicious bool
	}{
		{"Normal Chrome", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36", false},
		{"Normal Firefox", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0", false},
		{"Bot - explicit", "Googlebot/2.1", true},
		{"Bot - crawler", "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)", true},
		{"Scraper - curl", "curl/7.68.0", true},
		{"Scraper - python", "python-requests/2.25.1", true},
		{"Headless Chrome", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/91.0.4472.124 Safari/537.36", true},
		{"Empty UA", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/openrtb2/auction", nil)
			req.Header.Set("User-Agent", tt.userAgent)

			result := detector.Validate(context.Background(), req, "test-pub", "example.com")

			hasSuspiciousUA := false
			for _, signal := range result.Signals {
				if signal.Type == "suspicious_ua" {
					hasSuspiciousUA = true
					break
				}
			}

			if hasSuspiciousUA != tt.shouldBeSuspicious {
				t.Errorf("UA: %q - expected suspicious=%v, got suspicious=%v (score=%d)",
					tt.userAgent, tt.shouldBeSuspicious, hasSuspiciousUA, result.Score)
			}
		})
	}
}

func TestIVTDetector_RefererValidation(t *testing.T) {
	detector := NewIVTDetector(nil)

	tests := []struct {
		name            string
		referer         string
		domain          string
		shouldBeInvalid bool
	}{
		{"Valid referer", "https://example.com/page", "example.com", false},
		{"Valid subdomain", "https://www.example.com/page", "example.com", false},
		{"Mismatch", "https://malicious.com/page", "example.com", true},
		{"Empty referer", "", "example.com", false}, // Not invalid unless RequireReferer is true
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/openrtb2/auction", nil)
			req.Header.Set("User-Agent", "Mozilla/5.0 (normal browser)")
			req.Header.Set("Referer", tt.referer)

			result := detector.Validate(context.Background(), req, "test-pub", tt.domain)

			hasInvalidReferer := false
			for _, signal := range result.Signals {
				if signal.Type == "invalid_referer" {
					hasInvalidReferer = true
					break
				}
			}

			if hasInvalidReferer != tt.shouldBeInvalid {
				t.Errorf("Referer: %q, Domain: %q - expected invalid=%v, got invalid=%v",
					tt.referer, tt.domain, tt.shouldBeInvalid, hasInvalidReferer)
			}
		})
	}
}

func TestIVTDetector_Scoring(t *testing.T) {
	detector := NewIVTDetector(nil)

	tests := []struct {
		name        string
		userAgent   string
		referer     string
		domain      string
		expectScore int // Approximate score
		expectBlock bool
	}{
		{
			name:        "Clean request",
			userAgent:   "Mozilla/5.0 (normal browser)",
			referer:     "https://example.com",
			domain:      "example.com",
			expectScore: 0,
			expectBlock: false,
		},
		{
			name:        "Bot UA only (high severity = 50 points)",
			userAgent:   "Googlebot/2.1",
			referer:     "https://example.com",
			domain:      "example.com",
			expectScore: 50,
			expectBlock: false, // Below 70 threshold
		},
		{
			name:        "Bot UA + Referer mismatch (50 + 50 = 100)",
			userAgent:   "curl/7.68.0",
			referer:     "https://malicious.com",
			domain:      "example.com",
			expectScore: 100,
			expectBlock: true, // >= 70 threshold
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Enable blocking for this test
			config := DefaultIVTConfig()
			config.BlockingEnabled = true
			detector.SetConfig(config)

			req := httptest.NewRequest("POST", "/openrtb2/auction", nil)
			req.Header.Set("User-Agent", tt.userAgent)
			req.Header.Set("Referer", tt.referer)

			result := detector.Validate(context.Background(), req, "test-pub", tt.domain)

			if result.Score != tt.expectScore {
				t.Errorf("Expected score %d, got %d (signals: %d)",
					tt.expectScore, result.Score, len(result.Signals))
			}

			if result.ShouldBlock != tt.expectBlock {
				t.Errorf("Expected block=%v, got block=%v (score=%d)",
					tt.expectBlock, result.ShouldBlock, result.Score)
			}
		})
	}
}

func TestIVTDetector_Disabled(t *testing.T) {
	config := DefaultIVTConfig()
	config.MonitoringEnabled = false
	detector := NewIVTDetector(config)

	req := httptest.NewRequest("POST", "/openrtb2/auction", nil)
	req.Header.Set("User-Agent", "curl/7.68.0") // Suspicious

	result := detector.Validate(context.Background(), req, "test-pub", "example.com")

	if !result.IsValid {
		t.Error("Expected valid when IVT detection is disabled")
	}

	if len(result.Signals) > 0 {
		t.Errorf("Expected no signals when disabled, got %d", len(result.Signals))
	}
}

func TestIVTDetector_Metrics(t *testing.T) {
	config := DefaultIVTConfig()
	config.BlockingEnabled = false // Monitoring mode (don't block, just flag)
	detector := NewIVTDetector(config)

	// Run some clean validations
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("POST", "/openrtb2/auction", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		req.Header.Set("Referer", "https://example.com")
		detector.Validate(context.Background(), req, "test-pub", "example.com")
	}

	// Run some suspicious ones with score >= 70 (should be flagged but not blocked in monitoring mode)
	// curl UA (50) + referer mismatch (50) = 100 score -> flagged
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("POST", "/openrtb2/auction", nil)
		req.Header.Set("User-Agent", "curl/7.68.0")
		req.Header.Set("Referer", "https://malicious.com")
		detector.Validate(context.Background(), req, "test-pub", "example.com")
	}

	metrics := detector.GetMetrics()

	if metrics.TotalChecked != 15 {
		t.Errorf("Expected 15 total checks, got %d", metrics.TotalChecked)
	}

	if metrics.TotalFlagged != 5 {
		t.Errorf("Expected 5 flagged, got %d", metrics.TotalFlagged)
	}

	if metrics.SuspiciousUA != 5 {
		t.Errorf("Expected 5 suspicious UA signals, got %d", metrics.SuspiciousUA)
	}

	if metrics.InvalidReferer != 5 {
		t.Errorf("Expected 5 invalid referer signals, got %d", metrics.InvalidReferer)
	}

	if metrics.TotalBlocked != 0 {
		t.Errorf("Expected 0 blocked (monitoring mode), got %d", metrics.TotalBlocked)
	}
}

func TestIVTDetector_ClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		xff        string
		xri        string
		expectedIP string
	}{
		{"Direct connection", "192.168.1.1:12345", "", "", "192.168.1.1"},
		{"X-Forwarded-For", "127.0.0.1:12345", "203.0.113.1, 198.51.100.1", "", "203.0.113.1"},
		{"X-Real-IP", "127.0.0.1:12345", "", "203.0.113.1", "203.0.113.1"},
		{"Both headers (XFF wins)", "127.0.0.1:12345", "203.0.113.1", "198.51.100.1", "203.0.113.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xri != "" {
				req.Header.Set("X-Real-IP", tt.xri)
			}

			ip := getClientIP(req)
			if ip != tt.expectedIP {
				t.Errorf("Expected IP %s, got %s", tt.expectedIP, ip)
			}
		})
	}
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://example.com/path", "example.com"},
		{"http://example.com", "example.com"},
		{"https://www.example.com:8080/path", "www.example.com"},
		{"example.com/path?query=1", "example.com"},
		{"https://sub.domain.example.com/page", "sub.domain.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := extractDomain(tt.url)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestIVTDetector_GetConfig(t *testing.T) {
	config := DefaultIVTConfig()
	config.MonitoringEnabled = true
	config.BlockingEnabled = true

	detector := NewIVTDetector(config)

	retrievedConfig := detector.GetConfig()

	if retrievedConfig == nil {
		t.Fatal("Expected non-nil config")
	}

	if !retrievedConfig.MonitoringEnabled {
		t.Error("Expected config.MonitoringEnabled to be true")
	}

	if !retrievedConfig.BlockingEnabled {
		t.Error("Expected config.BlockingEnabled to be true")
	}
}

// TestIVTDetector_ConcurrentConfigReload tests thread-safe config updates
// This test verifies the fix for the sync.Once reset race condition.
// Previously, SetConfig would reset sync.Once while compilePatterns was executing,
// causing a data race. The fix uses a version counter approach instead.
func TestIVTDetector_ConcurrentConfigReload(t *testing.T) {
	detector := NewIVTDetector(nil)

	// Number of concurrent goroutines
	const numReaders = 50
	const numWriters = 10
	const iterations = 100

	// Use channels to synchronize start and completion
	start := make(chan struct{})
	done := make(chan struct{}, numReaders+numWriters)

	// Start reader goroutines that continuously validate requests
	for i := 0; i < numReaders; i++ {
		go func() {
			<-start // Wait for signal to start
			for j := 0; j < iterations; j++ {
				req := httptest.NewRequest("POST", "/openrtb2/auction", nil)
				req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
				req.Header.Set("Referer", "https://example.com")

				// This calls compilePatterns() internally
				result := detector.Validate(context.Background(), req, "test-pub", "example.com")

				// Basic sanity check - should not panic or return nil
				if result == nil {
					t.Error("Validate returned nil result")
				}
			}
			done <- struct{}{}
		}()
	}

	// Start writer goroutines that continuously update config
	for i := 0; i < numWriters; i++ {
		go func(writerID int) {
			<-start // Wait for signal to start
			for j := 0; j < iterations; j++ {
				config := DefaultIVTConfig()
				// Alternate between different patterns to ensure recompilation happens
				if j%2 == 0 {
					config.SuspiciousUAPatterns = []string{`(?i)bot`, `(?i)crawler`}
				} else {
					config.SuspiciousUAPatterns = []string{`(?i)spider`, `(?i)scraper`, `(?i)curl`}
				}
				// This used to reset sync.Once, causing a race condition
				detector.SetConfig(config)
			}
			done <- struct{}{}
		}(i)
	}

	// Start all goroutines simultaneously
	close(start)

	// Wait for all goroutines to complete
	for i := 0; i < numReaders+numWriters; i++ {
		<-done
	}
}

// TestIVTDetector_ConcurrentConfigReload_RaceDetector is designed to trigger
// the race detector if the sync.Once reset race condition exists.
// Run with: go test -race -run TestIVTDetector_ConcurrentConfigReload_RaceDetector
func TestIVTDetector_ConcurrentConfigReload_RaceDetector(t *testing.T) {
	detector := NewIVTDetector(nil)

	// Channels for tight synchronization
	ready := make(chan struct{})
	start := make(chan struct{})
	done := make(chan struct{}, 2)

	// Goroutine 1: calls Validate (which calls compilePatterns)
	go func() {
		ready <- struct{}{}
		<-start
		for i := 0; i < 1000; i++ {
			req := httptest.NewRequest("POST", "/openrtb2/auction", nil)
			req.Header.Set("User-Agent", "curl/7.68.0") // Triggers pattern matching
			detector.Validate(context.Background(), req, "test-pub", "example.com")
		}
		done <- struct{}{}
	}()

	// Goroutine 2: calls SetConfig (which used to reset sync.Once)
	go func() {
		ready <- struct{}{}
		<-start
		for i := 0; i < 1000; i++ {
			config := DefaultIVTConfig()
			config.SuspiciousUAPatterns = []string{`(?i)bot`, `(?i)curl`}
			detector.SetConfig(config)
		}
		done <- struct{}{}
	}()

	// Wait for both goroutines to be ready
	<-ready
	<-ready

	// Start both simultaneously
	close(start)

	// Wait for completion
	<-done
	<-done
}

// TestIVTDetector_PatternRecompilation verifies patterns are recompiled after config change
func TestIVTDetector_PatternRecompilation(t *testing.T) {
	// Start with config that does NOT match "testbot"
	config := DefaultIVTConfig()
	config.SuspiciousUAPatterns = []string{`(?i)curl`, `(?i)wget`}
	detector := NewIVTDetector(config)

	// First validation - "testbot" should NOT be flagged
	req := httptest.NewRequest("POST", "/openrtb2/auction", nil)
	req.Header.Set("User-Agent", "testbot/1.0")
	req.Header.Set("Referer", "https://example.com")

	result := detector.Validate(context.Background(), req, "test-pub", "example.com")

	hasSuspiciousUA := false
	for _, signal := range result.Signals {
		if signal.Type == "suspicious_ua" {
			hasSuspiciousUA = true
			break
		}
	}

	if hasSuspiciousUA {
		t.Error("Expected 'testbot' NOT to be flagged with initial patterns")
	}

	// Update config to include pattern that matches "testbot"
	config2 := DefaultIVTConfig()
	config2.SuspiciousUAPatterns = []string{`(?i)testbot`}
	detector.SetConfig(config2)

	// Second validation - "testbot" should now be flagged
	req2 := httptest.NewRequest("POST", "/openrtb2/auction", nil)
	req2.Header.Set("User-Agent", "testbot/1.0")
	req2.Header.Set("Referer", "https://example.com")

	result2 := detector.Validate(context.Background(), req2, "test-pub", "example.com")

	hasSuspiciousUA = false
	for _, signal := range result2.Signals {
		if signal.Type == "suspicious_ua" {
			hasSuspiciousUA = true
			break
		}
	}

	if !hasSuspiciousUA {
		t.Error("Expected 'testbot' to be flagged after config update")
	}
}

// TestIVTDetector_VersionIncrement verifies the version counter increments correctly
func TestIVTDetector_VersionIncrement(t *testing.T) {
	detector := NewIVTDetector(nil)

	// Initial version should be 1 (set in NewIVTDetector)
	initialVersion := detector.patternsVersion.Load()
	if initialVersion != 1 {
		t.Errorf("Expected initial version 1, got %d", initialVersion)
	}

	// After SetConfig, version should increment
	for i := 0; i < 5; i++ {
		config := DefaultIVTConfig()
		detector.SetConfig(config)
	}

	finalVersion := detector.patternsVersion.Load()
	expectedVersion := uint64(6) // 1 initial + 5 increments
	if finalVersion != expectedVersion {
		t.Errorf("Expected version %d after 5 SetConfig calls, got %d", expectedVersion, finalVersion)
	}
}
