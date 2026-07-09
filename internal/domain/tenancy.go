package domain

import (
	"time"
)

type Organization struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	BudgetCents int64     `json:"budget_cents"`
	CreatedAt   time.Time `json:"created_at"`
}

type Team struct {
	ID          string    `json:"id"`
	OrgID       string    `json:"org_id"`
	Name        string    `json:"name"`
	BudgetCents int64     `json:"budget_cents"`
	CreatedAt   time.Time `json:"created_at"`
}

type Project struct {
	ID        string    `json:"id"`
	TeamID    string    `json:"team_id"`
	OrgID     string    `json:"org_id,omitempty"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	OIDCSub   string    `json:"oidc_sub,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type ProjectMember struct {
	ProjectID string `json:"project_id"`
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
}

type APIKeyRecord struct {
	ID             string
	ProjectID      string
	TeamID         string
	OrgID          string
	Name           string
	ModelAllowlist []string
	RequiredTags   map[string]string
	RateLimitRPM   int
}
