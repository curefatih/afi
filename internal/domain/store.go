package domain

import "context"

type Store interface {
	Open() (*Store, error)
	Close() error
	Migrate(migrationsDir string) error
	ListOrganizations(ctx context.Context) ([]Organization, error)
	GetOrganization(ctx context.Context, id string) (*Organization, error)
	CreateOrganization(ctx context.Context, name string, budgetCents int64) (*Organization, error)
	UpdateOrganization(ctx context.Context, id, name string, budgetCents int64) error
	DeleteOrganization(ctx context.Context, id string) error
	ListTeams(ctx context.Context, orgID string) ([]Team, error)
	GetTeam(ctx context.Context, id string) (*Team, error)
	CreateTeam(ctx context.Context, orgID, name string, budgetCents int64) (*Team, error)
	UpdateTeam(ctx context.Context, id, name string, budgetCents int64) error
	DeleteTeam(ctx context.Context, id string) error
	ListProjects(ctx context.Context, teamID string) ([]Project, error)
	GetProject(ctx context.Context, id string) (*Project, error)
	CreateProject(ctx context.Context, teamID, name string) (*Project, error)
	UpdateProject(ctx context.Context, id, name string) error
	DeleteProject(ctx context.Context, id string) error
	ListUsers(ctx context.Context) ([]User, error)
	GetUser(ctx context.Context, id string) (*User, error)
	CreateUser(ctx context.Context, email, oidcSub string) (*User, error)
	UpdateUser(ctx context.Context, id, email, oidcSub string) error
	DeleteUser(ctx context.Context, id string) error
	ListProjectMembers(ctx context.Context, projectID string) ([]ProjectMember, error)
	AddProjectMember(ctx context.Context, projectID, userID, role string) error
	RemoveProjectMember(ctx context.Context, projectID, userID string) error
	UpdateProjectMemberRole(ctx context.Context, projectID, userID, role string) error

	// API keys
	LookupAPIKey(ctx context.Context, rawKey string) (*APIKeyRecord, error)
	CreateAPIKey(ctx context.Context, projectID, name, rawKey string, modelAllowlist []string, requiredTags map[string]string, rateLimitRPM int) (*APIKeyRecord, error)
	UpdateAPIKey(ctx context.Context, id, name string, modelAllowlist []string, requiredTags map[string]string, rateLimitRPM int) (*APIKeyRecord, error)
	DeleteAPIKey(ctx context.Context, id string) error
}
