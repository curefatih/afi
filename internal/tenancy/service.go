package tenancy

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/curefatih/afi/internal/kernel"
)

var timeNowUTC = func() time.Time { return time.Now().UTC() }

// CreateOrganization creates an org and assigns the creator as owner.
func CreateOrganization(ctx context.Context, repo OrganizationRepository, id, name, creatorUserID string) (*Organization, error) {
	if creatorUserID == "" {
		return nil, kernel.ErrInvalidRequest
	}
	o, err := NewOrganization(id, name, timeNowUTC())
	if err != nil {
		return nil, err
	}
	if err := repo.CreateWithOwner(ctx, *o, creatorUserID); err != nil {
		return nil, err
	}
	return o, nil
}

// CreateTeam creates a team and assigns the creator as team owner.
func CreateTeam(ctx context.Context, repo TeamRepository, id, orgID, name, creatorUserID string) (*Team, error) {
	if creatorUserID == "" {
		return nil, kernel.ErrInvalidRequest
	}
	t, err := NewTeam(id, orgID, name, timeNowUTC())
	if err != nil {
		return nil, err
	}
	if err := repo.CreateWithOwner(ctx, *t, creatorUserID); err != nil {
		return nil, err
	}
	return t, nil
}

// CreateProject validates and persists a project.
func CreateProject(ctx context.Context, repo ProjectRepository, id, orgID, teamID, name string) (*Project, error) {
	p, err := NewProject(id, orgID, teamID, name, timeNowUTC())
	if err != nil {
		return nil, err
	}
	if err := repo.Insert(ctx, *p); err != nil {
		return nil, err
	}
	return p, nil
}

// CreateEnvironment validates and persists an environment under a project.
func CreateEnvironment(ctx context.Context, repo EnvironmentRepository, projects ProjectRepository, id, orgID, projectID, name, slug string) (*Environment, error) {
	projOrg, err := projects.OrgID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if projOrg != orgID {
		return nil, fmt.Errorf("%w: project does not belong to organization", kernel.ErrInvalidRequest)
	}
	e, err := NewEnvironment(id, orgID, projectID, name, slug, timeNowUTC())
	if err != nil {
		return nil, err
	}
	if err := repo.Insert(ctx, *e); err != nil {
		return nil, err
	}
	return e, nil
}

// AddOrgMemberByEmail invites an existing user as a member.
func AddOrgMemberByEmail(ctx context.Context, orgs OrganizationRepository, users UserLookup, orgID, email string) (*OrgMember, error) {
	userID, name, userEmail, err := users.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if err := orgs.AddMember(ctx, orgID, userID, OrgRoleMember); err != nil {
		return nil, err
	}
	return &OrgMember{UserID: userID, Email: userEmail, Name: name, Role: OrgRoleMember}, nil
}

