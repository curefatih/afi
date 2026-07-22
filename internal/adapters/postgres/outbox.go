package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/curefatih/afi/internal/usage"
	"github.com/curefatih/afi/internal/workers"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UsageOutbox writes and drains the usage_outbox table.
type UsageOutbox struct {
	Pool *pgxpool.Pool
}

// Enqueue implements a write-side port for the gateway request path.
func (o *UsageOutbox) Enqueue(ctx context.Context, payload []byte) error {
	_, err := o.Pool.Exec(ctx, `INSERT INTO usage_outbox (payload) VALUES ($1)`, payload)
	return err
}

func (o *UsageOutbox) ClaimBatch(ctx context.Context, limit int) ([]workers.OutboxRow, error) {
	rows, err := o.Pool.Query(ctx, `
		SELECT id, payload FROM usage_outbox
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

func (o *UsageOutbox) MarkProcessed(ctx context.Context, id int64) error {
	_, err := o.Pool.Exec(ctx, `UPDATE usage_outbox SET processed_at=NOW() WHERE id=$1`, id)
	return err
}

// UsageSink writes usage_events from drained outbox payloads.
type UsageSink struct {
	Pool *pgxpool.Pool
}

func (s *UsageSink) InsertUsage(ctx context.Context, e usage.Event, costUSD *float64) error {
	e.MarkBYOK()
	modality := usage.NormalizeModality(e.Modality)
	metrics := e.Metrics
	if metrics == nil {
		metrics = map[string]any{}
	}
	metricsJSON, err := json.Marshal(metrics)
	if err != nil {
		return err
	}
	tags := e.Tags
	if tags == nil {
		tags = map[string]string{}
	}
	tagsJSON, err := json.Marshal(tags)
	if err != nil {
		return err
	}
	_, err = s.Pool.Exec(ctx, `
		INSERT INTO usage_events (
			organization_id, project_id, api_key_id, credential_id, used_byok, model, status,
			latency_ms, prompt_tokens, completion_tokens, cost_usd, modality, metrics, tags
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
	`, e.OrganizationID, e.ProjectID, e.APIKeyID, nullIfEmpty(e.CredentialID), e.UsedBYOK, e.Model, e.Status,
		e.LatencyMs, e.PromptTokens, e.CompletionTokens, costUSD, modality, metricsJSON, tagsJSON)
	return err
}

// PriceLookup reads model_prices for cost computation.
type PriceLookup struct {
	Pool *pgxpool.Pool
}

func (p *PriceLookup) LookupModelPrice(ctx context.Context, providerType, model string) (float64, float64, bool, error) {
	var in, out float64
	err := p.Pool.QueryRow(ctx, `
		SELECT input_per_mtok, output_per_mtok
		FROM model_prices WHERE provider_type=$1 AND model=$2
	`, providerType, model).Scan(&in, &out)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, 0, false, nil
	}
	if err != nil {
		return 0, 0, false, err
	}
	return in, out, true, nil
}
