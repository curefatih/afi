package controlplane

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/curefatih/afi/internal/access"
	"github.com/curefatih/afi/internal/adapters/postgres"
	"github.com/curefatih/afi/internal/gatewayconfig"
	"github.com/curefatih/afi/internal/identity"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/snapshot"
	"github.com/curefatih/afi/internal/tenancy"
	"github.com/curefatih/afi/internal/usage"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	OrgRoleOwner  = tenancy.OrgRoleOwner
	OrgRoleAdmin  = tenancy.OrgRoleAdmin
	OrgRoleMember = tenancy.OrgRoleMember

	TeamRoleOwner  = tenancy.TeamRoleOwner
	TeamRoleMember = tenancy.TeamRoleMember
)

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

// UsageEvent is the control-plane read model (persisted row + joins).
// Emit/outbox use the canonical usage.Event.
type UsageEvent = usage.Record
type UsageFilter = usage.Filter
type UsageSummaryBucket = usage.SummaryBucket
type ModelPrice = usage.ModelPrice
type ProviderHealth = usage.ProviderHealth

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
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

func (s *Store) ListTeams(ctx context.Context, orgID, userID string) ([]Team, error) {
	return tenancy.ListVisibleTeams(ctx, s.teamsRepo(), s.organizations(), orgID, userID)
}

func (s *Store) CreateTeam(ctx context.Context, orgID, name, creatorUserID string) (*Team, error) {
	return tenancy.CreateTeam(ctx, s.teamsRepo(), newID("team"), orgID, name, creatorUserID)
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

func (s *Store) AddTeamMember(ctx context.Context, teamID, userID string) (*TeamMember, error) {
	return tenancy.AddTeamMember(ctx, s.teamsRepo(), s.organizations(), teamID, userID)
}

func (s *Store) RemoveTeamMember(ctx context.Context, teamID, userID string) error {
	return tenancy.RemoveTeamMember(ctx, s.teamsRepo(), teamID, userID)
}

func (s *Store) CanAccessTeam(ctx context.Context, teamID, userID string) (bool, error) {
	return tenancy.CanAccessTeam(ctx, s.teamsRepo(), s.organizations(), teamID, userID)
}

func (s *Store) CanManageTeam(ctx context.Context, teamID, userID string) (bool, error) {
	return tenancy.CanManageTeam(ctx, s.teamsRepo(), s.organizations(), teamID, userID)
}

func (s *Store) ListProjects(ctx context.Context, orgID, userID string) ([]Project, error) {
	return tenancy.ListVisibleProjects(ctx, s.projectsRepo(), s.organizations(), orgID, userID)
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

func (s *Store) usageQueries() *postgres.UsageQueries {
	return postgres.NewUsageQueries(s.pool)
}

// ListProviderHealth aggregates usage_events for models routed to each org provider.
func (s *Store) ListProviderHealth(ctx context.Context, orgID string, from, to time.Time) ([]ProviderHealth, error) {
	return s.usageQueries().ListProviderHealth(ctx, orgID, from, to)
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
	return s.usageQueries().InsertRecord(ctx, e)
}

func (s *Store) ListUsage(ctx context.Context, orgID string, f UsageFilter) ([]UsageEvent, error) {
	return s.usageQueries().List(ctx, orgID, f)
}

func (s *Store) SummarizeUsage(ctx context.Context, orgID string, f UsageFilter) ([]UsageSummaryBucket, error) {
	return s.usageQueries().Summarize(ctx, orgID, f)
}

func (s *Store) LookupModelPrice(ctx context.Context, providerType, model string) (ModelPrice, bool, error) {
	return s.usageQueries().LookupModelPrice(ctx, providerType, model)
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
	return postgres.NewSnapshotSourceLoader(s.pool).Load(ctx)
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
