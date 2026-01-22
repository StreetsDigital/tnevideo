package filterpipeline

import (
	"context"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// FilterContext provides contextual information to filters during execution.
type FilterContext struct {
	// AccountID is the account associated with the request
	AccountID string
	// Endpoint is the current endpoint being processed
	Endpoint string
	// StartTime is when the request processing started
	StartTime time.Time
	// Metadata holds arbitrary key-value pairs for filter communication
	Metadata map[string]interface{}
}

// PreFilterRequest represents the request payload for pre-filters.
type PreFilterRequest struct {
	// Request is the wrapped OpenRTB bid request
	Request *openrtb_ext.RequestWrapper
	// Context provides filter execution context
	Context FilterContext
}

// PreFilterResponse represents the response from a pre-filter.
type PreFilterResponse struct {
	// Request is the potentially modified request
	Request *openrtb_ext.RequestWrapper
	// Reject indicates if the request should be rejected
	Reject bool
	// RejectReason provides the reason for rejection
	RejectReason string
	// Errors contains any errors encountered during filtering
	Errors []string
	// Warnings contains any warnings generated during filtering
	Warnings []string
	// Metadata holds data to pass to subsequent filters
	Metadata map[string]interface{}
}

// PostFilterRequest represents the request payload for post-filters.
type PostFilterRequest struct {
	// Request is the original bid request
	Request *openrtb_ext.RequestWrapper
	// Response is the bid response to be filtered
	Response *openrtb2.BidResponse
	// Context provides filter execution context
	Context FilterContext
}

// PostFilterResponse represents the response from a post-filter.
type PostFilterResponse struct {
	// Response is the potentially modified response
	Response *openrtb2.BidResponse
	// Reject indicates if the response should be rejected
	Reject bool
	// RejectReason provides the reason for rejection
	RejectReason string
	// Errors contains any errors encountered during filtering
	Errors []string
	// Warnings contains any warnings generated during filtering
	Warnings []string
	// Metadata holds data for analytics or logging
	Metadata map[string]interface{}
}

// PreFilter is the interface for filters that run before auction processing.
// Pre-filters can enrich requests, validate input, add identity data, etc.
type PreFilter interface {
	// Name returns the unique identifier for this filter
	Name() string

	// Execute processes the request and returns a filtered result
	Execute(ctx context.Context, req PreFilterRequest) (PreFilterResponse, error)

	// Enabled determines if this filter should be executed based on configuration
	Enabled(accountID string) bool

	// Priority returns the execution priority (lower values execute first)
	Priority() int
}

// PostFilter is the interface for filters that run after auction processing.
// Post-filters can modify responses, enforce policies, add tracking, etc.
type PostFilter interface {
	// Name returns the unique identifier for this filter
	Name() string

	// Execute processes the response and returns a filtered result
	Execute(ctx context.Context, req PostFilterRequest) (PostFilterResponse, error)

	// Enabled determines if this filter should be executed based on configuration
	Enabled(accountID string) bool

	// Priority returns the execution priority (lower values execute first)
	Priority() int
}

// FilterMetrics provides methods for recording filter execution metrics.
type FilterMetrics interface {
	// RecordFilterExecution records that a filter was executed
	RecordFilterExecution(filterName string, filterType string, duration time.Duration)

	// RecordFilterError records that a filter encountered an error
	RecordFilterError(filterName string, filterType string)

	// RecordFilterRejection records that a filter rejected a request/response
	RecordFilterRejection(filterName string, filterType string, reason string)
}
