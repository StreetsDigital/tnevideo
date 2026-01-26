package exchange

import (
	"testing"

	"github.com/thenexusengine/tne_springwire/internal/adapters"
)

// BenchmarkBidSliceAllocation compares direct allocation vs pool
func BenchmarkBidSliceAllocation(b *testing.B) {
	b.Run("DirectAllocation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var validBids []ValidatedBid
			// Simulate typical auction: 10 bids
			for j := 0; j < 10; j++ {
				validBids = append(validBids, ValidatedBid{
					BidderCode: "bidder",
					DemandType: adapters.DemandTypePlatform,
				})
			}
			// Simulates slice going out of scope
			_ = validBids
		}
	})

	b.Run("PooledAllocation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			validBidsPtr := getValidBidsSlice()
			validBids := *validBidsPtr
			// Simulate typical auction: 10 bids
			for j := 0; j < 10; j++ {
				validBids = append(validBids, ValidatedBid{
					BidderCode: "bidder",
					DemandType: adapters.DemandTypePlatform,
				})
			}
			*validBidsPtr = validBids
			putValidBidsSlice(validBidsPtr)
		}
	})
}

// BenchmarkErrorSliceAllocation compares direct allocation vs pool
func BenchmarkErrorSliceAllocation(b *testing.B) {
	b.Run("DirectAllocation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var errors []error
			// Simulate typical auction: 2-3 errors
			for j := 0; j < 3; j++ {
				errors = append(errors, &BidValidationError{
					BidID:  "bid123",
					Reason: "test error",
				})
			}
			_ = errors
		}
	})

	b.Run("PooledAllocation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			errorsPtr := getValidationErrorsSlice()
			errors := *errorsPtr
			// Simulate typical auction: 2-3 errors
			for j := 0; j < 3; j++ {
				errors = append(errors, &BidValidationError{
					BidID:  "bid123",
					Reason: "test error",
				})
			}
			*errorsPtr = errors
			putValidationErrorsSlice(errorsPtr)
		}
	})
}

// BenchmarkPoolContention tests pool performance under concurrent access
func BenchmarkPoolContention(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			validBidsPtr := getValidBidsSlice()
			validBids := *validBidsPtr
			for j := 0; j < 10; j++ {
				validBids = append(validBids, ValidatedBid{
					BidderCode: "bidder",
					DemandType: adapters.DemandTypePlatform,
				})
			}
			*validBidsPtr = validBids
			putValidBidsSlice(validBidsPtr)
		}
	})
}
