package gatewayconfig

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/kernel"
)

// RequestPolicy is the write-model CEL allow-policy for an organization.
type RequestPolicy struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	Name           string    `json:"name"`
	Expression     string    `json:"expression"`
	Enabled        bool      `json:"enabled"`
	Priority       int       `json:"priority"`
	CreatedAt      time.Time `json:"created_at"`
}

// ExpressionValidator checks CEL expressions without owning the engine.
type ExpressionValidator interface {
	Validate(expression string) error
}

// PolicyRepository persists write-model request policies.
type PolicyRepository interface {
	ListByOrg(ctx context.Context, orgID string) ([]RequestPolicy, error)
	Insert(ctx context.Context, p RequestPolicy) error
	Get(ctx context.Context, policyID string) (*RequestPolicy, error)
	Update(ctx context.Context, p RequestPolicy) (*RequestPolicy, error)
	UpdatePriorities(ctx context.Context, orgID string, items []PolicyPriorityUpdate) error
	Delete(ctx context.Context, policyID string) error
	OrgID(ctx context.Context, policyID string) (string, error)
}

// PolicyPriorityUpdate is one policy's new priority in a batch reorder.
type PolicyPriorityUpdate struct {
	ID       string `json:"id"`
	Priority int    `json:"priority"`
}

// NewRequestPolicy builds a validated policy entity.
func NewRequestPolicy(id, orgID, name, expression string, enabled bool, priority int, now time.Time, v ExpressionValidator) (*RequestPolicy, error) {
	name = strings.TrimSpace(name)
	expression = strings.TrimSpace(expression)
	if id == "" || orgID == "" {
		return nil, fmt.Errorf("%w: id and organization_id required", kernel.ErrInvalidRequest)
	}
	if name == "" || expression == "" {
		return nil, fmt.Errorf("%w: name and expression required", kernel.ErrInvalidRequest)
	}
	if v != nil {
		if err := v.Validate(expression); err != nil {
			return nil, fmt.Errorf("%w: invalid CEL expression: %v", kernel.ErrInvalidRequest, err)
		}
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return &RequestPolicy{
		ID:             id,
		OrganizationID: orgID,
		Name:           name,
		Expression:     expression,
		Enabled:        enabled,
		Priority:       priority,
		CreatedAt:      now.UTC(),
	}, nil
}

// ApplyPolicyPatch mutates a policy with optional field updates.
func ApplyPolicyPatch(cur *RequestPolicy, name, expression *string, enabled *bool, priority *int, v ExpressionValidator) error {
	if cur == nil {
		return kernel.ErrNotFound
	}
	if name != nil {
		n := strings.TrimSpace(*name)
		if n == "" {
			return fmt.Errorf("%w: name required", kernel.ErrInvalidRequest)
		}
		cur.Name = n
	}
	if expression != nil {
		e := strings.TrimSpace(*expression)
		if e == "" {
			return fmt.Errorf("%w: expression required", kernel.ErrInvalidRequest)
		}
		if v != nil {
			if err := v.Validate(e); err != nil {
				return fmt.Errorf("%w: invalid CEL expression: %v", kernel.ErrInvalidRequest, err)
			}
		}
		cur.Expression = e
	}
	if enabled != nil {
		cur.Enabled = *enabled
	}
	if priority != nil {
		cur.Priority = *priority
	}
	return nil
}
