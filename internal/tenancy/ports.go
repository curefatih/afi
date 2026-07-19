package tenancy

import "context"

// OrganizationRepository persists organizations and membership.
type OrganizationRepository interface {
	Count(ctx context.Context) (int64, error)
	ListForUser(ctx context.Context, userID string) ([]Organization, error)
	CreateWithOwner(ctx context.Context, org Organization, ownerUserID string) error
	ListMembers(ctx context.Context, orgID string) ([]OrgMember, error)
	AddMember(ctx context.Context, orgID, userID, role string) error
	GetMemberRole(ctx context.Context, userID, orgID string) (string, error)
	CountOwners(ctx context.Context, orgID string) (int, error)
	ApplyRoleChange(ctx context.Context, orgID, actorUserID, targetUserID, newRole string, demoteActor bool) error
	GetMember(ctx context.Context, orgID, userID string) (*OrgMember, error)
}

// TeamRepository persists teams.
type TeamRepository interface {
	ListByOrg(ctx context.Context, orgID string) ([]Team, error)
	ListByOrgForUser(ctx context.Context, orgID, userID string) ([]Team, error)
	Get(ctx context.Context, teamID string) (*Team, error)
	OrgID(ctx context.Context, teamID string) (string, error)
	ListMembers(ctx context.Context, teamID string) ([]TeamMember, error)
	GetMember(ctx context.Context, teamID, userID string) (*TeamMember, error)
	GetMemberRole(ctx context.Context, teamID, userID string) (string, error)
	AddMember(ctx context.Context, teamID, userID, role string) error
	RemoveMember(ctx context.Context, teamID, userID string) error
	CountOwners(ctx context.Context, teamID string) (int, error)
	CreateWithOwner(ctx context.Context, team Team, ownerUserID string) error
}

// ProjectRepository persists projects.
type ProjectRepository interface {
	ListByOrg(ctx context.Context, orgID string) ([]Project, error)
	ListByOrgForUser(ctx context.Context, orgID, userID string) ([]Project, error)
	Insert(ctx context.Context, p Project) error
	OrgID(ctx context.Context, projectID string) (string, error)
}

// UserLookup resolves users by email for invites.
type UserLookup interface {
	FindByEmail(ctx context.Context, email string) (userID, name, userEmail string, err error)
}
