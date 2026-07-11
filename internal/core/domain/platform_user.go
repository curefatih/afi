package domain

import "time"

type ActionPermission string

const (
	// Organization Scope Management Permissions
	PermOrgUserRead   ActionPermission = "org:user:read"
	PermOrgUserWrite  ActionPermission = "org:user:write"
	PermOrgRoleWrite  ActionPermission = "org:role:write"
	PermProjectCreate ActionPermission = "org:project:create"
	PermOrgSpendRead  ActionPermission = "org:spend:read"

	// Project Scope Management Permissions
	PermProjectKeyWrite ActionPermission = "project:key:write"
	PermProjectKeyRead  ActionPermission = "project:key:read"
	PermModelRouteWrite ActionPermission = "project:route:write"
	PermLLMInference    ActionPermission = "project:llm:inference"
)

type PlatformUser struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Provider     string    `json:"provider"`
	ExternalID   string    `json:"external_id"`
	CreatedAt    time.Time `json:"created_at"`
}

type CustomRole struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Scope       string             `json:"scope"`
	TargetID    string             `json:"target_id"`
	Permissions []ActionPermission `json:"permissions"`
}

type UserAssignment struct {
	UserID    string `json:"user_id"`
	OrgID     string `json:"org_id"`
	ProjectID string `json:"project_id,omitempty"`
	RoleName  string `json:"role_name"`
}

type Token struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}
