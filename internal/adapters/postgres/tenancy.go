package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/curefatih/afi/internal/gatewayconfig"
	"github.com/curefatih/afi/internal/identity"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/tenancy"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Users implements identity.UserRepository.
type Users struct {
	Pool *pgxpool.Pool
}

func NewUsers(pool *pgxpool.Pool) *Users {
	return &Users{Pool: pool}
}

func (u *Users) GetByEmail(ctx context.Context, email string) (*identity.User, error) {
	user := &identity.User{}
	var passwordHash *string
	err := u.Pool.QueryRow(ctx, `
		SELECT id, email, name, role, password_hash, created_at
		FROM users WHERE email = $1
	`, email).Scan(&user.ID, &user.Email, &user.Name, &user.Role, &passwordHash, &user.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if passwordHash != nil {
		user.PasswordHash = *passwordHash
	}
	return user, nil
}

func (u *Users) GetByID(ctx context.Context, id string) (*identity.User, error) {
	user := &identity.User{}
	var passwordHash *string
	err := u.Pool.QueryRow(ctx, `
		SELECT id, email, name, role, password_hash, created_at
		FROM users WHERE id = $1
	`, id).Scan(&user.ID, &user.Email, &user.Name, &user.Role, &passwordHash, &user.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if passwordHash != nil {
		user.PasswordHash = *passwordHash
	}
	return user, nil
}

func (u *Users) Create(ctx context.Context, user identity.User) error {
	var passwordHash any
	if user.PasswordHash != "" {
		passwordHash = user.PasswordHash
	}
	_, err := u.Pool.Exec(ctx, `
		INSERT INTO users (id, email, name, role, password_hash, created_at)
		VALUES ($1,$2,$3,$4,$5,$6)
	`, user.ID, user.Email, user.Name, user.Role, passwordHash, user.CreatedAt)
	return err
}

func (u *Users) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	tag, err := u.Pool.Exec(ctx, `
		UPDATE users SET password_hash = $2 WHERE id = $1
	`, userID, passwordHash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return kernel.ErrNotFound
	}
	return nil
}

// PasswordResets implements identity.PasswordResetRepository.
type PasswordResets struct {
	Pool *pgxpool.Pool
}

func NewPasswordResets(pool *pgxpool.Pool) *PasswordResets {
	return &PasswordResets{Pool: pool}
}

func (r *PasswordResets) Create(ctx context.Context, token identity.PasswordResetToken) error {
	_, err := r.Pool.Exec(ctx, `
		INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at, created_at, used_at)
		VALUES ($1,$2,$3,$4,$5,$6)
	`, token.ID, token.UserID, token.TokenHash, token.ExpiresAt, token.CreatedAt, token.UsedAt)
	return err
}

func (r *PasswordResets) DeleteUnusedForUser(ctx context.Context, userID string) error {
	_, err := r.Pool.Exec(ctx, `
		DELETE FROM password_reset_tokens WHERE user_id = $1 AND used_at IS NULL
	`, userID)
	return err
}

func (r *PasswordResets) GetByTokenHash(ctx context.Context, hash string) (*identity.PasswordResetToken, error) {
	var t identity.PasswordResetToken
	err := r.Pool.QueryRow(ctx, `
		SELECT id, user_id, token_hash, expires_at, created_at, used_at
		FROM password_reset_tokens WHERE token_hash = $1
	`, hash).Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.CreatedAt, &t.UsedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *PasswordResets) Consume(ctx context.Context, id string, usedAt time.Time) error {
	tag, err := r.Pool.Exec(ctx, `
		UPDATE password_reset_tokens SET used_at = $2 WHERE id = $1 AND used_at IS NULL
	`, id, usedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return kernel.ErrNotFound
	}
	return nil
}

// Organizations implements tenancy.OrganizationRepository.
type Organizations struct {
	Pool *pgxpool.Pool
}

func NewOrganizations(pool *pgxpool.Pool) *Organizations {
	return &Organizations{Pool: pool}
}

func (o *Organizations) Count(ctx context.Context) (int64, error) {
	var n int64
	err := o.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM organizations`).Scan(&n)
	return n, err
}

func (o *Organizations) Get(ctx context.Context, orgID string) (*tenancy.Organization, error) {
	var item tenancy.Organization
	err := o.Pool.QueryRow(ctx, `
		SELECT id, name, COALESCE(mail_provider, ''), COALESCE(byok_selector_header, ''), created_at
		FROM organizations WHERE id = $1
	`, orgID).Scan(&item.ID, &item.Name, &item.MailProvider, &item.BYOKSelectorHeader, &item.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	return &item, err
}

func (o *Organizations) ListForUser(ctx context.Context, userID string) ([]tenancy.Organization, error) {
	rows, err := o.Pool.Query(ctx, `
		SELECT org.id, org.name, COALESCE(org.mail_provider, ''), COALESCE(org.byok_selector_header, ''), org.created_at
		FROM organizations org
		JOIN organization_members m ON m.organization_id = org.id
		WHERE m.user_id = $1
		ORDER BY org.created_at
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []tenancy.Organization
	for rows.Next() {
		var item tenancy.Organization
		if err := rows.Scan(&item.ID, &item.Name, &item.MailProvider, &item.BYOKSelectorHeader, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (o *Organizations) SetMailProvider(ctx context.Context, orgID, provider string) error {
	tag, err := o.Pool.Exec(ctx, `
		UPDATE organizations SET mail_provider = $1 WHERE id = $2
	`, provider, orgID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return kernel.ErrNotFound
	}
	return nil
}

func (o *Organizations) GetDefaultRetry(ctx context.Context, orgID string) (*gatewayconfig.RetryConfig, error) {
	var raw []byte
	err := o.Pool.QueryRow(ctx, `
		SELECT default_retry FROM organizations WHERE id = $1
	`, orgID).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return DecodeRetry(raw), nil
}

func (o *Organizations) SetDefaultRetry(ctx context.Context, orgID string, retry *gatewayconfig.RetryConfig) error {
	raw, err := encodeRetry(retry)
	if err != nil {
		return err
	}
	tag, err := o.Pool.Exec(ctx, `
		UPDATE organizations SET default_retry = $1 WHERE id = $2
	`, raw, orgID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return kernel.ErrNotFound
	}
	return nil
}

func (o *Organizations) GetObjectStore(ctx context.Context, orgID string) (*gatewayconfig.ObjectStoreConfig, error) {
	var raw []byte
	err := o.Pool.QueryRow(ctx, `
		SELECT object_store FROM organizations WHERE id = $1
	`, orgID).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return DecodeObjectStore(raw), nil
}

func (o *Organizations) SetObjectStore(ctx context.Context, orgID string, cfg *gatewayconfig.ObjectStoreConfig) error {
	raw, err := encodeObjectStore(cfg)
	if err != nil {
		return err
	}
	tag, err := o.Pool.Exec(ctx, `
		UPDATE organizations SET object_store = $1 WHERE id = $2
	`, raw, orgID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return kernel.ErrNotFound
	}
	return nil
}

func (o *Organizations) CreateWithOwner(ctx context.Context, org tenancy.Organization, ownerUserID string) error {
	tx, err := o.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		INSERT INTO organizations (id, name, mail_provider, created_at) VALUES ($1,$2,$3,$4)
	`, org.ID, org.Name, org.MailProvider, org.CreatedAt); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO organization_members (organization_id, user_id, role) VALUES ($1,$2,$3)
	`, org.ID, ownerUserID, tenancy.OrgRoleOwner); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (o *Organizations) ListMembers(ctx context.Context, orgID string) ([]tenancy.OrgMember, error) {
	rows, err := o.Pool.Query(ctx, `
		SELECT u.id, u.email, u.name, m.role
		FROM organization_members m
		JOIN users u ON u.id = m.user_id
		WHERE m.organization_id = $1
		ORDER BY u.email
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []tenancy.OrgMember
	for rows.Next() {
		var m tenancy.OrgMember
		if err := rows.Scan(&m.UserID, &m.Email, &m.Name, &m.Role); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (o *Organizations) AddMember(ctx context.Context, orgID, userID, role string) error {
	_, err := o.Pool.Exec(ctx, `
		INSERT INTO organization_members (organization_id, user_id, role) VALUES ($1,$2,$3)
		ON CONFLICT DO NOTHING
	`, orgID, userID, role)
	return err
}

func (o *Organizations) GetMemberRole(ctx context.Context, userID, orgID string) (string, error) {
	var role string
	err := o.Pool.QueryRow(ctx, `
		SELECT role FROM organization_members WHERE user_id=$1 AND organization_id=$2
	`, userID, orgID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", kernel.ErrNotFound
	}
	return role, err
}

func (o *Organizations) CountOwners(ctx context.Context, orgID string) (int, error) {
	var n int
	err := o.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM organization_members
		WHERE organization_id=$1 AND role=$2
	`, orgID, tenancy.OrgRoleOwner).Scan(&n)
	return n, err
}

func (o *Organizations) ApplyRoleChange(ctx context.Context, orgID, actorUserID, targetUserID, newRole string, demoteActor bool) error {
	tx, err := o.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if demoteActor {
		if _, err := tx.Exec(ctx, `
			UPDATE organization_members SET role=$1 WHERE organization_id=$2 AND user_id=$3
		`, tenancy.OrgRoleAdmin, orgID, actorUserID); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE organization_members SET role=$1 WHERE organization_id=$2 AND user_id=$3
	`, newRole, orgID, targetUserID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (o *Organizations) GetMember(ctx context.Context, orgID, userID string) (*tenancy.OrgMember, error) {
	var m tenancy.OrgMember
	err := o.Pool.QueryRow(ctx, `
		SELECT u.id, u.email, u.name, m.role
		FROM organization_members m
		JOIN users u ON u.id = m.user_id
		WHERE m.organization_id=$1 AND m.user_id=$2
	`, orgID, userID).Scan(&m.UserID, &m.Email, &m.Name, &m.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	return &m, err
}

// Teams implements tenancy.TeamRepository.
type Teams struct {
	Pool *pgxpool.Pool
}

func NewTeams(pool *pgxpool.Pool) *Teams {
	return &Teams{Pool: pool}
}

func (t *Teams) ListByOrg(ctx context.Context, orgID string) ([]tenancy.Team, error) {
	rows, err := t.Pool.Query(ctx, `
		SELECT id, organization_id, name, created_at, updated_at
		FROM teams WHERE organization_id = $1 ORDER BY created_at
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []tenancy.Team
	for rows.Next() {
		var item tenancy.Team
		if err := rows.Scan(&item.ID, &item.OrganizationID, &item.Name, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.TeamID = item.ID
		out = append(out, item)
	}
	return out, rows.Err()
}

func (t *Teams) ListByOrgForUser(ctx context.Context, orgID, userID string) ([]tenancy.Team, error) {
	rows, err := t.Pool.Query(ctx, `
		SELECT t.id, t.organization_id, t.name, t.created_at, t.updated_at
		FROM teams t
		JOIN team_members tm ON tm.team_id = t.id
		WHERE t.organization_id = $1 AND tm.user_id = $2
		ORDER BY t.created_at
	`, orgID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []tenancy.Team
	for rows.Next() {
		var item tenancy.Team
		if err := rows.Scan(&item.ID, &item.OrganizationID, &item.Name, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.TeamID = item.ID
		out = append(out, item)
	}
	return out, rows.Err()
}

func (t *Teams) Get(ctx context.Context, teamID string) (*tenancy.Team, error) {
	item := &tenancy.Team{}
	err := t.Pool.QueryRow(ctx, `
		SELECT id, organization_id, name, created_at, updated_at
		FROM teams WHERE id = $1
	`, teamID).Scan(&item.ID, &item.OrganizationID, &item.Name, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	item.TeamID = item.ID
	return item, nil
}

func (t *Teams) OrgID(ctx context.Context, teamID string) (string, error) {
	var orgID string
	err := t.Pool.QueryRow(ctx, `SELECT organization_id FROM teams WHERE id = $1`, teamID).Scan(&orgID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", kernel.ErrNotFound
	}
	return orgID, err
}

func (t *Teams) ListMembers(ctx context.Context, teamID string) ([]tenancy.TeamMember, error) {
	rows, err := t.Pool.Query(ctx, `
		SELECT u.id, u.name, u.email, tm.role
		FROM team_members tm
		JOIN users u ON u.id = tm.user_id
		WHERE tm.team_id = $1
		ORDER BY u.email
	`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []tenancy.TeamMember
	for rows.Next() {
		var m tenancy.TeamMember
		if err := rows.Scan(&m.UserID, &m.Name, &m.Email, &m.Role); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (t *Teams) GetMember(ctx context.Context, teamID, userID string) (*tenancy.TeamMember, error) {
	var m tenancy.TeamMember
	err := t.Pool.QueryRow(ctx, `
		SELECT u.id, u.name, u.email, tm.role
		FROM team_members tm
		JOIN users u ON u.id = tm.user_id
		WHERE tm.team_id = $1 AND tm.user_id = $2
	`, teamID, userID).Scan(&m.UserID, &m.Name, &m.Email, &m.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	return &m, err
}

func (t *Teams) GetMemberRole(ctx context.Context, teamID, userID string) (string, error) {
	var role string
	err := t.Pool.QueryRow(ctx, `
		SELECT role FROM team_members WHERE team_id = $1 AND user_id = $2
	`, teamID, userID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", kernel.ErrNotFound
	}
	return role, err
}

func (t *Teams) AddMember(ctx context.Context, teamID, userID, role string) error {
	_, err := t.Pool.Exec(ctx, `
		INSERT INTO team_members (team_id, user_id, role) VALUES ($1,$2,$3)
	`, teamID, userID, role)
	return err
}

func (t *Teams) RemoveMember(ctx context.Context, teamID, userID string) error {
	tag, err := t.Pool.Exec(ctx, `
		DELETE FROM team_members WHERE team_id = $1 AND user_id = $2
	`, teamID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return kernel.ErrNotFound
	}
	return nil
}

func (t *Teams) SetMemberRole(ctx context.Context, teamID, userID, role string) error {
	tag, err := t.Pool.Exec(ctx, `
		UPDATE team_members SET role = $1 WHERE team_id = $2 AND user_id = $3
	`, role, teamID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return kernel.ErrNotFound
	}
	return nil
}

func (t *Teams) CountOwners(ctx context.Context, teamID string) (int, error) {
	var n int
	err := t.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM team_members WHERE team_id = $1 AND role = $2
	`, teamID, tenancy.TeamRoleOwner).Scan(&n)
	return n, err
}

func (t *Teams) CreateWithOwner(ctx context.Context, team tenancy.Team, ownerUserID string) error {
	tx, err := t.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		INSERT INTO teams (id, organization_id, name, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5)
	`, team.ID, team.OrganizationID, team.Name, team.CreatedAt, team.UpdatedAt); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO team_members (team_id, user_id, role) VALUES ($1,$2,$3)
	`, team.ID, ownerUserID, tenancy.TeamRoleOwner); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Projects implements tenancy.ProjectRepository.
type Projects struct {
	Pool *pgxpool.Pool
}

func NewProjects(pool *pgxpool.Pool) *Projects {
	return &Projects{Pool: pool}
}

func (p *Projects) ListByOrg(ctx context.Context, orgID string) ([]tenancy.Project, error) {
	rows, err := p.Pool.Query(ctx, `
		SELECT id, organization_id, COALESCE(team_id, ''), name, created_at, updated_at
		FROM projects WHERE organization_id = $1 ORDER BY created_at
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []tenancy.Project
	for rows.Next() {
		var item tenancy.Project
		if err := rows.Scan(&item.ID, &item.OrganizationID, &item.TeamID, &item.Name, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (p *Projects) ListByOrgForUser(ctx context.Context, orgID, userID string) ([]tenancy.Project, error) {
	rows, err := p.Pool.Query(ctx, `
		SELECT p.id, p.organization_id, COALESCE(p.team_id, ''), p.name, p.created_at, p.updated_at
		FROM projects p
		JOIN team_members tm ON tm.team_id = p.team_id
		WHERE p.organization_id = $1 AND tm.user_id = $2
		ORDER BY p.created_at
	`, orgID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []tenancy.Project
	for rows.Next() {
		var item tenancy.Project
		if err := rows.Scan(&item.ID, &item.OrganizationID, &item.TeamID, &item.Name, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (p *Projects) Insert(ctx context.Context, item tenancy.Project) error {
	var team any
	if item.TeamID != "" {
		team = item.TeamID
	}
	_, err := p.Pool.Exec(ctx, `
		INSERT INTO projects (id, organization_id, team_id, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, item.ID, item.OrganizationID, team, item.Name, item.CreatedAt, item.UpdatedAt)
	return err
}

func (p *Projects) OrgID(ctx context.Context, projectID string) (string, error) {
	var orgID string
	err := p.Pool.QueryRow(ctx, `SELECT organization_id FROM projects WHERE id = $1`, projectID).Scan(&orgID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", kernel.ErrNotFound
	}
	return orgID, err
}
