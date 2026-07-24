package postgres

import (
	"context"
	"errors"

	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/tenancy"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Environments implements tenancy.EnvironmentRepository.
type Environments struct {
	Pool *pgxpool.Pool
}

func NewEnvironments(pool *pgxpool.Pool) *Environments {
	return &Environments{Pool: pool}
}

func (e *Environments) ListByProject(ctx context.Context, projectID string) ([]tenancy.Environment, error) {
	rows, err := e.Pool.Query(ctx, `
		SELECT id, organization_id, project_id, name, slug, created_at
		FROM environments WHERE project_id = $1 ORDER BY created_at
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []tenancy.Environment
	for rows.Next() {
		var item tenancy.Environment
		if err := rows.Scan(&item.ID, &item.OrganizationID, &item.ProjectID, &item.Name, &item.Slug, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (e *Environments) Get(ctx context.Context, environmentID string) (*tenancy.Environment, error) {
	var item tenancy.Environment
	err := e.Pool.QueryRow(ctx, `
		SELECT id, organization_id, project_id, name, slug, created_at
		FROM environments WHERE id = $1
	`, environmentID).Scan(&item.ID, &item.OrganizationID, &item.ProjectID, &item.Name, &item.Slug, &item.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (e *Environments) Insert(ctx context.Context, item tenancy.Environment) error {
	_, err := e.Pool.Exec(ctx, `
		INSERT INTO environments (id, organization_id, project_id, name, slug, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, item.ID, item.OrganizationID, item.ProjectID, item.Name, item.Slug, item.CreatedAt)
	return err
}

func (e *Environments) Delete(ctx context.Context, environmentID string) error {
	tag, err := e.Pool.Exec(ctx, `DELETE FROM environments WHERE id = $1`, environmentID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return kernel.ErrNotFound
	}
	return nil
}

func (e *Environments) OrgID(ctx context.Context, environmentID string) (string, error) {
	var orgID string
	err := e.Pool.QueryRow(ctx, `SELECT organization_id FROM environments WHERE id = $1`, environmentID).Scan(&orgID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", kernel.ErrNotFound
	}
	return orgID, err
}
