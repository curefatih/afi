package domain

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrNoRouteMatched = errors.New("no routing rules matched the request context")
	ErrInvalidRule    = errors.New("routing rule contains invalid conditions or configurations")
)

// MatchOperator defines how a condition evaluates the incoming request metadata.
type MatchOperator string

const (
	OpEquals     MatchOperator = "EQUALS"
	OpNotEquals  MatchOperator = "NOT_EQUALS"
	OpContains   MatchOperator = "CONTAINS"
	OpIn         MatchOperator = "IN"
	OpStartsWith MatchOperator = "STARTS_WITH"
)

// MatchField specifies which field in the RequestContext or InternalRequest to evaluate.
type MatchField string

const (
	FieldOrganizationID MatchField = "OrganizationID"
	FieldTeamID         MatchField = "TeamID"
	FieldProjectID      MatchField = "ProjectID"
	FieldUserID         MatchField = "UserID"
	FieldAPIKeyType     MatchField = "APIKeyType"
	FieldRequestedModel MatchField = "RequestedModel"
)

// Condition defines a single rule predicate (e.g., OrganizationID EQUALS "org_123").
type Condition struct {
	Field    MatchField    `json:"field"`
	Operator MatchOperator `json:"operator"`
	Value    string        `json:"value"` // Comma-separated for the "IN" operator
}

// Evaluate checks if the incoming request payload matches this specific condition.
func (c *Condition) Evaluate(req *InternalRequest) bool {
	var fieldValue string

	switch c.Field {
	case FieldOrganizationID:
		fieldValue = req.Metadata.OrganizationID
	case FieldTeamID:
		fieldValue = req.Metadata.TeamID
	case FieldProjectID:
		fieldValue = req.Metadata.ProjectID
	case FieldUserID:
		fieldValue = req.Metadata.UserID
	case FieldAPIKeyType:
		fieldValue = req.Metadata.APIKeyType
	case FieldRequestedModel:
		fieldValue = req.Model
	default:
		return false
	}

	switch c.Operator {
	case OpEquals:
		return fieldValue == c.Value
	case OpNotEquals:
		return fieldValue != c.Value
	case OpContains:
		return strings.Contains(fieldValue, c.Value)
	case OpStartsWith:
		return strings.HasPrefix(fieldValue, c.Value)
	case OpIn:
		items := strings.Split(c.Value, ",")
		for _, item := range items {
			if fieldValue == strings.TrimSpace(item) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// TargetDestination defines where the traffic should be routed and how it should be transformed.
type TargetDestination struct {
	Provider    string             `json:"provider"`           // e.g., "openai", "anthropic"
	TargetModel string             `json:"target_model"`       // e.g., "claude-3-5-sonnet"
	Fallback    *TargetDestination `json:"fallback,omitempty"` // Nested failover strategy
}

// RoutingRule combines conditions and a destination. It is ordered by priority.
type RoutingRule struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Priority    int               `json:"priority"` // Higher number evaluated first
	IsActive    bool              `json:"is_active"`
	Conditions  []Condition       `json:"conditions"`
	Target      TargetDestination `json:"target"`
}

// Matches checks if all conditions inside the rule evaluate to true (AND logic).
// An empty list of conditions acts as a wildcard catch-all rule.
func (r *RoutingRule) Matches(req *InternalRequest) bool {
	if !r.IsActive {
		return false
	}

	for _, cond := range r.Conditions {
		if !cond.Evaluate(req) {
			return false
		}
	}
	return true
}

// Validate ensures that the rule config contains necessary targets and configurations.
func (r *RoutingRule) Validate() error {
	if r.ID == "" {
		return fmt.Errorf("%w: rule ID cannot be empty", ErrInvalidRule)
	}
	if r.Target.Provider == "" || r.Target.TargetModel == "" {
		return fmt.Errorf("%w: target provider and target model are required", ErrInvalidRule)
	}
	return nil
}
