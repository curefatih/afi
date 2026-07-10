package domain

type Organization struct{ ID string }
type Team struct {
	ID    string
	OrgID string
}
type Project struct {
	ID     string
	TeamID string
	OrgID  string
}
type User struct{ ID string }

type Membership struct {
	UserID   string
	TargetID string // Can be OrgID, TeamID, or ProjectID
	Role     string // Admin, Member, Viewer
}
