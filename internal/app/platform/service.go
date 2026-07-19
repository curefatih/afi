package platform

import (
	"context"
	"time"

	"github.com/curefatih/afi/internal/access"
	"github.com/curefatih/afi/internal/credentials"
	"github.com/curefatih/afi/internal/gatewayconfig"
	"github.com/curefatih/afi/internal/snapshot"
	"github.com/curefatih/afi/internal/tenancy"
	"github.com/curefatih/afi/internal/usage"
)

// SnapshotPublisher publishes compiled gateway snapshots after config changes.
type SnapshotPublisher interface {
	PublishSnapshot(ctx context.Context) error
}

// ConfigAPI is the persistence surface for platform reads and mutations.
type ConfigAPI interface {
	ListOrganizationsForUser(ctx context.Context, userID string) ([]tenancy.Organization, error)
	ListOrgMembers(ctx context.Context, orgID string) ([]tenancy.OrgMember, error)
	ListTeams(ctx context.Context, orgID, userID string) ([]tenancy.Team, error)
	GetTeam(ctx context.Context, teamID string) (*tenancy.Team, error)
	ListTeamMembers(ctx context.Context, teamID string) ([]tenancy.TeamMember, error)
	ListProjects(ctx context.Context, orgID, userID string) ([]tenancy.Project, error)
	CreateProject(ctx context.Context, orgID, teamID, name string) (*tenancy.Project, error)

	ListAPIKeys(ctx context.Context, projectID string) ([]access.APIKey, error)
	ListOrgAPIKeys(ctx context.Context, orgID string) ([]access.APIKey, error)
	CreateAPIKey(ctx context.Context, orgID, kind, ownerUserID, projectID, name, rawKey string) (*access.APIKey, error)
	DeleteAPIKey(ctx context.Context, keyID string) error

	ListProviders(ctx context.Context, orgID string) ([]gatewayconfig.Provider, error)
	ListProviderHealth(ctx context.Context, orgID string, from, to time.Time) ([]usage.ProviderHealth, error)
	CreateProvider(ctx context.Context, orgID, name, typ, baseURL, apiKeyEnv string, caps snapshot.ProviderCapabilities) (*gatewayconfig.Provider, error)
	UpdateProvider(ctx context.Context, providerID, name, baseURL, apiKeyEnv string) (*gatewayconfig.Provider, error)
	DeleteProvider(ctx context.Context, providerID string) error

	ListRoutes(ctx context.Context, orgID string) ([]gatewayconfig.Route, error)
	CreateRoute(ctx context.Context, orgID, model, providerID, targetModel string, fallbacks []gatewayconfig.RouteFallback) (*gatewayconfig.Route, error)
	UpdateRoute(ctx context.Context, routeID, model, providerID, targetModel string, fallbacks []gatewayconfig.RouteFallback) (*gatewayconfig.Route, error)
	DeleteRoute(ctx context.Context, routeID string) error

	ListUsage(ctx context.Context, orgID string, f usage.Filter) ([]usage.Record, error)
	SummarizeUsage(ctx context.Context, orgID string, f usage.Filter) ([]usage.SummaryBucket, error)

	ListQuotas(ctx context.Context, orgID string) ([]gatewayconfig.Quota, error)
	CreateQuota(ctx context.Context, orgID, scopeType, scopeID, metric string, limitValue int64, window string) (*gatewayconfig.Quota, error)
	UpdateQuota(ctx context.Context, quotaID string, limitValue int64) (*gatewayconfig.Quota, error)
	DeleteQuota(ctx context.Context, quotaID string) error

	ListPolicies(ctx context.Context, orgID string) ([]gatewayconfig.RequestPolicy, error)
	CreatePolicy(ctx context.Context, orgID, name, expression string, enabled bool, priority int) (*gatewayconfig.RequestPolicy, error)
	UpdatePolicy(ctx context.Context, policyID string, name, expression *string, enabled *bool, priority *int) (*gatewayconfig.RequestPolicy, error)
	DeletePolicy(ctx context.Context, policyID string) error

	ListCredentials(ctx context.Context, orgID string) ([]credentials.Credential, error)
	CreateCredential(ctx context.Context, orgID, name, providerType, storageKind, secretRef, secretValue string) (*credentials.Credential, error)
	UpdateCredential(ctx context.Context, credentialID, name, status string) (*credentials.Credential, error)
	RotateCredential(ctx context.Context, credentialID, secretRef, secretValue string) (*credentials.Credential, error)
	DeleteCredential(ctx context.Context, credentialID string) error
	ListCredentialAssignments(ctx context.Context, orgID string) ([]credentials.Assignment, error)
	AssignCredential(ctx context.Context, credentialID, scopeType, scopeID, createdBy string) (*credentials.Assignment, error)
	DeleteCredentialAssignment(ctx context.Context, assignmentID string) error
}

// Service orchestrates platform queries and commands (mutate + publish + events).
type Service struct {
	API    ConfigAPI
	Snap   SnapshotPublisher
	Events EventRecorder
}

func New(api ConfigAPI, snap SnapshotPublisher) *Service {
	return &Service{API: api, Snap: snap, Events: NopRecorder{}}
}
