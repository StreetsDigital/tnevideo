package filterpipeline

import (
	"time"

	"github.com/prebid/prebid-server/v3/logger"
)

// NoOpFilterMetrics is a no-op implementation of FilterMetrics for testing.
type NoOpFilterMetrics struct{}

// RecordFilterExecution is a no-op implementation.
func (m *NoOpFilterMetrics) RecordFilterExecution(filterName string, filterType string, duration time.Duration) {
	logger.Debugf("Filter %s (%s) executed in %v", filterName, filterType, duration)
}

// RecordFilterError is a no-op implementation.
func (m *NoOpFilterMetrics) RecordFilterError(filterName string, filterType string) {
	logger.Errorf("Filter %s (%s) encountered an error", filterName, filterType)
}

// RecordFilterRejection is a no-op implementation.
func (m *NoOpFilterMetrics) RecordFilterRejection(filterName string, filterType string, reason string) {
	logger.Infof("Filter %s (%s) rejected: %s", filterName, filterType, reason)
}

// NewNoOpFilterMetrics creates a new no-op filter metrics instance.
func NewNoOpFilterMetrics() FilterMetrics {
	return &NoOpFilterMetrics{}
}