// UpdateOrgMemberRole applies ownership/admin/member transitions with sole-owner guards.
func UpdateOrgMemberRole(ctx context.Context, repo OrganizationRepository, orgID, actorUserID, targetUserID, role string) (*OrgMember, error) {
	parsed, err := ParseOrgRole(role)
	if err != nil {
		return nil, err
	}
	role = parsed.String()
	actorRole, err := repo.GetMemberRole(ctx, actorUserID, orgID)
	if err != nil {
		return nil, err
	}
	if err := AssertActorCanChangeRoles(actorRole); err != nil {
		return nil, err
	}
	targetRole, err := repo.GetMemberRole(ctx, targetUserID, orgID)
	if err != nil {
		return nil, err
	}
	owners, err := repo.CountOwners(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if err := AssertSoleOwnerSafe(actorUserID, targetUserID, targetRole, role, owners); err != nil {
		return nil, err
	}
	demoteActor := ShouldDemoteActorOnOwnerTransfer(actorUserID, targetUserID, role)
	if err := repo.ApplyRoleChange(ctx, orgID, actorUserID, targetUserID, role, demoteActor); err != nil {
		return nil, err
	}
	return repo.GetMember(ctx, orgID, targetUserID)
}

// IsOrgAdmin reports whether the user has admin privileges in the org.
func IsOrgAdmin(ctx context.Context, repo OrganizationRepository, userID, orgID string) (bool, error) {
	role, err := repo.GetMemberRole(ctx, userID, orgID)
	if errors.Is(err, kernel.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return IsOrgAdminRole(role), nil
}

// IsOrgOwner reports whether the user is the org owner.
func IsOrgOwner(ctx context.Context, repo OrganizationRepository, userID, orgID string) (bool, error) {
	role, err := repo.GetMemberRole(ctx, userID, orgID)
	if errors.Is(err, kernel.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return OrgRole(role).CanChangeRoles(), nil
}

// ListVisibleTeams returns all org teams for admins, otherwise only teams the user belongs to.
func ListVisibleTeams(ctx context.Context, teams TeamRepository, orgs OrganizationRepository, orgID, userID string) ([]Team, error) {
	admin, err := IsOrgAdmin(ctx, orgs, userID, orgID)
	if err != nil {
		return nil, err
	}
	if admin {
		return teams.ListByOrg(ctx, orgID)
	}
	return teams.ListByOrgForUser(ctx, orgID, userID)
}

// ListVisibleProjects returns all org projects for admins, otherwise projects on the user's teams.
func ListVisibleProjects(ctx context.Context, projects ProjectRepository, orgs OrganizationRepository, orgID, userID string) ([]Project, error) {
	admin, err := IsOrgAdmin(ctx, orgs, userID, orgID)
	if err != nil {
		return nil, err
	}
	if admin {
		return projects.ListByOrg(ctx, orgID)
	}
	return projects.ListByOrgForUser(ctx, orgID, userID)
}

// CanAccessTeam reports whether the user may view the team (org admin or team member).
func CanAccessTeam(ctx context.Context, teams TeamRepository, orgs OrganizationRepository, teamID, userID string) (bool, error) {
	orgID, err := teams.OrgID(ctx, teamID)
	if err != nil {
		return false, err
	}
	admin, err := IsOrgAdmin(ctx, orgs, userID, orgID)
	if err != nil {
		return false, err
	}
	if admin {
		return true, nil
	}
	_, err = teams.GetMemberRole(ctx, teamID, userID)
	if errors.Is(err, kernel.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// CanManageTeam reports whether the user may add/remove team members
// (org admin, or team owner/admin).
func CanManageTeam(ctx context.Context, teams TeamRepository, orgs OrganizationRepository, teamID, userID string) (bool, error) {
	orgID, err := teams.OrgID(ctx, teamID)
	if err != nil {
		return false, err
	}
	admin, err := IsOrgAdmin(ctx, orgs, userID, orgID)
	if err != nil {
		return false, err
	}
	if admin {
		return true, nil
	}
	role, err := teams.GetMemberRole(ctx, teamID, userID)
	if errors.Is(err, kernel.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return IsTeamManagerRole(role), nil
}

// CanChangeTeamRoles reports whether the user may change team member roles
// (org admin or team owner — not team admin).
func CanChangeTeamRoles(ctx context.Context, teams TeamRepository, orgs OrganizationRepository, teamID, userID string) (bool, error) {
	orgID, err := teams.OrgID(ctx, teamID)
	if err != nil {
		return false, err
	}
	admin, err := IsOrgAdmin(ctx, orgs, userID, orgID)
	if err != nil {
		return false, err
	}
	if admin {
		return true, nil
	}
	role, err := teams.GetMemberRole(ctx, teamID, userID)
	if errors.Is(err, kernel.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return TeamRole(role).CanChangeRoles(), nil
}

// AddTeamMember adds an existing org member to the team as a member.
func AddTeamMember(ctx context.Context, teams TeamRepository, orgs OrganizationRepository, teamID, userID string) (*TeamMember, error) {
	if userID == "" {
		return nil, kernel.ErrInvalidRequest
	}
	orgID, err := teams.OrgID(ctx, teamID)
	if err != nil {
		return nil, err
	}
	if _, err := orgs.GetMemberRole(ctx, userID, orgID); err != nil {
		if errors.Is(err, kernel.ErrNotFound) {
			return nil, kernel.ErrInvalidRequest
		}
		return nil, err
	}
	if _, err := teams.GetMemberRole(ctx, teamID, userID); err == nil {
		return nil, kernel.ErrInvalidRequest
	} else if !errors.Is(err, kernel.ErrNotFound) {
		return nil, err
	}
	if err := teams.AddMember(ctx, teamID, userID, TeamRoleMember); err != nil {
		return nil, err
	}
	return teams.GetMember(ctx, teamID, userID)
}

// RemoveTeamMember removes a user from the team. Cannot remove the sole team owner.
func RemoveTeamMember(ctx context.Context, teams TeamRepository, teamID, targetUserID string) error {
	if targetUserID == "" {
		return kernel.ErrInvalidRequest
	}
	role, err := teams.GetMemberRole(ctx, teamID, targetUserID)
	if err != nil {
		return err
	}
	if role == TeamRoleOwner {
		owners, err := teams.CountOwners(ctx, teamID)
		if err != nil {
			return err
		}
		if owners <= 1 {
			return kernel.ErrInvalidRequest
		}
	}
	return teams.RemoveMember(ctx, teamID, targetUserID)
}

// UpdateTeamMemberRole sets a team member's role. Only org admins and team owners
// may change roles. Cannot demote the sole team owner.
func UpdateTeamMemberRole(ctx context.Context, teams TeamRepository, orgs OrganizationRepository, teamID, actorUserID, targetUserID, role string) (*TeamMember, error) {
	parsed, err := ParseTeamRole(role)
	if err != nil {
		return nil, err
	}
	role = parsed.String()
	if actorUserID == "" || targetUserID == "" {
		return nil, kernel.ErrInvalidRequest
	}
	ok, err := CanChangeTeamRoles(ctx, teams, orgs, teamID, actorUserID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, kernel.ErrUnauthorized
	}
	targetRole, err := teams.GetMemberRole(ctx, teamID, targetUserID)
	if err != nil {
		return nil, err
	}
	if targetRole == role {
		return teams.GetMember(ctx, teamID, targetUserID)
	}
	owners, err := teams.CountOwners(ctx, teamID)
	if err != nil {
		return nil, err
	}
	if err := AssertSoleTeamOwnerSafe(targetRole, role, owners); err != nil {
		return nil, err
	}
	if err := teams.SetMemberRole(ctx, teamID, targetUserID, role); err != nil {
		return nil, err
	}
	return teams.GetMember(ctx, teamID, targetUserID)
}
