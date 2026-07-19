package postgres

import (
	"context"

	"github.com/curefatih/afi/internal/workers"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PlatformEventOutbox writes and drains the platform_event_outbox table.
type PlatformEventOutbox struct {
	Pool *pgxpool.Pool
}

func NewPlatformEventOutbox(pool *pgxpool.Pool) *PlatformEventOutbox {
	return &PlatformEventOutbox{Pool: pool}
}

func (o *PlatformEventOutbox) Enqueue(ctx context.Context, payload []byte) error {
	_, err := o.Pool.Exec(ctx, `INSERT INTO platform_event_outbox (payload) VALUES ($1)`, payload)
	return err
}

func (o *PlatformEventOutbox) ClaimBatch(ctx context.Context, limit int) ([]workers.OutboxRow, error) {
	rows, err := o.Pool.Query(ctx, `
		SELECT id, payload FROM platform_event_outbox
		WHERE processed_at IS NULL
		ORDER BY id ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []workers.OutboxRow
	for rows.Next() {
		var r workers.OutboxRow
		if err := rows.Scan(&r.ID, &r.Payload); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (o *PlatformEventOutbox) MarkProcessed(ctx context.Context, id int64) error {
	_, err := o.Pool.Exec(ctx, `UPDATE platform_event_outbox SET processed_at=NOW() WHERE id=$1`, id)
	return err
}
