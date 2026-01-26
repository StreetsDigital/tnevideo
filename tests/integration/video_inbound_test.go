//go:build integration
// +build integration

package integration

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thenexusengine/tne_springwire/pkg/vast"
)

// TestInboundVASTParsing tests receiving and parsing VAST tags from demand partners
func TestInboundVASTParsing(t *testing.T) {
	t.Run("Parse_basic_inline_VAST", func(t *testing.T) {
		vastXML := `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="4.0">
  <Ad id="test-ad-001">
    <InLine>
      <AdSystem>Test System</AdSystem>
      <AdTitle>Test Video Ad</AdTitle>
      <Impression id="imp-1"><![CDATA[https://tracking.example.com/impression?id=12345]]></Impression>
      <Error><![CDATA[https://tracking.example.com/error?id=12345&code=[ERRORCODE]]]></Error>
      <Creatives>
        <Creative id="creative-1">
          <Linear>
            <Duration>00:00:30</Duration>
            <MediaFiles>
              <MediaFile delivery="progressive" type="video/mp4" width="1920" height="1080" bitrate="5000">
                <![CDATA[https://cdn.example.com/video.mp4]]>
              </MediaFile>
            </MediaFiles>
            <TrackingEvents>
              <Tracking event="start"><![CDATA[https://tracking.example.com/start]]></Tracking>
              <Tracking event="firstQuartile"><![CDATA[https://tracking.example.com/25]]></Tracking>
              <Tracking event="midpoint"><![CDATA[https://tracking.example.com/50]]></Tracking>
              <Tracking event="thirdQuartile"><![CDATA[https://tracking.example.com/75]]></Tracking>
              <Tracking event="complete"><![CDATA[https://tracking.example.com/complete]]></Tracking>
            </TrackingEvents>
            <VideoClicks>
              <ClickThrough><![CDATA[https://advertiser.example.com/landing]]></ClickThrough>
              <ClickTracking><![CDATA[https://tracking.example.com/click]]></ClickTracking>
            </VideoClicks>
          </Linear>
        </Creative>
      </Creatives>
    </InLine>
  </Ad>
</VAST>`

		// Parse VAST
		v, err := vast.Parse([]byte(vastXML))
		require.NoError(t, err, "Should parse valid VAST")

		// Verify structure
		assert.Equal(t, "4.0", v.Version)
		assert.Len(t, v.Ads, 1, "Should have one ad")

		ad := v.Ads[0]
		assert.Equal(t, "test-ad-001", ad.ID)
		assert.NotNil(t, ad.InLine, "Should have InLine ad")

		// Verify InLine fields
		inline := ad.InLine
		assert.Equal(t, "Test System", inline.AdSystem.Value)
		assert.Equal(t, "Test Video Ad", inline.AdTitle)
		assert.Len(t, inline.Impressions, 1, "Should have one impression")
		assert.Equal(t, "https://tracking.example.com/impression?id=12345", inline.Impressions[0].Value)
		assert.Equal(t, "https://tracking.example.com/error?id=12345&code=[ERRORCODE]", inline.Error)

		// Verify creative
		assert.Len(t, inline.Creatives.Creative, 1, "Should have one creative")
		creative := inline.Creatives.Creative[0]
		assert.NotNil(t, creative.Linear, "Should have linear creative")

		// Verify duration
		linear := creative.Linear
		assert.Equal(t, vast.Duration("00:00:30"), linear.Duration)

		// Verify media files
		assert.Len(t, linear.MediaFiles.MediaFile, 1, "Should have one media file")
		mf := linear.MediaFiles.MediaFile[0]
		assert.Equal(t, "progressive", mf.Delivery)
		assert.Equal(t, "video/mp4", mf.Type)
		assert.Equal(t, 1920, mf.Width)
		assert.Equal(t, 1080, mf.Height)
		assert.Equal(t, 5000, mf.Bitrate)
		assert.Equal(t, "https://cdn.example.com/video.mp4", mf.Value)

		// Verify tracking events
		assert.Len(t, linear.TrackingEvents.Tracking, 5, "Should have 5 tracking events")

		// Verify click tracking
		assert.NotNil(t, linear.VideoClicks, "Should have video clicks")
		assert.Equal(t, "https://advertiser.example.com/landing", linear.VideoClicks.ClickThrough.Value)
		assert.Len(t, linear.VideoClicks.ClickTracking, 1)
	})

	t.Run("Parse_wrapper_VAST", func(t *testing.T) {
		vastXML := `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="4.0">
  <Ad id="wrapper-ad">
    <Wrapper>
      <AdSystem>Wrapper System</AdSystem>
      <VASTAdTagURI><![CDATA[https://dsp.example.com/vast?id=abc123]]></VASTAdTagURI>
      <Impression><![CDATA[https://wrapper-tracking.example.com/impression]]></Impression>
      <Error><![CDATA[https://wrapper-tracking.example.com/error]]></Error>
      <Creatives></Creatives>
    </Wrapper>
  </Ad>
</VAST>`

		v, err := vast.Parse([]byte(vastXML))
		require.NoError(t, err)

		assert.Len(t, v.Ads, 1)
		ad := v.Ads[0]
		assert.NotNil(t, ad.Wrapper, "Should have wrapper")
		assert.Nil(t, ad.InLine, "Should not have inline")

		wrapper := ad.Wrapper
		assert.Equal(t, "Wrapper System", wrapper.AdSystem.Value)
		assert.Equal(t, "https://dsp.example.com/vast?id=abc123", wrapper.VASTAdTagURI)
		assert.Len(t, wrapper.Impressions, 1)
		assert.Equal(t, "https://wrapper-tracking.example.com/impression", wrapper.Impressions[0].Value)
	})

	t.Run("Parse_empty_VAST", func(t *testing.T) {
		vastXML := `<?xml version="1.0"?><VAST version="4.0"></VAST>`

		v, err := vast.Parse([]byte(vastXML))
		require.NoError(t, err)

		assert.True(t, v.IsEmpty(), "Should be empty")
		assert.Len(t, v.Ads, 0)
	})

	t.Run("Parse_VAST_with_error", func(t *testing.T) {
		vastXML := `<?xml version="1.0"?>
<VAST version="4.0">
  <Error><![CDATA[https://tracking.example.com/error?code=204]]></Error>
</VAST>`

		v, err := vast.Parse([]byte(vastXML))
		require.NoError(t, err)

		assert.Equal(t, "https://tracking.example.com/error?code=204", v.Error)
		assert.True(t, v.IsEmpty())
	})

	t.Run("Parse_invalid_XML", func(t *testing.T) {
		invalidXML := `<VAST><Ad></VAST>` // Unclosed tag

		_, err := vast.Parse([]byte(invalidXML))
		assert.Error(t, err, "Should fail on invalid XML")
	})

	t.Run("Parse_malformed_duration", func(t *testing.T) {
		_, err := vast.ParseDuration("invalid")
		assert.Error(t, err, "Should fail on invalid duration format")
	})
}

