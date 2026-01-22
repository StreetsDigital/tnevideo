package filterpipeline

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

// Mock pre-filter for testing
type mockPreFilter struct {
	name     string
	priority int
	enabled  bool
	reject   bool
	err      error
	delay    time.Duration
}

func (m *mockPreFilter) Name() string                       { return m.name }
func (m *mockPreFilter) Priority() int                      { return m.priority }
func (m *mockPreFilter) Enabled(accountID string) bool      { return m.enabled }
func (m *mockPreFilter) Execute(ctx context.Context, req PreFilterRequest) (PreFilterResponse, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	resp := PreFilterResponse{
		Request:  req.Request,
		Reject:   m.reject,
		Metadata: map[string]interface{}{"executed": m.name},
	}
	if m.reject {
		resp.RejectReason = "mock rejection"
	}
	return resp, m.err
}

// Mock post-filter for testing
type mockPostFilter struct {
	name     string
	priority int
	enabled  bool
	reject   bool
	err      error
	delay    time.Duration
}

func (m *mockPostFilter) Name() string                      { return m.name }
func (m *mockPostFilter) Priority() int                     { return m.priority }
func (m *mockPostFilter) Enabled(accountID string) bool     { return m.enabled }
func (m *mockPostFilter) Execute(ctx context.Context, req PostFilterRequest) (PostFilterResponse, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	resp := PostFilterResponse{
		Response: req.Response,
		Reject:   m.reject,
		Metadata: map[string]interface{}{"executed": m.name},
	}
	if m.reject {
		resp.RejectReason = "mock rejection"
	}
	return resp, m.err
}

func TestNewPipeline(t *testing.T) {
	config := DefaultPipelineConfig()
	metrics := NewNoOpFilterMetrics()
	pipeline := NewPipeline(config, metrics)

	assert.NotNil(t, pipeline)
	assert.Equal(t, config.Enabled, pipeline.config.Enabled)
	assert.Empty(t, pipeline.preFilters)
	assert.Empty(t, pipeline.postFilters)
}

func TestRegisterPreFilter(t *testing.T) {
	tests := []struct {
		name        string
		filters     []PreFilter
		expectError bool
		errorMsg    string
	}{
		{
			name: "register_single_filter",
			filters: []PreFilter{
				&mockPreFilter{name: "filter1", priority: 10, enabled: true},
			},
			expectError: false,
		},
		{
			name: "register_multiple_filters_with_priority",
			filters: []PreFilter{
				&mockPreFilter{name: "filter1", priority: 20, enabled: true},
				&mockPreFilter{name: "filter2", priority: 10, enabled: true},
				&mockPreFilter{name: "filter3", priority: 30, enabled: true},
			},
			expectError: false,
		},
		{
			name: "register_duplicate_filter",
			filters: []PreFilter{
				&mockPreFilter{name: "filter1", priority: 10, enabled: true},
				&mockPreFilter{name: "filter1", priority: 20, enabled: true},
			},
			expectError: true,
			errorMsg:    "already registered",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipeline := NewPipeline(DefaultPipelineConfig(), NewNoOpFilterMetrics())

			var err error
			for _, filter := range tt.filters {
				err = pipeline.RegisterPreFilter(filter)
				if err != nil {
					break
				}
			}

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.Len(t, pipeline.preFilters, len(tt.filters))

				// Verify filters are sorted by priority
				if len(tt.filters) > 1 {
					for i := 0; i < len(pipeline.preFilters)-1; i++ {
						assert.LessOrEqual(t, pipeline.preFilters[i].Priority(), pipeline.preFilters[i+1].Priority())
					}
				}
			}
		})
	}
}

