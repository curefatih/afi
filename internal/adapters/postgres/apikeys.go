package postgres

import (
	"context"
	"errors"

	"github.com/curefatih/afi/internal/access"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// APIKeys implements access.APIKeyRepository.
type APIKeys struct {
	Pool *pgxpool.Pool
}

func NewAPIKeys(pool *pgxpool.Pool) *APIKeys {
	return &APIKeys{Pool: pool}
}

func scanAPIKey(scan func(dest ...any) error) (access.APIKey, error) {
	var k access.APIKey
	var projectID, ownerUserID, environmentID *string
	err := scan(&k.ID, &projectID, &k.OrganizationID, &environmentID, &k.Name, &k.Kind, &ownerUserID, &k.KeyPrefix, &k.CreatedAt)
	if err != nil {
		return k, err
	}
	if projectID != nil {
		k.ProjectID = *projectID
	}
	if ownerUserID != nil {
		k.OwnerUserID = *ownerUserID
	}
	if environmentID != nil {
		k.EnvironmentID = *environmentID
	}
	return k, nil
}

func (a *APIKeys) ListByProject(ctx context.Context, projectID string) ([]access.APIKey, error) {
	rows, err := a.Pool.Query(ctx, `
		SELECT id, project_id, organization_id, environment_id, name, kind, owner_user_id, key_prefix, created_at
		FROM api_keys WHERE project_id = $1 ORDER BY created_at
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []access.APIKey
	for rows.Next() {
		k, err := scanAPIKey(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (a *APIKeys) ListByOrg(ctx context.Context, orgID string) ([]access.APIKey, error) {
	rows, err := a.Pool.Query(ctx, `
		SELECT id, project_id, organization_id, environment_id, name, kind, owner_user_id, key_prefix, created_at
		FROM api_keys WHERE organization_id = $1 ORDER BY created_at
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []access.APIKey
	for rows.Next() {
		k, err := scanAPIKey(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (a *APIKeys) Get(ctx context.Context, keyID string) (*access.APIKey, error) {
	row := a.Pool.QueryRow(ctx, `
		SELECT id, project_id, organization_id, environment_id, name, kind, owner_user_id, key_prefix, created_at
		FROM api_keys WHERE id = $1
	`, keyID)
	k, err := scanAPIKey(row.Scan)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &k, nil
}

func (a *APIKeys) Insert(ctx context.Context, key access.APIKey, keyHash string) error {
	var project any
	if key.ProjectID != "" {
		project = key.ProjectID
	}
	var owner any
	if key.OwnerUserID != "" {
		owner = key.OwnerUserID
	}
	var env any
	if key.EnvironmentID != "" {
		env = key.EnvironmentID
	}
	_, err := a.Pool.Exec(ctx, `
		INSERT INTO api_keys (id, project_id, organization_id, environment_id, name, kind, owner_user_id, key_hash, key_prefix, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, key.ID, project, key.OrganizationID, env, key.Name, key.Kind, owner, keyHash, key.KeyPrefix, key.CreatedAt)
	return err
}

func (a *APIKeys) Delete(ctx context.Context, keyID string) error {
	tag, err := a.Pool.Exec(ctx, `DELETE FROM api_keys WHERE id=$1`, keyID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return kernel.ErrNotFound
	}
	return nil
}

func (a *APIKeys) OrgID(ctx context.Context, keyID string) (string, error) {
	var orgID string
	err := a.Pool.QueryRow(ctx, `SELECT organization_id FROM api_keys WHERE id = $1`, keyID).Scan(&orgID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", kernel.ErrNotFound
	}
	return orgID, err
}
