package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/access"
	"github.com/curefatih/afi/internal/adapters/postgres"
	"github.com/curefatih/afi/internal/gatewayconfig"
	"github.com/curefatih/afi/internal/identity"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/snapshot"
	"github.com/curefatih/afi/internal/tenancy"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	OrgRoleOwner  = tenancy.OrgRoleOwner
	OrgRoleAdmin  = tenancy.OrgRoleAdmin
	OrgRoleMember = tenancy.OrgRoleMember
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

type User = identity.User
type Organization = tenancy.Organization
type Team = tenancy.Team
type TeamMember = tenancy.TeamMember
type Project = tenancy.Project
type OrgMember = tenancy.OrgMember

// APIKey is the platform write-model key (canonical in access).
type APIKey = access.APIKey

// Provider / Route are platform write-model config (canonical in gatewayconfig).
type Provider = gatewayconfig.Provider
type RouteFallback = gatewayconfig.RouteFallback
type Route = gatewayconfig.Route

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

func (s *Store) users() *postgres.Users {
	return postgres.NewUsers(s.pool)
}

func (s *Store) organizations() *postgres.Organizations {
	return postgres.NewOrganizations(s.pool)
}

func (s *Store) teamsRepo() *postgres.Teams {
	return postgres.NewTeams(s.pool)
}

func (s *Store) projectsRepo() *postgres.Projects {
	return postgres.NewProjects(s.pool)
}

func (s *Store) CountOrgs(ctx context.Context) (int64, error) {
	return s.organizations().Count(ctx)
}

