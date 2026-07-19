package dataplane

import (
	"context"

	"github.com/curefatih/afi/internal/snapshot"
)

// CounterStore reads/writes durable quota counters (not config).
type CounterStore interface {
	Get(ctx context.Context, scopeType, scopeID, metric, window string) (int64, error)
	Incr(ctx context.Context, scopeType, scopeID, metric, window string, delta int64) (int64, error)
}

func (p *Pipeline) checkAndIncrRequests(ctx context.Context, snap *snapshot.Snapshot, key snapshot.APIKey) (denied bool, err error) {
	if p.Counters == nil {
		return false, nil
	}
	q, ok := snap.ResolveQuota(key, snapshot.MetricRequests)
	if !ok {
		return false, nil
	}
	used, err := p.Counters.Get(ctx, q.ScopeType, q.ScopeID, q.Metric, q.Window)
	if err != nil {
		return false, err
	}
	if used >= q.LimitValue {
		return true, nil
	}
	_, err = p.Counters.Incr(ctx, q.ScopeType, q.ScopeID, q.Metric, q.Window, 1)
	return false, err
}

func (p *Pipeline) incrTokens(ctx context.Context, snap *snapshot.Snapshot, key snapshot.APIKey, tokens int64) {
	if p.Counters == nil || tokens <= 0 {
		return
	}
	q, ok := snap.ResolveQuota(key, snapshot.MetricTokens)
	if !ok {
		return
	}
	_, _ = p.Counters.Incr(ctx, q.ScopeType, q.ScopeID, q.Metric, q.Window, tokens)
}
