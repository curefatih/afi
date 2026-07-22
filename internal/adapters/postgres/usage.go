package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/usage"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UsageQueries reads and writes usage_events for the control plane.
type UsageQueries struct {
	Pool *pgxpool.Pool
}

func NewUsageQueries(pool *pgxpool.Pool) *UsageQueries {
	return &UsageQueries{Pool: pool}
}

func (q *UsageQueries) InsertRecord(ctx context.Context, e usage.Record) error {
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
	_, err = q.Pool.Exec(ctx, `
		INSERT INTO usage_events (
			organization_id, project_id, api_key_id, credential_id, used_byok, model, status,
			latency_ms, prompt_tokens, completion_tokens, cost_usd, modality, metrics, tags
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
	`, e.OrganizationID, e.ProjectID, e.APIKeyID, nullIfEmpty(e.CredentialID), e.UsedBYOK, e.Model, e.Status,
		e.LatencyMs, e.PromptTokens, e.CompletionTokens, e.CostUSD, modality, metricsJSON, tagsJSON)
	return err
}

func usageWhere(orgID string, f usage.Filter) (string, []any) {
	args := []any{orgID}
	var b strings.Builder
	b.WriteString("e.organization_id=$1")
	n := 2
	if f.ProjectID != "" {
		b.WriteString(fmt.Sprintf(" AND e.project_id=$%d", n))
		args = append(args, f.ProjectID)
		n++
	}
	if f.APIKeyID != "" {
		b.WriteString(fmt.Sprintf(" AND e.api_key_id=$%d", n))
		args = append(args, f.APIKeyID)
		n++
	}
	if f.CredentialID != "" {
		b.WriteString(fmt.Sprintf(" AND e.credential_id=$%d", n))
		args = append(args, f.CredentialID)
		n++
	}
	if f.Model != "" {
		b.WriteString(fmt.Sprintf(" AND e.model=$%d", n))
		args = append(args, f.Model)
		n++
	}
	if f.Modality != "" {
		b.WriteString(fmt.Sprintf(" AND e.modality=$%d", n))
		args = append(args, f.Modality)
		n++
	}
	if f.ExcludeBYOK {
		b.WriteString(" AND e.used_byok=FALSE")
	}
	if f.BYOKOnly {
		b.WriteString(" AND e.used_byok=TRUE")
	}
	if f.From != nil {
		b.WriteString(fmt.Sprintf(" AND e.created_at >= $%d", n))
		args = append(args, *f.From)
		n++
	}
	if f.To != nil {
		b.WriteString(fmt.Sprintf(" AND e.created_at < $%d", n))
		args = append(args, *f.To)
	}
	return b.String(), args
}

