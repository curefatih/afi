package tenancy

import (
	"context"
	"errors"
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
	if err := ValidateOrgRole(role); err != nil {
		return nil, err
	}
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
