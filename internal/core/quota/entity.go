package quota

import "github.com/shopspring/decimal"

type Scope string

const (
	ScopeOrganization Scope = "ORGANIZATION"
	ScopeTeam         Scope = "TEAM"
	ScopeProject      Scope = "PROJECT"
	ScopeUser         Scope = "USER"
	ScopeAPIKey       Scope = "API_KEY"
)

type Quota struct {
	ID string

	Scope Scope

	TargetID string

	Metric Metric

	Limit decimal.Decimal

	Used decimal.Decimal

	Enabled bool
}
