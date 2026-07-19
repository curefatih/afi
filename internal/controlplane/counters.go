package controlplane

import "context"

// CounterAdapter implements dataplane.CounterStore using Store.
type CounterAdapter struct {
	Store *Store
}

func (c CounterAdapter) Get(ctx context.Context, scopeType, scopeID, metric, window string) (int64, error) {
	return c.Store.GetCounter(ctx, scopeType, scopeID, metric, window)
}

func (c CounterAdapter) Incr(ctx context.Context, scopeType, scopeID, metric, window string, delta int64) (int64, error) {
	return c.Store.IncrCounter(ctx, scopeType, scopeID, metric, window, delta)
}