func (q *UsageQueries) List(ctx context.Context, orgID string, f usage.Filter) ([]usage.Record, error) {
	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	where, args := usageWhere(orgID, f)
	args = append(args, limit)
	limitArg := len(args)
	rows, err := q.Pool.Query(ctx, fmt.Sprintf(`
		SELECT e.id, e.organization_id, e.project_id, e.api_key_id,
			COALESCE(e.credential_id, ''), e.used_byok,
			e.model, e.status,
			e.latency_ms, e.prompt_tokens, e.completion_tokens, e.cost_usd, e.created_at,
			e.modality, e.metrics, e.tags,
			COALESCE(k.name, ''), COALESCE(k.kind, ''),
			COALESCE(k.owner_user_id, ''), COALESCE(u.email, ''), COALESCE(u.name, ''),
			COALESCE(proj.name, ''), COALESCE(c.name, '')
		FROM usage_events e
		LEFT JOIN api_keys k ON k.id = e.api_key_id
		LEFT JOIN users u ON u.id = k.owner_user_id
		LEFT JOIN projects proj ON proj.id = NULLIF(e.project_id, '')
		LEFT JOIN provider_credentials c ON c.id = e.credential_id
		WHERE %s
		ORDER BY e.created_at DESC
		LIMIT $%d
	`, where, limitArg), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []usage.Record
	for rows.Next() {
		var e usage.Record
		var metricsJSON, tagsJSON []byte
		if err := rows.Scan(
			&e.ID, &e.OrganizationID, &e.ProjectID, &e.APIKeyID,
			&e.CredentialID, &e.UsedBYOK,
			&e.Model, &e.Status,
			&e.LatencyMs, &e.PromptTokens, &e.CompletionTokens, &e.CostUSD, &e.CreatedAt,
			&e.Modality, &metricsJSON, &tagsJSON,
			&e.KeyName, &e.KeyKind, &e.OwnerUserID, &e.OwnerEmail, &e.OwnerName,
			&e.ProjectName, &e.CredentialName,
		); err != nil {
			return nil, err
		}
		if len(metricsJSON) > 0 {
			_ = json.Unmarshal(metricsJSON, &e.Metrics)
		}
		if e.Metrics == nil {
			e.Metrics = map[string]any{}
		}
		if len(tagsJSON) > 0 {
			_ = json.Unmarshal(tagsJSON, &e.Tags)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func metricSumExpr(key string) string {
	return fmt.Sprintf(`COALESCE(SUM(CASE WHEN jsonb_typeof(e.metrics->'%s') = 'number'
		THEN (e.metrics->>'%s')::double precision ELSE 0 END), 0)`, key, key)
}

func (q *UsageQueries) Summarize(ctx context.Context, orgID string, f usage.Filter) ([]usage.SummaryBucket, error) {
	groupBy := f.GroupBy
	if groupBy == "" {
		groupBy = "day"
	}
	ff := f
	if ff.From == nil && ff.To == nil && groupBy == "day" {
		from := time.Now().UTC().AddDate(0, 0, -30)
		ff.From = &from
	}
	where, args := usageWhere(orgID, ff)

	var selectBucket, groupSQL, orderSQL, joinSQL string
	switch groupBy {
	case "day":
		selectBucket = `to_char(date_trunc('day', e.created_at AT TIME ZONE 'UTC'), 'YYYY-MM-DD')`
		groupSQL = `1`
		orderSQL = `1 ASC`
		joinSQL = ""
	case "model":
		selectBucket = `e.model`
		groupSQL = `1`
		orderSQL = `requests DESC`
		joinSQL = ""
	case "modality":
		selectBucket = `e.modality`
		groupSQL = `1`
		orderSQL = `requests DESC`
		joinSQL = ""
	case "byok":
		selectBucket = `CASE WHEN e.used_byok THEN 'byok' ELSE 'platform' END`
		groupSQL = `1`
		orderSQL = `requests DESC`
		joinSQL = ""
	case "key":
		selectBucket = `e.api_key_id`
		groupSQL = `1, COALESCE(k.name,''), COALESCE(k.kind,''), COALESCE(u.email,''), COALESCE(u.name,'')`
		orderSQL = `requests DESC`
		joinSQL = `
			LEFT JOIN api_keys k ON k.id = e.api_key_id
			LEFT JOIN users u ON u.id = k.owner_user_id`
	default:
		return nil, fmt.Errorf("%w: group_by must be day, model, key, modality, or byok", kernel.ErrInvalidRequest)
	}

	extraCols := ``
	if groupBy == "key" {
		extraCols = `, COALESCE(k.name,'') AS key_name, COALESCE(k.kind,'') AS key_kind,
			COALESCE(u.email,'') AS owner_email, COALESCE(u.name,'') AS owner_name`
	}

	query := fmt.Sprintf(`
		SELECT %s AS bucket,
			COUNT(*)::bigint AS requests,
			COALESCE(SUM(e.cost_usd), 0)::double precision AS cost_usd,
			COALESCE(SUM(e.prompt_tokens), 0)::bigint AS prompt_tokens,
			COALESCE(SUM(e.completion_tokens), 0)::bigint AS completion_tokens,
			%s AS characters,
			%s AS audio_seconds,
			%s AS images,
			%s AS tokens
			%s
		FROM usage_events e
		%s
		WHERE %s
		GROUP BY %s
		ORDER BY %s
	`, selectBucket,
		metricSumExpr("characters"),
		metricSumExpr("audio_seconds"),
		metricSumExpr("images"),
		metricSumExpr("tokens"),
		extraCols,
		joinSQL, where, groupSQL, orderSQL)

	rows, err := q.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []usage.SummaryBucket
	for rows.Next() {
		var b usage.SummaryBucket
		var characters, audioSeconds, images, tokens float64
		if groupBy == "key" {
			var keyName string
			if err := rows.Scan(
				&b.Bucket, &b.Requests, &b.CostUSD, &b.PromptTokens, &b.CompletionTokens,
				&characters, &audioSeconds, &images, &tokens,
				&keyName, &b.KeyKind, &b.OwnerEmail, &b.OwnerName,
			); err != nil {
				return nil, err
			}
			b.Label = keyName
			if b.Label == "" {
				b.Label = b.Bucket
			}
		} else {
			if err := rows.Scan(
				&b.Bucket, &b.Requests, &b.CostUSD, &b.PromptTokens, &b.CompletionTokens,
				&characters, &audioSeconds, &images, &tokens,
			); err != nil {
				return nil, err
			}
			b.Label = b.Bucket
			if groupBy == "byok" {
				switch b.Bucket {
				case "byok":
					b.Label = "BYOK"
				case "platform":
					b.Label = "Platform"
				}
			}
		}
		totals := map[string]float64{}
		if characters != 0 {
			totals["characters"] = characters
		}
		if audioSeconds != 0 {
			totals["audio_seconds"] = audioSeconds
		}
		if images != 0 {
			totals["images"] = images
		}
		if tokens != 0 {
			totals["tokens"] = tokens
		}
		if len(totals) > 0 {
			b.MetricsTotals = totals
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (q *UsageQueries) ListProviderHealth(ctx context.Context, orgID string, from, to time.Time) ([]usage.ProviderHealth, error) {
	if from.IsZero() {
		from = time.Now().UTC().Add(-24 * time.Hour)
	}
	if to.IsZero() {
		to = time.Now().UTC().Add(time.Hour)
	}
	rows, err := q.Pool.Query(ctx, `
		SELECT p.id, p.name, p.type,
			COUNT(e.id)::bigint AS requests,
			COUNT(e.id) FILTER (WHERE e.status IS NOT NULL AND e.status <> 'ok')::bigint AS errors,
			COALESCE(AVG(e.latency_ms), 0)::double precision AS avg_latency_ms
		FROM providers p
		LEFT JOIN routes r
			ON r.provider_id = p.id AND r.organization_id = p.organization_id
		LEFT JOIN usage_events e
			ON e.organization_id = p.organization_id
			AND e.model = r.model
			AND e.created_at >= $2 AND e.created_at < $3
		WHERE p.organization_id = $1
		GROUP BY p.id, p.name, p.type
		ORDER BY p.name
	`, orgID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []usage.ProviderHealth
	for rows.Next() {
		var h usage.ProviderHealth
		if err := rows.Scan(&h.ProviderID, &h.Name, &h.Type, &h.Requests, &h.Errors, &h.AvgLatencyMs); err != nil {
			return nil, err
		}
		if h.Requests > 0 {
			h.ErrorRate = float64(h.Errors) / float64(h.Requests)
		}
		h.Status = usage.ClassifyProviderHealth(h.Requests, h.Errors, h.ErrorRate)
		out = append(out, h)
	}
	return out, rows.Err()
}

func (q *UsageQueries) LookupModelPrice(ctx context.Context, providerType, model string) (usage.ModelPrice, bool, error) {
	var p usage.ModelPrice
	err := q.Pool.QueryRow(ctx, `
		SELECT provider_type, model, input_per_mtok, output_per_mtok
		FROM model_prices WHERE provider_type=$1 AND model=$2
	`, providerType, model).Scan(&p.ProviderType, &p.Model, &p.InputPerMTok, &p.OutputPerMTok)
	if errors.Is(err, pgx.ErrNoRows) {
		return usage.ModelPrice{}, false, nil
	}
	if err != nil {
		return usage.ModelPrice{}, false, err
	}
	return p, true, nil
}
