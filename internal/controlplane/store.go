package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/adapters/postgres"
	"github.com/curefatih/afi/internal/gatewayconfig"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/snapshot"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	OrgRoleOwner  = "owner"
	OrgRoleAdmin  = "admin"
	OrgRoleMember = "member"
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
	ProjectID      string    `json:"project_id,omitempty"`
	OrganizationID string    `json:"organization_id"`
	Name           string    `json:"name"`
	Kind           string    `json:"kind"`
	OwnerUserID    string    `json:"owner_user_id,omitempty"`
	KeyPrefix      string    `json:"key_prefix"`
	Key            string    `json:"key,omitempty"` // plaintext only on create
	CreatedAt      time.Time `json:"created_at"`
}

type Provider struct {
	ID             string                        `json:"id"`
	OrganizationID string                        `json:"organization_id"`
	Name           string                        `json:"name"`
	Type           string                        `json:"type"`
	BaseURL        string                        `json:"base_url"`
	APIKeyEnv      string                        `json:"api_key_env"`
	Capabilities   snapshot.ProviderCapabilities `json:"capabilities"`
	CreatedAt      time.Time                     `json:"created_at"`
}

type RouteFallback struct {
	ProviderID  string `json:"provider_id"`
	TargetModel string `json:"target_model"`
}

type Route struct {
	ID             string          `json:"id"`
	OrganizationID string          `json:"organization_id"`
	Model          string          `json:"model"`
	ProviderID     string          `json:"provider_id"`
	TargetModel    string          `json:"target_model"`
	Fallbacks      []RouteFallback `json:"fallbacks"`
	CreatedAt      time.Time       `json:"created_at"`
}

type UsageEvent struct {
	ID               int64          `json:"id"`
	OrganizationID   string         `json:"organization_id"`
	ProjectID        string         `json:"project_id"`
	APIKeyID         string         `json:"api_key_id"`
	Model            string         `json:"model"`
	Status           string         `json:"status"`
	LatencyMs        int64          `json:"latency_ms"`
	PromptTokens     int64          `json:"prompt_tokens"`
	CompletionTokens int64          `json:"completion_tokens"`
	Modality         string         `json:"modality"`
	Metrics          map[string]any `json:"metrics"`
	CostUSD          *float64       `json:"cost_usd,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	KeyName          string         `json:"key_name,omitempty"`
	KeyKind          string         `json:"key_kind,omitempty"`
	OwnerUserID      string         `json:"owner_user_id,omitempty"`
	OwnerEmail       string         `json:"owner_email,omitempty"`
	OwnerName        string         `json:"owner_name,omitempty"`
}

type UsageFilter struct {
	Limit     int
	ProjectID string
	APIKeyID  string
	Model     string
	Modality  string
	From      *time.Time
	To        *time.Time
	GroupBy   string // day | model | key | modality (summary only)
}

type UsageSummaryBucket struct {
	Bucket           string             `json:"bucket"`
	Label            string             `json:"label"`
	Requests         int64              `json:"requests"`
	CostUSD          float64            `json:"cost_usd"`
	PromptTokens     int64              `json:"prompt_tokens"`
	CompletionTokens int64              `json:"completion_tokens"`
	MetricsTotals    map[string]float64 `json:"metrics_totals,omitempty"`
	KeyKind          string             `json:"key_kind,omitempty"`
	OwnerEmail       string             `json:"owner_email,omitempty"`
	OwnerName        string             `json:"owner_name,omitempty"`
}

type ModelPrice struct {
	ProviderType  string
	Model         string
	InputPerMTok  float64
	OutputPerMTok float64
}

type ProviderHealth struct {
	ProviderID   string  `json:"provider_id"`
	Name         string  `json:"name"`
	Type         string  `json:"type"`
	Requests     int64   `json:"requests"`
	Errors       int64   `json:"errors"`
	ErrorRate    float64 `json:"error_rate"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
	Status       string  `json:"status"` // healthy | degraded | down | unknown
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

type OrgMember struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	Role   string `json:"role"`
}

func (s *Store) CreateOrganization(ctx context.Context, name, creatorUserID string) (*Organization, error) {
	o := &Organization{
		ID: newID("org"), Name: name, CreatedAt: time.Now().UTC(),
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO organizations (id, name, created_at) VALUES ($1,$2,$3)
	`, o.ID, o.Name, o.CreatedAt)
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO organization_members (organization_id, user_id, role) VALUES ($1,$2,$3)
	`, o.ID, creatorUserID, OrgRoleOwner)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return o, nil
}

