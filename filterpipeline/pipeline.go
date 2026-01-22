package filterpipeline

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/logger"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// Pipeline manages the execution of pre and post filters.
type Pipeline struct {
	preFilters  []PreFilter
	postFilters []PostFilter
	metrics     FilterMetrics
	config      PipelineConfig
	mu          sync.RWMutex
}

// PipelineConfig holds configuration for the filter pipeline.
type PipelineConfig struct {
	// Enabled determines if the pipeline is active
	Enabled bool
	// MaxPreFilterDuration is the maximum time allowed for all pre-filters
	MaxPreFilterDuration time.Duration
	// MaxPostFilterDuration is the maximum time allowed for all post-filters
	MaxPostFilterDuration time.Duration
	// ContinueOnError determines if pipeline continues after filter errors
	ContinueOnError bool
	// ParallelExecution enables concurrent filter execution (same priority)
	ParallelExecution bool
}

// DefaultPipelineConfig returns a pipeline configuration with sensible defaults.
func DefaultPipelineConfig() PipelineConfig {
	return PipelineConfig{
		Enabled:               true,
		MaxPreFilterDuration:  100 * time.Millisecond,
		MaxPostFilterDuration: 100 * time.Millisecond,
		ContinueOnError:       true,
		ParallelExecution:     false,
	}
}

// NewPipeline creates a new filter pipeline with the given configuration.
func NewPipeline(config PipelineConfig, metrics FilterMetrics) *Pipeline {
	return &Pipeline{
		preFilters:  make([]PreFilter, 0),
		postFilters: make([]PostFilter, 0),
		metrics:     metrics,
		config:      config,
	}
}

// RegisterPreFilter adds a pre-filter to the pipeline.
// Filters are executed in priority order (lower values first).
func (p *Pipeline) RegisterPreFilter(filter PreFilter) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check for duplicate filter names
	for _, existing := range p.preFilters {
		if existing.Name() == filter.Name() {
			return fmt.Errorf("pre-filter with name %q already registered", filter.Name())
		}
	}

	p.preFilters = append(p.preFilters, filter)
	p.sortPreFilters()
	logger.Infof("Registered pre-filter: %s (priority: %d)", filter.Name(), filter.Priority())
	return nil
}

// RegisterPostFilter adds a post-filter to the pipeline.
// Filters are executed in priority order (lower values first).
func (p *Pipeline) RegisterPostFilter(filter PostFilter) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check for duplicate filter names
	for _, existing := range p.postFilters {
		if existing.Name() == filter.Name() {
			return fmt.Errorf("post-filter with name %q already registered", filter.Name())
		}
	}

	p.postFilters = append(p.postFilters, filter)
	p.sortPostFilters()
	logger.Infof("Registered post-filter: %s (priority: %d)", filter.Name(), filter.Priority())
	return nil
}