func (s *Store) IsOrgMember(ctx context.Context, userID, orgID string) (bool, error) {
	_, err := s.organizations().GetMemberRole(ctx, userID, orgID)
	if errors.Is(err, kernel.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	return s.users().GetByEmail(ctx, email)
}

func (s *Store) GetUserByID(ctx context.Context, id string) (*User, error) {
	return s.users().GetByID(ctx, id)
}

func (s *Store) ListOrganizationsForUser(ctx context.Context, userID string) ([]Organization, error) {
	return s.organizations().ListForUser(ctx, userID)
}

func (s *Store) CreateOrganization(ctx context.Context, name, creatorUserID string) (*Organization, error) {
	return tenancy.CreateOrganization(ctx, s.organizations(), newID("org"), name, creatorUserID)
}

func (s *Store) ListOrgMembers(ctx context.Context, orgID string) ([]OrgMember, error) {
	return s.organizations().ListMembers(ctx, orgID)
}

func (s *Store) FindByEmail(ctx context.Context, email string) (userID, name, userEmail string, err error) {
	u, err := s.GetUserByEmail(ctx, email)
	if err != nil {
		return "", "", "", err
	}
	return u.ID, u.Name, u.Email, nil
}

func (s *Store) AddOrgMemberByEmail(ctx context.Context, orgID, email string) (*OrgMember, error) {
	return tenancy.AddOrgMemberByEmail(ctx, s.organizations(), s, orgID, email)
}

func (s *Store) GetOrgMemberRole(ctx context.Context, userID, orgID string) (string, error) {
	return s.organizations().GetMemberRole(ctx, userID, orgID)
}

func (s *Store) IsOrgAdmin(ctx context.Context, userID, orgID string) (bool, error) {
	return tenancy.IsOrgAdmin(ctx, s.organizations(), userID, orgID)
}

// UpdateOrgMemberRole changes a member's role. Only the org owner may call this.
// Promoting to owner transfers ownership (actor becomes admin). Cannot demote the sole owner.
func (s *Store) UpdateOrgMemberRole(ctx context.Context, orgID, actorUserID, targetUserID, role string) (*OrgMember, error) {
	return tenancy.UpdateOrgMemberRole(ctx, s.organizations(), orgID, actorUserID, targetUserID, role)
}

func (s *Store) ListTeams(ctx context.Context, orgID string) ([]Team, error) {
	return s.teamsRepo().ListByOrg(ctx, orgID)
}

func (s *Store) GetTeam(ctx context.Context, teamID string) (*Team, error) {
	return s.teamsRepo().Get(ctx, teamID)
}

func (s *Store) GetTeamOrgID(ctx context.Context, teamID string) (string, error) {
	return s.teamsRepo().OrgID(ctx, teamID)
}

func (s *Store) ListTeamMembers(ctx context.Context, teamID string) ([]TeamMember, error) {
	return s.teamsRepo().ListMembers(ctx, teamID)
}

func (s *Store) ListProjects(ctx context.Context, orgID string) ([]Project, error) {
	return s.projectsRepo().ListByOrg(ctx, orgID)
}

func (s *Store) CreateProject(ctx context.Context, orgID, teamID, name string) (*Project, error) {
	return tenancy.CreateProject(ctx, s.projectsRepo(), newID("proj"), orgID, teamID, name)
}

func (s *Store) apiKeys() *postgres.APIKeys {
	return postgres.NewAPIKeys(s.pool)
}

func (s *Store) ListAPIKeys(ctx context.Context, projectID string) ([]APIKey, error) {
	return s.apiKeys().ListByProject(ctx, projectID)
}

func (s *Store) ListOrgAPIKeys(ctx context.Context, orgID string) ([]APIKey, error) {
	return s.apiKeys().ListByOrg(ctx, orgID)
}

func (s *Store) GetAPIKey(ctx context.Context, keyID string) (*APIKey, error) {
	return s.apiKeys().Get(ctx, keyID)
}

func (s *Store) GetAPIKeyOrgID(ctx context.Context, keyID string) (string, error) {
	return s.apiKeys().OrgID(ctx, keyID)
}

func (s *Store) DeleteAPIKey(ctx context.Context, keyID string) error {
	return s.apiKeys().Delete(ctx, keyID)
}

func (s *Store) GetProjectOrgID(ctx context.Context, projectID string) (string, error) {
	return s.projectsRepo().OrgID(ctx, projectID)
}

// CreateAPIKey inserts a key. kind must be personal or service_account.
// Personal: ownerUserID required, projectID must be empty.
// Service account: ownerUserID empty, projectID optional (empty = org-wide).
func (s *Store) CreateAPIKey(ctx context.Context, orgID, kind, ownerUserID, projectID, name, rawKey string) (*APIKey, error) {
	return access.CreateAPIKey(ctx, s.apiKeys(), s, newID("key"), orgID, kind, ownerUserID, projectID, name, rawKey)
}

func (s *Store) providers() *postgres.Providers {
	return postgres.NewProviders(s.pool)
}

func (s *Store) routes() *postgres.Routes {
	return postgres.NewRoutes(s.pool)
}

func (s *Store) ListProviders(ctx context.Context, orgID string) ([]Provider, error) {
	return s.providers().ListByOrg(ctx, orgID)
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
	return gatewayconfig.CreateProvider(ctx, s.providers(), newID("prov"), orgID, name, typ, baseURL, apiKeyEnv, caps)
}

func (s *Store) UpdateProvider(ctx context.Context, providerID, name, baseURL, apiKeyEnv string) (*Provider, error) {
	return s.providers().Update(ctx, providerID, name, baseURL, apiKeyEnv)
}

func (s *Store) DeleteProvider(ctx context.Context, providerID string) error {
	return s.providers().Delete(ctx, providerID)
}

func (s *Store) GetProviderOrgID(ctx context.Context, providerID string) (string, error) {
	return s.providers().OrgID(ctx, providerID)
}

func (s *Store) ListRoutes(ctx context.Context, orgID string) ([]Route, error) {
	return s.routes().ListByOrg(ctx, orgID)
}

func (s *Store) CreateRoute(ctx context.Context, orgID, model, providerID, targetModel string, fallbacks []RouteFallback) (*Route, error) {
	return gatewayconfig.CreateRoute(ctx, s.routes(), newID("route"), orgID, model, providerID, targetModel, fallbacks)
}

func (s *Store) UpdateRoute(ctx context.Context, routeID, model, providerID, targetModel string, fallbacks []RouteFallback) (*Route, error) {
	return s.routes().Update(ctx, routeID, model, providerID, targetModel, fallbacks)
}

func (s *Store) DeleteRoute(ctx context.Context, routeID string) error {
	return s.routes().Delete(ctx, routeID)
}

func (s *Store) GetRouteOrgID(ctx context.Context, routeID string) (string, error) {
	return s.routes().OrgID(ctx, routeID)
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
		p.Capabilities = postgres.DecodeCapabilities(p.Type, caps)
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
		for _, f := range postgres.DecodeFallbacks(fb) {
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
