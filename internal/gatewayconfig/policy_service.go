package gatewayconfig

import (
	"context"
	"fmt"

	"github.com/curefatih/afi/internal/kernel"
)

// CreatePolicy validates and persists a new request policy.
func CreatePolicy(
	ctx context.Context,
	repo PolicyRepository,
	validator ExpressionValidator,
	id, orgID, name, expression string,
	actions []PolicyAction,
	enabled bool,
	priority int,
) (*RequestPolicy, error) {
	p, err := NewRequestPolicy(id, orgID, name, expression, actions, enabled, priority, timeNowUTC(), validator)
	if err != nil {
		return nil, err
	}
	if err := repo.Insert(ctx, *p); err != nil {
		return nil, err
	}
	return p, nil
}

// UpdatePolicy loads, patches, and persists a request policy.
func UpdatePolicy(
	ctx context.Context,
	repo PolicyRepository,
	validator ExpressionValidator,
	policyID string,
	name, expression *string,
	actions []PolicyAction,
	enabled *bool,
	priority *int,
) (*RequestPolicy, error) {
	cur, err := repo.Get(ctx, policyID)
	if err != nil {
		return nil, err
	}
	if err := ApplyPolicyPatch(cur, name, expression, actions, enabled, priority, validator); err != nil {
		return nil, err
	}
	return repo.Update(ctx, *cur)
}

// ReorderPolicies validates and applies a batch of priority updates for one org.
func ReorderPolicies(ctx context.Context, repo PolicyRepository, orgID string, items []PolicyPriorityUpdate) error {
	if orgID == "" {
		return fmt.Errorf("%w: organization_id required", kernel.ErrInvalidRequest)
	}
	if len(items) == 0 {
		return fmt.Errorf("%w: items required", kernel.ErrInvalidRequest)
	}
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		if item.ID == "" {
			return fmt.Errorf("%w: policy id required", kernel.ErrInvalidRequest)
		}
		if _, ok := seen[item.ID]; ok {
			return fmt.Errorf("%w: duplicate policy id %s", kernel.ErrInvalidRequest, item.ID)
		}
		seen[item.ID] = struct{}{}
	}
	return repo.UpdatePriorities(ctx, orgID, items)
}
