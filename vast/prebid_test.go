package vast

import (
	"strings"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
)

func TestMakeVASTFromBid_WithVASTInAdM(t *testing.T) {
	bid := &openrtb2.Bid{
		ID: "bid123",
		AdM: `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="4.2">
  <Ad>
    <InLine>
      <AdSystem>TestSystem</AdSystem>
      <AdTitle>Test Ad</AdTitle>
      <Impression><![CDATA[http://example.com/imp]]></Impression>
      <Creatives>
        <Creative>
          <Linear>
            <Duration>00:00:30</Duration>
            <MediaFiles>
              <MediaFile delivery="progressive" type="video/mp4">
                <![CDATA[http://example.com/video.mp4]]>
              </MediaFile>
            </MediaFiles>
          </Linear>
        </Creative>
      </Creatives>
    </InLine>
  </Ad>
</VAST>`,
	}

	config := PrebidVASTConfig{
		Bid: bid,
		ImpressionTrackers: []string{"http://prebid.com/imp"},
	}

	vastXML, err := MakeVASTFromBid(config)
	if err != nil {
		t.Fatalf("Failed to make VAST from bid: %v", err)
	}

	// Verify tracking was injected
	parsed, err := Parse(vastXML)
	if err != nil {
		t.Fatalf("Failed to parse generated VAST: %v", err)
	}

	impressions := parsed.GetImpressionURLs()
	if len(impressions) != 2 {
		t.Errorf("Expected 2 impressions, got %d", len(impressions))
	}
}

func TestMakeVASTFromBid_WithNURL(t *testing.T) {
	bid := &openrtb2.Bid{
		ID:    "bid456",
		NURL:  "http://ad-server.com/vast-tag",
		Price: 2.50,
	}

	config := PrebidVASTConfig{
		Bid: bid,
		ImpressionTrackers: []string{
			"http://prebid.com/impression",
		},
		EventTrackers: map[string][]string{
			"start":    {"http://prebid.com/start"},
			"complete": {"http://prebid.com/complete"},
		},
	}

	vastXML, err := MakeVASTFromBid(config)
	if err != nil {
		t.Fatalf("Failed to make VAST from bid: %v", err)
	}

	parsed, err := Parse(vastXML)
	if err != nil {
		t.Fatalf("Failed to parse generated VAST: %v", err)
	}

	if parsed.Version != "4.2" {
		t.Errorf("Expected version 4.2, got %s", parsed.Version)
	}

	if len(parsed.Ads) != 1 {
		t.Fatalf("Expected 1 ad, got %d", len(parsed.Ads))
	}

	wrapper := parsed.Ads[0].Wrapper
	if wrapper == nil {
		t.Fatal("Expected wrapper ad")
	}

	if wrapper.VASTAdTagURI.Value != bid.NURL {
		t.Errorf("Expected VAST tag URI %s, got %s", bid.NURL, wrapper.VASTAdTagURI.Value)
	}

	// Check title includes price
	if !strings.Contains(wrapper.AdTitle, "$2.50") {
		t.Errorf("Expected AdTitle to contain price, got: %s", wrapper.AdTitle)
	}
}

func TestMakeVASTWrapper(t *testing.T) {
	bid := &openrtb2.Bid{
		ID:   "wrapper-bid",
		NURL: "http://example.com/vast",
	}

	vastXML, err := MakeVASTWrapper(bid, "", []string{"http://prebid.com/imp"})
	if err != nil {
		t.Fatalf("Failed to make VAST wrapper: %v", err)
	}

	parsed, err := Parse(vastXML)
	if err != nil {
		t.Fatalf("Failed to parse wrapper: %v", err)
	}

	if parsed.Ads[0].Wrapper == nil {
		t.Fatal("Expected wrapper ad")
	}

	if parsed.Ads[0].Wrapper.VASTAdTagURI.Value != bid.NURL {
		t.Errorf("Expected NURL in wrapper, got %s", parsed.Ads[0].Wrapper.VASTAdTagURI.Value)
	}
}

