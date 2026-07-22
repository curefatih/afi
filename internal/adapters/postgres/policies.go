package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/curefatih/afi/internal/gatewayconfig"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Policies implements gatewayconfig.PolicyRepository.
type Policies struct {
	Pool *pgxpool.Pool
}

func NewPolicies(pool *pgxpool.Pool) *Policies {
	return &Policies{Pool: pool}
}

func scanPolicy(row pgx.Row) (*gatewayconfig.RequestPolicy, error) {
	item := &gatewayconfig.RequestPolicy{}
	var cfg []byte
	err := row.Scan(
		&item.ID, &item.OrganizationID, &item.Name, &item.Expression,
		&item.Action, &cfg, &item.Enabled, &item.Priority, &item.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(cfg) == 0 {
		item.ActionConfig = json.RawMessage(`{}`)
	} else {
		item.ActionConfig = cfg
	}
	return item, nil
}

func (p *Policies) ListByOrg(ctx context.Context, orgID string) ([]gatewayconfig.RequestPolicy, error) {
	rows, err := p.Pool.Query(ctx, `
		SELECT id, organization_id, name, expression, action, action_config, enabled, priority, created_at
		FROM request_policies WHERE organization_id=$1
		ORDER BY priority DESC, name ASC
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []gatewayconfig.RequestPolicy
	for rows.Next() {
		item, err := scanPolicy(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func (p *Policies) Insert(ctx context.Context, item gatewayconfig.RequestPolicy) error {
	cfg := item.ActionConfig
	if len(cfg) == 0 {
		cfg = json.RawMessage(`{}`)
	}
	_, err := p.Pool.Exec(ctx, `
		INSERT INTO request_policies (id, organization_id, name, expression, action, action_config, enabled, priority, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`, item.ID, item.OrganizationID, item.Name, item.Expression, item.Action, cfg, item.Enabled, item.Priority, item.CreatedAt)
	return err
}

func (p *Policies) Get(ctx context.Context, policyID string) (*gatewayconfig.RequestPolicy, error) {
	item, err := scanPolicy(p.Pool.QueryRow(ctx, `
		SELECT id, organization_id, name, expression, action, action_config, enabled, priority, created_at
		FROM request_policies WHERE id=$1
	`, policyID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	return item, err
}

func (p *Policies) Update(ctx context.Context, item gatewayconfig.RequestPolicy) (*gatewayconfig.RequestPolicy, error) {
	cfg := item.ActionConfig
	if len(cfg) == 0 {
		cfg = json.RawMessage(`{}`)
	}
	out, err := scanPolicy(p.Pool.QueryRow(ctx, `
		UPDATE request_policies SET name=$2, expression=$3, action=$4, action_config=$5, enabled=$6, priority=$7 WHERE id=$1
		RETURNING id, organization_id, name, expression, action, action_config, enabled, priority, created_at
	`, item.ID, item.Name, item.Expression, item.Action, cfg, item.Enabled, item.Priority))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	return out, err
}

func (p *Policies) UpdatePriorities(ctx context.Context, orgID string, items []gatewayconfig.PolicyPriorityUpdate) error {
	if len(items) == 0 {
		return nil
	}
	tx, err := p.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, item := range items {
		tag, err := tx.Exec(ctx, `
			UPDATE request_policies SET priority=$2
			WHERE id=$1 AND organization_id=$3
		`, item.ID, item.Priority, orgID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return kernel.ErrNotFound
		}
	}
	return tx.Commit(ctx)
}

func (p *Policies) Delete(ctx context.Context, policyID string) error {
	tag, err := p.Pool.Exec(ctx, `DELETE FROM request_policies WHERE id=$1`, policyID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return kernel.ErrNotFound
	}
	return nil
}

func (p *Policies) OrgID(ctx context.Context, policyID string) (string, error) {
	var orgID string
	err := p.Pool.QueryRow(ctx, `SELECT organization_id FROM request_policies WHERE id=$1`, policyID).Scan(&orgID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", kernel.ErrNotFound
	}
	return orgID, err
}
