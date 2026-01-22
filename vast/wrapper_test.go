package vast

import (
	"strings"
	"testing"
)

func TestNewDefaultWrapper(t *testing.T) {
	vast, err := NewDefaultWrapper(
		"TestAdServer",
		"http://example.com/vast-tag",
		[]string{"http://example.com/impression1", "http://example.com/impression2"},
	)

	if err != nil {
		t.Fatalf("Failed to create default wrapper: %v", err)
	}

	if vast.Version != "4.2" {
		t.Errorf("Expected version 4.2, got %s", vast.Version)
	}

	if len(vast.Ads) != 1 {
		t.Fatalf("Expected 1 ad, got %d", len(vast.Ads))
	}

	wrapper := vast.Ads[0].Wrapper
	if wrapper == nil {
		t.Fatal("Expected wrapper ad")
	}

	if wrapper.AdSystem.Value != "TestAdServer" {
		t.Errorf("Expected AdSystem TestAdServer, got %s", wrapper.AdSystem.Value)
	}

	if wrapper.VASTAdTagURI.Value != "http://example.com/vast-tag" {
		t.Errorf("Expected VASTAdTagURI http://example.com/vast-tag, got %s", wrapper.VASTAdTagURI.Value)
	}

	if len(wrapper.Impressions) != 2 {
		t.Fatalf("Expected 2 impressions, got %d", len(wrapper.Impressions))
	}

	expectedImpressions := []string{
		"http://example.com/impression1",
		"http://example.com/impression2",
	}

	for i, imp := range wrapper.Impressions {
		if imp.Value != expectedImpressions[i] {
			t.Errorf("Expected impression %s, got %s", expectedImpressions[i], imp.Value)
		}
	}
}

func TestNewDefaultWrapperXML(t *testing.T) {
	xmlStr, err := NewDefaultWrapperXML(
		"TestAdServer",
		"http://example.com/vast-tag",
		[]string{"http://example.com/impression"},
	)

	if err != nil {
		t.Fatalf("Failed to create default wrapper XML: %v", err)
	}

	if !strings.Contains(xmlStr, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Error("Expected XML header")
	}

	if !strings.Contains(xmlStr, `<VAST`) {
		t.Error("Expected VAST element")
	}

	if !strings.Contains(xmlStr, `version="4.2"`) {
		t.Error("Expected version 4.2")
	}

	if !strings.Contains(xmlStr, `<Wrapper>`) {
		t.Error("Expected Wrapper element")
	}

	if !strings.Contains(xmlStr, `TestAdServer`) {
		t.Error("Expected AdSystem value")
	}

	// Verify it parses back correctly
	vast, err := Parse(xmlStr)
	if err != nil {
		t.Fatalf("Failed to parse generated XML: %v", err)
	}

	if vast.Version != "4.2" {
		t.Errorf("Expected version 4.2 after parse, got %s", vast.Version)
	}
}

func TestWrapperBuilder(t *testing.T) {
	config := WrapperConfig{
		AdID:            "ad-123",
		AdSystem:        "TestSystem",
		AdSystemVersion: "2.0",
		AdTitle:         "Test Wrapper",
		VASTAdTagURI:    "http://example.com/vast",
		ImpressionURLs: []string{
			"http://example.com/imp1",
			"http://example.com/imp2",
		},
		ErrorURL: "http://example.com/error",
		TrackingEvents: map[string][]string{
			"start":    {"http://example.com/start1", "http://example.com/start2"},
			"complete": {"http://example.com/complete"},
		},
	}

	vast, err := NewWrapperBuilder("4.2").AddWrapperAd(config).Build()
	if err != nil {
		t.Fatalf("Failed to build wrapper: %v", err)
	}

	if len(vast.Ads) != 1 {
		t.Fatalf("Expected 1 ad, got %d", len(vast.Ads))
	}

	ad := vast.Ads[0]
	if ad.ID != "ad-123" {
		t.Errorf("Expected ad ID ad-123, got %s", ad.ID)
	}

	wrapper := ad.Wrapper
	if wrapper == nil {
		t.Fatal("Expected wrapper")
	}

	if wrapper.AdTitle != "Test Wrapper" {
		t.Errorf("Expected AdTitle 'Test Wrapper', got %s", wrapper.AdTitle)
	}

	if wrapper.AdSystem.Version != "2.0" {
		t.Errorf("Expected AdSystem version 2.0, got %s", wrapper.AdSystem.Version)
	}

	if len(wrapper.Impressions) != 2 {
		t.Fatalf("Expected 2 impressions, got %d", len(wrapper.Impressions))
	}

	if wrapper.Error == nil || wrapper.Error.Value != "http://example.com/error" {
		t.Error("Expected error URL")
	}

	if len(wrapper.Creatives) != 1 {
		t.Fatalf("Expected 1 creative, got %d", len(wrapper.Creatives))
	}

	linear := wrapper.Creatives[0].Linear
	if linear == nil {
		t.Fatal("Expected linear creative")
	}

	// Check tracking events
	trackingEventCount := make(map[string]int)
	for _, tracking := range linear.TrackingEvents {
		trackingEventCount[tracking.Event]++
	}

	if trackingEventCount["start"] != 2 {
		t.Errorf("Expected 2 start events, got %d", trackingEventCount["start"])
	}

	if trackingEventCount["complete"] != 1 {
		t.Errorf("Expected 1 complete event, got %d", trackingEventCount["complete"])
	}
}

