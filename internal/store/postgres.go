package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/curefatih/afi/internal/domain"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) Open() (*PostgresStore, error) {
	// need to open the database connection here or in NewPostgresStore. leaving it as a placeholder for now.
	return NewPostgresStore(s.db), nil
}

func (s *PostgresStore) Close() error {
	return s.db.Close()
}

func (s *PostgresStore) Migrate(migrationsDir string) error {
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`); err != nil {
		return fmt.Errorf("ensure migrations table: %w", err)
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, name := range files {
		version := strings.TrimSuffix(name, ".sql")
		var exists bool
		if err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`, version).Scan(&exists); err != nil {
			return err
		}
		if exists {
			continue
		}

		sqlBytes, err := os.ReadFile(filepath.Join(migrationsDir, name))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		tx, err := s.db.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(string(sqlBytes)); err != nil {
			tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %s: %w", name, err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func (s *PostgresStore) ListOrganizations(ctx context.Context) ([]domain.Organization, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, budget_cents, created_at FROM organizations ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out = make([]domain.Organization, 0)
	for rows.Next() {
		var o domain.Organization
		if err := rows.Scan(&o.ID, &o.Name, &o.BudgetCents, &o.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func (s *PostgresStore) GetOrganization(ctx context.Context, id string) (*domain.Organization, error) {
	var o domain.Organization
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, budget_cents, created_at FROM organizations WHERE id = $1
	`, id).Scan(&o.ID, &o.Name, &o.BudgetCents, &o.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("organization not found")
	}
	return &o, err
}

func (s *PostgresStore) CreateOrganization(ctx context.Context, name string, budgetCents int64) (*domain.Organization, error) {
	var o domain.Organization
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO organizations (name, budget_cents) VALUES ($1, $2)
		RETURNING id, name, budget_cents, created_at
	`, name, budgetCents).Scan(&o.ID, &o.Name, &o.BudgetCents, &o.CreatedAt)
	return &o, err
}

func (s *PostgresStore) UpdateOrganization(ctx context.Context, id, name string, budgetCents int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE organizations SET name = $2, budget_cents = $3 WHERE id = $1
	`, id, name, budgetCents)
	return err
}

func (s *PostgresStore) DeleteOrganization(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM organizations WHERE id = $1
	`, id)
	return err
}

func (s *PostgresStore) ListTeams(ctx context.Context, orgID string) ([]domain.Team, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, org_id, name, budget_cents, created_at FROM teams WHERE org_id = $1 ORDER BY name
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out = make([]domain.Team, 0)
	for rows.Next() {
		var t domain.Team
		if err := rows.Scan(&t.ID, &t.OrgID, &t.Name, &t.BudgetCents, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *PostgresStore) GetTeam(ctx context.Context, id string) (*domain.Team, error) {
	var t domain.Team
	err := s.db.QueryRowContext(ctx, `
		SELECT id, org_id, name, budget_cents, created_at FROM teams WHERE id = $1
	`, id).Scan(&t.ID, &t.OrgID, &t.Name, &t.BudgetCents, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("team not found")
	}
	return &t, err
}

func (s *PostgresStore) CreateTeam(ctx context.Context, orgID, name string, budgetCents int64) (*domain.Team, error) {
	var t domain.Team
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO teams (org_id, name, budget_cents) VALUES ($1, $2, $3)
		RETURNING id, org_id, name, budget_cents, created_at
	`, orgID, name, budgetCents).Scan(&t.ID, &t.OrgID, &t.Name, &t.BudgetCents, &t.CreatedAt)
	return &t, err
}

func (s *PostgresStore) UpdateTeam(ctx context.Context, id, name string, budgetCents int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE teams SET name = $2, budget_cents = $3 WHERE id = $1
	`, id, name, budgetCents)
	return err
}

func (s *PostgresStore) DeleteTeam(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM teams WHERE id = $1
	`, id)
	return err
}

func (s *PostgresStore) ListProjects(ctx context.Context, teamID string) ([]domain.Project, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, team_id, org_id, name, created_at FROM projects WHERE team_id = $1 ORDER BY name
	`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out = make([]domain.Project, 0)
	for rows.Next() {
		var p domain.Project
		if err := rows.Scan(&p.ID, &p.TeamID, &p.OrgID, &p.Name, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *PostgresStore) GetProject(ctx context.Context, id string) (*domain.Project, error) {
	var p domain.Project
	err := s.db.QueryRowContext(ctx, `
		SELECT id, team_id, org_id, name, created_at FROM projects WHERE id = $1
	`, id).Scan(&p.ID, &p.TeamID, &p.OrgID, &p.Name, &p.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("project not found")
	}
	return &p, err
}

func (s *PostgresStore) CreateProject(ctx context.Context, teamID, name string) (*domain.Project, error) {
	var p domain.Project
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO projects (team_id, name) VALUES ($1, $2)
		RETURNING id, team_id, org_id, name, created_at
	`, teamID, name).Scan(&p.ID, &p.TeamID, &p.OrgID, &p.Name, &p.CreatedAt)
	return &p, err
}

func (s *PostgresStore) UpdateProject(ctx context.Context, id, name string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE projects SET name = $2 WHERE id = $1
	`, id, name)
	return err
}

func (s *PostgresStore) DeleteProject(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM projects WHERE id = $1
	`, id)
	return err
}

func (s *PostgresStore) ListUsers(ctx context.Context) ([]domain.User, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, email, oidc_sub, created_at FROM users ORDER BY email
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out = make([]domain.User, 0)
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Email, &u.OIDCSub, &u.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *PostgresStore) GetUser(ctx context.Context, id string) (*domain.User, error) {
	var u domain.User
	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, oidc_sub, created_at FROM users WHERE id = $1
	`, id).Scan(&u.ID, &u.Email, &u.OIDCSub, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	return &u, err
}

func (s *PostgresStore) CreateUser(ctx context.Context, email, oidcSub string) (*domain.User, error) {
	var u domain.User
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO users (email, oidc_sub) VALUES ($1, $2)
		RETURNING id, email, oidc_sub, created_at
	`, email, oidcSub).Scan(&u.ID, &u.Email, &u.OIDCSub, &u.CreatedAt)
	return &u, err
}

func (s *PostgresStore) UpdateUser(ctx context.Context, id, email, oidcSub string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET email = $2, oidc_sub = $3 WHERE id = $1
	`, id, email, oidcSub)
	return err
}

func (s *PostgresStore) DeleteUser(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM users WHERE id = $1
	`, id)
	return err
}

func (s *PostgresStore) ListProjectMembers(ctx context.Context, projectID string) ([]domain.ProjectMember, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT project_id, user_id, email, role FROM project_members WHERE project_id = $1 ORDER BY email
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out = make([]domain.ProjectMember, 0)
	for rows.Next() {
		var pm domain.ProjectMember
		if err := rows.Scan(&pm.ProjectID, &pm.UserID, &pm.Email, &pm.Role); err != nil {
			return nil, err
		}
		out = append(out, pm)
	}
	return out, rows.Err()
}

func (s *PostgresStore) AddProjectMember(ctx context.Context, projectID, userID, role string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO project_members (project_id, user_id, role) VALUES ($1, $2, $3)
	`, projectID, userID, role)
	return err
}

func (s *PostgresStore) RemoveProjectMember(ctx context.Context, projectID, userID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM project_members WHERE project_id = $1 AND user_id = $2
	`, projectID, userID)
	return err
}

func (s *PostgresStore) UpdateProjectMemberRole(ctx context.Context, projectID, userID, role string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE project_members SET role = $3 WHERE project_id = $1 AND user_id = $2
	`, projectID, userID, role)
	return err
}
