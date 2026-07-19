package postgres

import (
	"context"
	"errors"

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
	err := u.Pool.QueryRow(ctx, `
		SELECT id, email, name, role, password_hash, created_at
		FROM users WHERE email = $1
	`, email).Scan(&user.ID, &user.Email, &user.Name, &user.Role, &user.PasswordHash, &user.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	return user, err
}

func (u *Users) GetByID(ctx context.Context, id string) (*identity.User, error) {
	user := &identity.User{}
	err := u.Pool.QueryRow(ctx, `
		SELECT id, email, name, role, password_hash, created_at
		FROM users WHERE id = $1
	`, id).Scan(&user.ID, &user.Email, &user.Name, &user.Role, &user.PasswordHash, &user.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	return user, err
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

func (o *Organizations) ListForUser(ctx context.Context, userID string) ([]tenancy.Organization, error) {
	rows, err := o.Pool.Query(ctx, `
		SELECT org.id, org.name, org.created_at
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
		if err := rows.Scan(&item.ID, &item.Name, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (o *Organizations) CreateWithOwner(ctx context.Context, org tenancy.Organization, ownerUserID string) error {
	tx, err := o.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		INSERT INTO organizations (id, name, created_at) VALUES ($1,$2,$3)
	`, org.ID, org.Name, org.CreatedAt); err != nil {
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