func TestWrapperBuilderMultipleAds(t *testing.T) {
	builder := NewWrapperBuilder("4.2")

	config1 := WrapperConfig{
		AdID:           "ad-1",
		AdSystem:       "System1",
		VASTAdTagURI:   "http://example.com/vast1",
		ImpressionURLs: []string{"http://example.com/imp1"},
	}

	config2 := WrapperConfig{
		AdID:           "ad-2",
		AdSystem:       "System2",
		VASTAdTagURI:   "http://example.com/vast2",
		ImpressionURLs: []string{"http://example.com/imp2"},
	}

	vast, err := builder.AddWrapperAd(config1).AddWrapperAd(config2).Build()
	if err != nil {
		t.Fatalf("Failed to build multiple wrapper ads: %v", err)
	}

	if len(vast.Ads) != 2 {
		t.Fatalf("Expected 2 ads, got %d", len(vast.Ads))
	}

	if vast.Ads[0].ID != "ad-1" {
		t.Errorf("Expected first ad ID ad-1, got %s", vast.Ads[0].ID)
	}

	if vast.Ads[1].ID != "ad-2" {
		t.Errorf("Expected second ad ID ad-2, got %s", vast.Ads[1].ID)
	}
}

func TestWrapperBuilderValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      WrapperConfig
		expectError bool
	}{
		{
			name: "missing VASTAdTagURI",
			config: WrapperConfig{
				AdSystem:       "Test",
				ImpressionURLs: []string{"http://example.com/imp"},
			},
			expectError: true,
		},
		{
			name: "missing impressions",
			config: WrapperConfig{
				AdSystem:     "Test",
				VASTAdTagURI: "http://example.com/vast",
			},
			expectError: true,
		},
		{
			name: "valid config",
			config: WrapperConfig{
				AdSystem:       "Test",
				VASTAdTagURI:   "http://example.com/vast",
				ImpressionURLs: []string{"http://example.com/imp"},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewWrapperBuilder("4.2").AddWrapperAd(tt.config).Build()
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestDefaultWrapperValidation(t *testing.T) {
	tests := []struct {
		name        string
		adSystem    string
		vastTagURI  string
		impressions []string
		expectError bool
	}{
		{
			name:        "missing adSystem",
			adSystem:    "",
			vastTagURI:  "http://example.com/vast",
			impressions: []string{"http://example.com/imp"},
			expectError: true,
		},
		{
			name:        "missing vastTagURI",
			adSystem:    "Test",
			vastTagURI:  "",
			impressions: []string{"http://example.com/imp"},
			expectError: true,
		},
		{
			name:        "missing impressions",
			adSystem:    "Test",
			vastTagURI:  "http://example.com/vast",
			impressions: []string{},
			expectError: true,
		},
		{
			name:        "valid wrapper",
			adSystem:    "Test",
			vastTagURI:  "http://example.com/vast",
			impressions: []string{"http://example.com/imp"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewDefaultWrapper(tt.adSystem, tt.vastTagURI, tt.impressions)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}
