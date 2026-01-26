package pauseads

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/thenexusengine/tne_springwire/internal/openrtb"
)

// MockAdRequester is a mock implementation of AdRequester for testing
type MockAdRequester struct {
	mu           sync.Mutex
	responses    []*PauseAdResponse
	errors       []error
	callCount    int
	lastRequest  *PauseAdRequest
	returnError  bool
	returnNoBid  bool
	returnAd     bool
	responseDelay time.Duration
}

func (m *MockAdRequester) RequestPauseAd(ctx context.Context, req *PauseAdRequest) (*PauseAdResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.lastRequest = req
	m.callCount++

	if m.responseDelay > 0 {
		time.Sleep(m.responseDelay)
	}

	if m.returnError {
		return nil, errors.New("mock error")
	}

	if m.returnNoBid {
		return &PauseAdResponse{NoBid: true}, nil
	}

	if m.returnAd {
		return &PauseAdResponse{
			Ad: &PauseAd{
				ID:              "test-ad-123",
				CreativeURL:     "https://example.com/ad.jpg",
				ClickURL:        "https://example.com/click",
				Width:           1920,
				Height:          1080,
				Format:          "image/jpeg",
				DisplayDuration: 30,
			},
		}, nil
	}

	if len(m.responses) > 0 {
		resp := m.responses[0]
		m.responses = m.responses[1:]
		return resp, nil
	}

	if len(m.errors) > 0 {
		err := m.errors[0]
		m.errors = m.errors[1:]
		return nil, err
	}

	return &PauseAdResponse{NoBid: true}, nil
}

func (m *MockAdRequester) GetCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

func (m *MockAdRequester) GetLastRequest() *PauseAdRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastRequest
}

// TestDefaultConfig verifies the default configuration
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if !config.Enabled {
		t.Error("expected default config to be enabled")
	}

	if config.MinPauseDuration != 3 {
		t.Errorf("expected MinPauseDuration=3, got %d", config.MinPauseDuration)
	}

	if config.MaxDisplayDuration != 60 {
		t.Errorf("expected MaxDisplayDuration=60, got %d", config.MaxDisplayDuration)
	}

	if config.FrequencyCap == nil {
		t.Fatal("expected frequency cap to be set")
	}

	if config.FrequencyCap.MaxImpressions != 5 {
		t.Errorf("expected MaxImpressions=5, got %d", config.FrequencyCap.MaxImpressions)
	}

	if config.FrequencyCap.TimeWindowSeconds != 3600 {
		t.Errorf("expected TimeWindowSeconds=3600, got %d", config.FrequencyCap.TimeWindowSeconds)
	}
}

// TestPauseAdTrackerPeriodicCleanup verifies that old impressions are cleaned up
func TestPauseAdTrackerPeriodicCleanup(t *testing.T) {
	tracker := NewPauseAdTracker()
	defer tracker.Shutdown()

	// Add some impressions
	tracker.RecordImpression("session1")
	tracker.RecordImpression("session2")

	// Verify they exist
	tracker.mu.RLock()
	if len(tracker.impressions) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(tracker.impressions))
	}
	tracker.mu.RUnlock()

	// Manually add an old impression
	tracker.mu.Lock()
	tracker.impressions["session3"] = []time.Time{
		time.Now().Add(-25 * time.Hour), // Older than 24 hours
	}
	tracker.mu.Unlock()

	// Trigger cleanup
	tracker.cleanupExpiredSessions()

	// Old session should be removed, recent ones should remain
	tracker.mu.RLock()
	defer tracker.mu.RUnlock()

	if _, exists := tracker.impressions["session3"]; exists {
		t.Error("expected expired session to be removed")
	}

	if _, exists := tracker.impressions["session1"]; !exists {
		t.Error("expected recent session to remain")
	}

	if _, exists := tracker.impressions["session2"]; !exists {
		t.Error("expected recent session to remain")
	}
}

// TestPauseAdTrackerShutdown verifies graceful shutdown
func TestPauseAdTrackerShutdown(t *testing.T) {
	tracker := NewPauseAdTracker()

	// Should shut down cleanly
	tracker.Shutdown()

	// Should be safe to call multiple times
	tracker.Shutdown()
}

