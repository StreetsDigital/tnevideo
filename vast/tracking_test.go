package vast

import (
	"strings"
	"testing"
)

func TestAddImpressionTracking(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="4.2">
  <Ad>
    <InLine>
      <AdSystem>Test</AdSystem>
      <AdTitle>Test Ad</AdTitle>
      <Impression><![CDATA[http://example.com/original-imp]]></Impression>
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

	vast, err := Parse(xmlData)
	if err != nil {
		t.Fatalf("Failed to parse VAST: %v", err)
	}

	// Add impression tracking
	err = vast.AddImpressionTracking([]string{
		"http://example.com/tracking-imp1",
		"http://example.com/tracking-imp2",
	})
	if err != nil {
		t.Fatalf("Failed to add impression tracking: %v", err)
	}

	impressions := vast.Ads[0].InLine.Impressions
	if len(impressions) != 3 {
		t.Fatalf("Expected 3 impressions, got %d", len(impressions))
	}

	expectedURLs := []string{
		"http://example.com/original-imp",
		"http://example.com/tracking-imp1",
		"http://example.com/tracking-imp2",
	}

	for i, imp := range impressions {
		if imp.Value != expectedURLs[i] {
			t.Errorf("Expected impression %s, got %s", expectedURLs[i], imp.Value)
		}
	}
}

func TestAddVideoEventTracking(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="4.2">
  <Ad>
    <InLine>
      <AdSystem>Test</AdSystem>
      <AdTitle>Test Ad</AdTitle>
      <Impression><![CDATA[http://example.com/imp]]></Impression>
      <Creatives>
        <Creative>
          <Linear>
            <Duration>00:00:10</Duration>
            <TrackingEvents>
              <Tracking event="start"><![CDATA[http://example.com/original-start]]></Tracking>
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
		t.Fatalf("Failed to parse VAST: %v", err)
	}

	// Add start event tracking
	err = vast.AddVideoEventTracking("start", []string{
		"http://example.com/tracking-start1",
		"http://example.com/tracking-start2",
	})
	if err != nil {
		t.Fatalf("Failed to add start tracking: %v", err)
	}

	// Add complete event tracking
	err = vast.AddVideoEventTracking("complete", []string{
		"http://example.com/tracking-complete",
	})
	if err != nil {
		t.Fatalf("Failed to add complete tracking: %v", err)
	}

	trackingEvents := vast.Ads[0].InLine.Creatives[0].Linear.TrackingEvents
	if len(trackingEvents) != 4 {
		t.Fatalf("Expected 4 tracking events, got %d", len(trackingEvents))
	}

	startCount := 0
	completeCount := 0

	for _, tracking := range trackingEvents {
		if tracking.Event == "start" {
			startCount++
		} else if tracking.Event == "complete" {
			completeCount++
		}
	}

	if startCount != 3 {
		t.Errorf("Expected 3 start events, got %d", startCount)
	}

	if completeCount != 1 {
		t.Errorf("Expected 1 complete event, got %d", completeCount)
	}
}

func TestAddVideoEventTrackingInvalidEvent(t *testing.T) {
	vast := &VAST{
		Version: "4.2",
		Ads: []Ad{
			{
				InLine: &InLine{
					AdSystem:    AdSystem{Value: "Test"},
					AdTitle:     "Test",
					Impressions: []Impression{{Value: "http://example.com/imp"}},
					Creatives: []Creative{
						{Linear: &Linear{Duration: "00:00:10"}},
					},
				},
			},
		},
	}

	err := vast.AddVideoEventTracking("invalidEvent", []string{"http://example.com/track"})
	if err == nil {
		t.Error("Expected error for invalid event")
	}

	if !strings.Contains(err.Error(), "invalid tracking event") {
		t.Errorf("Expected 'invalid tracking event' error, got: %v", err)
	}
}

func TestAddErrorTracking(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="4.2">
  <Ad>
    <InLine>
      <AdSystem>Test</AdSystem>
      <AdTitle>Test Ad</AdTitle>
      <Impression><![CDATA[http://example.com/imp]]></Impression>
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

	vast, err := Parse(xmlData)
	if err != nil {
		t.Fatalf("Failed to parse VAST: %v", err)
	}

	err = vast.AddErrorTracking("http://example.com/error")
	if err != nil {
		t.Fatalf("Failed to add error tracking: %v", err)
	}

	if vast.Ads[0].InLine.Error == nil {
		t.Fatal("Expected error URL to be set")
	}

	if vast.Ads[0].InLine.Error.Value != "http://example.com/error" {
		t.Errorf("Expected error URL http://example.com/error, got %s", vast.Ads[0].InLine.Error.Value)
	}
}

func TestAddClickTracking(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="4.2">
  <Ad>
    <InLine>
      <AdSystem>Test</AdSystem>
      <AdTitle>Test Ad</AdTitle>
      <Impression><![CDATA[http://example.com/imp]]></Impression>
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

	vast, err := Parse(xmlData)
	if err != nil {
		t.Fatalf("Failed to parse VAST: %v", err)
	}

	err = vast.AddClickTracking([]string{
		"http://example.com/click1",
		"http://example.com/click2",
	})
	if err != nil {
		t.Fatalf("Failed to add click tracking: %v", err)
	}

	videoClicks := vast.Ads[0].InLine.Creatives[0].Linear.VideoClicks
	if videoClicks == nil {
		t.Fatal("Expected VideoClicks to be set")
	}

	if len(videoClicks.ClickTracking) != 2 {
		t.Fatalf("Expected 2 click tracking URLs, got %d", len(videoClicks.ClickTracking))
	}

	expectedURLs := []string{
		"http://example.com/click1",
		"http://example.com/click2",
	}

	for i, click := range videoClicks.ClickTracking {
		if click.Value != expectedURLs[i] {
			t.Errorf("Expected click URL %s, got %s", expectedURLs[i], click.Value)
		}
	}
}

func TestGetImpressionURLs(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="4.2">
  <Ad>
    <InLine>
      <AdSystem>Test</AdSystem>
      <AdTitle>Test Ad</AdTitle>
      <Impression><![CDATA[http://example.com/imp1]]></Impression>
      <Impression><![CDATA[http://example.com/imp2]]></Impression>
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

	vast, err := Parse(xmlData)
	if err != nil {
		t.Fatalf("Failed to parse VAST: %v", err)
	}

	urls := vast.GetImpressionURLs()
	if len(urls) != 2 {
		t.Fatalf("Expected 2 impression URLs, got %d", len(urls))
	}

	expectedURLs := []string{
		"http://example.com/imp1",
		"http://example.com/imp2",
	}

	for i, url := range urls {
		if url != expectedURLs[i] {
			t.Errorf("Expected URL %s, got %s", expectedURLs[i], url)
		}
	}
}

func TestGetTrackingEvents(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="4.2">
  <Ad>
    <InLine>
      <AdSystem>Test</AdSystem>
      <AdTitle>Test Ad</AdTitle>
      <Impression><![CDATA[http://example.com/imp]]></Impression>
      <Creatives>
        <Creative>
          <Linear>
            <Duration>00:00:10</Duration>
            <TrackingEvents>
              <Tracking event="start"><![CDATA[http://example.com/start1]]></Tracking>
              <Tracking event="start"><![CDATA[http://example.com/start2]]></Tracking>
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
		t.Fatalf("Failed to parse VAST: %v", err)
	}

	startURLs := vast.GetTrackingEvents("start")
	if len(startURLs) != 2 {
		t.Fatalf("Expected 2 start URLs, got %d", len(startURLs))
	}

	completeURLs := vast.GetTrackingEvents("complete")
	if len(completeURLs) != 1 {
		t.Fatalf("Expected 1 complete URL, got %d", len(completeURLs))
	}

	midpointURLs := vast.GetTrackingEvents("midpoint")
	if len(midpointURLs) != 0 {
		t.Errorf("Expected 0 midpoint URLs, got %d", len(midpointURLs))
	}
}

func TestTrackingInjector(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="4.2">
  <Ad>
    <InLine>
      <AdSystem>Test</AdSystem>
      <AdTitle>Test Ad</AdTitle>
      <Impression><![CDATA[http://example.com/imp]]></Impression>
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

	injector, err := NewTrackingInjectorFromXML(xmlData)
	if err != nil {
		t.Fatalf("Failed to create tracking injector: %v", err)
	}

	// Chain multiple injections
	modifiedXML, err := injector.
		InjectImpressions([]string{"http://example.com/new-imp"}).
		InjectVideoEvent("start", []string{"http://example.com/start"}).
		InjectVideoEvent("complete", []string{"http://example.com/complete"}).
		InjectError("http://example.com/error").
		InjectClickTracking([]string{"http://example.com/click"}).
		ToXML()

	if err != nil {
		t.Fatalf("Failed to generate modified XML: %v", err)
	}

	// Verify the modifications
	modifiedVast, err := Parse(modifiedXML)
	if err != nil {
		t.Fatalf("Failed to parse modified XML: %v", err)
	}

	// Check impressions
	impressions := modifiedVast.GetImpressionURLs()
	if len(impressions) != 2 {
		t.Errorf("Expected 2 impressions after injection, got %d", len(impressions))
	}

	// Check tracking events
	startURLs := modifiedVast.GetTrackingEvents("start")
	if len(startURLs) != 1 {
		t.Errorf("Expected 1 start tracking URL, got %d", len(startURLs))
	}

	completeURLs := modifiedVast.GetTrackingEvents("complete")
	if len(completeURLs) != 1 {
		t.Errorf("Expected 1 complete tracking URL, got %d", len(completeURLs))
	}

	// Check error tracking
	if modifiedVast.Ads[0].InLine.Error == nil {
		t.Error("Expected error URL to be set")
	}

	// Check click tracking
	clicks := modifiedVast.Ads[0].InLine.Creatives[0].Linear.VideoClicks
	if clicks == nil || len(clicks.ClickTracking) != 1 {
		t.Error("Expected 1 click tracking URL")
	}
}

func TestTrackingInjectorWithWrapper(t *testing.T) {
	vast, err := NewDefaultWrapper(
		"TestSystem",
		"http://example.com/vast",
		[]string{"http://example.com/imp"},
	)
	if err != nil {
		t.Fatalf("Failed to create wrapper: %v", err)
	}

	injector := NewTrackingInjector(vast)

	modifiedXML, err := injector.
		InjectImpressions([]string{"http://example.com/extra-imp"}).
		InjectError("http://example.com/error").
		ToXML()

	if err != nil {
		t.Fatalf("Failed to generate XML: %v", err)
	}

	modifiedVast, err := Parse(modifiedXML)
	if err != nil {
		t.Fatalf("Failed to parse modified XML: %v", err)
	}

	impressions := modifiedVast.GetImpressionURLs()
	if len(impressions) != 2 {
		t.Errorf("Expected 2 impressions, got %d", len(impressions))
	}

	if modifiedVast.Ads[0].Wrapper.Error == nil {
		t.Error("Expected error URL in wrapper")
	}
}
