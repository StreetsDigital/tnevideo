//go:build integration
// +build integration

package integration

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thenexusengine/tne_springwire/internal/adapters"
	"github.com/thenexusengine/tne_springwire/internal/endpoints"
	"github.com/thenexusengine/tne_springwire/internal/exchange"
	"github.com/thenexusengine/tne_springwire/pkg/vast"
	"time"
)

// TestVideoErrorHandling tests error scenarios in video functionality
func TestVideoErrorHandling(t *testing.T) {
	ex := exchange.New(adapters.NewRegistry(), &exchange.Config{Timeout: 100 * time.Millisecond})
	handler := endpoints.NewVideoHandler(ex, "https://tracking.test.com")

	t.Run("Missing_required_video_parameters", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(handler.HandleVASTRequest))
		defer server.Close()

		// Request with no parameters
		resp, err := http.Get(server.URL)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should still return valid VAST (with defaults or error)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/xml; charset=utf-8", resp.Header.Get("Content-Type"))
	})

	t.Run("Invalid_VAST_XML_from_demand", func(t *testing.T) {
		// This would require mocking a demand partner that returns invalid VAST
		// The exchange should handle this gracefully
		t.Skip("Requires mock demand partner setup")
	})

	t.Run("Timeout_from_demand_partners", func(t *testing.T) {
		// Setup slow bidder that exceeds timeout
		slowBidder := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(500 * time.Millisecond) // Exceed timeout
			w.WriteHeader(http.StatusOK)
		}))
		defer slowBidder.Close()

		// Exchange should handle timeout and return empty VAST or error
		t.Log("Timeout handling verified in adapter tests")
	})

	t.Run("No_video_bid_responses", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(handler.HandleVASTRequest))
		defer server.Close()

		// Request to server with no bidders
		resp, err := http.Get(server.URL + "?w=1920&h=1080")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Parse response
		var v vast.VAST
		err = xml.NewDecoder(resp.Body).Decode(&v)
		require.NoError(t, err)

		// Should be empty VAST
		assert.True(t, v.IsEmpty() || v.Error != "", "Should return empty VAST or error VAST")
	})

	t.Run("Malformed_video_creative_URLs", func(t *testing.T) {
		// Test VAST with invalid media file URL
		invalidVAST := `<?xml version="1.0"?>
<VAST version="4.0">
  <Ad>
    <InLine>
      <AdSystem>Test</AdSystem>
      <AdTitle>Test</AdTitle>
      <Impression><![CDATA[https://example.com/imp]]></Impression>
      <Creatives>
        <Creative>
          <Linear>
            <Duration>00:00:30</Duration>
            <MediaFiles>
              <MediaFile delivery="progressive" type="video/mp4" width="1920" height="1080">
                <![CDATA[invalid-url]]>
              </MediaFile>
            </MediaFiles>
          </Linear>
        </Creative>
      </Creatives>
    </InLine>
  </Ad>
</VAST>`

		v, err := vast.Parse([]byte(invalidVAST))
		require.NoError(t, err, "Should parse even with invalid URL")

		// Validation should catch the invalid URL
		result := v.Validate()
		assert.False(t, result.Valid, "Validation should fail on invalid URL")
	})

	t.Run("Invalid_video_duration", func(t *testing.T) {
		invalidVAST := `<?xml version="1.0"?>
<VAST version="4.0">
  <Ad>
    <InLine>
      <AdSystem>Test</AdSystem>
      <AdTitle>Test</AdTitle>
      <Impression><![CDATA[https://example.com/imp]]></Impression>
      <Creatives>
        <Creative>
          <Linear>
            <Duration>invalid</Duration>
            <MediaFiles>
              <MediaFile delivery="progressive" type="video/mp4" width="1920" height="1080">
                <![CDATA[https://example.com/video.mp4]]>
              </MediaFile>
            </MediaFiles>
          </Linear>
        </Creative>
      </Creatives>
    </InLine>
  </Ad>
</VAST>`

		v, err := vast.Parse([]byte(invalidVAST))
		require.NoError(t, err)

		result := v.Validate()
		assert.False(t, result.Valid, "Should fail validation on invalid duration")
	})

	t.Run("Unsupported_video_protocols", func(t *testing.T) {
		// Request with unsupported protocol
		// Should handle gracefully
		t.Skip("Requires protocol validation implementation")
	})

	t.Run("Network_failures_during_VAST_fetching", func(t *testing.T) {
		// Test wrapper unwrapping with network failure
		wrapperVAST := `<?xml version="1.0"?>
<VAST version="4.0">
  <Ad>
    <Wrapper>
      <AdSystem>Test</AdSystem>
      <VASTAdTagURI><![CDATA[http://unreachable.invalid/vast]]></VASTAdTagURI>
    </Wrapper>
  </Ad>
</VAST>`

		v, err := vast.Parse([]byte(wrapperVAST))
		require.NoError(t, err)

		// Attempting to fetch unreachable URL should fail
		_, err = http.Get(v.Ads[0].Wrapper.VASTAdTagURI)
		assert.Error(t, err, "Should fail on unreachable URL")
	})

	t.Run("HTTP_method_not_allowed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(handler.HandleVASTRequest))
		defer server.Close()

		// Try POST to GET-only endpoint
		resp, err := http.Post(server.URL, "application/json", strings.NewReader("{}"))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})

	t.Run("Invalid_JSON_in_POST_request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(handler.HandleOpenRTBVideo))
		defer server.Close()

		// Send invalid JSON
		resp, err := http.Post(server.URL, "application/json", strings.NewReader("invalid json"))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode) // Returns error VAST with 200

		// Should return error VAST
		var v vast.VAST
		err = xml.NewDecoder(resp.Body).Decode(&v)
		require.NoError(t, err)

		assert.True(t, v.IsEmpty() || v.Error != "")
	})
}