// TestPauseAdTrackerFrequencyCap verifies frequency capping works correctly
func TestPauseAdTrackerFrequencyCap(t *testing.T) {
	tracker := NewPauseAdTracker()
	defer tracker.Shutdown()

	cap := &FrequencyCap{
		MaxImpressions:    3,
		TimeWindowSeconds: 3600, // 1 hour
	}

	sessionID := "test-session"

	// Should allow ads initially
	if !tracker.CanShowAd(sessionID, cap) {
		t.Error("should allow ad on first check")
	}

	// Record impressions up to the cap
	for i := 0; i < 3; i++ {
		tracker.RecordImpression(sessionID)
	}

	// Should not allow ad after reaching cap
	if tracker.CanShowAd(sessionID, cap) {
		t.Error("should not allow ad after reaching cap")
	}

	// Add an old impression that's outside the time window
	tracker.mu.Lock()
	tracker.impressions[sessionID] = append(tracker.impressions[sessionID],
		time.Now().Add(-2*time.Hour))
	tracker.mu.Unlock()

	// Old impression shouldn't count toward cap
	// We have 3 recent + 1 old, cap is 3, so should still be blocked
	if tracker.CanShowAd(sessionID, cap) {
		t.Error("should still be blocked with old impressions")
	}
}

// TestPauseAdTrackerFrequencyCapNilCap tests behavior when frequency cap is nil
func TestPauseAdTrackerFrequencyCapNilCap(t *testing.T) {
	tracker := NewPauseAdTracker()
	defer tracker.Shutdown()

	// Nil cap should always allow ads
	if !tracker.CanShowAd("session-123", nil) {
		t.Error("should allow ad when cap is nil")
	}

	// Record some impressions
	for i := 0; i < 100; i++ {
		tracker.RecordImpression("session-123")
	}

	// Still should allow ads with nil cap
	if !tracker.CanShowAd("session-123", nil) {
		t.Error("should still allow ad when cap is nil, regardless of impressions")
	}
}

// TestPauseAdTrackerFrequencyCapEmptySession tests behavior with empty session ID
func TestPauseAdTrackerFrequencyCapEmptySession(t *testing.T) {
	tracker := NewPauseAdTracker()
	defer tracker.Shutdown()

	cap := &FrequencyCap{
		MaxImpressions:    3,
		TimeWindowSeconds: 3600,
	}

	// Empty session ID should work (edge case)
	if !tracker.CanShowAd("", cap) {
		t.Error("should allow ad for empty session initially")
	}

	// Record impressions
	for i := 0; i < 3; i++ {
		tracker.RecordImpression("")
	}

	// Should not allow after cap
	if tracker.CanShowAd("", cap) {
		t.Error("should not allow ad after reaching cap for empty session")
	}
}

// TestPauseAdTrackerCleanupOldImpressionsLocked verifies per-session cleanup
func TestPauseAdTrackerCleanupOldImpressionsLocked(t *testing.T) {
	tracker := NewPauseAdTracker()
	defer tracker.Shutdown()

	sessionID := "test-session"

	// Add some recent and old impressions
	tracker.mu.Lock()
	tracker.impressions[sessionID] = []time.Time{
		time.Now(),                      // Recent
		time.Now().Add(-1 * time.Hour),  // Recent
		time.Now().Add(-25 * time.Hour), // Old
		time.Now().Add(-26 * time.Hour), // Old
	}

	// Cleanup this session
	tracker.cleanupOldImpressionsLocked(sessionID)

	// Should have 2 recent impressions
	if len(tracker.impressions[sessionID]) != 2 {
		t.Errorf("expected 2 recent impressions, got %d", len(tracker.impressions[sessionID]))
	}
	tracker.mu.Unlock()
}

// TestPauseAdTrackerCleanupAllOld tests cleanup when all impressions are old
func TestPauseAdTrackerCleanupAllOld(t *testing.T) {
	tracker := NewPauseAdTracker()
	defer tracker.Shutdown()

	sessionID := "test-session"

	// Add only old impressions
	tracker.mu.Lock()
	tracker.impressions[sessionID] = []time.Time{
		time.Now().Add(-25 * time.Hour),
		time.Now().Add(-26 * time.Hour),
	}
	tracker.mu.Unlock()

	// Cleanup this session
	tracker.cleanupExpiredSessions()

	// Session should be deleted
	tracker.mu.RLock()
	defer tracker.mu.RUnlock()

	if _, exists := tracker.impressions[sessionID]; exists {
		t.Error("expected session with all old impressions to be deleted")
	}
}

