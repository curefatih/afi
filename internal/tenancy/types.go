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
	TeamRoleAdmin  = "admin"
	TeamRoleMember = "member"
)

// Organization is a tenant root.
type Organization struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	MailProvider       string    `json:"mail_provider,omitempty"`
	BYOKSelectorHeader string    `json:"byok_selector_header,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}

const (
	InviteStatusPending  = "pending"
	InviteStatusAccepted = "accepted"
	InviteStatusRevoked  = "revoked"
	InviteStatusExpired  = "expired"
)

// OrgInvite is a pending (or historical) organization membership invite.
type OrgInvite struct {
	ID              string     `json:"id"`
	OrganizationID  string     `json:"organization_id"`
	Email           string     `json:"email"`
	Role            string     `json:"role"`
	InvitedByUserID string     `json:"invited_by_user_id"`
	Status          string     `json:"status"`
	ExpiresAt       time.Time  `json:"expires_at"`
	CreatedAt       time.Time  `json:"created_at"`
	AcceptedAt      *time.Time `json:"accepted_at,omitempty"`
}

// InviteOutcome is returned by InviteOrgMember.
type InviteOutcome struct {
	Status string     `json:"status"` // "added" | "invited"
	Member *OrgMember `json:"member,omitempty"`
	Invite *OrgInvite `json:"invite,omitempty"`
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

// ValidateTeamRole checks role is owner/admin/member.
func ValidateTeamRole(role string) error {
	switch role {
	case TeamRoleOwner, TeamRoleAdmin, TeamRoleMember:
		return nil
	default:
		return fmt.Errorf("%w: role must be owner, admin, or member", kernel.ErrInvalidRequest)
	}
}

// IsTeamManagerRole reports whether the team role may manage team members.
func IsTeamManagerRole(role string) bool {
	return TeamRole(role).IsManager()
}

// AssertSoleTeamOwnerSafe blocks demoting the last team owner.
func AssertSoleTeamOwnerSafe(targetRole, newRole string, ownerCount int) error {
	if targetRole == TeamRoleOwner && newRole != TeamRoleOwner && ownerCount <= 1 {
		return fmt.Errorf("%w: cannot demote the sole team owner", kernel.ErrInvalidRequest)
	}
	return nil
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