// TestVASTWrapperUnwrapping tests unwrapping wrapper VAST chains
func TestVASTWrapperUnwrapping(t *testing.T) {
	t.Run("Unwrap_single_level_wrapper", func(t *testing.T) {
		// Setup mock servers
		finalVAST := `<?xml version="1.0"?>
<VAST version="4.0">
  <Ad id="final-ad">
    <InLine>
      <AdSystem>Final System</AdSystem>
      <AdTitle>Final Ad</AdTitle>
      <Impression><![CDATA[https://final.example.com/imp]]></Impression>
      <Creatives>
        <Creative>
          <Linear>
            <Duration>00:00:15</Duration>
            <MediaFiles>
              <MediaFile delivery="progressive" type="video/mp4" width="640" height="480">
                <![CDATA[https://final-cdn.example.com/video.mp4]]>
              </MediaFile>
            </MediaFiles>
          </Linear>
        </Creative>
      </Creatives>
    </InLine>
  </Ad>
</VAST>`

		// Mock final VAST server
		finalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(finalVAST))
		}))
		defer finalServer.Close()

		// Wrapper VAST pointing to final server
		wrapperVAST := fmt.Sprintf(`<?xml version="1.0"?>
<VAST version="4.0">
  <Ad id="wrapper-ad">
    <Wrapper>
      <AdSystem>Wrapper</AdSystem>
      <VASTAdTagURI><![CDATA[%s]]></VASTAdTagURI>
      <Impression><![CDATA[https://wrapper.example.com/imp]]></Impression>
    </Wrapper>
  </Ad>
</VAST>`, finalServer.URL)

		// Parse wrapper
		wrapperParsed, err := vast.Parse([]byte(wrapperVAST))
		require.NoError(t, err)
		assert.NotNil(t, wrapperParsed.Ads[0].Wrapper)

		// Fetch final VAST
		vastURL := wrapperParsed.Ads[0].Wrapper.VASTAdTagURI
		resp, err := http.Get(vastURL)
		require.NoError(t, err)
		defer resp.Body.Close()

		finalBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		finalParsed, err := vast.Parse(finalBody)
		require.NoError(t, err)

		// Verify final VAST
		assert.NotNil(t, finalParsed.Ads[0].InLine)
		assert.Equal(t, "Final Ad", finalParsed.Ads[0].InLine.AdTitle)

		// In real implementation, impressions would be combined from both levels
		t.Log("Wrapper unwrapping successful - final creative retrieved")
	})

	t.Run("Handle_unreachable_wrapper_URL", func(t *testing.T) {
		wrapperVAST := `<?xml version="1.0"?>
<VAST version="4.0">
  <Ad id="wrapper-ad">
    <Wrapper>
      <AdSystem>Wrapper</AdSystem>
      <VASTAdTagURI><![CDATA[https://unreachable.invalid/vast]]></VASTAdTagURI>
      <Impression><![CDATA[https://wrapper.example.com/imp]]></Impression>
    </Wrapper>
  </Ad>
</VAST>`

		v, err := vast.Parse([]byte(wrapperVAST))
		require.NoError(t, err)

		vastURL := v.Ads[0].Wrapper.VASTAdTagURI

		// Attempt to fetch with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 2*http.DefaultClient.Timeout)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "GET", vastURL, nil)
		require.NoError(t, err)

		_, err = http.DefaultClient.Do(req)
		assert.Error(t, err, "Should fail on unreachable URL")
		t.Log("Network error handled correctly")
	})
}

