package vast

import (
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	vastXML := `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="4.0">
  <Ad id="ad-123">
    <InLine>
      <AdSystem>TNEVideo</AdSystem>
      <AdTitle>Test Ad</AdTitle>
      <Impression id="imp1"><![CDATA[https://example.com/impression]]></Impression>
      <Creatives>
        <Creative id="creative-1">
          <Linear>
            <Duration>00:00:30</Duration>
            <MediaFiles>
              <MediaFile delivery="progressive" type="video/mp4" width="1920" height="1080"><![CDATA[https://example.com/video.mp4]]></MediaFile>
            </MediaFiles>
          </Linear>
        </Creative>
      </Creatives>
    </InLine>
  </Ad>
</VAST>`

	v, err := Parse([]byte(vastXML))
	if err != nil {
		t.Fatalf("Failed to parse VAST: %v", err)
	}

	if v.Version != "4.0" {
		t.Errorf("Expected version 4.0, got %s", v.Version)
	}

	if len(v.Ads) != 1 {
		t.Fatalf("Expected 1 ad, got %d", len(v.Ads))
	}

	ad := v.Ads[0]
	if ad.ID != "ad-123" {
		t.Errorf("Expected ad ID ad-123, got %s", ad.ID)
	}

	if ad.InLine == nil {
		t.Fatal("Expected InLine ad")
	}

	if ad.InLine.AdTitle != "Test Ad" {
		t.Errorf("Expected ad title 'Test Ad', got %s", ad.InLine.AdTitle)
	}
}

func TestBuilder(t *testing.T) {
	v, err := NewBuilder("4.0").
		AddAd("test-ad").
		WithInLine("TNEVideo", "Test Ad").
		WithImpression("https://example.com/impression", "imp1").
		WithError("https://example.com/error").
		WithLinearCreative("creative-1", 30*time.Second).
		WithMediaFile("https://example.com/video.mp4", "video/mp4", 1920, 1080).
		WithTracking(EventStart, "https://example.com/start").
		WithTracking(EventComplete, "https://example.com/complete").
		WithClickThrough("https://example.com/click").
		EndLinear().
		Done().
		Build()

	if err != nil {
		t.Fatalf("Failed to build VAST: %v", err)
	}

	if v.Version != "4.0" {
		t.Errorf("Expected version 4.0, got %s", v.Version)
	}

	if len(v.Ads) != 1 {
		t.Fatalf("Expected 1 ad, got %d", len(v.Ads))
	}

	if v.Ads[0].InLine == nil {
		t.Fatal("Expected InLine ad")
	}

	linear := v.GetLinearCreative()
	if linear == nil {
		t.Fatal("Expected linear creative")
	}

	if len(linear.MediaFiles.MediaFile) != 1 {
		t.Fatalf("Expected 1 media file, got %d", len(linear.MediaFiles.MediaFile))
	}

	mf := linear.MediaFiles.MediaFile[0]
	if mf.Width != 1920 || mf.Height != 1080 {
		t.Errorf("Expected 1920x1080, got %dx%d", mf.Width, mf.Height)
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		hasError bool
	}{
		{"00:00:30", 30 * time.Second, false},
		{"00:01:00", 1 * time.Minute, false},
		{"01:00:00", 1 * time.Hour, false},
		{"00:30:00", 30 * time.Minute, false},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		d, err := ParseDuration(tt.input)
		if tt.hasError {
			if err == nil {
				t.Errorf("Expected error for input %s", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for input %s: %v", tt.input, err)
			}
			if d != tt.expected {
				t.Errorf("Expected %v for input %s, got %v", tt.expected, tt.input, d)
			}
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input    time.Duration
		expected string
	}{
		{30 * time.Second, "00:00:30"},
		{1 * time.Minute, "00:01:00"},
		{1 * time.Hour, "01:00:00"},
		{90 * time.Second, "00:01:30"},
	}

	for _, tt := range tests {
		result := FormatDuration(tt.input)
		if result != tt.expected {
			t.Errorf("Expected %s for input %v, got %s", tt.expected, tt.input, result)
		}
	}
}

func TestMarshal(t *testing.T) {
	v := &VAST{
		Version: "4.0",
		Ads: []Ad{
			{
				ID: "test-ad",
				InLine: &InLine{
					AdSystem: AdSystem{Value: "TNEVideo"},
					AdTitle:  "Test",
					Creatives: Creatives{
						Creative: []Creative{},
					},
				},
			},
		},
	}

	data, err := v.Marshal()
	if err != nil {
		t.Fatalf("Failed to marshal VAST: %v", err)
	}

	if len(data) == 0 {
		t.Error("Expected non-empty output")
	}

	// Verify it can be parsed back
	parsed, err := Parse(data)
	if err != nil {
		t.Fatalf("Failed to parse marshaled VAST: %v", err)
	}

	if parsed.Version != v.Version {
		t.Errorf("Version mismatch after round-trip")
	}
}

func TestIsEmpty(t *testing.T) {
	empty := &VAST{Version: "4.0"}
	if !empty.IsEmpty() {
		t.Error("Expected empty VAST")
	}

	notEmpty := &VAST{
		Version: "4.0",
		Ads:     []Ad{{ID: "test"}},
	}
	if notEmpty.IsEmpty() {
		t.Error("Expected non-empty VAST")
	}
}

func TestCreateEmptyVAST(t *testing.T) {
	v := CreateEmptyVAST()
	if v.Version != "4.0" {
		t.Errorf("Expected version 4.0, got %s", v.Version)
	}
	if !v.IsEmpty() {
		t.Error("Expected empty VAST")
	}
}

func TestCreateErrorVAST(t *testing.T) {
	v := CreateErrorVAST("https://example.com/error")
	if v.Error != "https://example.com/error" {
		t.Errorf("Expected error URL, got %s", v.Error)
	}
}
