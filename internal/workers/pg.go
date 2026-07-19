package workers

import (
	"context"
	"encoding/json"

	"github.com/curefatih/afi/internal/controlplane"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PGOutbox struct {
	Pool *pgxpool.Pool
}

func (p *PGOutbox) ClaimBatch(ctx context.Context, limit int) ([]OutboxRow, error) {
	rows, err := p.Pool.Query(ctx, `
		SELECT id, payload FROM usage_outbox
		WHERE processed_at IS NULL
		ORDER BY id ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OutboxRow
	for rows.Next() {
		var r OutboxRow
		if err := rows.Scan(&r.ID, &r.Payload); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (p *PGOutbox) MarkProcessed(ctx context.Context, id int64) error {
	_, err := p.Pool.Exec(ctx, `UPDATE usage_outbox SET processed_at=NOW() WHERE id=$1`, id)
	return err
}

type PGUsageSink struct {
	Store *controlplane.Store
}

func (p *PGUsageSink) InsertUsage(ctx context.Context, e UsagePayload, costUSD *float64) error {
	return p.Store.InsertUsage(ctx, controlplane.UsageEvent{
		OrganizationID:   e.OrganizationID,
		ProjectID:        e.ProjectID,
		APIKeyID:         e.APIKeyID,
		Model:            e.Model,
		Status:           e.Status,
		LatencyMs:        e.LatencyMs,
		PromptTokens:     e.PromptTokens,
		CompletionTokens: e.CompletionTokens,
		Modality:         e.Modality,
		Metrics:          e.Metrics,
		CostUSD:          costUSD,
	})
}

type PGPriceLookup struct {
	Store *controlplane.Store
}

func (p *PGPriceLookup) LookupModelPrice(ctx context.Context, providerType, model string) (float64, float64, bool, error) {
	price, ok, err := p.Store.LookupModelPrice(ctx, providerType, model)
	if err != nil || !ok {
		return 0, 0, ok, err
	}
	return price.InputPerMTok, price.OutputPerMTok, true, nil
}

func EncodeUsage(e UsagePayload) ([]byte, error) {
	return json.Marshal(e)
}