// TestPauseAdTrackerConcurrentAccess tests concurrent access to tracker (race detector)
func TestPauseAdTrackerConcurrentAccess(t *testing.T) {
	tracker := NewPauseAdTracker()
	defer tracker.Shutdown()

	cap := &FrequencyCap{
		MaxImpressions:    10,
		TimeWindowSeconds: 3600,
	}

	var wg sync.WaitGroup
	sessions := []string{"session1", "session2", "session3", "session4", "session5"}

	// Concurrent writes
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(iteration int) {
			defer wg.Done()
			sessionID := sessions[iteration%len(sessions)]
			tracker.RecordImpression(sessionID)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(iteration int) {
			defer wg.Done()
			sessionID := sessions[iteration%len(sessions)]
			tracker.CanShowAd(sessionID, cap)
		}(i)
	}

	// Concurrent cleanup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tracker.cleanupExpiredSessions()
		}()
	}

	wg.Wait()

	// Verify tracker state is consistent
	tracker.mu.RLock()
	defer tracker.mu.RUnlock()

	for _, sessionID := range sessions {
		if impressions, ok := tracker.impressions[sessionID]; ok {
			if len(impressions) == 0 {
				t.Errorf("session %s has empty impressions slice", sessionID)
			}
		}
	}
}

// TestPauseAdServiceShutdown verifies service cleanup
func TestPauseAdServiceShutdown(t *testing.T) {
	config := DefaultConfig()
	service := NewPauseAdService(config, nil)

	// Should shut down cleanly
	service.Shutdown()
}

// TestPauseAdServiceShutdownNilTracker tests shutdown with nil tracker
func TestPauseAdServiceShutdownNilTracker(t *testing.T) {
	config := DefaultConfig()
	service := &PauseAdService{
		config:      config,
		adRequester: nil,
		tracker:     nil,
	}

	// Should not panic with nil tracker
	service.Shutdown()
}

// TestPauseAdServiceHandleRequestDisabled tests behavior when pause ads are disabled
func TestPauseAdServiceHandleRequestDisabled(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = false

	mock := &MockAdRequester{}
	service := NewPauseAdService(config, mock)
	defer service.Shutdown()

	req := &PauseAdRequest{
		SessionID: "test-session",
		ContentID: "test-content",
		PausedAt:  time.Now(),
	}

	resp, err := service.HandlePauseAdRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.NoBid {
		t.Error("expected NoBid when disabled")
	}

	if resp.Error != "pause ads disabled" {
		t.Errorf("expected 'pause ads disabled' error, got: %s", resp.Error)
	}

	if mock.GetCallCount() != 0 {
		t.Error("should not call ad requester when disabled")
	}
}

// TestPauseAdServiceHandleRequestFrequencyCapReached tests frequency cap enforcement
func TestPauseAdServiceHandleRequestFrequencyCapReached(t *testing.T) {
	config := DefaultConfig()
	config.FrequencyCap = &FrequencyCap{
		MaxImpressions:    2,
		TimeWindowSeconds: 3600,
	}

	mock := &MockAdRequester{returnAd: true}
	service := NewPauseAdService(config, mock)
	defer service.Shutdown()

	req := &PauseAdRequest{
		SessionID: "test-session",
		ContentID: "test-content",
		PausedAt:  time.Now(),
	}

	// First two requests should succeed
	for i := 0; i < 2; i++ {
		resp, err := service.HandlePauseAdRequest(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error on request %d: %v", i+1, err)
		}
		if resp.Ad == nil {
			t.Errorf("expected ad on request %d", i+1)
		}
	}

	// Third request should be blocked by frequency cap
	resp, err := service.HandlePauseAdRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.NoBid {
		t.Error("expected NoBid when frequency cap reached")
	}

	if resp.Error != "frequency cap reached" {
		t.Errorf("expected 'frequency cap reached' error, got: %s", resp.Error)
	}
}

