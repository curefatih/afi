package snapshot

import (
	"context"
	"time"
)

// Store is the persistence port for compiled gateway snapshots.
// Adapters live under internal/adapters (e.g. postgres).
type Store interface {
	Put(ctx context.Context, snap *Snapshot) (int64, error)
	Latest(ctx context.Context) (*Snapshot, error)
	Watch(ctx context.Context, pollInterval time.Duration, onUpdate func(*Snapshot)) error
}
