package controlplane

import (
	"context"
	"time"

	"github.com/curefatih/afi/internal/snapshot"
)

// snapshotPublisher is implemented by *Seeder.
type snapshotPublisher interface {
	PublishSnapshot(ctx context.Context) error
}

// membershipChecker resolves org membership for authorization.
type membershipChecker interface {
	IsOrgMember(ctx context.Context, userID, orgID string) (bool, error)
	IsOrgAdmin(ctx context.Context, userID, orgID string) (bool, error)
	GetTeamOrgID(ctx context.Context, teamID string) (string, error)
	GetProjectOrgID(ctx context.Context, projectID string) (string, error)
	GetProviderOrgID(ctx context.Context, providerID string) (string, error)
	GetRouteOrgID(ctx context.Context, routeID string) (string, error)
	GetQuotaOrgID(ctx context.Context, quotaID string) (string, error)
	GetPolicyOrgID(ctx context.Context, policyID string) (string, error)
	GetAPIKeyOrgID(ctx context.Context, keyID string) (string, error)
}

// platformAPI is the control-plane persistence surface used by HTTP handlers.
type platformAPI interface {
	membershipChecker
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
	ListOrganizationsForUser(ctx context.Context, userID string) ([]Organization, error)
	CreateOrganization(ctx context.Context, name, creatorUserID string) (*Organization, error)
	ListOrgMembers(ctx context.Context, orgID string) ([]OrgMember, error)
	AddOrgMemberByEmail(ctx context.Context, orgID, email string) (*OrgMember, error)
	UpdateOrgMemberRole(ctx context.Context, orgID, actorUserID, targetUserID, role string) (*OrgMember, error)
	ListTeams(ctx context.Context, orgID string) ([]Team, error)
	CreateTeam(ctx context.Context, orgID, name, creatorUserID string) (*Team, error)
	GetTeam(ctx context.Context, teamID string) (*Team, error)
	ListTeamMembers(ctx context.Context, teamID string) ([]TeamMember, error)
	ListProjects(ctx context.Context, orgID string) ([]Project, error)
	CreateProject(ctx context.Context, orgID, teamID, name string) (*Project, error)
	ListAPIKeys(ctx context.Context, projectID string) ([]APIKey, error)
	ListOrgAPIKeys(ctx context.Context, orgID string) ([]APIKey, error)
	GetAPIKey(ctx context.Context, keyID string) (*APIKey, error)
	CreateAPIKey(ctx context.Context, orgID, kind, ownerUserID, projectID, name, rawKey string) (*APIKey, error)
	DeleteAPIKey(ctx context.Context, keyID string) error
	ListProviders(ctx context.Context, orgID string) ([]Provider, error)
	ListProviderHealth(ctx context.Context, orgID string, from, to time.Time) ([]ProviderHealth, error)
	CreateProvider(ctx context.Context, orgID, name, typ, baseURL, apiKeyEnv string, caps snapshot.ProviderCapabilities) (*Provider, error)
	UpdateProvider(ctx context.Context, providerID, name, baseURL, apiKeyEnv string) (*Provider, error)
	DeleteProvider(ctx context.Context, providerID string) error
	ListRoutes(ctx context.Context, orgID string) ([]Route, error)
	CreateRoute(ctx context.Context, orgID, model, providerID, targetModel string, fallbacks []RouteFallback) (*Route, error)
	UpdateRoute(ctx context.Context, routeID, model, providerID, targetModel string, fallbacks []RouteFallback) (*Route, error)
	DeleteRoute(ctx context.Context, routeID string) error
	ListUsage(ctx context.Context, orgID string, f UsageFilter) ([]UsageEvent, error)
	SummarizeUsage(ctx context.Context, orgID string, f UsageFilter) ([]UsageSummaryBucket, error)
	ListQuotas(ctx context.Context, orgID string) ([]Quota, error)
	CreateQuota(ctx context.Context, orgID, scopeType, scopeID, metric string, limitValue int64, window string) (*Quota, error)
	UpdateQuota(ctx context.Context, quotaID string, limitValue int64) (*Quota, error)
	DeleteQuota(ctx context.Context, quotaID string) error
	ListPolicies(ctx context.Context, orgID string) ([]RequestPolicy, error)
	CreatePolicy(ctx context.Context, orgID, name, expression string, enabled bool, priority int) (*RequestPolicy, error)
	UpdatePolicy(ctx context.Context, policyID string, name, expression *string, enabled *bool, priority *int) (*RequestPolicy, error)
	DeletePolicy(ctx context.Context, policyID string) error
}
