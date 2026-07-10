package domain

// internal/core/domain/budget.go
type BudgetScope string

const (
	ScopeOrg     BudgetScope = "ORGANIZATION"
	ScopeTeam    BudgetScope = "TEAM"
	ScopeProject BudgetScope = "PROJECT"
	ScopeUser    BudgetScope = "USER"
	ScopeKey     BudgetScope = "API_KEY"
)

type BudgetLimit struct {
	Scope    BudgetScope
	TargetID string // Can be OrgID, ProjectID, or KeyHash
	MaxCost  float64
	UsedCost float64
}