func (s *Store) ListOrgMembers(ctx context.Context, orgID string) ([]OrgMember, error) {
	rows, err := s.pool.Query(ctx, `
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
	var out []OrgMember
	for rows.Next() {
		var m OrgMember
		if err := rows.Scan(&m.UserID, &m.Email, &m.Name, &m.Role); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) AddOrgMemberByEmail(ctx context.Context, orgID, email string) (*OrgMember, error) {
	user, err := s.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO organization_members (organization_id, user_id, role) VALUES ($1,$2,$3)
		ON CONFLICT DO NOTHING
	`, orgID, user.ID, OrgRoleMember)
	if err != nil {
		return nil, err
	}
	return &OrgMember{
		UserID: user.ID, Email: user.Email, Name: user.Name, Role: OrgRoleMember,
	}, nil
}

func (s *Store) GetOrgMemberRole(ctx context.Context, userID, orgID string) (string, error) {
	var role string
	err := s.pool.QueryRow(ctx, `
		SELECT role FROM organization_members WHERE user_id=$1 AND organization_id=$2
	`, userID, orgID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", kernel.ErrNotFound
	}
	return role, err
}

func (s *Store) IsOrgAdmin(ctx context.Context, userID, orgID string) (bool, error) {
	role, err := s.GetOrgMemberRole(ctx, userID, orgID)
	if errors.Is(err, kernel.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return role == OrgRoleOwner || role == OrgRoleAdmin, nil
}

// UpdateOrgMemberRole changes a member's role. Only the org owner may call this.
// Promoting to owner transfers ownership (actor becomes admin). Cannot demote the sole owner.
func (s *Store) UpdateOrgMemberRole(ctx context.Context, orgID, actorUserID, targetUserID, role string) (*OrgMember, error) {
	switch role {
	case OrgRoleOwner, OrgRoleAdmin, OrgRoleMember:
	default:
		return nil, fmt.Errorf("%w: role must be owner, admin, or member", kernel.ErrInvalidRequest)
	}
	actorRole, err := s.GetOrgMemberRole(ctx, actorUserID, orgID)
	if err != nil {
		return nil, err
	}
	if actorRole != OrgRoleOwner {
		return nil, kernel.ErrUnauthorized
	}
	targetRole, err := s.GetOrgMemberRole(ctx, targetUserID, orgID)
	if err != nil {
		return nil, err
	}
	if targetUserID == actorUserID && role != OrgRoleOwner {
		var owners int
		if err := s.pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM organization_members
			WHERE organization_id=$1 AND role=$2
		`, orgID, OrgRoleOwner).Scan(&owners); err != nil {
			return nil, err
		}
		if owners <= 1 {
			return nil, fmt.Errorf("%w: cannot demote the sole owner", kernel.ErrInvalidRequest)
		}
	}
	if targetRole == OrgRoleOwner && role != OrgRoleOwner {
		var owners int
		if err := s.pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM organization_members
			WHERE organization_id=$1 AND role=$2
		`, orgID, OrgRoleOwner).Scan(&owners); err != nil {
			return nil, err
		}
		if owners <= 1 && targetUserID != actorUserID {
			return nil, fmt.Errorf("%w: cannot demote the sole owner", kernel.ErrInvalidRequest)
		}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if role == OrgRoleOwner && targetUserID != actorUserID {
		if _, err := tx.Exec(ctx, `
			UPDATE organization_members SET role=$1 WHERE organization_id=$2 AND user_id=$3
		`, OrgRoleAdmin, orgID, actorUserID); err != nil {
			return nil, err
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE organization_members SET role=$1 WHERE organization_id=$2 AND user_id=$3
	`, role, orgID, targetUserID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	var m OrgMember
	err = s.pool.QueryRow(ctx, `
		SELECT u.id, u.email, u.name, m.role
		FROM organization_members m
		JOIN users u ON u.id = m.user_id
		WHERE m.organization_id=$1 AND m.user_id=$2
	`, orgID, targetUserID).Scan(&m.UserID, &m.Email, &m.Name, &m.Role)
	if err != nil {
		return nil, err
	}
	return &m, nil
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

func scanAPIKey(scan func(dest ...any) error) (APIKey, error) {
	var k APIKey
	var projectID, ownerUserID *string
	err := scan(&k.ID, &projectID, &k.OrganizationID, &k.Name, &k.Kind, &ownerUserID, &k.KeyPrefix, &k.CreatedAt)
	if err != nil {
		return k, err
	}
	if projectID != nil {
		k.ProjectID = *projectID
	}
	if ownerUserID != nil {
		k.OwnerUserID = *ownerUserID
	}
	return k, nil
}

func (s *Store) ListAPIKeys(ctx context.Context, projectID string) ([]APIKey, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, project_id, organization_id, name, kind, owner_user_id, key_prefix, created_at
		FROM api_keys WHERE project_id = $1 ORDER BY created_at
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []APIKey
	for rows.Next() {
		k, err := scanAPIKey(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (s *Store) ListOrgAPIKeys(ctx context.Context, orgID string) ([]APIKey, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, project_id, organization_id, name, kind, owner_user_id, key_prefix, created_at
		FROM api_keys WHERE organization_id = $1 ORDER BY created_at
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []APIKey
	for rows.Next() {
		k, err := scanAPIKey(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (s *Store) GetAPIKey(ctx context.Context, keyID string) (*APIKey, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, project_id, organization_id, name, kind, owner_user_id, key_prefix, created_at
		FROM api_keys WHERE id = $1
	`, keyID)
	k, err := scanAPIKey(row.Scan)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &k, nil
}

func (s *Store) GetAPIKeyOrgID(ctx context.Context, keyID string) (string, error) {
	var orgID string
	err := s.pool.QueryRow(ctx, `SELECT organization_id FROM api_keys WHERE id = $1`, keyID).Scan(&orgID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", kernel.ErrNotFound
	}
	return orgID, err
}

func (s *Store) DeleteAPIKey(ctx context.Context, keyID string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM api_keys WHERE id=$1`, keyID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return kernel.ErrNotFound
	}
	return nil
}

func (s *Store) GetProjectOrgID(ctx context.Context, projectID string) (string, error) {
	var orgID string
	err := s.pool.QueryRow(ctx, `SELECT organization_id FROM projects WHERE id = $1`, projectID).Scan(&orgID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", kernel.ErrNotFound
	}
	return orgID, err
}

// CreateAPIKey inserts a key. kind must be personal or service_account.
// Personal: ownerUserID required, projectID must be empty.
// Service account: ownerUserID empty, projectID optional (empty = org-wide).
func (s *Store) CreateAPIKey(ctx context.Context, orgID, kind, ownerUserID, projectID, name, rawKey string) (*APIKey, error) {
	if kind == "" {
		kind = snapshot.KeyKindServiceAccount
	}
	switch kind {
	case snapshot.KeyKindPersonal:
		if ownerUserID == "" {
			return nil, fmt.Errorf("personal keys require owner")
		}
		if projectID != "" {
			return nil, fmt.Errorf("personal keys cannot have a project")
		}
	case snapshot.KeyKindServiceAccount:
		if ownerUserID != "" {
			return nil, fmt.Errorf("service account keys cannot have an owner")
		}
	default:
		return nil, fmt.Errorf("invalid key kind %q", kind)
	}
	if projectID != "" {
		projOrg, err := s.GetProjectOrgID(ctx, projectID)
		if err != nil {
			return nil, err
		}
		if projOrg != orgID {
			return nil, fmt.Errorf("project not in organization")
		}
	}

	k := &APIKey{
		ID:             newID("key"),
		ProjectID:      projectID,
		OrganizationID: orgID,
		Name:           name,
		Kind:           kind,
		OwnerUserID:    ownerUserID,
		KeyPrefix:      KeyPrefix(rawKey),
		Key:            rawKey,
		CreatedAt:      time.Now().UTC(),
	}
	var project any
	if projectID != "" {
		project = projectID
	}
	var owner any
	if ownerUserID != "" {
		owner = ownerUserID
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO api_keys (id, project_id, organization_id, name, kind, owner_user_id, key_hash, key_prefix, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, k.ID, project, k.OrganizationID, k.Name, k.Kind, owner, HashAPIKey(rawKey), k.KeyPrefix, k.CreatedAt)
	if err != nil {
		return nil, err
	}
	return k, nil
}

func (s *Store) ListProviders(ctx context.Context, orgID string) ([]Provider, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, organization_id, name, type, base_url, api_key_env, capabilities, created_at
		FROM providers WHERE organization_id = $1 ORDER BY created_at
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Provider
	for rows.Next() {
		var p Provider
		var caps []byte
		if err := rows.Scan(&p.ID, &p.OrganizationID, &p.Name, &p.Type, &p.BaseURL, &p.APIKeyEnv, &caps, &p.CreatedAt); err != nil {
			return nil, err
		}
		p.Capabilities = decodeCapabilities(p.Type, caps)
		out = append(out, p)
	}
	return out, rows.Err()
}

func classifyProviderHealth(requests, errors int64, errorRate float64) string {
	if requests == 0 {
		return "unknown"
	}
	if errorRate >= 0.9 || (requests >= 3 && errors == requests) {
		return "down"
	}
	if errorRate >= 0.2 {
		return "degraded"
	}
	return "healthy"
}

// ListProviderHealth aggregates usage_events for models routed to each org provider.
func (s *Store) ListProviderHealth(ctx context.Context, orgID string, from, to time.Time) ([]ProviderHealth, error) {
	if from.IsZero() {
		from = time.Now().UTC().Add(-24 * time.Hour)
	}
	if to.IsZero() {
		to = time.Now().UTC().Add(time.Hour)
	}
	rows, err := s.pool.Query(ctx, `
		SELECT p.id, p.name, p.type,
			COUNT(e.id)::bigint AS requests,
			COUNT(e.id) FILTER (WHERE e.status IS NOT NULL AND e.status <> 'ok')::bigint AS errors,
			COALESCE(AVG(e.latency_ms), 0)::double precision AS avg_latency_ms
		FROM providers p
		LEFT JOIN routes r
			ON r.provider_id = p.id AND r.organization_id = p.organization_id
		LEFT JOIN usage_events e
			ON e.organization_id = p.organization_id
			AND e.model = r.model
			AND e.created_at >= $2 AND e.created_at < $3
		WHERE p.organization_id = $1
		GROUP BY p.id, p.name, p.type
		ORDER BY p.name
	`, orgID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProviderHealth
	for rows.Next() {
		var h ProviderHealth
		if err := rows.Scan(&h.ProviderID, &h.Name, &h.Type, &h.Requests, &h.Errors, &h.AvgLatencyMs); err != nil {
			return nil, err
		}
		if h.Requests > 0 {
			h.ErrorRate = float64(h.Errors) / float64(h.Requests)
		}
		h.Status = classifyProviderHealth(h.Requests, h.Errors, h.ErrorRate)
		out = append(out, h)
	}
	return out, rows.Err()
}

func (s *Store) CreateProvider(ctx context.Context, orgID, name, typ, baseURL, apiKeyEnv string, caps snapshot.ProviderCapabilities) (*Provider, error) {
	caps = snapshot.NormalizeCapabilities(typ, caps)
	p := &Provider{
		ID: newID("prov"), OrganizationID: orgID, Name: name, Type: typ,
		BaseURL: baseURL, APIKeyEnv: apiKeyEnv, Capabilities: caps, CreatedAt: time.Now().UTC(),
	}
	raw, err := json.Marshal(caps)
	if err != nil {
		return nil, err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO providers (id, organization_id, name, type, base_url, api_key_env, capabilities, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`, p.ID, p.OrganizationID, p.Name, p.Type, p.BaseURL, p.APIKeyEnv, raw, p.CreatedAt)
	return p, err
}

func decodeCapabilities(typ string, raw []byte) snapshot.ProviderCapabilities {
	var c snapshot.ProviderCapabilities
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &c)
	}
	return snapshot.NormalizeCapabilities(typ, c)
}

