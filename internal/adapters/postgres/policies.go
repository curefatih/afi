package postgres

import (
	"context"
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

func (p *Policies) ListByOrg(ctx context.Context, orgID string) ([]gatewayconfig.RequestPolicy, error) {
	rows, err := p.Pool.Query(ctx, `
		SELECT id, organization_id, name, expression, enabled, priority, created_at
		FROM request_policies WHERE organization_id=$1
		ORDER BY priority DESC, name ASC
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []gatewayconfig.RequestPolicy
	for rows.Next() {
		var item gatewayconfig.RequestPolicy
		if err := rows.Scan(
			&item.ID, &item.OrganizationID, &item.Name, &item.Expression,
			&item.Enabled, &item.Priority, &item.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (p *Policies) Insert(ctx context.Context, item gatewayconfig.RequestPolicy) error {
	_, err := p.Pool.Exec(ctx, `
		INSERT INTO request_policies (id, organization_id, name, expression, enabled, priority, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
	`, item.ID, item.OrganizationID, item.Name, item.Expression, item.Enabled, item.Priority, item.CreatedAt)
	return err
}

func (p *Policies) Get(ctx context.Context, policyID string) (*gatewayconfig.RequestPolicy, error) {
	item := &gatewayconfig.RequestPolicy{}
	err := p.Pool.QueryRow(ctx, `
		SELECT id, organization_id, name, expression, enabled, priority, created_at
		FROM request_policies WHERE id=$1
	`, policyID).Scan(
		&item.ID, &item.OrganizationID, &item.Name, &item.Expression,
		&item.Enabled, &item.Priority, &item.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	return item, err
}

func (p *Policies) Update(ctx context.Context, item gatewayconfig.RequestPolicy) (*gatewayconfig.RequestPolicy, error) {
	out := &gatewayconfig.RequestPolicy{}
	err := p.Pool.QueryRow(ctx, `
		UPDATE request_policies SET name=$2, expression=$3, enabled=$4, priority=$5 WHERE id=$1
		RETURNING id, organization_id, name, expression, enabled, priority, created_at
	`, item.ID, item.Name, item.Expression, item.Enabled, item.Priority).Scan(
		&out.ID, &out.OrganizationID, &out.Name, &out.Expression, &out.Enabled, &out.Priority, &out.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	return out, err
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
