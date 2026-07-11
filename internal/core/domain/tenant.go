package domain

import "time"

type Organization struct {
	ID        string
	Name      string
	CreatedAt time.Time
}
type Team struct {
	ID        string
	Name      string
	CreatedAt time.Time
	OrgID     string
}
type Project struct {
	ID        string
	TeamID    string
	OrgID     string
	Name      string
	CreatedAt time.Time
}
type User struct{ ID string }

type Membership struct {
	UserID         string
	OrganizationID string
	TargetID       string // Can be OrgID, TeamID, or ProjectID
	Role           string // Admin, Member, Viewer
	CreatedAt      time.Time
}
