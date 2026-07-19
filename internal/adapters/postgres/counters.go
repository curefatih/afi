package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Counters implements dataplane.CounterStore for lifetime (total) quota windows.
type Counters struct {
	Pool *pgxpool.Pool
}

func (c *Counters) Get(ctx context.Context, scopeType, scopeID, metric, window string) (int64, error) {
	var used int64
	err := c.Pool.QueryRow(ctx, `
		SELECT used FROM quota_counters
		WHERE scope_type=$1 AND scope_id=$2 AND metric=$3 AND time_window=$4
	`, scopeType, scopeID, metric, window).Scan(&used)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return used, err
}

func (c *Counters) Incr(ctx context.Context, scopeType, scopeID, metric, window string, delta int64) (int64, error) {
	var used int64
	err := c.Pool.QueryRow(ctx, `
		INSERT INTO quota_counters (scope_type, scope_id, metric, time_window, used, updated_at)
		VALUES ($1,$2,$3,$4,$5,NOW())
		ON CONFLICT (scope_type, scope_id, metric, time_window)
		DO UPDATE SET used = quota_counters.used + EXCLUDED.used, updated_at = NOW()
		RETURNING used
	`, scopeType, scopeID, metric, window, delta).Scan(&used)
	return used, err
}
