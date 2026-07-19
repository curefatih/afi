package gatewayconfig

import "context"

// CreatePolicy validates and persists a new request policy.
func CreatePolicy(
	ctx context.Context,
	repo PolicyRepository,
	validator ExpressionValidator,
	id, orgID, name, expression string,
	enabled bool,
	priority int,
) (*RequestPolicy, error) {
	p, err := NewRequestPolicy(id, orgID, name, expression, enabled, priority, timeNowUTC(), validator)
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
	enabled *bool,
	priority *int,
) (*RequestPolicy, error) {
	cur, err := repo.Get(ctx, policyID)
	if err != nil {
		return nil, err
	}
	if err := ApplyPolicyPatch(cur, name, expression, enabled, priority, validator); err != nil {
		return nil, err
	}
	return repo.Update(ctx, *cur)
}