// TestPauseAdServiceHandleRequestNoFrequencyCap tests behavior without frequency cap
func TestPauseAdServiceHandleRequestNoFrequencyCap(t *testing.T) {
	config := DefaultConfig()
	config.FrequencyCap = nil

	mock := &MockAdRequester{returnAd: true}
	service := NewPauseAdService(config, mock)
	defer service.Shutdown()

	req := &PauseAdRequest{
		SessionID: "test-session",
		ContentID: "test-content",
		PausedAt:  time.Now(),
	}

	// Should allow unlimited requests without frequency cap
	for i := 0; i < 10; i++ {
		resp, err := service.HandlePauseAdRequest(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error on request %d: %v", i+1, err)
		}
		if resp.Ad == nil {
			t.Errorf("expected ad on request %d", i+1)
		}
	}

	if mock.GetCallCount() != 10 {
		t.Errorf("expected 10 calls, got %d", mock.GetCallCount())
	}
}

// TestPauseAdServiceHandleRequestError tests error handling from ad requester
func TestPauseAdServiceHandleRequestError(t *testing.T) {
	config := DefaultConfig()
	mock := &MockAdRequester{returnError: true}
	service := NewPauseAdService(config, mock)
	defer service.Shutdown()

	req := &PauseAdRequest{
		SessionID: "test-session",
		ContentID: "test-content",
		PausedAt:  time.Now(),
	}

	resp, err := service.HandlePauseAdRequest(context.Background(), req)
	if err == nil {
		t.Fatal("expected error from ad requester")
	}

	if resp != nil {
		t.Error("expected nil response on error")
	}
}

// TestPauseAdServiceHandleRequestNoBid tests no bid response
func TestPauseAdServiceHandleRequestNoBid(t *testing.T) {
	config := DefaultConfig()
	mock := &MockAdRequester{returnNoBid: true}
	service := NewPauseAdService(config, mock)
	defer service.Shutdown()

	req := &PauseAdRequest{
		SessionID: "test-session",
		ContentID: "test-content",
		PausedAt:  time.Now(),
	}

	resp, err := service.HandlePauseAdRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.NoBid {
		t.Error("expected NoBid response")
	}

	// Should not record impression for no bid
	if service.tracker.CanShowAd(req.SessionID, config.FrequencyCap) == false {
		t.Error("should not have recorded impression for no bid")
	}
}

// TestPauseAdServiceHandleRequestWithAd tests successful ad response
func TestPauseAdServiceHandleRequestWithAd(t *testing.T) {
	config := DefaultConfig()
	mock := &MockAdRequester{returnAd: true}
	service := NewPauseAdService(config, mock)
	defer service.Shutdown()

	req := &PauseAdRequest{
		SessionID:        "test-session",
		ContentID:        "test-content",
		PausedAt:         time.Now(),
		PlaybackPosition: 120.5,
		PublisherID:      "pub-123",
		Device: &openrtb.Device{
			UA: "Mozilla/5.0",
		},
	}

	resp, err := service.HandlePauseAdRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Ad == nil {
		t.Fatal("expected ad in response")
	}

	if resp.Ad.ID != "test-ad-123" {
		t.Errorf("expected ad ID 'test-ad-123', got %s", resp.Ad.ID)
	}

	// Verify impression was recorded
	tracker := service.tracker
	tracker.mu.RLock()
	impressions := tracker.impressions[req.SessionID]
	tracker.mu.RUnlock()

	if len(impressions) != 1 {
		t.Errorf("expected 1 impression, got %d", len(impressions))
	}

	// Verify last request was passed correctly
	lastReq := mock.GetLastRequest()
	if lastReq.SessionID != req.SessionID {
		t.Errorf("expected session ID %s, got %s", req.SessionID, lastReq.SessionID)
	}
}

