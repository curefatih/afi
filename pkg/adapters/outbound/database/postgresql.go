package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/curefatih/afi/internal/core/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

// =========================================================================
// 👤 PlatformUserRepository Implementation
// =========================================================================

func (s *PostgresStore) SaveUser(ctx context.Context, u *domain.PlatformUser) error {
	query := `INSERT INTO platform_users (id, email, password_hash, provider, external_id, created_at) 
	          VALUES ($1, $2, $3, $4, $5, $6)
	          ON CONFLICT (id) DO UPDATE SET email = $2, password_hash = $3`
	_, err := s.pool.Exec(ctx, query, u.ID, u.Email, u.PasswordHash, u.Provider, u.ExternalID, u.CreatedAt)
	return err
}

func (s *PostgresStore) GetUserByEmail(ctx context.Context, email string) (*domain.PlatformUser, error) {
	query := `SELECT id, email, password_hash, provider, external_id, created_at FROM platform_users WHERE email = $1`
	var u domain.PlatformUser
	err := s.pool.QueryRow(ctx, query, email).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Provider, &u.ExternalID, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("user not found: %s", email)
	}
	return &u, err
}

func (s *PostgresStore) SaveCustomRole(ctx context.Context, r *domain.CustomRole) error {
	query := `INSERT INTO custom_roles (id, name, scope, target_id, permissions) VALUES ($1, $2, $3, $4, $5)`
	_, err := s.pool.Exec(ctx, query, r.ID, r.Name, r.Scope, r.TargetID, r.Permissions)
	return err
}

func (s *PostgresStore) SaveRoleAssignment(ctx context.Context, a *domain.UserAssignment) error {
	query := `INSERT INTO user_assignments (user_id, org_id, project_id, role_name) VALUES ($1, $2, $3, $4)
	          ON CONFLICT (user_id, org_id, COALESCE(project_id, 'GLOBAL_ORG_SCOPE')) DO UPDATE SET role_name = $4`
	_, err := s.pool.Exec(ctx, query, a.UserID, a.OrgID, a.ProjectID, a.RoleName)
	return err
}

