package vast

import (
	"strings"
	"testing"
)

// Test VAST 2.0 parsing
func TestParseVAST20(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="2.0">
  <Ad id="12345">
    <InLine>
      <AdSystem version="1.0">TestAdServer</AdSystem>
      <AdTitle>Test Ad Title</AdTitle>
      <Impression><![CDATA[http://example.com/impression]]></Impression>
      <Creatives>
        <Creative>
          <Linear>
            <Duration>00:00:30</Duration>
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

	vast, err := Parse(xmlData)
	if err != nil {
		t.Fatalf("Failed to parse VAST 2.0: %v", err)
	}

	if vast.Version != "2.0" {
		t.Errorf("Expected version 2.0, got %s", vast.Version)
	}

	if len(vast.Ads) != 1 {
		t.Fatalf("Expected 1 ad, got %d", len(vast.Ads))
	}

	ad := vast.Ads[0]
	if ad.ID != "12345" {
		t.Errorf("Expected ad ID 12345, got %s", ad.ID)
	}

	if ad.InLine == nil {
		t.Fatal("Expected InLine ad")
	}

	if ad.InLine.AdSystem.Value != "TestAdServer" {
		t.Errorf("Expected AdSystem TestAdServer, got %s", ad.InLine.AdSystem.Value)
	}

	if ad.InLine.AdTitle != "Test Ad Title" {
		t.Errorf("Expected AdTitle 'Test Ad Title', got %s", ad.InLine.AdTitle)
	}

	if len(ad.InLine.Impressions) != 1 {
		t.Fatalf("Expected 1 impression, got %d", len(ad.InLine.Impressions))
	}

	if ad.InLine.Impressions[0].Value != "http://example.com/impression" {
		t.Errorf("Expected impression URL http://example.com/impression, got %s", ad.InLine.Impressions[0].Value)
	}
}

// Test VAST 3.0 parsing with tracking events
func TestParseVAST30WithTracking(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="3.0">
  <Ad id="67890">
    <InLine>
      <AdSystem>TestAdServer</AdSystem>
      <AdTitle>Test Ad</AdTitle>
      <Impression><![CDATA[http://example.com/impression]]></Impression>
      <Creatives>
        <Creative>
          <Linear>
            <Duration>00:00:15</Duration>
            <TrackingEvents>
              <Tracking event="start"><![CDATA[http://example.com/start]]></Tracking>
              <Tracking event="firstQuartile"><![CDATA[http://example.com/firstQuartile]]></Tracking>
              <Tracking event="midpoint"><![CDATA[http://example.com/midpoint]]></Tracking>
              <Tracking event="thirdQuartile"><![CDATA[http://example.com/thirdQuartile]]></Tracking>
              <Tracking event="complete"><![CDATA[http://example.com/complete]]></Tracking>
            </TrackingEvents>
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

	vast, err := Parse(xmlData)
	if err != nil {
		t.Fatalf("Failed to parse VAST 3.0: %v", err)
	}

	if vast.Version != "3.0" {
		t.Errorf("Expected version 3.0, got %s", vast.Version)
	}

	creative := vast.Ads[0].InLine.Creatives[0]
	if creative.Linear == nil {
		t.Fatal("Expected Linear creative")
	}

	trackingEvents := creative.Linear.TrackingEvents
	if len(trackingEvents) != 5 {
		t.Fatalf("Expected 5 tracking events, got %d", len(trackingEvents))
	}

	expectedEvents := map[string]string{
		"start":          "http://example.com/start",
		"firstQuartile":  "http://example.com/firstQuartile",
		"midpoint":       "http://example.com/midpoint",
		"thirdQuartile":  "http://example.com/thirdQuartile",
		"complete":       "http://example.com/complete",
	}

	for _, tracking := range trackingEvents {
		expectedURL, ok := expectedEvents[tracking.Event]
		if !ok {
			t.Errorf("Unexpected tracking event: %s", tracking.Event)
			continue
		}
		if tracking.Value != expectedURL {
			t.Errorf("Expected URL %s for event %s, got %s", expectedURL, tracking.Event, tracking.Value)
		}
	}
}

// Test VAST 4.2 wrapper parsing
func TestParseVAST42Wrapper(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="4.2">
  <Ad id="wrapper-ad">
    <Wrapper>
      <AdSystem version="2.0">WrapperSystem</AdSystem>
      <VASTAdTagURI><![CDATA[http://example.com/vast-tag]]></VASTAdTagURI>
      <Impression><![CDATA[http://example.com/wrapper-impression]]></Impression>
      <Creatives>
        <Creative>
          <Linear>
            <TrackingEvents>
              <Tracking event="start"><![CDATA[http://example.com/wrapper-start]]></Tracking>
            </TrackingEvents>
          </Linear>
        </Creative>
      </Creatives>
    </Wrapper>
  </Ad>
</VAST>`

	vast, err := Parse(xmlData)
	if err != nil {
		t.Fatalf("Failed to parse VAST 4.2 wrapper: %v", err)
	}

	if vast.Version != "4.2" {
		t.Errorf("Expected version 4.2, got %s", vast.Version)
	}

	ad := vast.Ads[0]
	if ad.Wrapper == nil {
		t.Fatal("Expected Wrapper ad")
	}

	if ad.Wrapper.VASTAdTagURI.Value != "http://example.com/vast-tag" {
		t.Errorf("Expected VASTAdTagURI http://example.com/vast-tag, got %s", ad.Wrapper.VASTAdTagURI.Value)
	}

	if len(ad.Wrapper.Impressions) != 1 {
		t.Fatalf("Expected 1 impression, got %d", len(ad.Wrapper.Impressions))
	}
}

// Test invalid VAST version
func TestParseInvalidVersion(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="5.0">
  <Ad>
    <InLine>
      <AdSystem>Test</AdSystem>
      <AdTitle>Test</AdTitle>
      <Impression><![CDATA[http://example.com/impression]]></Impression>
      <Creatives>
        <Creative>
          <Linear>
            <Duration>00:00:10</Duration>
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

	_, err := Parse(xmlData)
	if err == nil {
		t.Fatal("Expected error for unsupported version 5.0")
	}

	if !strings.Contains(err.Error(), "unsupported VAST version") {
		t.Errorf("Expected 'unsupported VAST version' error, got: %v", err)
	}
}

// Test VAST marshaling
func TestVASTMarshal(t *testing.T) {
	vast := &VAST{
		Version: "4.2",
		Ads: []Ad{
			{
				ID: "test-ad",
				InLine: &InLine{
					AdSystem: AdSystem{
						Version: "1.0",
						Value:   "TestSystem",
					},
					AdTitle: "Test Ad",
					Impressions: []Impression{
						{Value: "http://example.com/impression"},
					},
					Creatives: []Creative{
						{
							Linear: &Linear{
								Duration: "00:00:30",
								MediaFiles: []MediaFile{
									{
										Delivery: "progressive",
										Type:     "video/mp4",
										Width:    1920,
										Height:   1080,
										Value:    "http://example.com/video.mp4",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	xmlStr, err := vast.Marshal()
	if err != nil {
		t.Fatalf("Failed to marshal VAST: %v", err)
	}

	if !strings.Contains(xmlStr, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Error("Expected XML header in output")
	}

	if !strings.Contains(xmlStr, `<VAST`) {
		t.Error("Expected VAST element in output")
	}

	if !strings.Contains(xmlStr, `version="4.2"`) {
		t.Error("Expected version attribute in output")
	}

	// Verify it can be parsed back
	parsedVast, err := Parse(xmlStr)
	if err != nil {
		t.Fatalf("Failed to parse marshaled VAST: %v", err)
	}

	if parsedVast.Version != "4.2" {
		t.Errorf("Expected version 4.2 after round-trip, got %s", parsedVast.Version)
	}
}

// Test VAST validation
func TestVASTValidation(t *testing.T) {
	tests := []struct {
		name        string
		vast        *VAST
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid inline ad",
			vast: &VAST{
				Version: "4.2",
				Ads: []Ad{
					{
						InLine: &InLine{
							AdSystem:    AdSystem{Value: "Test"},
							AdTitle:     "Test Ad",
							Impressions: []Impression{{Value: "http://example.com/imp"}},
							Creatives: []Creative{
								{Linear: &Linear{Duration: "00:00:10"}},
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "missing version",
			vast: &VAST{
				Ads: []Ad{
					{InLine: &InLine{AdTitle: "Test"}},
				},
			},
			expectError: true,
			errorMsg:    "version is required",
		},
		{
			name: "no ads",
			vast: &VAST{
				Version: "4.2",
				Ads:     []Ad{},
			},
			expectError: true,
			errorMsg:    "at least one Ad",
		},
		{
			name: "wrapper without VASTAdTagURI",
			vast: &VAST{
				Version: "4.2",
				Ads: []Ad{
					{
						Wrapper: &Wrapper{
							AdSystem:    AdSystem{Value: "Test"},
							Impressions: []Impression{{Value: "http://example.com/imp"}},
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "VASTAdTagURI",
		},
		{
			name: "inline without impressions",
			vast: &VAST{
				Version: "4.2",
				Ads: []Ad{
					{
						InLine: &InLine{
							AdSystem:  AdSystem{Value: "Test"},
							AdTitle:   "Test",
							Creatives: []Creative{{Linear: &Linear{}}},
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "at least one Impression",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.vast.Validate()
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errorMsg)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}