// ExecutePreFilters runs all enabled pre-filters on the request.
func (p *Pipeline) ExecutePreFilters(ctx context.Context, req *openrtb_ext.RequestWrapper, accountID, endpoint string) (*openrtb_ext.RequestWrapper, error) {
	if !p.config.Enabled {
		return req, nil
	}

	p.mu.RLock()
	filters := p.getEnabledPreFilters(accountID)
	p.mu.RUnlock()

	if len(filters) == 0 {
		return req, nil
	}

	// Create filter context
	filterCtx := FilterContext{
		AccountID: accountID,
		Endpoint:  endpoint,
		StartTime: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	// Apply timeout to the entire pre-filter execution
	execCtx, cancel := context.WithTimeout(ctx, p.config.MaxPreFilterDuration)
	defer cancel()

	filterReq := PreFilterRequest{
		Request: req,
		Context: filterCtx,
	}

	var allErrors []string
	var allWarnings []string

	for _, filter := range filters {
		startTime := time.Now()
		resp, err := filter.Execute(execCtx, filterReq)
		duration := time.Since(startTime)

		// Record metrics
		if p.metrics != nil {
			p.metrics.RecordFilterExecution(filter.Name(), "pre", duration)
		}

		if err != nil {
			logger.Errorf("Pre-filter %s failed: %v", filter.Name(), err)
			if p.metrics != nil {
				p.metrics.RecordFilterError(filter.Name(), "pre")
			}
			if !p.config.ContinueOnError {
				return req, fmt.Errorf("pre-filter %s failed: %w", filter.Name(), err)
			}
			allErrors = append(allErrors, fmt.Sprintf("%s: %v", filter.Name(), err))
			continue
		}

		// Handle rejection
		if resp.Reject {
			logger.Infof("Pre-filter %s rejected request: %s", filter.Name(), resp.RejectReason)
			if p.metrics != nil {
				p.metrics.RecordFilterRejection(filter.Name(), "pre", resp.RejectReason)
			}
			return req, &FilterRejectionError{
				FilterName: filter.Name(),
				Reason:     resp.RejectReason,
			}
		}

		// Collect errors and warnings
		allErrors = append(allErrors, resp.Errors...)
		allWarnings = append(allWarnings, resp.Warnings...)

		// Update request for next filter
		if resp.Request != nil {
			filterReq.Request = resp.Request
		}

		// Merge metadata
		for k, v := range resp.Metadata {
			filterReq.Context.Metadata[k] = v
		}

		logger.Debugf("Pre-filter %s executed successfully in %v", filter.Name(), duration)
	}

	// Log accumulated warnings
	for _, warning := range allWarnings {
		logger.Warnf("Pre-filter warning: %s", warning)
	}

	return filterReq.Request, nil
}

// ExecutePostFilters runs all enabled post-filters on the response.
func (p *Pipeline) ExecutePostFilters(ctx context.Context, req *openrtb_ext.RequestWrapper, resp *openrtb2.BidResponse, accountID, endpoint string) (*openrtb2.BidResponse, error) {
	if !p.config.Enabled {
		return resp, nil
	}

	p.mu.RLock()
	filters := p.getEnabledPostFilters(accountID)
	p.mu.RUnlock()

	if len(filters) == 0 {
		return resp, nil
	}

	// Create filter context
	filterCtx := FilterContext{
		AccountID: accountID,
		Endpoint:  endpoint,
		StartTime: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	// Apply timeout to the entire post-filter execution
	execCtx, cancel := context.WithTimeout(ctx, p.config.MaxPostFilterDuration)
	defer cancel()

	filterReq := PostFilterRequest{
		Request:  req,
		Response: resp,
		Context:  filterCtx,
	}

	var allErrors []string
	var allWarnings []string

	for _, filter := range filters {
		startTime := time.Now()
		filterResp, err := filter.Execute(execCtx, filterReq)
		duration := time.Since(startTime)

		// Record metrics
		if p.metrics != nil {
			p.metrics.RecordFilterExecution(filter.Name(), "post", duration)
		}

		if err != nil {
			logger.Errorf("Post-filter %s failed: %v", filter.Name(), err)
			if p.metrics != nil {
				p.metrics.RecordFilterError(filter.Name(), "post")
			}
			if !p.config.ContinueOnError {
				return resp, fmt.Errorf("post-filter %s failed: %w", filter.Name(), err)
			}
			allErrors = append(allErrors, fmt.Sprintf("%s: %v", filter.Name(), err))
			continue
		}

		// Handle rejection
		if filterResp.Reject {
			logger.Infof("Post-filter %s rejected response: %s", filter.Name(), filterResp.RejectReason)
			if p.metrics != nil {
				p.metrics.RecordFilterRejection(filter.Name(), "post", filterResp.RejectReason)
			}
			return resp, &FilterRejectionError{
				FilterName: filter.Name(),
				Reason:     filterResp.RejectReason,
			}
		}

		// Collect errors and warnings
		allErrors = append(allErrors, filterResp.Errors...)
		allWarnings = append(allWarnings, filterResp.Warnings...)

		// Update response for next filter
		if filterResp.Response != nil {
			filterReq.Response = filterResp.Response
		}

		// Merge metadata
		for k, v := range filterResp.Metadata {
			filterReq.Context.Metadata[k] = v
		}

		logger.Debugf("Post-filter %s executed successfully in %v", filter.Name(), duration)
	}

	// Log accumulated warnings
	for _, warning := range allWarnings {
		logger.Warnf("Post-filter warning: %s", warning)
	}

	return filterReq.Response, nil
}

// getEnabledPreFilters returns all pre-filters that are enabled for the given account.
func (p *Pipeline) getEnabledPreFilters(accountID string) []PreFilter {
	enabled := make([]PreFilter, 0, len(p.preFilters))
	for _, filter := range p.preFilters {
		if filter.Enabled(accountID) {
			enabled = append(enabled, filter)
		}
	}
	return enabled
}

// getEnabledPostFilters returns all post-filters that are enabled for the given account.
func (p *Pipeline) getEnabledPostFilters(accountID string) []PostFilter {
	enabled := make([]PostFilter, 0, len(p.postFilters))
	for _, filter := range p.postFilters {
		if filter.Enabled(accountID) {
			enabled = append(enabled, filter)
		}
	}
	return enabled
}

// sortPreFilters sorts pre-filters by priority (lower values first).
func (p *Pipeline) sortPreFilters() {
	sort.Slice(p.preFilters, func(i, j int) bool {
		return p.preFilters[i].Priority() < p.preFilters[j].Priority()
	})
}

// sortPostFilters sorts post-filters by priority (lower values first).
func (p *Pipeline) sortPostFilters() {
	sort.Slice(p.postFilters, func(i, j int) bool {
		return p.postFilters[i].Priority() < p.postFilters[j].Priority()
	})
}

// FilterRejectionError is returned when a filter rejects a request or response.
type FilterRejectionError struct {
	FilterName string
	Reason     string
}

func (e *FilterRejectionError) Error() string {
	return fmt.Sprintf("filter %q rejected: %s", e.FilterName, e.Reason)
}
