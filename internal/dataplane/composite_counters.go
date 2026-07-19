package dataplane

import (
	"context"
	"fmt"

	"github.com/curefatih/afi/internal/snapshot"
)

// CompositeCounters routes lifetime quotas to Postgres and timed windows to Redis.
type CompositeCounters struct {
	Total CounterStore // Postgres (window=total)
	Timed CounterStore // Redis (minute/hour/day)
}

func (c CompositeCounters) storeFor(window string) (CounterStore, error) {
	switch window {
	case snapshot.WindowTotal, "":
		if c.Total == nil {
			return nil, fmt.Errorf("total counter store not configured")
		}
		return c.Total, nil
	case snapshot.WindowMinute, snapshot.WindowHour, snapshot.WindowDay:
		if c.Timed == nil {
			return nil, fmt.Errorf("redis rate limits required for window %q (set AFI_REDIS_URL)", window)
		}
		return c.Timed, nil
	default:
		return nil, fmt.Errorf("unsupported quota window %q", window)
	}
}

func (c CompositeCounters) Get(ctx context.Context, scopeType, scopeID, metric, window string) (int64, error) {
	s, err := c.storeFor(window)
	if err != nil {
		return 0, err
	}
	return s.Get(ctx, scopeType, scopeID, metric, window)
}

func (c CompositeCounters) Incr(ctx context.Context, scopeType, scopeID, metric, window string, delta int64) (int64, error) {
	s, err := c.storeFor(window)
	if err != nil {
		return 0, err
	}
	return s.Incr(ctx, scopeType, scopeID, metric, window, delta)
}
