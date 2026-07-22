package snapshot

import (
	"math"
	"time"
)

// Backoff strategies for route retry (compiled snapshot).
const (
	BackoffFixed       = "fixed"
	BackoffExponential = "exponential"
)

// RetryConfig controls same-target retries before failover.
// Nil means no same-target retries (single attempt per route target).
type RetryConfig struct {
	MaxAttempts int           `json:"max_attempts"`
	Backoff     BackoffConfig `json:"backoff"`
}

// BackoffConfig selects the delay between retry attempts.
type BackoffConfig struct {
	Strategy   string  `json:"strategy"`
	BaseDelay  string  `json:"base_delay"`
	MaxDelay   string  `json:"max_delay,omitempty"`
	Multiplier float64 `json:"multiplier,omitempty"`
}

// Delay returns the wait before retry attempt index n (0 = first retry after the initial failure).
func (c *RetryConfig) Delay(n int) time.Duration {
	if c == nil || n < 0 {
		return 0
	}
	base, err := time.ParseDuration(c.Backoff.BaseDelay)
	if err != nil || base < 0 {
		return 0
	}
	switch c.Backoff.Strategy {
	case BackoffFixed:
		return base
	case BackoffExponential:
		mult := c.Backoff.Multiplier
		if mult == 0 {
			mult = 2
		}
		d := time.Duration(float64(base) * math.Pow(mult, float64(n)))
		if c.Backoff.MaxDelay != "" {
			if max, err := time.ParseDuration(c.Backoff.MaxDelay); err == nil && d > max {
				return max
			}
		}
		return d
	default:
		return 0
	}
}