// TestVASTTrackingEventExtraction tests extracting tracking URLs by event type
func TestVASTTrackingEventExtraction(t *testing.T) {
	vastXML := `<?xml version="1.0"?>
<VAST version="4.0">
  <Ad id="tracking-ad">
    <InLine>
      <AdSystem>Test</AdSystem>
      <AdTitle>Tracking Test</AdTitle>
      <Impression><![CDATA[https://track.example.com/imp]]></Impression>
      <Creatives>
        <Creative>
          <Linear>
            <Duration>00:00:30</Duration>
            <MediaFiles>
              <MediaFile delivery="progressive" type="video/mp4" width="640" height="480">
                <![CDATA[https://cdn.example.com/video.mp4]]>
              </MediaFile>
            </MediaFiles>
            <TrackingEvents>
              <Tracking event="start"><![CDATA[https://track.example.com/start/1]]></Tracking>
              <Tracking event="start"><![CDATA[https://track.example.com/start/2]]></Tracking>
              <Tracking event="firstQuartile"><![CDATA[https://track.example.com/25]]></Tracking>
              <Tracking event="midpoint"><![CDATA[https://track.example.com/50]]></Tracking>
              <Tracking event="thirdQuartile"><![CDATA[https://track.example.com/75]]></Tracking>
              <Tracking event="complete"><![CDATA[https://track.example.com/complete]]></Tracking>
              <Tracking event="mute"><![CDATA[https://track.example.com/mute]]></Tracking>
              <Tracking event="unmute"><![CDATA[https://track.example.com/unmute]]></Tracking>
              <Tracking event="pause"><![CDATA[https://track.example.com/pause]]></Tracking>
              <Tracking event="resume"><![CDATA[https://track.example.com/resume]]></Tracking>
              <Tracking event="skip"><![CDATA[https://track.example.com/skip]]></Tracking>
            </TrackingEvents>
          </Linear>
        </Creative>
      </Creatives>
    </InLine>
  </Ad>
</VAST>`

	v, err := vast.Parse([]byte(vastXML))
	require.NoError(t, err)

	linear := v.GetLinearCreative()
	require.NotNil(t, linear)

	// Extract tracking URLs by event type
	trackingByEvent := make(map[string][]string)
	for _, tracking := range linear.TrackingEvents.Tracking {
		trackingByEvent[tracking.Event] = append(trackingByEvent[tracking.Event], tracking.Value)
	}

	// Verify start events (multiple)
	assert.Len(t, trackingByEvent[vast.EventStart], 2, "Should have 2 start events")
	assert.Contains(t, trackingByEvent[vast.EventStart], "https://track.example.com/start/1")
	assert.Contains(t, trackingByEvent[vast.EventStart], "https://track.example.com/start/2")

	// Verify quartile events
	assert.Len(t, trackingByEvent[vast.EventFirstQuartile], 1)
	assert.Len(t, trackingByEvent[vast.EventMidpoint], 1)
	assert.Len(t, trackingByEvent[vast.EventThirdQuartile], 1)
	assert.Len(t, trackingByEvent[vast.EventComplete], 1)

	// Verify interactive events
	assert.Len(t, trackingByEvent[vast.EventMute], 1)
	assert.Len(t, trackingByEvent[vast.EventUnmute], 1)
	assert.Len(t, trackingByEvent[vast.EventPause], 1)
	assert.Len(t, trackingByEvent[vast.EventResume], 1)
	assert.Len(t, trackingByEvent[vast.EventSkip], 1)

	t.Logf("Total event types: %d", len(trackingByEvent))
}

