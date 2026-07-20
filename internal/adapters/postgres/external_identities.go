package postgres

import (
	"context"
	"errors"

	"github.com/curefatih/afi/internal/identity"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ExternalIdentities implements identity.ExternalIdentityRepository.
type ExternalIdentities struct {
	Pool *pgxpool.Pool
}

func NewExternalIdentities(pool *pgxpool.Pool) *ExternalIdentities {
	return &ExternalIdentities{Pool: pool}
}

func (r *ExternalIdentities) GetByProviderSubject(ctx context.Context, provider, subject string) (*identity.ExternalIdentity, error) {
	item := &identity.ExternalIdentity{}
	err := r.Pool.QueryRow(ctx, `
		SELECT id, user_id, provider, issuer, subject, email, created_at
		FROM external_identities
		WHERE provider = $1 AND subject = $2
	`, provider, subject).Scan(
		&item.ID, &item.UserID, &item.Provider, &item.Issuer, &item.Subject, &item.Email, &item.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (r *ExternalIdentities) Create(ctx context.Context, item identity.ExternalIdentity) error {
	_, err := r.Pool.Exec(ctx, `
		INSERT INTO external_identities (id, user_id, provider, issuer, subject, email, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
	`, item.ID, item.UserID, item.Provider, item.Issuer, item.Subject, item.Email, item.CreatedAt)
	return err
}
