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
	KeyPrefix      string    `json:"key_prefix"`
	Key            string    `json:"key,omitempty"` // plaintext only on create
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

type UsageEvent struct {
	ID               int64     `json:"id"`
	OrganizationID   string    `json:"organization_id"`
	ProjectID        string    `json:"project_id"`
	APIKeyID         string    `json:"api_key_id"`
	Model            string    `json:"model"`
	Status           string    `json:"status"`
	LatencyMs        int64     `json:"latency_ms"`
	PromptTokens     int64     `json:"prompt_tokens"`
	CompletionTokens int64     `json:"completion_tokens"`
	CreatedAt        time.Time `json:"created_at"`
}

func (s *Store) CountOrgs(ctx context.Context) (int64, error) {
	var n int64
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM organizations`).Scan(&n)
	return n, err
}

func (s *Store) IsOrgMember(ctx context.Context, userID, orgID string) (bool, error) {
	var ok bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM organization_members
			WHERE user_id = $1 AND organization_id = $2
		)
	`, userID, orgID).Scan(&ok)
	return ok, err
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

func (s *Store) GetTeamOrgID(ctx context.Context, teamID string) (string, error) {
	var orgID string
	err := s.pool.QueryRow(ctx, `SELECT organization_id FROM teams WHERE id = $1`, teamID).Scan(&orgID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", kernel.ErrNotFound
	}
	return orgID, err
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
		SELECT id, project_id, organization_id, name, key_prefix, created_at
		FROM api_keys WHERE project_id = $1 ORDER BY created_at
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []APIKey
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.ID, &k.ProjectID, &k.OrganizationID, &k.Name, &k.KeyPrefix, &k.CreatedAt); err != nil {
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

func (s *Store) CreateAPIKey(ctx context.Context, orgID, projectID, name, rawKey string) (*APIKey, error) {
	k := &APIKey{
		ID:             newID("key"),
		ProjectID:      projectID,
		OrganizationID: orgID,
		Name:           name,
		KeyPrefix:      KeyPrefix(rawKey),
		Key:            rawKey,
		CreatedAt:      time.Now().UTC(),
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO api_keys (id, project_id, organization_id, name, key_hash, key_prefix, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, k.ID, k.ProjectID, k.OrganizationID, k.Name, HashAPIKey(rawKey), k.KeyPrefix, k.CreatedAt)
	if err != nil {
		return nil, err
	}
	return k, nil
}

func (s *Store) ListProviders(ctx context.Context, orgID string) ([]Provider, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, organization_id, name, type, base_url, api_key_env, created_at
		FROM providers WHERE organization_id = $1 ORDER BY created_at
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Provider
	for rows.Next() {
		var p Provider
		if err := rows.Scan(&p.ID, &p.OrganizationID, &p.Name, &p.Type, &p.BaseURL, &p.APIKeyEnv, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) CreateProvider(ctx context.Context, orgID, name, typ, baseURL, apiKeyEnv string) (*Provider, error) {
	p := &Provider{
		ID: newID("prov"), OrganizationID: orgID, Name: name, Type: typ,
		BaseURL: baseURL, APIKeyEnv: apiKeyEnv, CreatedAt: time.Now().UTC(),
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO providers (id, organization_id, name, type, base_url, api_key_env, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
	`, p.ID, p.OrganizationID, p.Name, p.Type, p.BaseURL, p.APIKeyEnv, p.CreatedAt)
	return p, err
}

func (s *Store) UpdateProvider(ctx context.Context, providerID, name, baseURL, apiKeyEnv string) (*Provider, error) {
	p := &Provider{}
	err := s.pool.QueryRow(ctx, `
		UPDATE providers SET name=$2, base_url=$3, api_key_env=$4
		WHERE id=$1
		RETURNING id, organization_id, name, type, base_url, api_key_env, created_at
	`, providerID, name, baseURL, apiKeyEnv).Scan(
		&p.ID, &p.OrganizationID, &p.Name, &p.Type, &p.BaseURL, &p.APIKeyEnv, &p.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	return p, err
}

func (s *Store) DeleteProvider(ctx context.Context, providerID string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM providers WHERE id=$1`, providerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return kernel.ErrNotFound
	}
	return nil
}

func (s *Store) GetProviderOrgID(ctx context.Context, providerID string) (string, error) {
	var orgID string
	err := s.pool.QueryRow(ctx, `SELECT organization_id FROM providers WHERE id=$1`, providerID).Scan(&orgID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", kernel.ErrNotFound
	}
	return orgID, err
}

func (s *Store) ListRoutes(ctx context.Context, orgID string) ([]Route, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, organization_id, model, provider_id, target_model, created_at
		FROM routes WHERE organization_id=$1 ORDER BY created_at
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Route
	for rows.Next() {
		var r Route
		if err := rows.Scan(&r.ID, &r.OrganizationID, &r.Model, &r.ProviderID, &r.TargetModel, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) CreateRoute(ctx context.Context, orgID, model, providerID, targetModel string) (*Route, error) {
	r := &Route{
		ID: newID("route"), OrganizationID: orgID, Model: model,
		ProviderID: providerID, TargetModel: targetModel, CreatedAt: time.Now().UTC(),
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO routes (id, organization_id, model, provider_id, target_model, created_at)
		VALUES ($1,$2,$3,$4,$5,$6)
	`, r.ID, r.OrganizationID, r.Model, r.ProviderID, r.TargetModel, r.CreatedAt)
	return r, err
}

func (s *Store) UpdateRoute(ctx context.Context, routeID, model, providerID, targetModel string) (*Route, error) {
	r := &Route{}
	err := s.pool.QueryRow(ctx, `
		UPDATE routes SET model=$2, provider_id=$3, target_model=$4
		WHERE id=$1
		RETURNING id, organization_id, model, provider_id, target_model, created_at
	`, routeID, model, providerID, targetModel).Scan(
		&r.ID, &r.OrganizationID, &r.Model, &r.ProviderID, &r.TargetModel, &r.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	return r, err
}

func (s *Store) DeleteRoute(ctx context.Context, routeID string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM routes WHERE id=$1`, routeID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return kernel.ErrNotFound
	}
	return nil
}

func (s *Store) GetRouteOrgID(ctx context.Context, routeID string) (string, error) {
	var orgID string
	err := s.pool.QueryRow(ctx, `SELECT organization_id FROM routes WHERE id=$1`, routeID).Scan(&orgID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", kernel.ErrNotFound
	}
	return orgID, err
}

func (s *Store) InsertUsage(ctx context.Context, e UsageEvent) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO usage_events (
			organization_id, project_id, api_key_id, model, status,
			latency_ms, prompt_tokens, completion_tokens
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`, e.OrganizationID, e.ProjectID, e.APIKeyID, e.Model, e.Status,
		e.LatencyMs, e.PromptTokens, e.CompletionTokens)
	return err
}

func (s *Store) ListUsage(ctx context.Context, orgID string, limit int) ([]UsageEvent, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, organization_id, project_id, api_key_id, model, status,
			latency_ms, prompt_tokens, completion_tokens, created_at
		FROM usage_events WHERE organization_id=$1
		ORDER BY created_at DESC LIMIT $2
	`, orgID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UsageEvent
	for rows.Next() {
		var e UsageEvent
		if err := rows.Scan(
			&e.ID, &e.OrganizationID, &e.ProjectID, &e.APIKeyID, &e.Model, &e.Status,
			&e.LatencyMs, &e.PromptTokens, &e.CompletionTokens, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

type Quota struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	ScopeType      string    `json:"scope_type"`
	ScopeID        string    `json:"scope_id"`
	Metric         string    `json:"metric"`
	LimitValue     int64     `json:"limit_value"`
	Window         string    `json:"window"`
	CreatedAt      time.Time `json:"created_at"`
}

func (s *Store) ListQuotas(ctx context.Context, orgID string) ([]Quota, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, organization_id, scope_type, scope_id, metric, limit_value, time_window, created_at
		FROM quotas WHERE organization_id=$1 ORDER BY created_at
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Quota
	for rows.Next() {
		var q Quota
		if err := rows.Scan(&q.ID, &q.OrganizationID, &q.ScopeType, &q.ScopeID, &q.Metric, &q.LimitValue, &q.Window, &q.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, rows.Err()
}

func (s *Store) CreateQuota(ctx context.Context, orgID, scopeType, scopeID, metric string, limitValue int64, window string) (*Quota, error) {
	if window == "" {
		window = snapshot.WindowTotal
	}
	q := &Quota{
		ID: newID("quota"), OrganizationID: orgID, ScopeType: scopeType, ScopeID: scopeID,
		Metric: metric, LimitValue: limitValue, Window: window, CreatedAt: time.Now().UTC(),
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO quotas (id, organization_id, scope_type, scope_id, metric, limit_value, time_window, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`, q.ID, q.OrganizationID, q.ScopeType, q.ScopeID, q.Metric, q.LimitValue, q.Window, q.CreatedAt)
	return q, err
}

func (s *Store) UpdateQuota(ctx context.Context, quotaID string, limitValue int64) (*Quota, error) {
	q := &Quota{}
	err := s.pool.QueryRow(ctx, `
		UPDATE quotas SET limit_value=$2 WHERE id=$1
		RETURNING id, organization_id, scope_type, scope_id, metric, limit_value, time_window, created_at
	`, quotaID, limitValue).Scan(
		&q.ID, &q.OrganizationID, &q.ScopeType, &q.ScopeID, &q.Metric, &q.LimitValue, &q.Window, &q.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	return q, err
}

func (s *Store) DeleteQuota(ctx context.Context, quotaID string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM quotas WHERE id=$1`, quotaID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return kernel.ErrNotFound
	}
	return nil
}

func (s *Store) GetQuotaOrgID(ctx context.Context, quotaID string) (string, error) {
	var orgID string
	err := s.pool.QueryRow(ctx, `SELECT organization_id FROM quotas WHERE id=$1`, quotaID).Scan(&orgID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", kernel.ErrNotFound
	}
	return orgID, err
}

func (s *Store) GetCounter(ctx context.Context, scopeType, scopeID, metric, window string) (int64, error) {
	var used int64
	err := s.pool.QueryRow(ctx, `
		SELECT used FROM quota_counters
		WHERE scope_type=$1 AND scope_id=$2 AND metric=$3 AND time_window=$4
	`, scopeType, scopeID, metric, window).Scan(&used)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return used, err
}

func (s *Store) IncrCounter(ctx context.Context, scopeType, scopeID, metric, window string, delta int64) (int64, error) {
	var used int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO quota_counters (scope_type, scope_id, metric, time_window, used, updated_at)
		VALUES ($1,$2,$3,$4,$5,NOW())
		ON CONFLICT (scope_type, scope_id, metric, time_window)
		DO UPDATE SET used = quota_counters.used + EXCLUDED.used, updated_at = NOW()
		RETURNING used
	`, scopeType, scopeID, metric, window, delta).Scan(&used)
	return used, err
}

func (s *Store) EnqueueUsage(ctx context.Context, payload []byte) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO usage_outbox (payload) VALUES ($1)`, payload)
	return err
}

func (s *Store) LoadSnapshotSource(ctx context.Context) (snapshot.Source, error) {
	var src snapshot.Source

	keyRows, err := s.pool.Query(ctx, `
		SELECT id, key_hash, key_prefix, project_id, organization_id, name FROM api_keys
	`)
	if err != nil {
		return src, err
	}
	defer keyRows.Close()
	for keyRows.Next() {
		var k snapshot.APIKey
		if err := keyRows.Scan(&k.ID, &k.KeyHash, &k.KeyPrefix, &k.ProjectID, &k.OrganizationID, &k.Name); err != nil {
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
		SELECT organization_id, model, provider_id, target_model FROM routes
	`)
	if err != nil {
		return src, err
	}
	defer routeRows.Close()
	for routeRows.Next() {
		var r snapshot.Route
		if err := routeRows.Scan(&r.OrganizationID, &r.Model, &r.ProviderID, &r.TargetModel); err != nil {
			return src, err
		}
		src.Routes = append(src.Routes, r)
	}
	if err := routeRows.Err(); err != nil {
		return src, err
	}

	quotaRows, err := s.pool.Query(ctx, `
		SELECT id, organization_id, scope_type, scope_id, metric, limit_value, time_window FROM quotas
	`)
	if err != nil {
		return src, err
	}
	defer quotaRows.Close()
	for quotaRows.Next() {
		var q snapshot.Quota
		if err := quotaRows.Scan(&q.ID, &q.OrganizationID, &q.ScopeType, &q.ScopeID, &q.Metric, &q.LimitValue, &q.Window); err != nil {
			return src, err
		}
		src.Quotas = append(src.Quotas, q)
	}
	return src, quotaRows.Err()
}