func (s *PostgresStore) GetUserPermissions(ctx context.Context, userID, orgID, projectID string) ([]domain.ActionPermission, error) {
	// Evaluates standard system fallback rules: checks both specific project scope assignments and organization wide assignments.
	query := `
		SELECT DISTINCT unnest(c.permissions)
		FROM user_assignments ua
		JOIN custom_roles c ON ua.role_name = c.id OR ua.role_name = c.name
		WHERE ua.user_id = $1 AND ua.org_id = $2 AND (ua.project_id = $3 OR ua.project_id IS NULL)`

	rows, err := s.pool.Query(ctx, query, userID, orgID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []domain.ActionPermission
	for rows.Next() {
		var p domain.ActionPermission
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	return perms, nil
}

// =========================================================================
// 🔑 AuthRepository Implementation (Data Plane Fast Path)
// =========================================================================

func (s *PostgresStore) SaveAPIKey(ctx context.Context, key *domain.APIKey) error {
	query := `INSERT INTO api_keys (key_hash, key_type, target_id, is_active, created_at) VALUES ($1, $2, $3, $4, $5)`
	_, err := s.pool.Exec(ctx, query, key.HashedKey, key.Type, key.UserID, key.IsActive, time.Now())
	return err
}

func (s *PostgresStore) GetContextByKeyHash(ctx context.Context, hash string) (*domain.RequestContext, error) {
	query := `
		SELECT k.target_id, p.team_id, t.org_id
		FROM api_keys k
		LEFT JOIN projects p ON k.target_id = p.id
		LEFT JOIN teams t ON p.team_id = t.id
		WHERE k.key_hash = $1 AND k.is_active = TRUE`

	var ctxMeta domain.RequestContext
	err := s.pool.QueryRow(ctx, query, hash).Scan(&ctxMeta.ProjectID, &ctxMeta.TeamID, &ctxMeta.OrganizationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("invalid or inactive api gateway credential key")
	}
	return &ctxMeta, err
}

// =========================================================================
// 💰 BudgetRepository Implementation
// =========================================================================

func (s *PostgresStore) GetLimit(ctx context.Context, scope domain.BudgetScope, targetID string) (*domain.BudgetLimit, error) {
	query := `SELECT target_id, scope, max_limit, current_usage FROM budgets WHERE target_id = $1 AND scope = $2`
	var b domain.BudgetLimit
	err := s.pool.QueryRow(ctx, query, targetID, scope).Scan(&b.TargetID, &b.Scope, &b.MaxCost, &b.UsedCost)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // Return empty if no budget barrier constraints configured
	}
	return &b, err
}

func (s *PostgresStore) IncrementUsage(ctx context.Context, scope domain.BudgetScope, targetID string, amount float64) error {
	query := `UPDATE budgets SET current_usage = current_usage + $1, updated_at = CURRENT_TIMESTAMP 
	          WHERE target_id = $2 AND scope = $3`
	_, err := s.pool.Exec(ctx, query, amount, targetID, scope)
	return err
}

// =========================================================================
// 🪝 PluginRepository Implementation
// =========================================================================

func (s *PostgresStore) SavePlugin(ctx context.Context, p *domain.CustomPlugin) error {
	query := `INSERT INTO plugins (project_id, stage, script, is_active, updated_at) VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
	          ON CONFLICT (project_id, stage) DO UPDATE SET script = $3, is_active = $4, updated_at = CURRENT_TIMESTAMP`
	_, err := s.pool.Exec(ctx, query, p.ProjectID, p.Stage, p.Script, p.IsActive)
	return err
}

func (s *PostgresStore) GetActivePlugin(ctx context.Context, projectID string, stage domain.HookStage) (*domain.CustomPlugin, error) {
	query := `SELECT project_id, stage, script, is_active FROM plugins WHERE project_id = $1 AND stage = $2 AND is_active = TRUE`
	var p domain.CustomPlugin
	err := s.pool.QueryRow(ctx, query, projectID, stage).Scan(&p.ProjectID, &p.Stage, &p.Script, &p.IsActive)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &p, err
}

// =========================================================================
// 🏢 TenantRepository Implementation
// =========================================================================

func (s *PostgresStore) SaveOrganization(ctx context.Context, org *domain.Organization) error {
	query := `INSERT INTO organizations (id, name, created_at) VALUES ($1, $2, $3) ON CONFLICT (id) DO UPDATE SET name = $2`
	_, err := s.pool.Exec(ctx, query, org.ID, org.Name, org.CreatedAt)
	return err
}

func (s *PostgresStore) SaveTeam(ctx context.Context, team *domain.Team) error {
	query := `INSERT INTO teams (id, org_id, name, created_at) VALUES ($1, $2, $3, $4) ON CONFLICT (id) DO UPDATE SET name = $3`
	_, err := s.pool.Exec(ctx, query, team.ID, team.OrgID, team.Name, team.CreatedAt)
	return err
}

func (s *PostgresStore) SaveProject(ctx context.Context, p *domain.Project) error {
	query := `INSERT INTO projects (id, team_id, name, created_at) VALUES ($1, $2, $3, $4) ON CONFLICT (id) DO UPDATE SET name = $3`
	_, err := s.pool.Exec(ctx, query, p.ID, p.TeamID, p.Name, p.CreatedAt)
	return err
}

func (s *PostgresStore) SaveMembership(ctx context.Context, m *domain.Membership) error {
	// If membership translates straight to user assignments, reuse role logic map matrices
	query := `INSERT INTO user_assignments (user_id, org_id, project_id, role_name) VALUES ($1, $2, $3, $4)
	          ON CONFLICT (user_id, org_id, COALESCE(project_id, 'GLOBAL_ORG_SCOPE')) DO UPDATE SET role_name = $4`
	_, err := s.pool.Exec(ctx, query, m.UserID, m.OrganizationID, m.TargetID, m.Role)
	return err
}

func (s *PostgresStore) GetProviderKey(ctx context.Context, projectID string, provider string) (string, error) {
	query := `SELECT key FROM provider_keys WHERE project_id = $1 AND provider = $2`
	var key string
	err := s.pool.QueryRow(ctx, query, projectID, provider).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("provider key not found: %s", provider)
	}
	return key, err
}

// RouterService implementation
func (s *PostgresStore) Route(req *domain.InternalRequest) (domain.TargetDestination, error) {
	return domain.TargetDestination{}, errors.New("not implemented")
}

func (a *PostgresStore) AddRule(ctx context.Context, rule domain.RoutingRule) error {
	return errors.New("not implemented")
}
