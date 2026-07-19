package postgres

import (
	"context"
	"errors"

	"github.com/curefatih/afi/internal/gatewayconfig"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Quotas implements gatewayconfig.QuotaRepository.
type Quotas struct {
	Pool *pgxpool.Pool
}

func NewQuotas(pool *pgxpool.Pool) *Quotas {
	return &Quotas{Pool: pool}
}

func (q *Quotas) ListByOrg(ctx context.Context, orgID string) ([]gatewayconfig.Quota, error) {
	rows, err := q.Pool.Query(ctx, `
		SELECT id, organization_id, scope_type, scope_id, metric, limit_value, time_window, created_at
		FROM quotas WHERE organization_id=$1 ORDER BY created_at
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []gatewayconfig.Quota
	for rows.Next() {
		var item gatewayconfig.Quota
		if err := rows.Scan(
			&item.ID, &item.OrganizationID, &item.ScopeType, &item.ScopeID,
			&item.Metric, &item.LimitValue, &item.Window, &item.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (q *Quotas) Insert(ctx context.Context, item gatewayconfig.Quota) error {
	_, err := q.Pool.Exec(ctx, `
		INSERT INTO quotas (id, organization_id, scope_type, scope_id, metric, limit_value, time_window, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`, item.ID, item.OrganizationID, item.ScopeType, item.ScopeID, item.Metric, item.LimitValue, item.Window, item.CreatedAt)
	return err
}

func (q *Quotas) UpdateLimit(ctx context.Context, quotaID string, limitValue int64) (*gatewayconfig.Quota, error) {
	item := &gatewayconfig.Quota{}
	err := q.Pool.QueryRow(ctx, `
		UPDATE quotas SET limit_value=$2 WHERE id=$1
		RETURNING id, organization_id, scope_type, scope_id, metric, limit_value, time_window, created_at
	`, quotaID, limitValue).Scan(
		&item.ID, &item.OrganizationID, &item.ScopeType, &item.ScopeID,
		&item.Metric, &item.LimitValue, &item.Window, &item.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	return item, err
}

func (q *Quotas) Delete(ctx context.Context, quotaID string) error {
	tag, err := q.Pool.Exec(ctx, `DELETE FROM quotas WHERE id=$1`, quotaID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return kernel.ErrNotFound
	}
	return nil
}

func (q *Quotas) OrgID(ctx context.Context, quotaID string) (string, error) {
	var orgID string
	err := q.Pool.QueryRow(ctx, `SELECT organization_id FROM quotas WHERE id=$1`, quotaID).Scan(&orgID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", kernel.ErrNotFound
	}
	return orgID, err
}
