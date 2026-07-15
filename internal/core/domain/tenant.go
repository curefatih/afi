package domain

import "time"

type Organization struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}
type Team struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	OrgID     string    `json:"org_id"`
}
type Project struct {
	ID        string    `json:"id"`
	TeamID    string    `json:"team_id"`
	OrgID     string    `json:"org_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}
type User struct {
	ID string `json:"id"`
}

type Membership struct {
	UserID         string    `json:"user_id"`
	OrganizationID string    `json:"organization_id,omitempty"`
	TargetID       string    `json:"target_id,omitempty"` // Could be OrgID, TeamID, or ProjectID
	Role           string    `json:"role"`                // Admin, Member, Viewer
	CreatedAt      time.Time `json:"created_at"`
}
