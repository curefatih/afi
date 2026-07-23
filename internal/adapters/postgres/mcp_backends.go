package postgres

import (
	"context"
	"errors"

	"github.com/curefatih/afi/internal/gatewayconfig"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MCPBackends implements gatewayconfig.MCPBackendRepository.
type MCPBackends struct {
	Pool *pgxpool.Pool
}

func NewMCPBackends(pool *pgxpool.Pool) *MCPBackends {
	return &MCPBackends{Pool: pool}
}

func (p *MCPBackends) ListByOrg(ctx context.Context, orgID string) ([]gatewayconfig.MCPBackend, error) {
	rows, err := p.Pool.Query(ctx, `
		SELECT id, organization_id, alias, name, base_url, api_key_env, method_allowlist, enabled, created_at
		FROM mcp_backends WHERE organization_id=$1
		ORDER BY alias ASC
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []gatewayconfig.MCPBackend
	for rows.Next() {
		var item gatewayconfig.MCPBackend
		if err := rows.Scan(
			&item.ID, &item.OrganizationID, &item.Alias, &item.Name, &item.BaseURL, &item.APIKeyEnv,
			&item.MethodAllowlist, &item.Enabled, &item.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (p *MCPBackends) Insert(ctx context.Context, item gatewayconfig.MCPBackend) error {
	_, err := p.Pool.Exec(ctx, `
		INSERT INTO mcp_backends (id, organization_id, alias, name, base_url, api_key_env, method_allowlist, enabled, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`, item.ID, item.OrganizationID, item.Alias, item.Name, item.BaseURL, item.APIKeyEnv,
		item.MethodAllowlist, item.Enabled, item.CreatedAt)
	return err
}

func (p *MCPBackends) Get(ctx context.Context, id string) (*gatewayconfig.MCPBackend, error) {
	item := &gatewayconfig.MCPBackend{}
	err := p.Pool.QueryRow(ctx, `
		SELECT id, organization_id, alias, name, base_url, api_key_env, method_allowlist, enabled, created_at
		FROM mcp_backends WHERE id=$1
	`, id).Scan(
		&item.ID, &item.OrganizationID, &item.Alias, &item.Name, &item.BaseURL, &item.APIKeyEnv,
		&item.MethodAllowlist, &item.Enabled, &item.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	return item, err
}

func (p *MCPBackends) Update(ctx context.Context, item gatewayconfig.MCPBackend) (*gatewayconfig.MCPBackend, error) {
	out := &gatewayconfig.MCPBackend{}
	err := p.Pool.QueryRow(ctx, `
		UPDATE mcp_backends SET alias=$2, name=$3, base_url=$4, api_key_env=$5, method_allowlist=$6, enabled=$7
		WHERE id=$1
		RETURNING id, organization_id, alias, name, base_url, api_key_env, method_allowlist, enabled, created_at
	`, item.ID, item.Alias, item.Name, item.BaseURL, item.APIKeyEnv, item.MethodAllowlist, item.Enabled).Scan(
		&out.ID, &out.OrganizationID, &out.Alias, &out.Name, &out.BaseURL, &out.APIKeyEnv,
		&out.MethodAllowlist, &out.Enabled, &out.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	return out, err
}

func (p *MCPBackends) Delete(ctx context.Context, id string) error {
	tag, err := p.Pool.Exec(ctx, `DELETE FROM mcp_backends WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return kernel.ErrNotFound
	}
	return nil
}

func (p *MCPBackends) OrgID(ctx context.Context, id string) (string, error) {
	var orgID string
	err := p.Pool.QueryRow(ctx, `SELECT organization_id FROM mcp_backends WHERE id=$1`, id).Scan(&orgID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", kernel.ErrNotFound
	}
	return orgID, err
}