func TestRegisterPostFilter(t *testing.T) {
	tests := []struct {
		name        string
		filters     []PostFilter
		expectError bool
		errorMsg    string
	}{
		{
			name: "register_single_filter",
			filters: []PostFilter{
				&mockPostFilter{name: "filter1", priority: 10, enabled: true},
			},
			expectError: false,
		},
		{
			name: "register_multiple_filters_with_priority",
			filters: []PostFilter{
				&mockPostFilter{name: "filter1", priority: 20, enabled: true},
				&mockPostFilter{name: "filter2", priority: 10, enabled: true},
				&mockPostFilter{name: "filter3", priority: 30, enabled: true},
			},
			expectError: false,
		},
		{
			name: "register_duplicate_filter",
			filters: []PostFilter{
				&mockPostFilter{name: "filter1", priority: 10, enabled: true},
				&mockPostFilter{name: "filter1", priority: 20, enabled: true},
			},
			expectError: true,
			errorMsg:    "already registered",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipeline := NewPipeline(DefaultPipelineConfig(), NewNoOpFilterMetrics())

			var err error
			for _, filter := range tt.filters {
				err = pipeline.RegisterPostFilter(filter)
				if err != nil {
					break
				}
			}

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.Len(t, pipeline.postFilters, len(tt.filters))

				// Verify filters are sorted by priority
				if len(tt.filters) > 1 {
					for i := 0; i < len(pipeline.postFilters)-1; i++ {
						assert.LessOrEqual(t, pipeline.postFilters[i].Priority(), pipeline.postFilters[i+1].Priority())
					}
				}
			}
		})
	}
}

