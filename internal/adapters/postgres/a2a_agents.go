package postgres

import (
	"context"
	"errors"

	"github.com/curefatih/afi/internal/gatewayconfig"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// A2AAgents implements gatewayconfig.A2AAgentRepository.
type A2AAgents struct {
	Pool *pgxpool.Pool
}

func NewA2AAgents(pool *pgxpool.Pool) *A2AAgents {
	return &A2AAgents{Pool: pool}
}

func (p *A2AAgents) ListByOrg(ctx context.Context, orgID string) ([]gatewayconfig.A2AAgent, error) {
	rows, err := p.Pool.Query(ctx, `
		SELECT id, organization_id, alias, name, upstream_url, card_url, card_cache, api_key_env, auth_scheme, enabled, created_at
		FROM a2a_agents WHERE organization_id=$1
		ORDER BY alias ASC
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []gatewayconfig.A2AAgent
	for rows.Next() {
		var item gatewayconfig.A2AAgent
		var cardCache []byte
		if err := rows.Scan(
			&item.ID, &item.OrganizationID, &item.Alias, &item.Name, &item.UpstreamURL, &item.CardURL,
			&cardCache, &item.APIKeyEnv, &item.AuthScheme, &item.Enabled, &item.CreatedAt,
		); err != nil {
			return nil, err
		}
		if len(cardCache) > 0 {
			item.CardCache = cardCache
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (p *A2AAgents) Insert(ctx context.Context, item gatewayconfig.A2AAgent) error {
	_, err := p.Pool.Exec(ctx, `
		INSERT INTO a2a_agents (id, organization_id, alias, name, upstream_url, card_url, card_cache, api_key_env, auth_scheme, enabled, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	`, item.ID, item.OrganizationID, item.Alias, item.Name, item.UpstreamURL, item.CardURL,
		nullableJSON(item.CardCache), item.APIKeyEnv, item.AuthScheme, item.Enabled, item.CreatedAt)
	return err
}

func (p *A2AAgents) Get(ctx context.Context, id string) (*gatewayconfig.A2AAgent, error) {
	item := &gatewayconfig.A2AAgent{}
	var cardCache []byte
	err := p.Pool.QueryRow(ctx, `
		SELECT id, organization_id, alias, name, upstream_url, card_url, card_cache, api_key_env, auth_scheme, enabled, created_at
		FROM a2a_agents WHERE id=$1
	`, id).Scan(
		&item.ID, &item.OrganizationID, &item.Alias, &item.Name, &item.UpstreamURL, &item.CardURL,
		&cardCache, &item.APIKeyEnv, &item.AuthScheme, &item.Enabled, &item.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if len(cardCache) > 0 {
		item.CardCache = cardCache
	}
	return item, nil
}

func (p *A2AAgents) Update(ctx context.Context, item gatewayconfig.A2AAgent) (*gatewayconfig.A2AAgent, error) {
	out := &gatewayconfig.A2AAgent{}
	var cardCache []byte
	err := p.Pool.QueryRow(ctx, `
		UPDATE a2a_agents SET alias=$2, name=$3, upstream_url=$4, card_url=$5, card_cache=$6, api_key_env=$7, auth_scheme=$8, enabled=$9
		WHERE id=$1
		RETURNING id, organization_id, alias, name, upstream_url, card_url, card_cache, api_key_env, auth_scheme, enabled, created_at
	`, item.ID, item.Alias, item.Name, item.UpstreamURL, item.CardURL, nullableJSON(item.CardCache),
		item.APIKeyEnv, item.AuthScheme, item.Enabled).Scan(
		&out.ID, &out.OrganizationID, &out.Alias, &out.Name, &out.UpstreamURL, &out.CardURL,
		&cardCache, &out.APIKeyEnv, &out.AuthScheme, &out.Enabled, &out.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if len(cardCache) > 0 {
		out.CardCache = cardCache
	}
	return out, nil
}

func (p *A2AAgents) Delete(ctx context.Context, id string) error {
	tag, err := p.Pool.Exec(ctx, `DELETE FROM a2a_agents WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return kernel.ErrNotFound
	}
	return nil
}

func (p *A2AAgents) OrgID(ctx context.Context, id string) (string, error) {
	var orgID string
	err := p.Pool.QueryRow(ctx, `SELECT organization_id FROM a2a_agents WHERE id=$1`, id).Scan(&orgID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", kernel.ErrNotFound
	}
	return orgID, err
}

func nullableJSON(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return b
}
