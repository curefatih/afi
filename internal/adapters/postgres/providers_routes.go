package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/curefatih/afi/internal/gatewayconfig"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/snapshot"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Providers implements gatewayconfig.ProviderRepository.
type Providers struct {
	Pool *pgxpool.Pool
}

func NewProviders(pool *pgxpool.Pool) *Providers {
	return &Providers{Pool: pool}
}

func DecodeCapabilities(typ string, raw []byte) snapshot.ProviderCapabilities {
	return decodeCapabilities(typ, raw)
}

func decodeCapabilities(typ string, raw []byte) snapshot.ProviderCapabilities {
	var c snapshot.ProviderCapabilities
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &c)
	}
	return snapshot.NormalizeCapabilities(typ, c)
}

func (p *Providers) ListByOrg(ctx context.Context, orgID string) ([]gatewayconfig.Provider, error) {
	rows, err := p.Pool.Query(ctx, `
		SELECT id, organization_id, name, type, base_url, api_key_env, capabilities, created_at
		FROM providers WHERE organization_id = $1 ORDER BY created_at
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []gatewayconfig.Provider
	for rows.Next() {
		var item gatewayconfig.Provider
		var caps []byte
		if err := rows.Scan(&item.ID, &item.OrganizationID, &item.Name, &item.Type, &item.BaseURL, &item.APIKeyEnv, &caps, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.Capabilities = decodeCapabilities(item.Type, caps)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (p *Providers) Insert(ctx context.Context, item gatewayconfig.Provider) error {
	raw, err := json.Marshal(item.Capabilities)
	if err != nil {
		return err
	}
	_, err = p.Pool.Exec(ctx, `
		INSERT INTO providers (id, organization_id, name, type, base_url, api_key_env, capabilities, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`, item.ID, item.OrganizationID, item.Name, item.Type, item.BaseURL, item.APIKeyEnv, raw, item.CreatedAt)
	return err
}

func (p *Providers) Update(ctx context.Context, providerID, name, baseURL, apiKeyEnv string) (*gatewayconfig.Provider, error) {
	item := &gatewayconfig.Provider{}
	var caps []byte
	err := p.Pool.QueryRow(ctx, `
		UPDATE providers SET name=$2, base_url=$3, api_key_env=$4
		WHERE id=$1
		RETURNING id, organization_id, name, type, base_url, api_key_env, capabilities, created_at
	`, providerID, name, baseURL, apiKeyEnv).Scan(
		&item.ID, &item.OrganizationID, &item.Name, &item.Type, &item.BaseURL, &item.APIKeyEnv, &caps, &item.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	item.Capabilities = decodeCapabilities(item.Type, caps)
	return item, nil
}

func (p *Providers) Delete(ctx context.Context, providerID string) error {
	tag, err := p.Pool.Exec(ctx, `DELETE FROM providers WHERE id=$1`, providerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return kernel.ErrNotFound
	}
	return nil
}

func (p *Providers) OrgID(ctx context.Context, providerID string) (string, error) {
	var orgID string
	err := p.Pool.QueryRow(ctx, `SELECT organization_id FROM providers WHERE id=$1`, providerID).Scan(&orgID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", kernel.ErrNotFound
	}
	return orgID, err
}

// Routes implements gatewayconfig.RouteRepository.
type Routes struct {
	Pool *pgxpool.Pool
}

func NewRoutes(pool *pgxpool.Pool) *Routes {
	return &Routes{Pool: pool}
}

func DecodeFallbacks(raw []byte) []gatewayconfig.RouteFallback {
	return decodeFallbacks(raw)
}

func decodeFallbacks(raw []byte) []gatewayconfig.RouteFallback {
	if len(raw) == 0 {
		return []gatewayconfig.RouteFallback{}
	}
	var out []gatewayconfig.RouteFallback
	if err := json.Unmarshal(raw, &out); err != nil || out == nil {
		return []gatewayconfig.RouteFallback{}
	}
	return out
}

func DecodeRetry(raw []byte) *gatewayconfig.RetryConfig {
	return decodeRetry(raw)
}

func decodeRetry(raw []byte) *gatewayconfig.RetryConfig {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var out gatewayconfig.RetryConfig
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return &out
}

func encodeRetry(c *gatewayconfig.RetryConfig) ([]byte, error) {
	if c == nil {
		return nil, nil
	}
	return json.Marshal(c)
}

func (r *Routes) ListByOrg(ctx context.Context, orgID string) ([]gatewayconfig.Route, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT id, organization_id, model, provider_id, target_model, fallbacks, retry, routing_strategy, weight, created_at
		FROM routes WHERE organization_id=$1 ORDER BY created_at
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []gatewayconfig.Route
	for rows.Next() {
		var item gatewayconfig.Route
		var fb, retryRaw []byte
		if err := rows.Scan(
			&item.ID, &item.OrganizationID, &item.Model, &item.ProviderID, &item.TargetModel,
			&fb, &retryRaw, &item.RoutingStrategy, &item.Weight, &item.CreatedAt,
		); err != nil {
			return nil, err
		}
		item.Fallbacks = decodeFallbacks(fb)
		item.Retry = decodeRetry(retryRaw)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *Routes) Insert(ctx context.Context, item gatewayconfig.Route) error {
	fb, err := json.Marshal(item.Fallbacks)
	if err != nil {
		return err
	}
	retryRaw, err := encodeRetry(item.Retry)
	if err != nil {
		return err
	}
	_, err = r.Pool.Exec(ctx, `
		INSERT INTO routes (id, organization_id, model, provider_id, target_model, fallbacks, retry, routing_strategy, weight, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
	`, item.ID, item.OrganizationID, item.Model, item.ProviderID, item.TargetModel, fb, retryRaw, item.RoutingStrategy, item.Weight, item.CreatedAt)
	return err
}

func (r *Routes) Update(ctx context.Context, routeID, model, providerID, targetModel string, fallbacks []gatewayconfig.RouteFallback, retry *gatewayconfig.RetryConfig, strategy string, weight int) (*gatewayconfig.Route, error) {
	if fallbacks == nil {
		fallbacks = []gatewayconfig.RouteFallback{}
	}
	fb, err := json.Marshal(fallbacks)
	if err != nil {
		return nil, err
	}
	retryRaw, err := encodeRetry(retry)
	if err != nil {
		return nil, err
	}
	item := &gatewayconfig.Route{}
	var raw, gotRetry []byte
	err = r.Pool.QueryRow(ctx, `
		UPDATE routes SET model=$2, provider_id=$3, target_model=$4, fallbacks=$5, retry=$6, routing_strategy=$7, weight=$8
		WHERE id=$1
		RETURNING id, organization_id, model, provider_id, target_model, fallbacks, retry, routing_strategy, weight, created_at
	`, routeID, model, providerID, targetModel, fb, retryRaw, strategy, weight).Scan(
		&item.ID, &item.OrganizationID, &item.Model, &item.ProviderID, &item.TargetModel,
		&raw, &gotRetry, &item.RoutingStrategy, &item.Weight, &item.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	item.Fallbacks = decodeFallbacks(raw)
	item.Retry = decodeRetry(gotRetry)
	return item, nil
}

func (r *Routes) Delete(ctx context.Context, routeID string) error {
	tag, err := r.Pool.Exec(ctx, `DELETE FROM routes WHERE id=$1`, routeID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return kernel.ErrNotFound
	}
	return nil
}

func (r *Routes) OrgID(ctx context.Context, routeID string) (string, error) {
	var orgID string
	err := r.Pool.QueryRow(ctx, `SELECT organization_id FROM routes WHERE id=$1`, routeID).Scan(&orgID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", kernel.ErrNotFound
	}
	return orgID, err
}