func TestExecutePreFilters(t *testing.T) {
	tests := []struct {
		name          string
		config        PipelineConfig
		filters       []PreFilter
		expectedError bool
		errorType     string
	}{
		{
			name:   "disabled_pipeline",
			config: PipelineConfig{Enabled: false},
			filters: []PreFilter{
				&mockPreFilter{name: "filter1", priority: 10, enabled: true},
			},
			expectedError: false,
		},
		{
			name:   "successful_execution",
			config: DefaultPipelineConfig(),
			filters: []PreFilter{
				&mockPreFilter{name: "filter1", priority: 10, enabled: true},
				&mockPreFilter{name: "filter2", priority: 20, enabled: true},
			},
			expectedError: false,
		},
		{
			name:   "filter_rejection",
			config: DefaultPipelineConfig(),
			filters: []PreFilter{
				&mockPreFilter{name: "filter1", priority: 10, enabled: true},
				&mockPreFilter{name: "filter2", priority: 20, enabled: true, reject: true},
			},
			expectedError: true,
			errorType:     "rejection",
		},
		{
			name: "filter_error_continue",
			config: PipelineConfig{
				Enabled:               true,
				MaxPreFilterDuration:  1 * time.Second,
				MaxPostFilterDuration: 1 * time.Second,
				ContinueOnError:       true,
			},
			filters: []PreFilter{
				&mockPreFilter{name: "filter1", priority: 10, enabled: true, err: errors.New("filter error")},
				&mockPreFilter{name: "filter2", priority: 20, enabled: true},
			},
			expectedError: false,
		},
		{
			name: "filter_error_stop",
			config: PipelineConfig{
				Enabled:               true,
				MaxPreFilterDuration:  1 * time.Second,
				MaxPostFilterDuration: 1 * time.Second,
				ContinueOnError:       false,
			},
			filters: []PreFilter{
				&mockPreFilter{name: "filter1", priority: 10, enabled: true, err: errors.New("filter error")},
				&mockPreFilter{name: "filter2", priority: 20, enabled: true},
			},
			expectedError: true,
			errorType:     "filter error",
		},
		{
			name:   "disabled_filter_skipped",
			config: DefaultPipelineConfig(),
			filters: []PreFilter{
				&mockPreFilter{name: "filter1", priority: 10, enabled: true},
				&mockPreFilter{name: "filter2", priority: 20, enabled: false},
				&mockPreFilter{name: "filter3", priority: 30, enabled: true},
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipeline := NewPipeline(tt.config, NewNoOpFilterMetrics())

			for _, filter := range tt.filters {
				err := pipeline.RegisterPreFilter(filter)
				assert.NoError(t, err)
			}

			req := &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID: "test-request",
					Imp: []openrtb2.Imp{
						{ID: "imp1"},
					},
				},
			}

			result, err := pipeline.ExecutePreFilters(context.Background(), req, "test-account", "/openrtb2/auction")

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorType != "" {
					assert.Contains(t, err.Error(), tt.errorType)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestExecutePostFilters(t *testing.T) {
	tests := []struct {
		name          string
		config        PipelineConfig
		filters       []PostFilter
		expectedError bool
		errorType     string
	}{
		{
			name:   "disabled_pipeline",
			config: PipelineConfig{Enabled: false},
			filters: []PostFilter{
				&mockPostFilter{name: "filter1", priority: 10, enabled: true},
			},
			expectedError: false,
		},
		{
			name:   "successful_execution",
			config: DefaultPipelineConfig(),
			filters: []PostFilter{
				&mockPostFilter{name: "filter1", priority: 10, enabled: true},
				&mockPostFilter{name: "filter2", priority: 20, enabled: true},
			},
			expectedError: false,
		},
		{
			name:   "filter_rejection",
			config: DefaultPipelineConfig(),
			filters: []PostFilter{
				&mockPostFilter{name: "filter1", priority: 10, enabled: true},
				&mockPostFilter{name: "filter2", priority: 20, enabled: true, reject: true},
			},
			expectedError: true,
			errorType:     "rejection",
		},
		{
			name: "filter_error_continue",
			config: PipelineConfig{
				Enabled:               true,
				MaxPreFilterDuration:  1 * time.Second,
				MaxPostFilterDuration: 1 * time.Second,
				ContinueOnError:       true,
			},
			filters: []PostFilter{
				&mockPostFilter{name: "filter1", priority: 10, enabled: true, err: errors.New("filter error")},
				&mockPostFilter{name: "filter2", priority: 20, enabled: true},
			},
			expectedError: false,
		},
		{
			name: "filter_error_stop",
			config: PipelineConfig{
				Enabled:               true,
				MaxPreFilterDuration:  1 * time.Second,
				MaxPostFilterDuration: 1 * time.Second,
				ContinueOnError:       false,
			},
			filters: []PostFilter{
				&mockPostFilter{name: "filter1", priority: 10, enabled: true, err: errors.New("filter error")},
				&mockPostFilter{name: "filter2", priority: 20, enabled: true},
			},
			expectedError: true,
			errorType:     "filter error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipeline := NewPipeline(tt.config, NewNoOpFilterMetrics())

			for _, filter := range tt.filters {
				err := pipeline.RegisterPostFilter(filter)
				assert.NoError(t, err)
			}

			req := &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID: "test-request",
				},
			}

			resp := &openrtb_ext.BidResponseWrapper{
				BidResponse: &openrtb2.BidResponse{
					ID: "test-response",
					SeatBid: []openrtb2.SeatBid{
						{
							Seat: "bidder1",
							Bid: []*openrtb2.Bid{
								{ID: "bid1", Price: 1.5},
							},
						},
					},
				},
			}

			result, err := pipeline.ExecutePostFilters(context.Background(), req, resp, "test-account", "/openrtb2/auction")

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorType != "" {
					assert.Contains(t, err.Error(), tt.errorType)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestFilterPriority(t *testing.T) {
	pipeline := NewPipeline(DefaultPipelineConfig(), NewNoOpFilterMetrics())

	// Register pre-filters in random order
	_ = pipeline.RegisterPreFilter(&mockPreFilter{name: "filter3", priority: 30, enabled: true})
	_ = pipeline.RegisterPreFilter(&mockPreFilter{name: "filter1", priority: 10, enabled: true})
	_ = pipeline.RegisterPreFilter(&mockPreFilter{name: "filter2", priority: 20, enabled: true})

	// Verify they are sorted by priority
	assert.Equal(t, "filter1", pipeline.preFilters[0].Name())
	assert.Equal(t, "filter2", pipeline.preFilters[1].Name())
	assert.Equal(t, "filter3", pipeline.preFilters[2].Name())

	// Register post-filters in random order
	_ = pipeline.RegisterPostFilter(&mockPostFilter{name: "filter3", priority: 30, enabled: true})
	_ = pipeline.RegisterPostFilter(&mockPostFilter{name: "filter1", priority: 10, enabled: true})
	_ = pipeline.RegisterPostFilter(&mockPostFilter{name: "filter2", priority: 20, enabled: true})

	// Verify they are sorted by priority
	assert.Equal(t, "filter1", pipeline.postFilters[0].Name())
	assert.Equal(t, "filter2", pipeline.postFilters[1].Name())
	assert.Equal(t, "filter3", pipeline.postFilters[2].Name())
}

func TestFilterRejectionError(t *testing.T) {
	err := &FilterRejectionError{
		FilterName: "test_filter",
		Reason:     "test reason",
	}

	assert.Contains(t, err.Error(), "test_filter")
	assert.Contains(t, err.Error(), "test reason")
}
