package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/curefatih/afi/internal/snapshot"
	goredis "github.com/redis/go-redis/v9"
)

// Counters implements dataplane.CounterStore with fixed-window buckets in Redis.
// Used for minute / hour / day rate limits. Lifetime (total) quotas stay on Postgres.
type Counters struct {
	Client *goredis.Client
	Now    func() time.Time // optional; defaults to time.Now
}

func (c *Counters) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func windowBucket(window string, t time.Time) (bucket string, ttl time.Duration, err error) {
	switch window {
	case snapshot.WindowMinute:
		sec := t.Unix()
		return fmt.Sprintf("%d", sec/60), 2 * time.Minute, nil
	case snapshot.WindowHour:
		sec := t.Unix()
		return fmt.Sprintf("%d", sec/3600), 2 * time.Hour, nil
	case snapshot.WindowDay:
		day := t.UTC().Format("2006-01-02")
		return day, 48 * time.Hour, nil
	default:
		return "", 0, fmt.Errorf("redis counters do not support window %q", window)
	}
}

func quotaKey(scopeType, scopeID, metric, window, bucket string) string {
	return fmt.Sprintf("afi:quota:%s:%s:%s:%s:%s", scopeType, scopeID, metric, window, bucket)
}

func (c *Counters) Get(ctx context.Context, scopeType, scopeID, metric, window string) (int64, error) {
	if c == nil || c.Client == nil {
		return 0, fmt.Errorf("redis counters not configured")
	}
	bucket, _, err := windowBucket(window, c.now())
	if err != nil {
		return 0, err
	}
	n, err := c.Client.Get(ctx, quotaKey(scopeType, scopeID, metric, window, bucket)).Int64()
	if err == goredis.Nil {
		return 0, nil
	}
	return n, err
}

func (c *Counters) Incr(ctx context.Context, scopeType, scopeID, metric, window string, delta int64) (int64, error) {
	if c == nil || c.Client == nil {
		return 0, fmt.Errorf("redis counters not configured")
	}
	bucket, ttl, err := windowBucket(window, c.now())
	if err != nil {
		return 0, err
	}
	key := quotaKey(scopeType, scopeID, metric, window, bucket)
	pipe := c.Client.TxPipeline()
	incr := pipe.IncrBy(ctx, key, delta)
	pipe.Expire(ctx, key, ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	return incr.Val(), nil
}
