package gatewayconfig

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/snapshot"
)

// Backoff strategies for route retry.
const (
	BackoffFixed       = "fixed"
	BackoffExponential = "exponential"
)

// RetryConfig controls same-target retries before failover.
// Nil means no same-target retries (single attempt per route target).
type RetryConfig struct {
	MaxAttempts int           `json:"max_attempts"` // includes first try; must be >= 1 when set
	Backoff     BackoffConfig `json:"backoff"`
}

// BackoffConfig selects the delay between retry attempts.
type BackoffConfig struct {
	Strategy   string  `json:"strategy"`             // fixed | exponential
	BaseDelay  string  `json:"base_delay"`           // duration, e.g. "100ms"
	MaxDelay   string  `json:"max_delay,omitempty"`  // exponential only
	Multiplier float64 `json:"multiplier,omitempty"` // exponential only; default 2 when omitted
}

// NormalizeRetry validates and returns a canonical retry config.
// A nil input returns (nil, nil).
func NormalizeRetry(c *RetryConfig) (*RetryConfig, error) {
	if c == nil {
		return nil, nil
	}
	out := *c
	out.Backoff.Strategy = strings.TrimSpace(strings.ToLower(out.Backoff.Strategy))
	out.Backoff.BaseDelay = strings.TrimSpace(out.Backoff.BaseDelay)
	out.Backoff.MaxDelay = strings.TrimSpace(out.Backoff.MaxDelay)
	if err := ValidateRetry(&out); err != nil {
		return nil, err
	}
	if out.Backoff.Strategy == BackoffExponential && out.Backoff.Multiplier == 0 {
		out.Backoff.Multiplier = 2
	}
	return &out, nil
}

// ValidateRetry ensures retry fields are well-formed.
func ValidateRetry(c *RetryConfig) error {
	if c == nil {
		return nil
	}
	if c.MaxAttempts < 1 {
		return fmt.Errorf("%w: max_attempts must be >= 1", kernel.ErrInvalidRequest)
	}
	switch c.Backoff.Strategy {
	case BackoffFixed, BackoffExponential:
	case "":
		return fmt.Errorf("%w: backoff.strategy is required", kernel.ErrInvalidRequest)
	default:
		return fmt.Errorf("%w: backoff.strategy must be fixed or exponential", kernel.ErrInvalidRequest)
	}
	if c.Backoff.BaseDelay == "" {
		return fmt.Errorf("%w: backoff.base_delay is required", kernel.ErrInvalidRequest)
	}
	base, err := time.ParseDuration(c.Backoff.BaseDelay)
	if err != nil {
		return fmt.Errorf("%w: backoff.base_delay: %v", kernel.ErrInvalidRequest, err)
	}
	if base < 0 {
		return fmt.Errorf("%w: backoff.base_delay must be >= 0", kernel.ErrInvalidRequest)
	}

	switch c.Backoff.Strategy {
	case BackoffFixed:
		if c.Backoff.MaxDelay != "" {
			return fmt.Errorf("%w: backoff.max_delay is only valid for exponential", kernel.ErrInvalidRequest)
		}
		if c.Backoff.Multiplier != 0 {
			return fmt.Errorf("%w: backoff.multiplier is only valid for exponential", kernel.ErrInvalidRequest)
		}
	case BackoffExponential:
		if c.Backoff.Multiplier != 0 && c.Backoff.Multiplier < 1 {
			return fmt.Errorf("%w: backoff.multiplier must be >= 1", kernel.ErrInvalidRequest)
		}
		if c.Backoff.MaxDelay != "" {
			max, err := time.ParseDuration(c.Backoff.MaxDelay)
			if err != nil {
				return fmt.Errorf("%w: backoff.max_delay: %v", kernel.ErrInvalidRequest, err)
			}
			if max < base {
				return fmt.Errorf("%w: backoff.max_delay must be >= base_delay", kernel.ErrInvalidRequest)
			}
		}
	}
	return nil
}

// Delay returns the wait before retry attempt index n (0 = first retry after the initial failure).
// Negative n or a nil/invalid config yields 0.
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

// ToSnapshot copies this config into the compiled snapshot shape.
func (c *RetryConfig) ToSnapshot() *snapshot.RetryConfig {
	if c == nil {
		return nil
	}
	return &snapshot.RetryConfig{
		MaxAttempts: c.MaxAttempts,
		Backoff: snapshot.BackoffConfig{
			Strategy:   c.Backoff.Strategy,
			BaseDelay:  c.Backoff.BaseDelay,
			MaxDelay:   c.Backoff.MaxDelay,
			Multiplier: c.Backoff.Multiplier,
		},
	}
}