// TestVASTMediaFileSelection tests selecting best media file
func TestVASTMediaFileSelection(t *testing.T) {
	vastXML := `<?xml version="1.0"?>
<VAST version="4.0">
  <Ad id="multi-bitrate-ad">
    <InLine>
      <AdSystem>Test</AdSystem>
      <AdTitle>Multi-Bitrate Ad</AdTitle>
      <Impression><![CDATA[https://track.example.com/imp]]></Impression>
      <Creatives>
        <Creative>
          <Linear>
            <Duration>00:00:30</Duration>
            <MediaFiles>
              <MediaFile delivery="progressive" type="video/mp4" width="1920" height="1080" bitrate="5000">
                <![CDATA[https://cdn.example.com/video-1080p.mp4]]>
              </MediaFile>
              <MediaFile delivery="progressive" type="video/mp4" width="1280" height="720" bitrate="2500">
                <![CDATA[https://cdn.example.com/video-720p.mp4]]>
              </MediaFile>
              <MediaFile delivery="progressive" type="video/webm" width="1920" height="1080" bitrate="4000">
                <![CDATA[https://cdn.example.com/video-1080p.webm]]>
              </MediaFile>
              <MediaFile delivery="progressive" type="video/mp4" width="640" height="480" bitrate="1000">
                <![CDATA[https://cdn.example.com/video-480p.mp4]]>
              </MediaFile>
            </MediaFiles>
          </Linear>
        </Creative>
      </Creatives>
    </InLine>
  </Ad>
</VAST>`

	v, err := vast.Parse([]byte(vastXML))
	require.NoError(t, err)

	mediaFiles := v.GetMediaFiles()
	assert.Len(t, mediaFiles, 4, "Should have 4 media file options")

	// Test selection logic (prefer MP4, highest bitrate within limits)
	bestMP4 := selectBestMediaFile(mediaFiles, "video/mp4", 1920, 1080, 6000)
	assert.NotNil(t, bestMP4)
	assert.Equal(t, 5000, bestMP4.Bitrate, "Should select highest bitrate MP4")
	assert.Equal(t, 1920, bestMP4.Width)

	// Test selecting for lower resolution
	best720p := selectBestMediaFile(mediaFiles, "video/mp4", 1280, 720, 3000)
	assert.NotNil(t, best720p)
	assert.Equal(t, 2500, best720p.Bitrate)
	assert.Equal(t, 720, best720p.Height)
}

