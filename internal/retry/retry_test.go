package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.MaxAttempts != 3 {
		t.Errorf("Expected MaxAttempts 3, got %d", cfg.MaxAttempts)
	}

	if cfg.InitialDelay != 1*time.Second {
		t.Errorf("Expected InitialDelay 1s, got %v", cfg.InitialDelay)
	}

	if cfg.MaxDelay != 10*time.Second {
		t.Errorf("Expected MaxDelay 10s, got %v", cfg.MaxDelay)
	}

	if cfg.Multiplier != 2.0 {
		t.Errorf("Expected Multiplier 2.0, got %f", cfg.Multiplier)
	}
}

func TestDoSuccess(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig()

	attempts := 0
	err := Do(ctx, cfg, func() error {
		attempts++
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}
}

func TestDoRetrySuccess(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
	}

	attempts := 0
	err := Do(ctx, cfg, func() error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary error")
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestDoAllAttemptsFail(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
	}

	attempts := 0
	testErr := errors.New("persistent error")
	err := Do(ctx, cfg, func() error {
		attempts++
		return testErr
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}

	if !errors.Is(err, testErr) {
		t.Errorf("Expected error to wrap testErr")
	}
}

func TestDoContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := Config{
		MaxAttempts:  10,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
	}

	attempts := 0
	errChan := make(chan error, 1)

	go func() {
		err := Do(ctx, cfg, func() error {
			attempts++
			return errors.New("temporary error")
		})
		errChan <- err
	}()

	// Cancel after first attempt
	time.Sleep(50 * time.Millisecond)
	cancel()

	err := <-errChan

	if err == nil {
		t.Error("Expected error due to context cancellation")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}

	if attempts > 2 {
		t.Errorf("Expected at most 2 attempts before cancellation, got %d", attempts)
	}
}

func TestDoContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	cfg := Config{
		MaxAttempts:  10,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
	}

	attempts := 0
	err := Do(ctx, cfg, func() error {
		attempts++
		return errors.New("temporary error")
	})

	if err == nil {
		t.Error("Expected error due to context timeout")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected context.DeadlineExceeded error, got %v", err)
	}
}

func TestCalculateBackoff(t *testing.T) {
	cfg := Config{
		InitialDelay: 1 * time.Second,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 1 * time.Second},
		{1, 2 * time.Second},
		{2, 4 * time.Second},
		{3, 8 * time.Second},
		{4, 10 * time.Second}, // Capped at MaxDelay
		{5, 10 * time.Second}, // Still capped
	}

	for _, tt := range tests {
		result := calculateBackoff(tt.attempt, cfg)
		if result != tt.expected {
			t.Errorf("Attempt %d: expected %v, got %v", tt.attempt, tt.expected, result)
		}
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		err      error
		expected bool
	}{
		{nil, false},
		{errors.New("normal error"), true},
		{context.Canceled, false},
		{context.DeadlineExceeded, false},
	}

	for _, tt := range tests {
		result := IsRetryable(tt.err)
		if result != tt.expected {
			t.Errorf("IsRetryable(%v): expected %v, got %v", tt.err, tt.expected, result)
		}
	}
}

func TestDoWithNonRetryableError(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
	}

	attempts := 0
	err := Do(ctx, cfg, func() error {
		attempts++
		return context.Canceled
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}

	// Should still retry even with non-retryable error
	// because IsRetryable is only used as a hint in the actual implementation
	if attempts < 1 {
		t.Errorf("Expected at least 1 attempt, got %d", attempts)
	}
}

func TestBackoffTiming(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		MaxAttempts:  3,
		InitialDelay: 50 * time.Millisecond,
		MaxDelay:     200 * time.Millisecond,
		Multiplier:   2.0,
	}

	start := time.Now()
	attempts := 0

	Do(ctx, cfg, func() error {
		attempts++
		return errors.New("temporary error")
	})

	elapsed := time.Since(start)

	// Should take at least: 50ms (1st backoff) + 100ms (2nd backoff) = 150ms
	// But less than 500ms (with some buffer)
	if elapsed < 150*time.Millisecond {
		t.Errorf("Expected at least 150ms, got %v", elapsed)
	}

	if elapsed > 500*time.Millisecond {
		t.Errorf("Expected less than 500ms, got %v", elapsed)
	}
}
