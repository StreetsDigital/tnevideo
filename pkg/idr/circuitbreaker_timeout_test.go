package idr

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestCircuitBreakerCallbackTimeout verifies that blocking callbacks are properly timed out
func TestCircuitBreakerCallbackTimeout(t *testing.T) {
	var callbackStarted sync.WaitGroup
	callbackStarted.Add(1)

	config := &CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          1 * time.Second,
		OnStateChange: func(from, to string) {
			callbackStarted.Done()
			// Block forever - this should be timed out by the circuit breaker
			select {}
		},
	}

	cb := NewCircuitBreaker(config)

	// Start time to measure total execution time
	start := time.Now()

	// Force a state change that will trigger the blocking callback
	cb.ForceOpen()

	// Wait for callback to start
	callbackStarted.Wait()

	// The setState call should return quickly despite the blocking callback
	// We give it 6 seconds (5 second timeout + 1 second buffer)
	elapsed := time.Since(start)
	if elapsed > 6*time.Second {
		t.Errorf("setState took too long: %v (expected < 6s)", elapsed)
	}

	// Close should complete within the timeout period
	closeStart := time.Now()
	done := make(chan struct{})
	go func() {
		cb.Close()
		close(done)
	}()

	select {
	case <-done:
		closeElapsed := time.Since(closeStart)
		// Should complete within timeout + small buffer
		if closeElapsed > 6*time.Second {
			t.Errorf("Close() took too long: %v (expected < 6s)", closeElapsed)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Close() blocked for more than 10 seconds - callback timeout not working")
	}
}

// TestCircuitBreakerCallbackNonBlocking verifies normal callbacks still work
func TestCircuitBreakerCallbackNonBlocking(t *testing.T) {
	var callbackCalled bool
	var mu sync.Mutex

	config := &CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          1 * time.Second,
		OnStateChange: func(from, to string) {
			mu.Lock()
			callbackCalled = true
			mu.Unlock()
		},
	}

	cb := NewCircuitBreaker(config)
	cb.ForceOpen()

	// Wait for callback to complete
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	if !callbackCalled {
		t.Error("Callback was not called")
	}
	mu.Unlock()

	// Close should complete quickly
	cb.Close()
}

// TestCircuitBreakerCallbackPanicRecovery verifies panic recovery in callbacks
func TestCircuitBreakerCallbackPanicRecovery(t *testing.T) {
	var panicOccurred atomic.Bool

	config := &CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          1 * time.Second,
		OnStateChange: func(from, to string) {
			panicOccurred.Store(true)
			panic("test panic in callback")
		},
	}

	cb := NewCircuitBreaker(config)

	// This should not crash the program despite the panic
	cb.ForceOpen()

	// Wait for callback goroutine to execute and panic
	time.Sleep(100 * time.Millisecond)

	if !panicOccurred.Load() {
		t.Error("Expected callback to be called and panic")
	}

	// Circuit breaker should still be functional
	if !cb.IsOpen() {
		t.Error("Expected circuit to be open after ForceOpen")
	}

	// Close should complete successfully
	cb.Close()
}

// TestCircuitBreakerCallbackOuterGoroutineExits verifies that outer goroutine exits on timeout
func TestCircuitBreakerCallbackOuterGoroutineExits(t *testing.T) {
	var callbackStarted sync.WaitGroup
	var callbackBlocking sync.WaitGroup
	callbackStarted.Add(1)
	callbackBlocking.Add(1)

	config := &CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          1 * time.Second,
		OnStateChange: func(from, to string) {
			callbackStarted.Done()
			// Block until test releases it
			callbackBlocking.Wait()
		},
	}

	cb := NewCircuitBreaker(config)

	// Trigger state change (this spawns goroutine but returns immediately)
	start := time.Now()
	cb.ForceOpen()
	elapsed := time.Since(start)

	// setState should return immediately since callback runs in goroutine
	if elapsed > 100*time.Millisecond {
		t.Errorf("setState blocked unexpectedly: %v (expected < 100ms)", elapsed)
	}

	if !cb.IsOpen() {
		t.Error("Expected circuit to be open after ForceOpen")
	}

	// Wait for callback to start
	callbackStarted.Wait()

	// Close() should timeout and return within ~5 seconds, not block forever
	closeStart := time.Now()
	done := make(chan struct{})
	go func() {
		cb.Close()
		close(done)
	}()

	select {
	case <-done:
		closeElapsed := time.Since(closeStart)
		// Should complete within timeout + buffer (5s timeout + 1s buffer)
		if closeElapsed > 6*time.Second {
			t.Errorf("Close() took too long: %v (expected < 6s)", closeElapsed)
		}
		if closeElapsed < 4900*time.Millisecond {
			t.Errorf("Close() completed too quickly: %v (expected ~5s for timeout)", closeElapsed)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Close() blocked for more than 10 seconds - outer goroutine not exiting properly")
	}

	// Release the blocking callback to clean up
	callbackBlocking.Done()
	time.Sleep(50 * time.Millisecond) // Let callback goroutine finish
}
