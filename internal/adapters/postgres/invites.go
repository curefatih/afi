package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/tenancy"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Invites implements tenancy.InviteRepository.
type Invites struct {
	Pool *pgxpool.Pool
}

func NewInvites(pool *pgxpool.Pool) *Invites {
	return &Invites{Pool: pool}
}

func (i *Invites) Get(ctx context.Context, inviteID string) (*tenancy.OrgInvite, error) {
	inv := &tenancy.OrgInvite{}
	var accepted *time.Time
	err := i.Pool.QueryRow(ctx, `
		SELECT id, organization_id, email, role, invited_by_user_id, status, expires_at, created_at, accepted_at
		FROM organization_invites WHERE id = $1
	`, inviteID).Scan(
		&inv.ID, &inv.OrganizationID, &inv.Email, &inv.Role, &inv.InvitedByUserID,
		&inv.Status, &inv.ExpiresAt, &inv.CreatedAt, &accepted,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	inv.AcceptedAt = accepted
	return inv, nil
}

func (i *Invites) GetPendingByOrgEmail(ctx context.Context, orgID, email string) (*tenancy.OrgInvite, error) {
	inv := &tenancy.OrgInvite{}
	var accepted *time.Time
	err := i.Pool.QueryRow(ctx, `
		SELECT id, organization_id, email, role, invited_by_user_id, status, expires_at, created_at, accepted_at
		FROM organization_invites
		WHERE organization_id = $1 AND email = $2 AND status = $3
	`, orgID, email, tenancy.InviteStatusPending).Scan(
		&inv.ID, &inv.OrganizationID, &inv.Email, &inv.Role, &inv.InvitedByUserID,
		&inv.Status, &inv.ExpiresAt, &inv.CreatedAt, &accepted,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	inv.AcceptedAt = accepted
	return inv, nil
}

func (i *Invites) GetByTokenHash(ctx context.Context, tokenHash string) (*tenancy.OrgInvite, string, error) {
	inv := &tenancy.OrgInvite{}
	var accepted *time.Time
	var orgName string
	err := i.Pool.QueryRow(ctx, `
		SELECT i.id, i.organization_id, i.email, i.role, i.invited_by_user_id, i.status,
			i.expires_at, i.created_at, i.accepted_at, o.name
		FROM organization_invites i
		JOIN organizations o ON o.id = i.organization_id
		WHERE i.token_hash = $1
	`, tokenHash).Scan(
		&inv.ID, &inv.OrganizationID, &inv.Email, &inv.Role, &inv.InvitedByUserID,
		&inv.Status, &inv.ExpiresAt, &inv.CreatedAt, &accepted, &orgName,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", kernel.ErrNotFound
	}
	if err != nil {
		return nil, "", err
	}
	inv.AcceptedAt = accepted
	return inv, orgName, nil
}

func (i *Invites) ListByOrg(ctx context.Context, orgID string) ([]tenancy.OrgInvite, error) {
	rows, err := i.Pool.Query(ctx, `
		SELECT id, organization_id, email, role, invited_by_user_id, status, expires_at, created_at, accepted_at
		FROM organization_invites
		WHERE organization_id = $1
		ORDER BY created_at DESC
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []tenancy.OrgInvite
	for rows.Next() {
		var inv tenancy.OrgInvite
		var accepted *time.Time
		if err := rows.Scan(
			&inv.ID, &inv.OrganizationID, &inv.Email, &inv.Role, &inv.InvitedByUserID,
			&inv.Status, &inv.ExpiresAt, &inv.CreatedAt, &accepted,
		); err != nil {
			return nil, err
		}
		inv.AcceptedAt = accepted
		out = append(out, inv)
	}
	return out, rows.Err()
}

func (i *Invites) Insert(ctx context.Context, inv tenancy.OrgInvite, tokenHash string) error {
	_, err := i.Pool.Exec(ctx, `
		INSERT INTO organization_invites (
			id, organization_id, email, role, token_hash, invited_by_user_id, status, expires_at, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`, inv.ID, inv.OrganizationID, inv.Email, inv.Role, tokenHash, inv.InvitedByUserID,
		inv.Status, inv.ExpiresAt, inv.CreatedAt)
	return err
}

func (i *Invites) UpdateToken(ctx context.Context, inviteID, tokenHash string, expiresAt time.Time) error {
	tag, err := i.Pool.Exec(ctx, `
		UPDATE organization_invites SET token_hash = $1, expires_at = $2, status = $3
		WHERE id = $4
	`, tokenHash, expiresAt, tenancy.InviteStatusPending, inviteID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return kernel.ErrNotFound
	}
	return nil
}

func (i *Invites) MarkAccepted(ctx context.Context, inviteID string, at time.Time) error {
	tag, err := i.Pool.Exec(ctx, `
		UPDATE organization_invites SET status = $1, accepted_at = $2 WHERE id = $3
	`, tenancy.InviteStatusAccepted, at, inviteID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return kernel.ErrNotFound
	}
	return nil
}

func (i *Invites) MarkRevoked(ctx context.Context, inviteID string) error {
	tag, err := i.Pool.Exec(ctx, `
		UPDATE organization_invites SET status = $1 WHERE id = $2
	`, tenancy.InviteStatusRevoked, inviteID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return kernel.ErrNotFound
	}
	return nil
}
