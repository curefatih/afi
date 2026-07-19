package controlplane

import (
	"context"
	"errors"
	"time"

	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/snapshot"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	Role         string    `json:"role"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Organization struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Team struct {
	ID             string    `json:"id"`
	TeamID         string    `json:"team_id"`
	OrganizationID string    `json:"organization_id"`
	Name           string    `json:"name"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type TeamMember struct {
	UserID string `json:"user_id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}

type Project struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	TeamID         string    `json:"team_id,omitempty"`
	Name           string    `json:"name"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type APIKey struct {
	ID             string    `json:"id"`
	ProjectID      string    `json:"project_id"`
	OrganizationID string    `json:"organization_id"`
	Name           string    `json:"name"`
	KeyValue       string    `json:"key"`
	CreatedAt      time.Time `json:"created_at"`
}

type Provider struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	Name           string    `json:"name"`
	Type           string    `json:"type"`
	BaseURL        string    `json:"base_url"`
	APIKeyEnv      string    `json:"api_key_env"`
	CreatedAt      time.Time `json:"created_at"`
}

type Route struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	Model          string    `json:"model"`
	ProviderID     string    `json:"provider_id"`
	TargetModel    string    `json:"target_model"`
	CreatedAt      time.Time `json:"created_at"`
}

func (s *Store) CountOrgs(ctx context.Context) (int64, error) {
	var n int64
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM organizations`).Scan(&n)
	return n, err
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	u := &User{}
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, name, role, password_hash, created_at
		FROM users WHERE email = $1
	`, email).Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	return u, err
}

func (s *Store) GetUserByID(ctx context.Context, id string) (*User, error) {
	u := &User{}
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, name, role, password_hash, created_at
		FROM users WHERE id = $1
	`, id).Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	return u, err
}

func (s *Store) ListOrganizationsForUser(ctx context.Context, userID string) ([]Organization, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT o.id, o.name, o.created_at
		FROM organizations o
		JOIN organization_members m ON m.organization_id = o.id
		WHERE m.user_id = $1
		ORDER BY o.created_at
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Organization
	for rows.Next() {
		var o Organization
		if err := rows.Scan(&o.ID, &o.Name, &o.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func (s *Store) ListTeams(ctx context.Context, orgID string) ([]Team, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, organization_id, name, created_at, updated_at
		FROM teams WHERE organization_id = $1 ORDER BY created_at
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Team
	for rows.Next() {
		var t Team
		if err := rows.Scan(&t.ID, &t.OrganizationID, &t.Name, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		t.TeamID = t.ID
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) GetTeam(ctx context.Context, teamID string) (*Team, error) {
	t := &Team{}
	err := s.pool.QueryRow(ctx, `
		SELECT id, organization_id, name, created_at, updated_at
		FROM teams WHERE id = $1
	`, teamID).Scan(&t.ID, &t.OrganizationID, &t.Name, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	t.TeamID = t.ID
	return t, nil
}

func (s *Store) ListTeamMembers(ctx context.Context, teamID string) ([]TeamMember, error) {
	rows, err := s.pool.Query(ctx, `
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

	var out []TeamMember
	for rows.Next() {
		var m TeamMember
		if err := rows.Scan(&m.UserID, &m.Name, &m.Email, &m.Role); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) ListProjects(ctx context.Context, orgID string) ([]Project, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, organization_id, COALESCE(team_id, ''), name, created_at, updated_at
		FROM projects WHERE organization_id = $1 ORDER BY created_at
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.OrganizationID, &p.TeamID, &p.Name, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) CreateProject(ctx context.Context, orgID, teamID, name string) (*Project, error) {
	p := &Project{
		ID:             newID("proj"),
		OrganizationID: orgID,
		TeamID:         teamID,
		Name:           name,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	var team any
	if teamID != "" {
		team = teamID
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO projects (id, organization_id, team_id, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, p.ID, p.OrganizationID, team, p.Name, p.CreatedAt, p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Store) ListAPIKeys(ctx context.Context, projectID string) ([]APIKey, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, project_id, organization_id, name, key_value, created_at
		FROM api_keys WHERE project_id = $1 ORDER BY created_at
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []APIKey
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.ID, &k.ProjectID, &k.OrganizationID, &k.Name, &k.KeyValue, &k.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (s *Store) GetProjectOrgID(ctx context.Context, projectID string) (string, error) {
	var orgID string
	err := s.pool.QueryRow(ctx, `SELECT organization_id FROM projects WHERE id = $1`, projectID).Scan(&orgID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", kernel.ErrNotFound
	}
	return orgID, err
}

func (s *Store) CreateAPIKey(ctx context.Context, orgID, projectID, name, keyValue string) (*APIKey, error) {
	k := &APIKey{
		ID:             newID("key"),
		ProjectID:      projectID,
		OrganizationID: orgID,
		Name:           name,
		KeyValue:       keyValue,
		CreatedAt:      time.Now().UTC(),
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO api_keys (id, project_id, organization_id, name, key_value, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, k.ID, k.ProjectID, k.OrganizationID, k.Name, k.KeyValue, k.CreatedAt)
	if err != nil {
		return nil, err
	}
	return k, nil
}

func (s *Store) LoadSnapshotSource(ctx context.Context) (snapshot.Source, error) {
	var src snapshot.Source

	keyRows, err := s.pool.Query(ctx, `
		SELECT key_value, project_id, organization_id, name FROM api_keys
	`)
	if err != nil {
		return src, err
	}
	defer keyRows.Close()
	for keyRows.Next() {
		var k snapshot.APIKey
		if err := keyRows.Scan(&k.Key, &k.ProjectID, &k.OrganizationID, &k.Name); err != nil {
			return src, err
		}
		src.APIKeys = append(src.APIKeys, k)
	}
	if err := keyRows.Err(); err != nil {
		return src, err
	}

	provRows, err := s.pool.Query(ctx, `
		SELECT id, type, base_url, api_key_env, name FROM providers
	`)
	if err != nil {
		return src, err
	}
	defer provRows.Close()
	for provRows.Next() {
		var p snapshot.Provider
		if err := provRows.Scan(&p.ID, &p.Type, &p.BaseURL, &p.APIKeyEnv, &p.Name); err != nil {
			return src, err
		}
		src.Providers = append(src.Providers, p)
	}
	if err := provRows.Err(); err != nil {
		return src, err
	}

	routeRows, err := s.pool.Query(ctx, `
		SELECT model, provider_id, target_model FROM routes
	`)
	if err != nil {
		return src, err
	}
	defer routeRows.Close()
	for routeRows.Next() {
		var r snapshot.Route
		if err := routeRows.Scan(&r.Model, &r.ProviderID, &r.TargetModel); err != nil {
			return src, err
		}
		src.Routes = append(src.Routes, r)
	}
	return src, routeRows.Err()
}
