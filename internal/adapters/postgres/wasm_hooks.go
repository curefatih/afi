package postgres

import (
	"context"
	"errors"

	"github.com/curefatih/afi/internal/gatewayconfig"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WasmHooks implements gatewayconfig.WasmHookRepository.
type WasmHooks struct {
	Pool *pgxpool.Pool
}

func NewWasmHooks(pool *pgxpool.Pool) *WasmHooks {
	return &WasmHooks{Pool: pool}
}

func (p *WasmHooks) ListByOrg(ctx context.Context, orgID string) ([]gatewayconfig.WasmHook, error) {
	rows, err := p.Pool.Query(ctx, `
		SELECT id, organization_id, name, phase, module_uri, digest, enabled, priority, config, created_at
		FROM wasm_hooks WHERE organization_id=$1
		ORDER BY priority DESC, name ASC
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []gatewayconfig.WasmHook
	for rows.Next() {
		var item gatewayconfig.WasmHook
		if err := rows.Scan(
			&item.ID, &item.OrganizationID, &item.Name, &item.Phase, &item.ModuleURI, &item.Digest,
			&item.Enabled, &item.Priority, &item.Config, &item.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (p *WasmHooks) Insert(ctx context.Context, item gatewayconfig.WasmHook) error {
	_, err := p.Pool.Exec(ctx, `
		INSERT INTO wasm_hooks (id, organization_id, name, phase, module_uri, digest, enabled, priority, config, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
	`, item.ID, item.OrganizationID, item.Name, item.Phase, item.ModuleURI, item.Digest,
		item.Enabled, item.Priority, item.Config, item.CreatedAt)
	return err
}

func (p *WasmHooks) Get(ctx context.Context, id string) (*gatewayconfig.WasmHook, error) {
	item := &gatewayconfig.WasmHook{}
	err := p.Pool.QueryRow(ctx, `
		SELECT id, organization_id, name, phase, module_uri, digest, enabled, priority, config, created_at
		FROM wasm_hooks WHERE id=$1
	`, id).Scan(
		&item.ID, &item.OrganizationID, &item.Name, &item.Phase, &item.ModuleURI, &item.Digest,
		&item.Enabled, &item.Priority, &item.Config, &item.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	return item, err
}

func (p *WasmHooks) Update(ctx context.Context, item gatewayconfig.WasmHook) (*gatewayconfig.WasmHook, error) {
	out := &gatewayconfig.WasmHook{}
	err := p.Pool.QueryRow(ctx, `
		UPDATE wasm_hooks SET name=$2, phase=$3, module_uri=$4, digest=$5, enabled=$6, priority=$7, config=$8
		WHERE id=$1
		RETURNING id, organization_id, name, phase, module_uri, digest, enabled, priority, config, created_at
	`, item.ID, item.Name, item.Phase, item.ModuleURI, item.Digest, item.Enabled, item.Priority, item.Config).Scan(
		&out.ID, &out.OrganizationID, &out.Name, &out.Phase, &out.ModuleURI, &out.Digest,
		&out.Enabled, &out.Priority, &out.Config, &out.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	return out, err
}

func (p *WasmHooks) Delete(ctx context.Context, id string) error {
	tag, err := p.Pool.Exec(ctx, `DELETE FROM wasm_hooks WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return kernel.ErrNotFound
	}
	return nil
}

func (p *WasmHooks) OrgID(ctx context.Context, id string) (string, error) {
	var orgID string
	err := p.Pool.QueryRow(ctx, `SELECT organization_id FROM wasm_hooks WHERE id=$1`, id).Scan(&orgID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", kernel.ErrNotFound
	}
	return orgID, err
}
