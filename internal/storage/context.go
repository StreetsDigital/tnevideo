package storage

import (
	"context"
	"time"
)

// DefaultDBTimeout is the default timeout for database operations
const DefaultDBTimeout = 5 * time.Second

// withTimeout wraps a context with a default timeout if it doesn't already have a deadline
func withTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	// Check if context already has a deadline
	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		// Return context as-is with a no-op cancel function
		return ctx, func() {}
	}

	// Add default timeout
	return context.WithTimeout(ctx, timeout)
}