// TestVASTCompanionAdsParsing tests companion ads
func TestVASTCompanionAdsParsing(t *testing.T) {
	vastXML := `<?xml version="1.0"?>
<VAST version="4.0">
  <Ad id="companion-ad">
    <InLine>
      <AdSystem>Test</AdSystem>
      <AdTitle>Ad with Companions</AdTitle>
      <Impression><![CDATA[https://track.example.com/imp]]></Impression>
      <Creatives>
        <Creative>
          <Linear>
            <Duration>00:00:30</Duration>
            <MediaFiles>
              <MediaFile delivery="progressive" type="video/mp4" width="640" height="480">
                <![CDATA[https://cdn.example.com/video.mp4]]>
              </MediaFile>
            </MediaFiles>
          </Linear>
        </Creative>
        <Creative>
          <CompanionAds required="all">
            <Companion id="companion-1" width="300" height="250">
              <StaticResource creativeType="image/jpeg">
                <![CDATA[https://cdn.example.com/banner-300x250.jpg]]>
              </StaticResource>
              <CompanionClickThrough><![CDATA[https://advertiser.example.com/landing]]></CompanionClickThrough>
            </Companion>
            <Companion id="companion-2" width="728" height="90">
              <StaticResource creativeType="image/png">
                <![CDATA[https://cdn.example.com/banner-728x90.png]]>
              </StaticResource>
              <CompanionClickThrough><![CDATA[https://advertiser.example.com/landing]]></CompanionClickThrough>
            </Companion>
          </CompanionAds>
        </Creative>
      </Creatives>
    </InLine>
  </Ad>
</VAST>`

	v, err := vast.Parse([]byte(vastXML))
	require.NoError(t, err)

	ad := v.Ads[0]
	assert.Len(t, ad.InLine.Creatives.Creative, 2, "Should have 2 creatives")

	// Find companion creative
	var companionCreative *vast.Creative
	for i := range ad.InLine.Creatives.Creative {
		if ad.InLine.Creatives.Creative[i].CompanionAds != nil {
			companionCreative = &ad.InLine.Creatives.Creative[i]
			break
		}
	}
	require.NotNil(t, companionCreative, "Should have companion creative")

	companions := companionCreative.CompanionAds
	assert.Equal(t, "all", companions.Required)
	assert.Len(t, companions.Companion, 2)

	// Verify first companion
	c1 := companions.Companion[0]
	assert.Equal(t, "companion-1", c1.ID)
	assert.Equal(t, 300, c1.Width)
	assert.Equal(t, 250, c1.Height)
	assert.NotNil(t, c1.StaticResource)
	assert.Equal(t, "image/jpeg", c1.StaticResource.CreativeType)
	assert.Equal(t, "https://cdn.example.com/banner-300x250.jpg", c1.StaticResource.Value)

	// Verify second companion
	c2 := companions.Companion[1]
	assert.Equal(t, "companion-2", c2.ID)
	assert.Equal(t, 728, c2.Width)
	assert.Equal(t, 90, c2.Height)
}

// Helper function to select best media file
func selectBestMediaFile(mediaFiles []vast.MediaFile, preferredType string, targetWidth, targetHeight, maxBitrate int) *vast.MediaFile {
	var best *vast.MediaFile

	for i := range mediaFiles {
		mf := &mediaFiles[i]

		// Filter by type
		if mf.Type != preferredType {
			continue
		}

		// Filter by bitrate
		if mf.Bitrate > maxBitrate {
			continue
		}

		// Prefer exact dimension match, or closest
		if best == nil {
			best = mf
			continue
		}

		// Prefer higher bitrate
		if mf.Bitrate > best.Bitrate && mf.Width <= targetWidth && mf.Height <= targetHeight {
			best = mf
		}
	}

	return best
}
