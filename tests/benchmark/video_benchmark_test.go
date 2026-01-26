package benchmark

import (
	"encoding/xml"
	"testing"
	"time"

	"github.com/thenexusengine/tne_springwire/internal/exchange"
	"github.com/thenexusengine/tne_springwire/internal/openrtb"
	"github.com/thenexusengine/tne_springwire/pkg/vast"
)

// BenchmarkVASTGeneration benchmarks VAST XML generation
func BenchmarkVASTGeneration(b *testing.B) {
	b.Run("Simple_Inline_VAST", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := vast.NewBuilder("4.0").
				AddAd("test-ad").
				WithInLine("TNEVideo", "Test Ad").
				WithImpression("https://tracking.example.com/imp").
				WithLinearCreative("creative-1", 30*time.Second).
				WithMediaFile("https://cdn.example.com/video.mp4", "video/mp4", 1920, 1080).
				EndLinear().
				Done().
				Build()

			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Complex_VAST_With_Tracking", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := vast.NewBuilder("4.0").
				AddAd("test-ad").
				WithInLine("TNEVideo", "Test Ad").
				WithImpression("https://tracking.example.com/imp").
				WithError("https://tracking.example.com/error").
				WithLinearCreative("creative-1", 30*time.Second).
				WithMediaFile("https://cdn.example.com/video.mp4", "video/mp4", 1920, 1080).
				WithTracking(vast.EventStart, "https://tracking.example.com/start").
				WithTracking(vast.EventFirstQuartile, "https://tracking.example.com/25").
				WithTracking(vast.EventMidpoint, "https://tracking.example.com/50").
				WithTracking(vast.EventThirdQuartile, "https://tracking.example.com/75").
				WithTracking(vast.EventComplete, "https://tracking.example.com/complete").
				WithClickThrough("https://advertiser.example.com/landing").
				EndLinear().
				Done().
				Build()

			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("VAST_Marshal_To_XML", func(b *testing.B) {
		v, _ := vast.NewBuilder("4.0").
			AddAd("test-ad").
			WithInLine("TNEVideo", "Test Ad").
			WithImpression("https://tracking.example.com/imp").
			WithLinearCreative("creative-1", 30*time.Second).
			WithMediaFile("https://cdn.example.com/video.mp4", "video/mp4", 1920, 1080).
			EndLinear().
			Done().
			Build()

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := v.Marshal()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkVASTParsing benchmarks VAST XML parsing
func BenchmarkVASTParsing(b *testing.B) {
	vastXML := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<VAST version="4.0">
  <Ad id="test-ad">
    <InLine>
      <AdSystem>TNEVideo</AdSystem>
      <AdTitle>Test Ad</AdTitle>
      <Impression><![CDATA[https://tracking.example.com/imp]]></Impression>
      <Error><![CDATA[https://tracking.example.com/error]]></Error>
      <Creatives>
        <Creative id="creative-1">
          <Linear>
            <Duration>00:00:30</Duration>
            <MediaFiles>
              <MediaFile delivery="progressive" type="video/mp4" width="1920" height="1080">
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
            </VideoClicks>
          </Linear>
        </Creative>
      </Creatives>
    </InLine>
  </Ad>
</VAST>`)

	b.Run("Parse_VAST_XML", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := vast.Parse(vastXML)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Unmarshal_Raw_XML", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var v vast.VAST
			err := xml.Unmarshal(vastXML, &v)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Parse_And_Extract_MediaFiles", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			v, err := vast.Parse(vastXML)
			if err != nil {
				b.Fatal(err)
			}
			_ = v.GetMediaFiles()
		}
	})
}

// BenchmarkDurationOperations benchmarks duration parsing and formatting
func BenchmarkDurationOperations(b *testing.B) {
	b.Run("Parse_Duration", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := vast.ParseDuration("00:01:30")
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Format_Duration", func(b *testing.B) {
		duration := 90 * time.Second
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = vast.FormatDuration(duration)
		}
	})

	b.Run("Duration_Round_Trip", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			duration := 90 * time.Second
			formatted := vast.FormatDuration(duration)
			_, err := vast.ParseDuration(formatted)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkVASTResponseBuilder benchmarks building VAST from auction results
func BenchmarkVASTResponseBuilder(b *testing.B) {
	bidReq := createBenchmarkBidRequest()
	auctionResp := createBenchmarkAuctionResponse()
	builder := exchange.NewVASTResponseBuilder("https://tracking.example.com")

	b.Run("Build_VAST_From_Auction", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := builder.BuildVASTFromAuction(bidReq, auctionResp)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Complete_Auction_To_XML", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			vastResp, err := builder.BuildVASTFromAuction(bidReq, auctionResp)
			if err != nil {
				b.Fatal(err)
			}
			_, err = vastResp.Marshal()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkConcurrentOperations benchmarks concurrent VAST operations
func BenchmarkConcurrentOperations(b *testing.B) {
	b.Run("Concurrent_VAST_Generation", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, err := vast.NewBuilder("4.0").
					AddAd("test-ad").
					WithInLine("TNEVideo", "Test Ad").
					WithImpression("https://tracking.example.com/imp").
					WithLinearCreative("creative-1", 30*time.Second).
					WithMediaFile("https://cdn.example.com/video.mp4", "video/mp4", 1920, 1080).
					EndLinear().
					Done().
					Build()

				if err != nil {
					b.Fatal(err)
				}
			}
		})
	})

	vastXML := []byte(`<?xml version="1.0"?><VAST version="4.0"><Ad><InLine><AdSystem>Test</AdSystem><AdTitle>Test</AdTitle><Impression><![CDATA[https://example.com/imp]]></Impression><Creatives><Creative><Linear><Duration>00:00:30</Duration><MediaFiles><MediaFile delivery="progressive" type="video/mp4" width="1920" height="1080"><![CDATA[https://example.com/video.mp4]]></MediaFile></MediaFiles></Linear></Creative></Creatives></InLine></Ad></VAST>`)

	b.Run("Concurrent_VAST_Parsing", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, err := vast.Parse(vastXML)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	})
}

// BenchmarkVASTValidation benchmarks VAST validation
func BenchmarkVASTValidation(b *testing.B) {
	v, _ := vast.NewBuilder("4.0").
		AddAd("test-ad").
		WithInLine("TNEVideo", "Test Ad").
		WithImpression("https://tracking.example.com/imp").
		WithLinearCreative("creative-1", 30*time.Second).
		WithMediaFile("https://cdn.example.com/video.mp4", "video/mp4", 1920, 1080).
		EndLinear().
		Done().
		Build()

	b.Run("Validate_VAST", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result := v.Validate()
			if !result.Valid {
				b.Fatal("Validation failed")
			}
		}
	})
}

// BenchmarkMemoryAllocations tests memory efficiency
func BenchmarkMemoryAllocations(b *testing.B) {
	b.Run("VAST_Builder_Allocations", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			builder := vast.NewBuilder("4.0")
			builder.AddAd("test-ad")
			builder.WithInLine("TNEVideo", "Test Ad")
			builder.WithImpression("https://tracking.example.com/imp")
			linearBuilder := builder.WithLinearCreative("creative-1", 30*time.Second)
			linearBuilder.WithMediaFile("https://cdn.example.com/video.mp4", "video/mp4", 1920, 1080)
			linearBuilder.EndLinear()
			builder.Done()
			_, _ = builder.Build()
		}
	})
}

// Helper functions for benchmarks

func createBenchmarkBidRequest() *openrtb.BidRequest {
	skip := 1
	return &openrtb.BidRequest{
		ID: "benchmark-request",
		Imp: []openrtb.Imp{
			{
				ID: "1",
				Video: &openrtb.Video{
					Mimes:       []string{"video/mp4"},
					MinDuration: 5,
					MaxDuration: 30,
					Protocols:   []int{2, 3, 5, 6},
					W:           1920,
					H:           1080,
					Placement:   1,
					Linearity:   1,
					Skip:        &skip,
					SkipAfter:   5,
					MinBitrate:  1000,
					MaxBitrate:  5000,
				},
				BidFloor:    2.5,
				BidFloorCur: "USD",
			},
		},
		Device: &openrtb.Device{
			UA: "Mozilla/5.0",
			IP: "203.0.113.1",
		},
		TMax: 1000,
		Cur:  []string{"USD"},
		AT:   2,
	}
}

func createBenchmarkAuctionResponse() *exchange.AuctionResponse {
	return &exchange.AuctionResponse{
		BidResponse: &openrtb.BidResponse{
			ID: "benchmark-response",
			SeatBid: []openrtb.SeatBid{
				{
					Bid: []openrtb.Bid{
						{
							ID:    "bid-001",
							ImpID: "1",
							Price: 5.0,
							AdM:   "https://cdn.example.com/video.mp4",
							NURL:  "https://win.example.com/win",
							CRID:  "creative-001",
							W:     1920,
							H:     1080,
						},
					},
					Seat: "benchmark-bidder",
				},
			},
			Cur: "USD",
		},
	}
}
