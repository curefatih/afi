package controlplane

import "context"

// snapshotPublisher is implemented by *Seeder.
type snapshotPublisher interface {
	PublishSnapshot(ctx context.Context) error
}

// membershipChecker resolves org membership for authorization.
type membershipChecker interface {
	IsOrgMember(ctx context.Context, userID, orgID string) (bool, error)
	GetTeamOrgID(ctx context.Context, teamID string) (string, error)
	GetProjectOrgID(ctx context.Context, projectID string) (string, error)
	GetProviderOrgID(ctx context.Context, providerID string) (string, error)
	GetRouteOrgID(ctx context.Context, routeID string) (string, error)
}

// platformAPI is the control-plane persistence surface used by HTTP handlers.
type platformAPI interface {
	membershipChecker
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
	ListOrganizationsForUser(ctx context.Context, userID string) ([]Organization, error)
	ListTeams(ctx context.Context, orgID string) ([]Team, error)
	GetTeam(ctx context.Context, teamID string) (*Team, error)
	ListTeamMembers(ctx context.Context, teamID string) ([]TeamMember, error)
	ListProjects(ctx context.Context, orgID string) ([]Project, error)
	CreateProject(ctx context.Context, orgID, teamID, name string) (*Project, error)
	ListAPIKeys(ctx context.Context, projectID string) ([]APIKey, error)
	CreateAPIKey(ctx context.Context, orgID, projectID, name, rawKey string) (*APIKey, error)
	ListProviders(ctx context.Context, orgID string) ([]Provider, error)
	CreateProvider(ctx context.Context, orgID, name, typ, baseURL, apiKeyEnv string) (*Provider, error)
	UpdateProvider(ctx context.Context, providerID, name, baseURL, apiKeyEnv string) (*Provider, error)
	DeleteProvider(ctx context.Context, providerID string) error
	ListRoutes(ctx context.Context, orgID string) ([]Route, error)
	CreateRoute(ctx context.Context, orgID, model, providerID, targetModel string) (*Route, error)
	UpdateRoute(ctx context.Context, routeID, model, providerID, targetModel string) (*Route, error)
	DeleteRoute(ctx context.Context, routeID string) error
	ListUsage(ctx context.Context, orgID string, limit int) ([]UsageEvent, error)
}
