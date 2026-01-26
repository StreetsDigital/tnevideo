package fpd

import (
	"encoding/json"
	"testing"

	"github.com/thenexusengine/tne_springwire/internal/openrtb"
)

// BenchmarkBidderConfigMarshal compares old vs new approach
func BenchmarkBidderConfigMarshal(b *testing.B) {
	// Setup: Create a typical bidder config
	siteFPD := &SiteFPD{
		Name:   "example.com",
		Domain: "example.com",
		Cat:    []string{"IAB1", "IAB2"},
		Page:   "https://example.com/page",
	}
	
	appFPD := &AppFPD{
		Name:   "ExampleApp",
		Bundle: "com.example.app",
		Cat:    []string{"IAB1"},
	}
	
	userFPD := &UserFPD{
		YOB:    1990,
		Gender: "M",
	}

	config := BidderConfig{
		Bidders: []string{"*"},
		Config: &FPDConfig{
			ORTB2: &ORTB2Config{
				Site: siteFPD,
				App:  appFPD,
				User: userFPD,
			},
		},
	}

	configs := []BidderConfig{config}
	bidders := []string{"bidder1", "bidder2", "bidder3", "bidder4", "bidder5"}

	b.Run("OldApproach_MarshalPerBidder", func(b *testing.B) {
		p := NewProcessor(DefaultConfig())
		base := &ResolvedFPD{Imp: make(map[string]json.RawMessage)}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, bidder := range bidders {
				// Old approach: marshal per bidder (3 marshals per bidder)
				_ = p.applyBidderConfig(base, bidder, configs)
			}
		}
	})

	b.Run("NewApproach_PreMarshal", func(b *testing.B) {
		p := NewProcessor(DefaultConfig())
		base := &ResolvedFPD{Imp: make(map[string]json.RawMessage)}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Pre-marshal once
			cached := PrepareBidderConfigs(configs)
			for _, bidder := range bidders {
				// New approach: use pre-marshaled (0 marshals per bidder)
				_ = p.applyBidderConfigCached(base, bidder, cached)
			}
		}
	})
}

// BenchmarkProcessRequestWithBidderConfig tests full request processing
func BenchmarkProcessRequestWithBidderConfig(b *testing.B) {
	p := NewProcessor(&Config{
		Enabled:             true,
		BidderConfigEnabled: true,
		SiteEnabled:         true,
		UserEnabled:         true,
	})

	// Create a request with bidder config
	req := &openrtb.BidRequest{
		ID: "test-request",
		Imp: []openrtb.Imp{
			{ID: "imp1"},
		},
		Site: &openrtb.Site{
			Domain: "example.com",
		},
		User: &openrtb.User{
			ID: "user123",
		},
	}

	// Add bidder config
	bidderConfig := []BidderConfig{
		{
			Bidders: []string{"*"},
			Config: &FPDConfig{
				ORTB2: &ORTB2Config{
					Site: &SiteFPD{
						Name:   "Example Site",
						Cat:    []string{"IAB1", "IAB2"},
						Domain: "example.com",
					},
					User: &UserFPD{
						YOB:    1990,
						Gender: "M",
					},
				},
			},
		},
	}

	ext, _ := json.Marshal(map[string]interface{}{
		"prebid": map[string]interface{}{
			"bidderconfig": bidderConfig,
		},
	})
	req.Ext = ext

	bidders := []string{"bidder1", "bidder2", "bidder3", "bidder4", "bidder5"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = p.ProcessRequest(req, bidders)
	}
}

// BenchmarkPrepareBidderConfigs tests the preparation overhead
func BenchmarkPrepareBidderConfigs(b *testing.B) {
	configs := []BidderConfig{
		{
			Bidders: []string{"bidder1", "bidder2", "bidder3"},
			Config: &FPDConfig{
				ORTB2: &ORTB2Config{
					Site: &SiteFPD{
						Name:   "Site1",
						Domain: "example1.com",
						Cat:    []string{"IAB1"},
					},
				},
			},
		},
		{
			Bidders: []string{"bidder4", "bidder5"},
			Config: &FPDConfig{
				ORTB2: &ORTB2Config{
					User: &UserFPD{
						YOB:    1990,
						Gender: "M",
					},
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = PrepareBidderConfigs(configs)
	}
}