func TestInjectPrebidTracking(t *testing.T) {
	originalVAST := `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="4.2">
  <Ad>
    <InLine>
      <AdSystem>Test</AdSystem>
      <AdTitle>Test</AdTitle>
      <Impression><![CDATA[http://original.com/imp]]></Impression>
      <Creatives>
        <Creative>
          <Linear>
            <Duration>00:00:15</Duration>
            <MediaFiles>
              <MediaFile delivery="progressive" type="video/mp4">
                <![CDATA[http://example.com/video.mp4]]>
              </MediaFile>
            </MediaFiles>
          </Linear>
        </Creative>
      </Creatives>
    </InLine>
  </Ad>
</VAST>`

	config := PrebidVASTConfig{
		ImpressionTrackers: []string{"http://prebid.com/imp"},
		EventTrackers: map[string][]string{
			"start":    {"http://prebid.com/start"},
			"midpoint": {"http://prebid.com/midpoint"},
			"complete": {"http://prebid.com/complete"},
		},
		ClickTrackers: []string{"http://prebid.com/click"},
		ErrorTracker:  "http://prebid.com/error",
	}

	modifiedVAST, err := InjectPrebidTracking(originalVAST, config)
	if err != nil {
		t.Fatalf("Failed to inject tracking: %v", err)
	}

	parsed, err := Parse(modifiedVAST)
	if err != nil {
		t.Fatalf("Failed to parse modified VAST: %v", err)
	}

	// Verify impressions
	impressions := parsed.GetImpressionURLs()
	if len(impressions) != 2 {
		t.Errorf("Expected 2 impressions, got %d", len(impressions))
	}

	// Verify tracking events
	startEvents := parsed.GetTrackingEvents("start")
	if len(startEvents) != 1 {
		t.Errorf("Expected 1 start event, got %d", len(startEvents))
	}

	completeEvents := parsed.GetTrackingEvents("complete")
	if len(completeEvents) != 1 {
		t.Errorf("Expected 1 complete event, got %d", len(completeEvents))
	}
}

func TestGetVideoMetadata(t *testing.T) {
	vastXML := `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="4.2">
  <Ad>
    <InLine>
      <AdSystem>Test</AdSystem>
      <AdTitle>Test</AdTitle>
      <Impression><![CDATA[http://example.com/imp1]]></Impression>
      <Impression><![CDATA[http://example.com/imp2]]></Impression>
      <Creatives>
        <Creative>
          <Linear>
            <Duration>00:00:30</Duration>
            <VideoClicks>
              <ClickThrough><![CDATA[http://example.com/clickthrough]]></ClickThrough>
            </VideoClicks>
            <MediaFiles>
              <MediaFile delivery="progressive" type="video/mp4" width="1920" height="1080">
                <![CDATA[http://example.com/video.mp4]]>
              </MediaFile>
            </MediaFiles>
          </Linear>
        </Creative>
      </Creatives>
    </InLine>
  </Ad>
</VAST>`

	metadata, err := GetVideoMetadata(vastXML)
	if err != nil {
		t.Fatalf("Failed to get metadata: %v", err)
	}

	if metadata.Duration != "00:00:30" {
		t.Errorf("Expected duration 00:00:30, got %s", metadata.Duration)
	}

	if metadata.Width != 1920 {
		t.Errorf("Expected width 1920, got %d", metadata.Width)
	}

	if metadata.Height != 1080 {
		t.Errorf("Expected height 1080, got %d", metadata.Height)
	}

	if metadata.MediaFileURL != "http://example.com/video.mp4" {
		t.Errorf("Expected media file URL http://example.com/video.mp4, got %s", metadata.MediaFileURL)
	}

	if metadata.MediaType != "video/mp4" {
		t.Errorf("Expected media type video/mp4, got %s", metadata.MediaType)
	}

	if len(metadata.ImpressionURLs) != 2 {
		t.Errorf("Expected 2 impression URLs, got %d", len(metadata.ImpressionURLs))
	}

	if metadata.ClickThrough != "http://example.com/clickthrough" {
		t.Errorf("Expected clickthrough URL, got %s", metadata.ClickThrough)
	}
}

func TestDurationToSeconds(t *testing.T) {
	tests := []struct {
		duration string
		expected int
		hasError bool
	}{
		{"00:00:30", 30, false},
		{"00:01:00", 60, false},
		{"01:00:00", 3600, false},
		{"01:30:45", 5445, false},
		{"00:00:15.500", 15, false},
		{"", 0, true},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.duration, func(t *testing.T) {
			seconds, err := DurationToSeconds(tt.duration)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error for duration %s", tt.duration)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for duration %s: %v", tt.duration, err)
				}
				if seconds != tt.expected {
					t.Errorf("Expected %d seconds, got %d for duration %s", tt.expected, seconds, tt.duration)
				}
			}
		})
	}
}

func TestSecondsToDuration(t *testing.T) {
	tests := []struct {
		seconds  int
		expected string
	}{
		{30, "00:00:30"},
		{60, "00:01:00"},
		{3600, "01:00:00"},
		{5445, "01:30:45"},
		{0, "00:00:00"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			duration := SecondsToDuration(tt.seconds)
			if duration != tt.expected {
				t.Errorf("Expected %s, got %s for %d seconds", tt.expected, duration, tt.seconds)
			}
		})
	}
}

func TestIsVASTXML(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "valid VAST",
			content:  `<?xml version="1.0"?><VAST version="4.2"></VAST>`,
			expected: true,
		},
		{
			name:     "VAST lowercase",
			content:  `<vast version="4.2"></vast>`,
			expected: true,
		},
		{
			name:     "not VAST",
			content:  `<html><body>Not VAST</body></html>`,
			expected: false,
		},
		{
			name:     "empty string",
			content:  ``,
			expected: false,
		},
		{
			name:     "plain text",
			content:  `This is just plain text`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isVASTXML(tt.content)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for content: %s", tt.expected, result, tt.content)
			}
		})
	}
}
