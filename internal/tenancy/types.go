package tenancy

import (
	"fmt"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/kernel"
)

const (
	OrgRoleOwner  = "owner"
	OrgRoleAdmin  = "admin"
	OrgRoleMember = "member"

	TeamRoleOwner  = "owner"
	TeamRoleMember = "member"
)

// Organization is a tenant root.
type Organization struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// OrgMember is a user membership within an organization.
type OrgMember struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	Role   string `json:"role"`
}

// Team belongs to an organization.
type Team struct {
	ID             string    `json:"id"`
	TeamID         string    `json:"team_id"`
	OrganizationID string    `json:"organization_id"`
	Name           string    `json:"name"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// TeamMember is a user membership within a team.
type TeamMember struct {
	UserID string `json:"user_id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}

// Project belongs to an organization (optionally a team).
type Project struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	TeamID         string    `json:"team_id,omitempty"`
	Name           string    `json:"name"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ValidateOrgRole checks role is owner/admin/member.
func ValidateOrgRole(role string) error {
	switch role {
	case OrgRoleOwner, OrgRoleAdmin, OrgRoleMember:
		return nil
	default:
		return fmt.Errorf("%w: role must be owner, admin, or member", kernel.ErrInvalidRequest)
	}
}

// IsOrgAdminRole reports whether the role grants admin privileges.
func IsOrgAdminRole(role string) bool {
	return OrgRole(role).IsAdmin()
}

// AssertActorCanChangeRoles requires the actor to be the org owner.
func AssertActorCanChangeRoles(actorRole string) error {
	if !OrgRole(actorRole).CanChangeRoles() {
		return kernel.ErrUnauthorized
	}
	return nil
}

// AssertSoleOwnerSafe blocks demoting the last owner.
func AssertSoleOwnerSafe(actorUserID, targetUserID, targetRole, newRole string, ownerCount int) error {
	if targetUserID == actorUserID && newRole != OrgRoleOwner && ownerCount <= 1 {
		return fmt.Errorf("%w: cannot demote the sole owner", kernel.ErrInvalidRequest)
	}
	if targetRole == OrgRoleOwner && newRole != OrgRoleOwner && ownerCount <= 1 && targetUserID != actorUserID {
		return fmt.Errorf("%w: cannot demote the sole owner", kernel.ErrInvalidRequest)
	}
	return nil
}

// ShouldDemoteActorOnOwnerTransfer is true when promoting someone else to owner.
func ShouldDemoteActorOnOwnerTransfer(actorUserID, targetUserID, newRole string) bool {
	return newRole == OrgRoleOwner && targetUserID != actorUserID
}

// NewOrganization builds a validated org entity.
func NewOrganization(id, name string, now time.Time) (*Organization, error) {
	name = strings.TrimSpace(name)
	if id == "" {
		return nil, fmt.Errorf("%w: id required", kernel.ErrInvalidRequest)
	}
	if name == "" {
		return nil, fmt.Errorf("%w: name required", kernel.ErrInvalidRequest)
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return &Organization{ID: id, Name: name, CreatedAt: now.UTC()}, nil
}

// NewTeam builds a validated team entity.
func NewTeam(id, orgID, name string, now time.Time) (*Team, error) {
	name = strings.TrimSpace(name)
	if id == "" || orgID == "" {
		return nil, fmt.Errorf("%w: id and organization_id required", kernel.ErrInvalidRequest)
	}
	if name == "" {
		return nil, fmt.Errorf("%w: name required", kernel.ErrInvalidRequest)
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	return &Team{
		ID: id, TeamID: id, OrganizationID: orgID, Name: name,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

// NewProject builds a validated project entity.
func NewProject(id, orgID, teamID, name string, now time.Time) (*Project, error) {
	name = strings.TrimSpace(name)
	if id == "" || orgID == "" {
		return nil, fmt.Errorf("%w: id and organization_id required", kernel.ErrInvalidRequest)
	}
	if name == "" {
		return nil, fmt.Errorf("%w: name required", kernel.ErrInvalidRequest)
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	return &Project{
		ID: id, OrganizationID: orgID, TeamID: teamID, Name: name,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}