func (s *Store) UpdateProvider(ctx context.Context, providerID, name, baseURL, apiKeyEnv string) (*Provider, error) {
	p := &Provider{}
	var caps []byte
	err := s.pool.QueryRow(ctx, `
		UPDATE providers SET name=$2, base_url=$3, api_key_env=$4
		WHERE id=$1
		RETURNING id, organization_id, name, type, base_url, api_key_env, capabilities, created_at
	`, providerID, name, baseURL, apiKeyEnv).Scan(
		&p.ID, &p.OrganizationID, &p.Name, &p.Type, &p.BaseURL, &p.APIKeyEnv, &caps, &p.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	p.Capabilities = decodeCapabilities(p.Type, caps)
	return p, nil
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
		SELECT id, organization_id, model, provider_id, target_model, fallbacks, created_at
		FROM routes WHERE organization_id=$1 ORDER BY created_at
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Route
	for rows.Next() {
		var r Route
		var fb []byte
		if err := rows.Scan(&r.ID, &r.OrganizationID, &r.Model, &r.ProviderID, &r.TargetModel, &fb, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.Fallbacks = decodeFallbacks(fb)
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) CreateRoute(ctx context.Context, orgID, model, providerID, targetModel string, fallbacks []RouteFallback) (*Route, error) {
	if fallbacks == nil {
		fallbacks = []RouteFallback{}
	}
	r := &Route{
		ID: newID("route"), OrganizationID: orgID, Model: model,
		ProviderID: providerID, TargetModel: targetModel, Fallbacks: fallbacks,
		CreatedAt: time.Now().UTC(),
	}
	fb, err := json.Marshal(r.Fallbacks)
	if err != nil {
		return nil, err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO routes (id, organization_id, model, provider_id, target_model, fallbacks, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
	`, r.ID, r.OrganizationID, r.Model, r.ProviderID, r.TargetModel, fb, r.CreatedAt)
	return r, err
}

func (s *Store) UpdateRoute(ctx context.Context, routeID, model, providerID, targetModel string, fallbacks []RouteFallback) (*Route, error) {
	if fallbacks == nil {
		fallbacks = []RouteFallback{}
	}
	fb, err := json.Marshal(fallbacks)
	if err != nil {
		return nil, err
	}
	r := &Route{}
	var raw []byte
	err = s.pool.QueryRow(ctx, `
		UPDATE routes SET model=$2, provider_id=$3, target_model=$4, fallbacks=$5
		WHERE id=$1
		RETURNING id, organization_id, model, provider_id, target_model, fallbacks, created_at
	`, routeID, model, providerID, targetModel, fb).Scan(
		&r.ID, &r.OrganizationID, &r.Model, &r.ProviderID, &r.TargetModel, &raw, &r.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	r.Fallbacks = decodeFallbacks(raw)
	return r, nil
}

func decodeFallbacks(raw []byte) []RouteFallback {
	if len(raw) == 0 {
		return []RouteFallback{}
	}
	var out []RouteFallback
	if err := json.Unmarshal(raw, &out); err != nil || out == nil {
		return []RouteFallback{}
	}
	return out
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
	modality := e.Modality
	if modality == "" {
		modality = "chat"
	}
	metrics := e.Metrics
	if metrics == nil {
		metrics = map[string]any{}
	}
	metricsJSON, err := json.Marshal(metrics)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO usage_events (
			organization_id, project_id, api_key_id, model, status,
			latency_ms, prompt_tokens, completion_tokens, cost_usd, modality, metrics
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	`, e.OrganizationID, e.ProjectID, e.APIKeyID, e.Model, e.Status,
		e.LatencyMs, e.PromptTokens, e.CompletionTokens, e.CostUSD, modality, metricsJSON)
	return err
}

func usageWhere(orgID string, f UsageFilter) (string, []any) {
	args := []any{orgID}
	var b strings.Builder
	b.WriteString("e.organization_id=$1")
	n := 2
	if f.ProjectID != "" {
		b.WriteString(fmt.Sprintf(" AND e.project_id=$%d", n))
		args = append(args, f.ProjectID)
		n++
	}
	if f.APIKeyID != "" {
		b.WriteString(fmt.Sprintf(" AND e.api_key_id=$%d", n))
		args = append(args, f.APIKeyID)
		n++
	}
	if f.Model != "" {
		b.WriteString(fmt.Sprintf(" AND e.model=$%d", n))
		args = append(args, f.Model)
		n++
	}
	if f.Modality != "" {
		b.WriteString(fmt.Sprintf(" AND e.modality=$%d", n))
		args = append(args, f.Modality)
		n++
	}
	if f.From != nil {
		b.WriteString(fmt.Sprintf(" AND e.created_at >= $%d", n))
		args = append(args, *f.From)
		n++
	}
	if f.To != nil {
		b.WriteString(fmt.Sprintf(" AND e.created_at < $%d", n))
		args = append(args, *f.To)
	}
	return b.String(), args
}

func (s *Store) ListUsage(ctx context.Context, orgID string, f UsageFilter) ([]UsageEvent, error) {
	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	where, args := usageWhere(orgID, f)
	args = append(args, limit)
	limitArg := len(args)
	rows, err := s.pool.Query(ctx, fmt.Sprintf(`
		SELECT e.id, e.organization_id, e.project_id, e.api_key_id, e.model, e.status,
			e.latency_ms, e.prompt_tokens, e.completion_tokens, e.cost_usd, e.created_at,
			e.modality, e.metrics,
			COALESCE(k.name, ''), COALESCE(k.kind, ''),
			COALESCE(k.owner_user_id, ''), COALESCE(u.email, ''), COALESCE(u.name, '')
		FROM usage_events e
		LEFT JOIN api_keys k ON k.id = e.api_key_id
		LEFT JOIN users u ON u.id = k.owner_user_id
		WHERE %s
		ORDER BY e.created_at DESC
		LIMIT $%d
	`, where, limitArg), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UsageEvent
	for rows.Next() {
		var e UsageEvent
		var metricsJSON []byte
		if err := rows.Scan(
			&e.ID, &e.OrganizationID, &e.ProjectID, &e.APIKeyID, &e.Model, &e.Status,
			&e.LatencyMs, &e.PromptTokens, &e.CompletionTokens, &e.CostUSD, &e.CreatedAt,
			&e.Modality, &metricsJSON,
			&e.KeyName, &e.KeyKind, &e.OwnerUserID, &e.OwnerEmail, &e.OwnerName,
		); err != nil {
			return nil, err
		}
		if len(metricsJSON) > 0 {
			_ = json.Unmarshal(metricsJSON, &e.Metrics)
		}
		if e.Metrics == nil {
			e.Metrics = map[string]any{}
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func metricSumExpr(key string) string {
	return fmt.Sprintf(`COALESCE(SUM(CASE WHEN jsonb_typeof(e.metrics->'%s') = 'number'
		THEN (e.metrics->>'%s')::double precision ELSE 0 END), 0)`, key, key)
}

func (s *Store) SummarizeUsage(ctx context.Context, orgID string, f UsageFilter) ([]UsageSummaryBucket, error) {
	groupBy := f.GroupBy
	if groupBy == "" {
		groupBy = "day"
	}
	ff := f
	if ff.From == nil && ff.To == nil && groupBy == "day" {
		from := time.Now().UTC().AddDate(0, 0, -30)
		ff.From = &from
	}
	where, args := usageWhere(orgID, ff)

	var selectBucket, groupSQL, orderSQL, joinSQL string
	switch groupBy {
	case "day":
		selectBucket = `to_char(date_trunc('day', e.created_at AT TIME ZONE 'UTC'), 'YYYY-MM-DD')`
		groupSQL = `1`
		orderSQL = `1 ASC`
		joinSQL = ""
	case "model":
		selectBucket = `e.model`
		groupSQL = `1`
		orderSQL = `requests DESC`
		joinSQL = ""
	case "modality":
		selectBucket = `e.modality`
		groupSQL = `1`
		orderSQL = `requests DESC`
		joinSQL = ""
	case "key":
		selectBucket = `e.api_key_id`
		groupSQL = `1, COALESCE(k.name,''), COALESCE(k.kind,''), COALESCE(u.email,''), COALESCE(u.name,'')`
		orderSQL = `requests DESC`
		joinSQL = `
			LEFT JOIN api_keys k ON k.id = e.api_key_id
			LEFT JOIN users u ON u.id = k.owner_user_id`
	default:
		return nil, fmt.Errorf("%w: group_by must be day, model, key, or modality", kernel.ErrInvalidRequest)
	}

	q := fmt.Sprintf(`
		SELECT %s AS bucket,
			COUNT(*)::bigint AS requests,
			COALESCE(SUM(e.cost_usd), 0)::double precision AS cost_usd,
			COALESCE(SUM(e.prompt_tokens), 0)::bigint AS prompt_tokens,
			COALESCE(SUM(e.completion_tokens), 0)::bigint AS completion_tokens,
			%s AS characters,
			%s AS audio_seconds,
			%s AS images,
			%s AS tokens
			%s
		FROM usage_events e
		%s
		WHERE %s
		GROUP BY %s
		ORDER BY %s
	`, selectBucket,
		metricSumExpr("characters"),
		metricSumExpr("audio_seconds"),
		metricSumExpr("images"),
		metricSumExpr("tokens"),
		func() string {
			if groupBy == "key" {
				return `, COALESCE(k.name,'') AS key_name, COALESCE(k.kind,'') AS key_kind,
					COALESCE(u.email,'') AS owner_email, COALESCE(u.name,'') AS owner_name`
			}
			return ``
		}(),
		joinSQL, where, groupSQL, orderSQL)

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []UsageSummaryBucket
	for rows.Next() {
		var b UsageSummaryBucket
		var characters, audioSeconds, images, tokens float64
		if groupBy == "key" {
			var keyName string
			if err := rows.Scan(
				&b.Bucket, &b.Requests, &b.CostUSD, &b.PromptTokens, &b.CompletionTokens,
				&characters, &audioSeconds, &images, &tokens,
				&keyName, &b.KeyKind, &b.OwnerEmail, &b.OwnerName,
			); err != nil {
				return nil, err
			}
			b.Label = keyName
			if b.Label == "" {
				b.Label = b.Bucket
			}
		} else {
			if err := rows.Scan(
				&b.Bucket, &b.Requests, &b.CostUSD, &b.PromptTokens, &b.CompletionTokens,
				&characters, &audioSeconds, &images, &tokens,
			); err != nil {
				return nil, err
			}
			b.Label = b.Bucket
		}
		totals := map[string]float64{}
		if characters != 0 {
			totals["characters"] = characters
		}
		if audioSeconds != 0 {
			totals["audio_seconds"] = audioSeconds
		}
		if images != 0 {
			totals["images"] = images
		}
		if tokens != 0 {
			totals["tokens"] = tokens
		}
		if len(totals) > 0 {
			b.MetricsTotals = totals
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (s *Store) LookupModelPrice(ctx context.Context, providerType, model string) (ModelPrice, bool, error) {
	var p ModelPrice
	err := s.pool.QueryRow(ctx, `
		SELECT provider_type, model, input_per_mtok, output_per_mtok
		FROM model_prices WHERE provider_type=$1 AND model=$2
	`, providerType, model).Scan(&p.ProviderType, &p.Model, &p.InputPerMTok, &p.OutputPerMTok)
	if errors.Is(err, pgx.ErrNoRows) {
		return ModelPrice{}, false, nil
	}
	if err != nil {
		return ModelPrice{}, false, err
	}
	return p, true, nil
}

// Quota is the platform write-model quota (canonical type in gatewayconfig).
type Quota = gatewayconfig.Quota

func (s *Store) quotas() *postgres.Quotas {
	return postgres.NewQuotas(s.pool)
}

func (s *Store) ListQuotas(ctx context.Context, orgID string) ([]Quota, error) {
	return s.quotas().ListByOrg(ctx, orgID)
}

func (s *Store) CreateQuota(ctx context.Context, orgID, scopeType, scopeID, metric string, limitValue int64, window string) (*Quota, error) {
	return gatewayconfig.CreateQuota(ctx, s.quotas(), s, newID("quota"), orgID, scopeType, scopeID, metric, limitValue, window)
}

func (s *Store) ProjectBelongsToOrg(ctx context.Context, projectID, orgID string) error {
	projOrg, err := s.GetProjectOrgID(ctx, projectID)
	if errors.Is(err, kernel.ErrNotFound) {
		return fmt.Errorf("%w: project not found", kernel.ErrInvalidRequest)
	}
	if err != nil {
		return err
	}
	if projOrg != orgID {
		return fmt.Errorf("%w: project not in organization", kernel.ErrInvalidRequest)
	}
	return nil
}

func (s *Store) UserIsOrgMember(ctx context.Context, userID, orgID string) error {
	ok, err := s.IsOrgMember(ctx, userID, orgID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: user is not an organization member", kernel.ErrInvalidRequest)
	}
	return nil
}

func (s *Store) APIKeyBelongsToOrg(ctx context.Context, keyID, orgID string) error {
	k, err := s.GetAPIKey(ctx, keyID)
	if errors.Is(err, kernel.ErrNotFound) {
		return fmt.Errorf("%w: api key not found", kernel.ErrInvalidRequest)
	}
	if err != nil {
		return err
	}
	if k.OrganizationID != orgID {
		return fmt.Errorf("%w: api key not in organization", kernel.ErrInvalidRequest)
	}
	return nil
}

func (s *Store) UpdateQuota(ctx context.Context, quotaID string, limitValue int64) (*Quota, error) {
	if err := gatewayconfig.ValidateLimit(limitValue); err != nil {
		return nil, err
	}
	return s.quotas().UpdateLimit(ctx, quotaID, limitValue)
}

func (s *Store) DeleteQuota(ctx context.Context, quotaID string) error {
	return s.quotas().Delete(ctx, quotaID)
}

func (s *Store) GetQuotaOrgID(ctx context.Context, quotaID string) (string, error) {
	return s.quotas().OrgID(ctx, quotaID)
}

// GetCounter / IncrCounter / EnqueueUsage remain for transitional callers.
// Prefer internal/adapters/postgres for new gateway/worker wiring.
func (s *Store) GetCounter(ctx context.Context, scopeType, scopeID, metric, window string) (int64, error) {
	return (&postgres.Counters{Pool: s.pool}).Get(ctx, scopeType, scopeID, metric, window)
}

func (s *Store) IncrCounter(ctx context.Context, scopeType, scopeID, metric, window string, delta int64) (int64, error) {
	return (&postgres.Counters{Pool: s.pool}).Incr(ctx, scopeType, scopeID, metric, window, delta)
}

func (s *Store) EnqueueUsage(ctx context.Context, payload []byte) error {
	return (&postgres.UsageOutbox{Pool: s.pool}).Enqueue(ctx, payload)
}

func (s *Store) LoadSnapshotSource(ctx context.Context) (snapshot.Source, error) {
	var src snapshot.Source

	keyRows, err := s.pool.Query(ctx, `
		SELECT id, key_hash, key_prefix, project_id, organization_id, name, kind, owner_user_id FROM api_keys
	`)
	if err != nil {
		return src, err
	}
	defer keyRows.Close()
	for keyRows.Next() {
		var k snapshot.APIKey
		var projectID, ownerUserID *string
		if err := keyRows.Scan(&k.ID, &k.KeyHash, &k.KeyPrefix, &projectID, &k.OrganizationID, &k.Name, &k.Kind, &ownerUserID); err != nil {
			return src, err
		}
		if projectID != nil {
			k.ProjectID = *projectID
		}
		if ownerUserID != nil {
			k.OwnerUserID = *ownerUserID
		}
		src.APIKeys = append(src.APIKeys, k)
	}
	if err := keyRows.Err(); err != nil {
		return src, err
	}

	provRows, err := s.pool.Query(ctx, `
		SELECT id, type, base_url, api_key_env, name, capabilities FROM providers
	`)
	if err != nil {
		return src, err
	}
	defer provRows.Close()
	for provRows.Next() {
		var p snapshot.Provider
		var caps []byte
		if err := provRows.Scan(&p.ID, &p.Type, &p.BaseURL, &p.APIKeyEnv, &p.Name, &caps); err != nil {
			return src, err
		}
		p.Capabilities = decodeCapabilities(p.Type, caps)
		src.Providers = append(src.Providers, p)
	}
	if err := provRows.Err(); err != nil {
		return src, err
	}

	routeRows, err := s.pool.Query(ctx, `
		SELECT organization_id, model, provider_id, target_model, fallbacks FROM routes
	`)
	if err != nil {
		return src, err
	}
	defer routeRows.Close()
	for routeRows.Next() {
		var r snapshot.Route
		var fb []byte
		if err := routeRows.Scan(&r.OrganizationID, &r.Model, &r.ProviderID, &r.TargetModel, &fb); err != nil {
			return src, err
		}
		for _, f := range decodeFallbacks(fb) {
			r.Fallbacks = append(r.Fallbacks, snapshot.RouteTarget{
				ProviderID: f.ProviderID, TargetModel: f.TargetModel,
			})
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
	if err := quotaRows.Err(); err != nil {
		return src, err
	}

	polRows, err := s.pool.Query(ctx, `
		SELECT id, organization_id, name, expression, enabled, priority FROM request_policies
	`)
	if err != nil {
		return src, err
	}
	defer polRows.Close()
	for polRows.Next() {
		var p snapshot.Policy
		if err := polRows.Scan(&p.ID, &p.OrganizationID, &p.Name, &p.Expression, &p.Enabled, &p.Priority); err != nil {
			return src, err
		}
		src.Policies = append(src.Policies, p)
	}
	return src, polRows.Err()
}

// RequestPolicy is the platform write-model CEL policy (canonical in gatewayconfig).
type RequestPolicy = gatewayconfig.RequestPolicy

func (s *Store) policies() *postgres.Policies {
	return postgres.NewPolicies(s.pool)
}

func (s *Store) ListPolicies(ctx context.Context, orgID string) ([]RequestPolicy, error) {
	return s.policies().ListByOrg(ctx, orgID)
}

func (s *Store) CreatePolicy(ctx context.Context, orgID, name, expression string, enabled bool, priority int) (*RequestPolicy, error) {
	return gatewayconfig.CreatePolicy(ctx, s.policies(), celValidator{}, newID("pol"), orgID, name, expression, enabled, priority)
}

func (s *Store) UpdatePolicy(ctx context.Context, policyID string, name, expression *string, enabled *bool, priority *int) (*RequestPolicy, error) {
	return gatewayconfig.UpdatePolicy(ctx, s.policies(), celValidator{}, policyID, name, expression, enabled, priority)
}

func (s *Store) DeletePolicy(ctx context.Context, policyID string) error {
	return s.policies().Delete(ctx, policyID)
}

func (s *Store) GetPolicyOrgID(ctx context.Context, policyID string) (string, error) {
	return s.policies().OrgID(ctx, policyID)
}
