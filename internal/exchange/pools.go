// Package exchange - Performance optimization: sync.Pool for bid slice allocations
package exchange

import (
	"sync"
)

// Pools for reusing bid and error slices to reduce GC pressure
var (
	// validBidsPool provides pooled slices for ValidatedBid
	validBidsPool = sync.Pool{
		New: func() interface{} {
			// Pre-allocate with reasonable capacity for typical auctions
			s := make([]ValidatedBid, 0, 32)
			return &s
		},
	}

	// validationErrorsPool provides pooled slices for errors
	validationErrorsPool = sync.Pool{
		New: func() interface{} {
			s := make([]error, 0, 8)
			return &s
		},
	}
)

// getValidBidsSlice acquires a ValidatedBid slice from the pool
func getValidBidsSlice() *[]ValidatedBid {
	s := validBidsPool.Get().(*[]ValidatedBid)
	*s = (*s)[:0] // Reset length while keeping capacity
	return s
}

// putValidBidsSlice returns a ValidatedBid slice to the pool
func putValidBidsSlice(s *[]ValidatedBid) {
	if s == nil {
		return
	}
	// Clear references to prevent memory leaks
	for i := range *s {
		(*s)[i] = ValidatedBid{}
	}
	validBidsPool.Put(s)
}

// getValidationErrorsSlice acquires an error slice from the pool
func getValidationErrorsSlice() *[]error {
	s := validationErrorsPool.Get().(*[]error)
	*s = (*s)[:0] // Reset length while keeping capacity
	return s
}

// putValidationErrorsSlice returns an error slice to the pool
func putValidationErrorsSlice(s *[]error) {
	if s == nil {
		return
	}
	// Clear references to prevent memory leaks
	for i := range *s {
		(*s)[i] = nil
	}
	validationErrorsPool.Put(s)
}
