package storage

import (
	"context"
	"testing"
	"time"
)

func TestWithTimeout_NoExistingDeadline(t *testing.T) {
	ctx := context.Background()

	newCtx, cancel := withTimeout(ctx, 2*time.Second)
	defer cancel()

	deadline, hasDeadline := newCtx.Deadline()
	if !hasDeadline {
		t.Error("Expected context to have deadline")
	}

	if time.Until(deadline) > 2*time.Second {
		t.Error("Expected deadline to be approximately 2 seconds from now")
	}
}

func TestWithTimeout_ExistingDeadline(t *testing.T) {
	// Create context with existing deadline
	ctx, existingCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer existingCancel()

	existingDeadline, _ := ctx.Deadline()

	// withTimeout should preserve existing deadline
	newCtx, cancel := withTimeout(ctx, 5*time.Second)
	defer cancel()

	newDeadline, hasDeadline := newCtx.Deadline()
	if !hasDeadline {
		t.Error("Expected context to have deadline")
	}

	// Should be the same deadline (not extended)
	if !newDeadline.Equal(existingDeadline) {
		t.Errorf("Expected deadline to be preserved: got %v, want %v", newDeadline, existingDeadline)
	}
}

func TestWithTimeout_DefaultTimeout(t *testing.T) {
	ctx := context.Background()

	newCtx, cancel := withTimeout(ctx, DefaultDBTimeout)
	defer cancel()

	deadline, hasDeadline := newCtx.Deadline()
	if !hasDeadline {
		t.Error("Expected context to have deadline")
	}

	timeUntilDeadline := time.Until(deadline)
	if timeUntilDeadline > DefaultDBTimeout || timeUntilDeadline < DefaultDBTimeout-100*time.Millisecond {
		t.Errorf("Expected deadline around %v, got %v", DefaultDBTimeout, timeUntilDeadline)
	}
}

func TestWithTimeout_CancelFunction(t *testing.T) {
	ctx := context.Background()

	newCtx, cancel := withTimeout(ctx, 1*time.Second)

	// Cancel immediately
	cancel()

	// Context should be cancelled
	select {
	case <-newCtx.Done():
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected context to be cancelled")
	}
}

func TestWithTimeout_NoOpCancelForExistingDeadline(t *testing.T) {
	ctx, existingCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer existingCancel()

	// withTimeout returns no-op cancel when deadline exists
	_, cancel := withTimeout(ctx, 5*time.Second)

	// This should not panic
	cancel()
	cancel() // Multiple calls should be safe
}
