package snapshot

import (
	"testing"
	"time"
)

func TestRetryConfigDelay(t *testing.T) {
	t.Parallel()
	fixed := &RetryConfig{
		MaxAttempts: 2,
		Backoff:     BackoffConfig{Strategy: BackoffFixed, BaseDelay: "25ms"},
	}
	if fixed.Delay(0) != 25*time.Millisecond || fixed.Delay(5) != 25*time.Millisecond {
		t.Fatalf("fixed delay=%v %v", fixed.Delay(0), fixed.Delay(5))
	}

	exp := &RetryConfig{
		MaxAttempts: 5,
		Backoff: BackoffConfig{
			Strategy:   BackoffExponential,
			BaseDelay:  "10ms",
			MaxDelay:   "50ms",
			Multiplier: 3,
		},
	}
	if exp.Delay(0) != 10*time.Millisecond {
		t.Fatalf("exp0=%v", exp.Delay(0))
	}
	if exp.Delay(1) != 30*time.Millisecond {
		t.Fatalf("exp1=%v", exp.Delay(1))
	}
	if exp.Delay(2) != 50*time.Millisecond {
		t.Fatalf("exp2 capped=%v", exp.Delay(2))
	}
	if (*RetryConfig)(nil).Delay(0) != 0 {
		t.Fatal("nil delay")
	}
}