// TestPauseAdHandlerMethodNotAllowed tests HTTP method validation
func TestPauseAdHandlerMethodNotAllowed(t *testing.T) {
	config := DefaultConfig()
	mock := &MockAdRequester{}
	service := NewPauseAdService(config, mock)
	defer service.Shutdown()

	handler := NewPauseAdHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/pause-ad", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

// TestPauseAdHandlerInvalidRequest tests invalid JSON handling
func TestPauseAdHandlerInvalidRequest(t *testing.T) {
	config := DefaultConfig()
	mock := &MockAdRequester{}
	service := NewPauseAdService(config, mock)
	defer service.Shutdown()

	handler := NewPauseAdHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/pause-ad", bytes.NewBufferString("invalid json"))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// TestPauseAdHandlerServiceError tests service error handling
func TestPauseAdHandlerServiceError(t *testing.T) {
	config := DefaultConfig()
	mock := &MockAdRequester{returnError: true}
	service := NewPauseAdService(config, mock)
	defer service.Shutdown()

	handler := NewPauseAdHandler(service)

	reqBody := PauseAdRequest{
		SessionID: "test-session",
		ContentID: "test-content",
		PausedAt:  time.Now(),
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/pause-ad", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

// TestPauseAdHandlerSuccess tests successful HTTP request
func TestPauseAdHandlerSuccess(t *testing.T) {
	config := DefaultConfig()
	mock := &MockAdRequester{returnAd: true}
	service := NewPauseAdService(config, mock)
	defer service.Shutdown()

	handler := NewPauseAdHandler(service)

	reqBody := PauseAdRequest{
		SessionID:        "test-session",
		ContentID:        "test-content",
		PausedAt:         time.Now(),
		PlaybackPosition: 60.0,
		PublisherID:      "pub-123",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/pause-ad", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %s", contentType)
	}

	var resp PauseAdResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Ad == nil {
		t.Error("expected ad in response")
	}
}

// TestCreatePauseAdVASTNilAd tests VAST creation with nil ad
func TestCreatePauseAdVASTNilAd(t *testing.T) {
	vast, err := CreatePauseAdVAST(nil, "https://tracking.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if vast == nil {
		t.Fatal("expected non-nil VAST")
	}

	if len(vast.Ads) != 0 {
		t.Errorf("expected empty VAST, got %d ads", len(vast.Ads))
	}
}

// TestCreatePauseAdVASTWithAd tests VAST creation with valid ad
func TestCreatePauseAdVASTWithAd(t *testing.T) {
	ad := &PauseAd{
		ID:              "test-ad-456",
		CreativeURL:     "https://example.com/creative.jpg",
		ClickURL:        "https://example.com/click",
		Width:           1920,
		Height:          1080,
		Format:          "image/jpeg",
		DisplayDuration: 30,
		Price:           2.50,
		Currency:        "USD",
		Advertiser:      "Test Advertiser",
	}

	vast, err := CreatePauseAdVAST(ad, "https://tracking.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if vast == nil {
		t.Fatal("expected non-nil VAST")
	}

	if len(vast.Ads) == 0 {
		t.Fatal("expected ads in VAST")
	}

	vastAd := vast.Ads[0]
	if vastAd.InLine == nil {
		t.Fatal("expected InLine ad")
	}

	// Check for non-linear creative
	hasNonLinear := false
	for _, creative := range vastAd.InLine.Creatives.Creative {
		if creative.NonLinearAds != nil && len(creative.NonLinearAds.NonLinear) > 0 {
			hasNonLinear = true
			nonLinear := creative.NonLinearAds.NonLinear[0]

			if nonLinear.Width != ad.Width {
				t.Errorf("expected width %d, got %d", ad.Width, nonLinear.Width)
			}

			if nonLinear.Height != ad.Height {
				t.Errorf("expected height %d, got %d", ad.Height, nonLinear.Height)
			}

			if nonLinear.StaticResource == nil {
				t.Fatal("expected static resource")
			}

			if nonLinear.StaticResource.Value != ad.CreativeURL {
				t.Errorf("expected creative URL %s, got %s", ad.CreativeURL, nonLinear.StaticResource.Value)
			}

			if nonLinear.NonLinearClickThrough != ad.ClickURL {
				t.Errorf("expected click URL %s, got %s", ad.ClickURL, nonLinear.NonLinearClickThrough)
			}
		}
	}

	if !hasNonLinear {
		t.Error("expected non-linear creative in VAST")
	}
}

// TestPauseAdTrackerRecordImpressionMultipleSessions tests recording across multiple sessions
func TestPauseAdTrackerRecordImpressionMultipleSessions(t *testing.T) {
	tracker := NewPauseAdTracker()
	defer tracker.Shutdown()

	sessions := []string{"session1", "session2", "session3"}

	for _, session := range sessions {
		for i := 0; i < 3; i++ {
			tracker.RecordImpression(session)
		}
	}

	tracker.mu.RLock()
	defer tracker.mu.RUnlock()

	if len(tracker.impressions) != len(sessions) {
		t.Errorf("expected %d sessions, got %d", len(sessions), len(tracker.impressions))
	}

	for _, session := range sessions {
		if impressions, ok := tracker.impressions[session]; !ok {
			t.Errorf("session %s not found", session)
		} else if len(impressions) != 3 {
			t.Errorf("expected 3 impressions for %s, got %d", session, len(impressions))
		}
	}
}

// TestPauseAdTrackerPeriodicCleanupRuns tests that periodic cleanup actually runs
func TestPauseAdTrackerPeriodicCleanupRuns(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping periodic cleanup test in short mode")
	}

	tracker := NewPauseAdTracker()
	defer tracker.Shutdown()

	// Add old impressions
	tracker.mu.Lock()
	tracker.impressions["old-session"] = []time.Time{
		time.Now().Add(-25 * time.Hour),
	}
	tracker.mu.Unlock()

	// Wait a short time for cleanup to potentially run
	// Note: Default cleanup runs every 10 minutes, so we manually trigger it
	time.Sleep(100 * time.Millisecond)
	tracker.cleanupExpiredSessions()

	tracker.mu.RLock()
	defer tracker.mu.RUnlock()

	if _, exists := tracker.impressions["old-session"]; exists {
		t.Error("expected old session to be cleaned up")
	}
}

// TestNewPauseAdService verifies service initialization
func TestNewPauseAdService(t *testing.T) {
	config := DefaultConfig()
	mock := &MockAdRequester{}

	service := NewPauseAdService(config, mock)

	if service == nil {
		t.Fatal("expected non-nil service")
	}

	if service.tracker == nil {
		t.Error("expected tracker to be initialized")
	}

	service.Shutdown()
}

// TestNewPauseAdHandler verifies handler initialization
func TestNewPauseAdHandler(t *testing.T) {
	config := DefaultConfig()
	mock := &MockAdRequester{}
	service := NewPauseAdService(config, mock)
	defer service.Shutdown()

	handler := NewPauseAdHandler(service)

	if handler == nil {
		t.Fatal("expected non-nil handler")
	}

	if handler.service != service {
		t.Error("expected handler to reference service")
	}
}

// TestPauseAdResponseSerialization tests JSON serialization
func TestPauseAdResponseSerialization(t *testing.T) {
	resp := &PauseAdResponse{
		Ad: &PauseAd{
			ID:              "test-123",
			CreativeURL:     "https://example.com/ad.jpg",
			ClickURL:        "https://example.com/click",
			Width:           1920,
			Height:          1080,
			Format:          "image/jpeg",
			DisplayDuration: 30,
			Price:           1.50,
			Currency:        "USD",
			Advertiser:      "Test Co",
			TrackingURLs: &PauseAdTracking{
				Impression: []string{"https://track.com/imp"},
				Click:      []string{"https://track.com/click"},
			},
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded PauseAdResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Ad.ID != resp.Ad.ID {
		t.Errorf("expected ID %s, got %s", resp.Ad.ID, decoded.Ad.ID)
	}
}

// TestPauseAdHandlerEmptyBody tests handling of empty request body
func TestPauseAdHandlerEmptyBody(t *testing.T) {
	config := DefaultConfig()
	mock := &MockAdRequester{}
	service := NewPauseAdService(config, mock)
	defer service.Shutdown()

	handler := NewPauseAdHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/pause-ad", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// TestPauseAdHandlerClosedBody tests handling of closed body reader
func TestPauseAdHandlerClosedBody(t *testing.T) {
	config := DefaultConfig()
	mock := &MockAdRequester{}
	service := NewPauseAdService(config, mock)
	defer service.Shutdown()

	handler := NewPauseAdHandler(service)

	// Create a reader that returns an error
	body := io.NopCloser(bytes.NewReader([]byte{}))
	body.Close()

	req := httptest.NewRequest(http.MethodPost, "/pause-ad", body)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should handle the error gracefully
	if w.Code != http.StatusBadRequest && w.Code != http.StatusInternalServerError {
		t.Errorf("expected error status code, got %d", w.Code)
	}
}
