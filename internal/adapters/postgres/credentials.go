package postgres

import (
	"context"
	"errors"

	"github.com/curefatih/afi/internal/credentials"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Credentials implements credentials.Repository.
type Credentials struct {
	Pool *pgxpool.Pool
}

func NewCredentials(pool *pgxpool.Pool) *Credentials {
	return &Credentials{Pool: pool}
}

func (r *Credentials) ListByOrg(ctx context.Context, orgID string) ([]credentials.Credential, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT id, organization_id, name, provider_type, storage_kind, secret_ref,
		       encrypted_payload, key_version, status, created_at, updated_at
		FROM provider_credentials
		WHERE organization_id = $1
		ORDER BY created_at
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []credentials.Credential
	for rows.Next() {
		c, err := scanCredential(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c.Public())
	}
	return out, rows.Err()
}

func (r *Credentials) Get(ctx context.Context, id string) (*credentials.Credential, error) {
	row := r.Pool.QueryRow(ctx, `
		SELECT id, organization_id, name, provider_type, storage_kind, secret_ref,
		       encrypted_payload, key_version, status, created_at, updated_at
		FROM provider_credentials WHERE id = $1
	`, id)
	c, err := scanCredential(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Credentials) Insert(ctx context.Context, c credentials.Credential) error {
	_, err := r.Pool.Exec(ctx, `
		INSERT INTO provider_credentials (
			id, organization_id, name, provider_type, storage_kind, secret_ref,
			encrypted_payload, key_version, status, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	`, c.ID, c.OrganizationID, c.Name, c.ProviderType, c.StorageKind, nullIfEmpty(c.SecretRef),
		c.EncryptedPayload, c.KeyVersion, c.Status, c.CreatedAt, c.UpdatedAt)
	return err
}

func (r *Credentials) UpdateMeta(ctx context.Context, id, name, status string) (*credentials.Credential, error) {
	row := r.Pool.QueryRow(ctx, `
		UPDATE provider_credentials
		SET name = COALESCE(NULLIF($2, ''), name),
		    status = COALESCE(NULLIF($3, ''), status),
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, organization_id, name, provider_type, storage_kind, secret_ref,
		          encrypted_payload, key_version, status, created_at, updated_at
	`, id, name, status)
	c, err := scanCredential(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	pub := c.Public()
	return &pub, nil
}

func (r *Credentials) UpdateSecret(ctx context.Context, id string, secretRef string, payload []byte, keyVersion int) (*credentials.Credential, error) {
	row := r.Pool.QueryRow(ctx, `
		UPDATE provider_credentials
		SET secret_ref = $2,
		    encrypted_payload = $3,
		    key_version = $4,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, organization_id, name, provider_type, storage_kind, secret_ref,
		          encrypted_payload, key_version, status, created_at, updated_at
	`, id, nullIfEmpty(secretRef), payload, keyVersion)
	c, err := scanCredential(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	pub := c.Public()
	return &pub, nil
}

func (r *Credentials) Delete(ctx context.Context, id string) error {
	tag, err := r.Pool.Exec(ctx, `DELETE FROM provider_credentials WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return kernel.ErrNotFound
	}
	return nil
}

func (r *Credentials) OrgID(ctx context.Context, id string) (string, error) {
	var orgID string
	err := r.Pool.QueryRow(ctx, `SELECT organization_id FROM provider_credentials WHERE id = $1`, id).Scan(&orgID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", kernel.ErrNotFound
	}
	return orgID, err
}

func (r *Credentials) HasAssignments(ctx context.Context, credentialID string) (bool, error) {
	var n int
	err := r.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM credential_assignments WHERE credential_id = $1
	`, credentialID).Scan(&n)
	return n > 0, err
}

func (r *Credentials) ListAssignmentsByOrg(ctx context.Context, orgID string) ([]credentials.Assignment, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT id, credential_id, organization_id, provider_type, scope_type, scope_id, created_at, created_by
		FROM credential_assignments
		WHERE organization_id = $1
		ORDER BY created_at
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []credentials.Assignment
	for rows.Next() {
		var a credentials.Assignment
		var createdBy *string
		if err := rows.Scan(&a.ID, &a.CredentialID, &a.OrganizationID, &a.ProviderType, &a.ScopeType, &a.ScopeID, &a.CreatedAt, &createdBy); err != nil {
			return nil, err
		}
		if createdBy != nil {
			a.CreatedBy = *createdBy
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *Credentials) InsertAssignment(ctx context.Context, a credentials.Assignment) error {
	_, err := r.Pool.Exec(ctx, `
		INSERT INTO credential_assignments (
			id, credential_id, organization_id, provider_type, scope_type, scope_id, created_at, created_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`, a.ID, a.CredentialID, a.OrganizationID, a.ProviderType, a.ScopeType, a.ScopeID, a.CreatedAt, nullIfEmpty(a.CreatedBy))
	return err
}

func (r *Credentials) UpsertAssignment(ctx context.Context, a credentials.Assignment) (*credentials.Assignment, error) {
	// Replace existing slot for (scope_type, scope_id, provider_type).
	_, err := r.Pool.Exec(ctx, `
		DELETE FROM credential_assignments
		WHERE scope_type = $1 AND scope_id = $2 AND provider_type = $3
	`, a.ScopeType, a.ScopeID, a.ProviderType)
	if err != nil {
		return nil, err
	}
	if err := r.InsertAssignment(ctx, a); err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *Credentials) DeleteAssignment(ctx context.Context, id string) error {
	tag, err := r.Pool.Exec(ctx, `DELETE FROM credential_assignments WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return kernel.ErrNotFound
	}
	return nil
}

func (r *Credentials) AssignmentOrgID(ctx context.Context, id string) (string, error) {
	var orgID string
	err := r.Pool.QueryRow(ctx, `SELECT organization_id FROM credential_assignments WHERE id = $1`, id).Scan(&orgID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", kernel.ErrNotFound
	}
	return orgID, err
}

type scannable interface {
	Scan(dest ...any) error
}

func scanCredential(row scannable) (credentials.Credential, error) {
	var c credentials.Credential
	var secretRef *string
	var payload []byte
	err := row.Scan(
		&c.ID, &c.OrganizationID, &c.Name, &c.ProviderType, &c.StorageKind, &secretRef,
		&payload, &c.KeyVersion, &c.Status, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return c, err
	}
	if secretRef != nil {
		c.SecretRef = *secretRef
	}
	c.EncryptedPayload = payload
	return c, nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