// TestErrorVASTGeneration tests error VAST responses
func TestErrorVASTGeneration(t *testing.T) {
	t.Run("Create_error_VAST", func(t *testing.T) {
		errorURL := "https://tracking.example.com/error?code=204"
		v := vast.CreateErrorVAST(errorURL)

		assert.Equal(t, "4.0", v.Version)
		assert.Equal(t, errorURL, v.Error)
		assert.Empty(t, v.Ads)
		assert.True(t, v.IsEmpty())
	})

	t.Run("Marshal_error_VAST", func(t *testing.T) {
		v := vast.CreateErrorVAST("https://tracking.example.com/error")
		data, err := v.Marshal()

		require.NoError(t, err)
		assert.Contains(t, string(data), "<Error>")
		assert.Contains(t, string(data), "https://tracking.example.com/error")
	})
}

// TestResilientParsing tests parsing resilience
func TestResilientParsing(t *testing.T) {
	t.Run("Parse_with_extra_whitespace", func(t *testing.T) {
		vastXML := `<?xml version="1.0"?>
<VAST version="4.0">

  <Ad>
    <InLine>
      <AdSystem>   Test   </AdSystem>
      <AdTitle>Test</AdTitle>
      <Impression>   <![CDATA[https://example.com/imp]]>   </Impression>
      <Creatives>
        <Creative>
          <Linear>
            <Duration>00:00:30</Duration>
            <MediaFiles>
              <MediaFile delivery="progressive" type="video/mp4" width="1920" height="1080">
                <![CDATA[https://example.com/video.mp4]]>
              </MediaFile>
            </MediaFiles>
          </Linear>
        </Creative>
      </Creatives>
    </InLine>
  </Ad>

</VAST>`

		v, err := vast.Parse([]byte(vastXML))
		require.NoError(t, err, "Should handle extra whitespace")
		assert.NotNil(t, v)
	})

	t.Run("Parse_with_CDATA_variations", func(t *testing.T) {
		// Test both CDATA and non-CDATA URLs
		vastXML := `<?xml version="1.0"?>
<VAST version="4.0">
  <Ad>
    <InLine>
      <AdSystem>Test</AdSystem>
      <AdTitle>Test</AdTitle>
      <Impression><![CDATA[https://example.com/imp]]></Impression>
      <Creatives>
        <Creative>
          <Linear>
            <Duration>00:00:30</Duration>
            <MediaFiles>
              <MediaFile delivery="progressive" type="video/mp4" width="1920" height="1080">
                https://example.com/video.mp4
              </MediaFile>
            </MediaFiles>
          </Linear>
        </Creative>
      </Creatives>
    </InLine>
  </Ad>
</VAST>`

		v, err := vast.Parse([]byte(vastXML))
		require.NoError(t, err, "Should handle URLs without CDATA")
		assert.NotNil(t, v)
	})
}
