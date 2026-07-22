package dataplane

import (
	"context"
	"testing"
	"time"

	"github.com/curefatih/afi/internal/snapshot"
)

func TestMaxTriesFor(t *testing.T) {
	t.Parallel()
	if maxTriesFor(nil) != 1 {
		t.Fatal("nil")
	}
	if maxTriesFor(&snapshot.RetryConfig{MaxAttempts: 0}) != 1 {
		t.Fatal("zero")
	}
	if maxTriesFor(&snapshot.RetryConfig{MaxAttempts: 4}) != 4 {
		t.Fatal("four")
	}
}

func TestSleepBeforeRetryRespectsContext(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cfg := &snapshot.RetryConfig{
		MaxAttempts: 2,
		Backoff:     snapshot.BackoffConfig{Strategy: snapshot.BackoffFixed, BaseDelay: "1h"},
	}
	start := time.Now()
	err := sleepBeforeRetry(ctx, cfg, 0)
	if err == nil {
		t.Fatal("expected context error")
	}
	if time.Since(start) > 200*time.Millisecond {
		t.Fatalf("slept too long: %v", time.Since(start))
	}
}
