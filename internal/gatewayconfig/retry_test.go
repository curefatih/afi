package gatewayconfig

import (
	"errors"
	"testing"
	"time"

	"github.com/curefatih/afi/internal/kernel"
)

func TestNormalizeRetryNil(t *testing.T) {
	t.Parallel()
	got, err := NormalizeRetry(nil)
	if err != nil || got != nil {
		t.Fatalf("got=%v err=%v", got, err)
	}
}

func TestNormalizeRetryFixed(t *testing.T) {
	t.Parallel()
	got, err := NormalizeRetry(&RetryConfig{
		MaxAttempts: 3,
		Backoff:     BackoffConfig{Strategy: "FIXED", BaseDelay: " 100ms "},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Backoff.Strategy != BackoffFixed || got.Backoff.BaseDelay != "100ms" {
		t.Fatalf("got=%+v", got)
	}
	if got.Delay(0) != 100*time.Millisecond || got.Delay(2) != 100*time.Millisecond {
		t.Fatalf("delay=%v %v", got.Delay(0), got.Delay(2))
	}
}

func TestNormalizeRetryExponentialDefaultsMultiplier(t *testing.T) {
	t.Parallel()
	got, err := NormalizeRetry(&RetryConfig{
		MaxAttempts: 4,
		Backoff: BackoffConfig{
			Strategy:  BackoffExponential,
			BaseDelay: "100ms",
			MaxDelay:  "500ms",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Backoff.Multiplier != 2 {
		t.Fatalf("multiplier=%v", got.Backoff.Multiplier)
	}
	if got.Delay(0) != 100*time.Millisecond {
		t.Fatalf("delay0=%v", got.Delay(0))
	}
	if got.Delay(1) != 200*time.Millisecond {
		t.Fatalf("delay1=%v", got.Delay(1))
	}
	if got.Delay(2) != 400*time.Millisecond {
		t.Fatalf("delay2=%v", got.Delay(2))
	}
	if got.Delay(3) != 500*time.Millisecond {
		t.Fatalf("delay3 capped=%v", got.Delay(3))
	}
}

func TestValidateRetryRejects(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		cfg  RetryConfig
	}{
		{name: "attempts", cfg: RetryConfig{MaxAttempts: 0, Backoff: BackoffConfig{Strategy: BackoffFixed, BaseDelay: "1ms"}}},
		{name: "strategy", cfg: RetryConfig{MaxAttempts: 1, Backoff: BackoffConfig{Strategy: "linear", BaseDelay: "1ms"}}},
		{name: "fixed max", cfg: RetryConfig{MaxAttempts: 1, Backoff: BackoffConfig{Strategy: BackoffFixed, BaseDelay: "1ms", MaxDelay: "2ms"}}},
		{name: "fixed mult", cfg: RetryConfig{MaxAttempts: 1, Backoff: BackoffConfig{Strategy: BackoffFixed, BaseDelay: "1ms", Multiplier: 2}}},
		{name: "mult lt1", cfg: RetryConfig{MaxAttempts: 1, Backoff: BackoffConfig{Strategy: BackoffExponential, BaseDelay: "1ms", Multiplier: 0.5}}},
		{name: "max lt base", cfg: RetryConfig{MaxAttempts: 1, Backoff: BackoffConfig{Strategy: BackoffExponential, BaseDelay: "10ms", MaxDelay: "1ms"}}},
		{name: "bad duration", cfg: RetryConfig{MaxAttempts: 1, Backoff: BackoffConfig{Strategy: BackoffFixed, BaseDelay: "nope"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := NormalizeRetry(&tc.cfg)
			if !errors.Is(err, kernel.ErrInvalidRequest) {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestNewRouteWithRetry(t *testing.T) {
	t.Parallel()
	r, err := NewRoute("r1", "o1", "m", "prov", "m", nil, &RetryConfig{
		MaxAttempts: 2,
		Backoff:     BackoffConfig{Strategy: BackoffFixed, BaseDelay: "50ms"},
	}, "", 0, timeNowUTC())
	if err != nil {
		t.Fatal(err)
	}
	if r.Retry == nil || r.Retry.MaxAttempts != 2 {
		t.Fatalf("retry=%+v", r.Retry)
	}
	snap := r.Retry.ToSnapshot()
	if snap == nil || snap.Delay(0) != 50*time.Millisecond {
		t.Fatalf("snapshot=%+v", snap)
	}
}
