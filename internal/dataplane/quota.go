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
	type hit struct {
		q snapshot.Quota
	}
	var hits []hit
	for _, window := range snapshot.RequestWindows {
		q, ok := snap.ResolveQuota(key, snapshot.MetricRequests, window)
		if !ok {
			continue
		}
		used, err := p.Counters.Get(ctx, q.ScopeType, q.ScopeID, q.Metric, q.Window)
		if err != nil {
			return false, err
		}
		if used >= q.LimitValue {
			return true, nil
		}
		hits = append(hits, hit{q: q})
	}
	for _, h := range hits {
		if _, err := p.Counters.Incr(ctx, h.q.ScopeType, h.q.ScopeID, h.q.Metric, h.q.Window, 1); err != nil {
			return false, err
		}
	}
	return false, nil
}

func (p *Pipeline) incrTokens(ctx context.Context, snap *snapshot.Snapshot, key snapshot.APIKey, tokens int64) {
	if p.Counters == nil || tokens <= 0 {
		return
	}
	for _, window := range snapshot.RequestWindows {
		q, ok := snap.ResolveQuota(key, snapshot.MetricTokens, window)
		if !ok {
			continue
		}
		_, _ = p.Counters.Incr(ctx, q.ScopeType, q.ScopeID, q.Metric, q.Window, tokens)
	}
}
