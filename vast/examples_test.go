package vast_test

import (
	"fmt"
	"github.com/prebid/prebid-server/v3/vast"
)

// Example of parsing a VAST 2.0 document
func ExampleParse_vast20() {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="2.0">
  <Ad id="12345">
    <InLine>
      <AdSystem>MyAdServer</AdSystem>
      <AdTitle>Sample Video Ad</AdTitle>
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

	vastDoc, err := vast.Parse(xmlData)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Version: %s\n", vastDoc.Version)
	fmt.Printf("Ad ID: %s\n", vastDoc.Ads[0].ID)
	fmt.Printf("Ad Title: %s\n", vastDoc.Ads[0].InLine.AdTitle)

	// Output:
	// Version: 2.0
	// Ad ID: 12345
	// Ad Title: Sample Video Ad
}

// Example of parsing a VAST 4.2 wrapper
func ExampleParse_wrapper() {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="4.2">
  <Ad id="wrapper-123">
    <Wrapper>
      <AdSystem>WrapperSystem</AdSystem>
      <VASTAdTagURI><![CDATA[http://example.com/next-vast]]></VASTAdTagURI>
      <Impression><![CDATA[http://example.com/wrapper-impression]]></Impression>
    </Wrapper>
  </Ad>
</VAST>`

	vastDoc, err := vast.Parse(xmlData)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Version: %s\n", vastDoc.Version)
	fmt.Printf("Wrapper Tag URI: %s\n", vastDoc.Ads[0].Wrapper.VASTAdTagURI.Value)

	// Output:
	// Version: 4.2
	// Wrapper Tag URI: http://example.com/next-vast
}

// Example of creating a simple wrapper
func ExampleNewDefaultWrapper() {
	vastDoc, err := vast.NewDefaultWrapper(
		"PrebidServer",
		"http://example.com/vast-tag",
		[]string{
			"http://example.com/impression1",
			"http://example.com/impression2",
		},
	)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Version: %s\n", vastDoc.Version)
	fmt.Printf("Number of impressions: %d\n", len(vastDoc.Ads[0].Wrapper.Impressions))

	// Output:
	// Version: 4.2
	// Number of impressions: 2
}

// Example of creating a wrapper with XML output
func ExampleNewDefaultWrapperXML() {
	xmlString, err := vast.NewDefaultWrapperXML(
		"PrebidServer",
		"http://example.com/vast-tag",
		[]string{"http://example.com/impression"},
	)
	if err != nil {
		panic(err)
	}

	// Verify it's valid XML by parsing it back
	vastDoc, err := vast.Parse(xmlString)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Version: %s\n", vastDoc.Version)

	// Output:
	// Version: 4.2
}

// Example of building a complex wrapper with tracking
func ExampleWrapperBuilder() {
	config := vast.WrapperConfig{
		AdID:            "prebid-wrapper-001",
		AdSystem:        "PrebidServer",
		AdSystemVersion: "3.0",
		AdTitle:         "Prebid Video Wrapper",
		VASTAdTagURI:    "http://ad-server.com/vast-response",
		ImpressionURLs: []string{
			"http://prebid.com/impression?id=123",
		},
		ErrorURL: "http://prebid.com/error?id=123",
		TrackingEvents: map[string][]string{
			"start": {
				"http://prebid.com/event?type=start&id=123",
			},
			"firstQuartile": {
				"http://prebid.com/event?type=firstQuartile&id=123",
			},
			"midpoint": {
				"http://prebid.com/event?type=midpoint&id=123",
			},
			"thirdQuartile": {
				"http://prebid.com/event?type=thirdQuartile&id=123",
			},
			"complete": {
				"http://prebid.com/event?type=complete&id=123",
			},
		},
	}

	vastDoc, err := vast.NewWrapperBuilder("4.2").
		AddWrapperAd(config).
		Build()

	if err != nil {
		panic(err)
	}

	fmt.Printf("Ad ID: %s\n", vastDoc.Ads[0].ID)
	fmt.Printf("Ad Title: %s\n", vastDoc.Ads[0].Wrapper.AdTitle)

	// Output:
	// Ad ID: prebid-wrapper-001
	// Ad Title: Prebid Video Wrapper
}

// Example of injecting tracking URLs into existing VAST
func ExampleTrackingInjector() {
	originalXML := `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="4.2">
  <Ad>
    <InLine>
      <AdSystem>OriginalAdServer</AdSystem>
      <AdTitle>Original Ad</AdTitle>
      <Impression><![CDATA[http://original.com/impression]]></Impression>
      <Creatives>
        <Creative>
          <Linear>
            <Duration>00:00:15</Duration>
            <MediaFiles>
              <MediaFile delivery="progressive" type="video/mp4">
                <![CDATA[http://original.com/video.mp4]]>
              </MediaFile>
            </MediaFiles>
          </Linear>
        </Creative>
      </Creatives>
    </InLine>
  </Ad>
</VAST>`

	injector, err := vast.NewTrackingInjectorFromXML(originalXML)
	if err != nil {
		panic(err)
	}

	modifiedXML, err := injector.
		InjectImpressions([]string{"http://prebid.com/impression"}).
		InjectVideoEvent("start", []string{"http://prebid.com/start"}).
		InjectVideoEvent("complete", []string{"http://prebid.com/complete"}).
		ToXML()

	if err != nil {
		panic(err)
	}

	// Verify the injected tracking
	vastDoc, _ := vast.Parse(modifiedXML)
	impressions := vastDoc.GetImpressionURLs()
	startEvents := vastDoc.GetTrackingEvents("start")

	fmt.Printf("Total impressions: %d\n", len(impressions))
	fmt.Printf("Start events: %d\n", len(startEvents))

	// Output:
	// Total impressions: 2
	// Start events: 1
}

// Example of extracting tracking URLs from VAST
func ExampleVAST_GetImpressionURLs() {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="4.2">
  <Ad>
    <InLine>
      <AdSystem>Test</AdSystem>
      <AdTitle>Test Ad</AdTitle>
      <Impression><![CDATA[http://example.com/impression1]]></Impression>
      <Impression><![CDATA[http://example.com/impression2]]></Impression>
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

	vastDoc, _ := vast.Parse(xmlData)
	urls := vastDoc.GetImpressionURLs()

	for i, url := range urls {
		fmt.Printf("Impression %d: %s\n", i+1, url)
	}

	// Output:
	// Impression 1: http://example.com/impression1
	// Impression 2: http://example.com/impression2
}

// Example of extracting tracking events
func ExampleVAST_GetTrackingEvents() {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="4.2">
  <Ad>
    <InLine>
      <AdSystem>Test</AdSystem>
      <AdTitle>Test Ad</AdTitle>
      <Impression><![CDATA[http://example.com/impression]]></Impression>
      <Creatives>
        <Creative>
          <Linear>
            <Duration>00:00:30</Duration>
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

	vastDoc, _ := vast.Parse(xmlData)

	startURLs := vastDoc.GetTrackingEvents("start")
	completeURLs := vastDoc.GetTrackingEvents("complete")

	fmt.Printf("Start tracking URLs: %d\n", len(startURLs))
	fmt.Printf("Complete tracking URLs: %d\n", len(completeURLs))

	// Output:
	// Start tracking URLs: 2
	// Complete tracking URLs: 1
}

// Example of validating a VAST document
func ExampleVAST_Validate() {
	// Valid VAST document
	validVAST := &vast.VAST{
		Version: "4.2",
		Ads: []vast.Ad{
			{
				InLine: &vast.InLine{
					AdSystem:    vast.AdSystem{Value: "TestSystem"},
					AdTitle:     "Test Ad",
					Impressions: []vast.Impression{{Value: "http://example.com/imp"}},
					Creatives: []vast.Creative{
						{Linear: &vast.Linear{Duration: "00:00:30"}},
					},
				},
			},
		},
	}

	err := validVAST.Validate()
	fmt.Printf("Valid VAST: %v\n", err == nil)

	// Invalid VAST (missing impressions)
	invalidVAST := &vast.VAST{
		Version: "4.2",
		Ads: []vast.Ad{
			{
				InLine: &vast.InLine{
					AdSystem:  vast.AdSystem{Value: "TestSystem"},
					AdTitle:   "Test Ad",
					Creatives: []vast.Creative{},
				},
			},
		},
	}

	err = invalidVAST.Validate()
	fmt.Printf("Invalid VAST: %v\n", err != nil)

	// Output:
	// Valid VAST: true
	// Invalid VAST: true
}

// Example of marshaling VAST to XML
func ExampleVAST_Marshal() {
	vastDoc := &vast.VAST{
		Version: "4.2",
		Ads: []vast.Ad{
			{
				ID: "test-ad",
				InLine: &vast.InLine{
					AdSystem: vast.AdSystem{
						Version: "1.0",
						Value:   "TestSystem",
					},
					AdTitle: "Test Ad",
					Impressions: []vast.Impression{
						{Value: "http://example.com/impression"},
					},
					Creatives: []vast.Creative{
						{
							Linear: &vast.Linear{
								Duration: "00:00:15",
								MediaFiles: []vast.MediaFile{
									{
										Delivery: "progressive",
										Type:     "video/mp4",
										Width:    1280,
										Height:   720,
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

	xmlString, err := vastDoc.Marshal()
	if err != nil {
		panic(err)
	}

	// Verify it can be parsed back
	parsedVAST, _ := vast.Parse(xmlString)
	fmt.Printf("Version: %s\n", parsedVAST.Version)

	// Output:
	// Version: 4.2
}
