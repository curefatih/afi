package postgres

import (
	"context"
	"errors"

	"github.com/curefatih/afi/internal/access"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SigningKeys struct {
	Pool *pgxpool.Pool
}

func NewSigningKeys(pool *pgxpool.Pool) *SigningKeys {
	return &SigningKeys{Pool: pool}
}

func scanSigningKey(scan func(dest ...any) error) (access.SigningKey, error) {
	var k access.SigningKey
	var projectID, environmentID *string
	err := scan(&k.ID, &k.KeyID, &projectID, &k.OrganizationID, &environmentID, &k.Name, &k.Algorithm, &k.PublicKeyPEM, &k.Status, &k.CreatedAt, &k.UpdatedAt)
	if err != nil {
		return k, err
	}
	if projectID != nil {
		k.ProjectID = *projectID
	}
	if environmentID != nil {
		k.EnvironmentID = *environmentID
	}
	return k, nil
}

func (r *SigningKeys) ListByOrg(ctx context.Context, orgID string) ([]access.SigningKey, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT id, key_id, project_id, organization_id, environment_id, name, algorithm, public_key_pem, status, created_at, updated_at
		FROM signing_keys WHERE organization_id = $1 ORDER BY created_at
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []access.SigningKey
	for rows.Next() {
		k, err := scanSigningKey(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (r *SigningKeys) Get(ctx context.Context, id string) (*access.SigningKey, error) {
	row := r.Pool.QueryRow(ctx, `
		SELECT id, key_id, project_id, organization_id, environment_id, name, algorithm, public_key_pem, status, created_at, updated_at
		FROM signing_keys WHERE id = $1
	`, id)
	k, err := scanSigningKey(row.Scan)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &k, nil
}

func (r *SigningKeys) Insert(ctx context.Context, k access.SigningKey) error {
	_, err := r.Pool.Exec(ctx, `
		INSERT INTO signing_keys (
			id, key_id, project_id, organization_id, environment_id, name, algorithm, public_key_pem, status, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	`, k.ID, k.KeyID, nullIfEmpty(k.ProjectID), k.OrganizationID, nullIfEmpty(k.EnvironmentID), k.Name, k.Algorithm, k.PublicKeyPEM, k.Status, k.CreatedAt, k.UpdatedAt)
	return err
}

func (r *SigningKeys) UpdateMeta(ctx context.Context, id, name, status string) (*access.SigningKey, error) {
	row := r.Pool.QueryRow(ctx, `
		UPDATE signing_keys
		SET name = COALESCE(NULLIF($2, ''), name),
		    status = COALESCE(NULLIF($3, ''), status),
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, key_id, project_id, organization_id, environment_id, name, algorithm, public_key_pem, status, created_at, updated_at
	`, id, name, status)
	k, err := scanSigningKey(row.Scan)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &k, nil
}

func (r *SigningKeys) UpdatePublicKey(ctx context.Context, id, publicKeyPEM string) (*access.SigningKey, error) {
	row := r.Pool.QueryRow(ctx, `
		UPDATE signing_keys
		SET public_key_pem = $2,
		    status = 'active',
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, key_id, project_id, organization_id, environment_id, name, algorithm, public_key_pem, status, created_at, updated_at
	`, id, publicKeyPEM)
	k, err := scanSigningKey(row.Scan)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &k, nil
}

func (r *SigningKeys) Delete(ctx context.Context, id string) error {
	tag, err := r.Pool.Exec(ctx, `DELETE FROM signing_keys WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return kernel.ErrNotFound
	}
	return nil
}

func (r *SigningKeys) OrgID(ctx context.Context, id string) (string, error) {
	var orgID string
	err := r.Pool.QueryRow(ctx, `SELECT organization_id FROM signing_keys WHERE id = $1`, id).Scan(&orgID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", kernel.ErrNotFound
	}
	return orgID, err
}
