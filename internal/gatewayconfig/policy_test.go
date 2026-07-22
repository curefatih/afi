package gatewayconfig

import (
	"context"
	"errors"
	"testing"

	"github.com/curefatih/afi/internal/kernel"
)

type okValidator struct{}

func (okValidator) Validate(string) error       { return nil }
func (okValidator) ValidateString(string) error { return nil }

type badValidator struct{}

func (badValidator) Validate(string) error       { return errors.New("bad cel") }
func (badValidator) ValidateString(string) error { return errors.New("bad cel") }

func TestNewRequestPolicy(t *testing.T) {
	t.Parallel()
	p, err := NewRequestPolicy("pol_1", "org_1", "allow", "true", ActionDeny, nil, true, 10, timeNowUTC(), okValidator{})
	if err != nil || p.Name != "allow" || p.Action != ActionDeny {
		t.Fatalf("p=%+v err=%v", p, err)
	}
}

func TestNewRequestPolicySetHeader(t *testing.T) {
	t.Parallel()
	p, err := NewRequestPolicy("pol_1", "org_1", "hdr", "true", ActionSetHeader, []byte(`{"header":"X-A","value":"1"}`), true, 1, timeNowUTC(), okValidator{})
	if err != nil || p.Action != ActionSetHeader {
		t.Fatalf("p=%+v err=%v", p, err)
	}
}

func TestNewRequestPolicyRejectsBadCEL(t *testing.T) {
	t.Parallel()
	_, err := NewRequestPolicy("pol_1", "org_1", "allow", "true", ActionDeny, nil, true, 1, timeNowUTC(), badValidator{})
	if !errors.Is(err, kernel.ErrInvalidRequest) {
		t.Fatalf("err=%v", err)
	}
}

type memPolicyRepo struct {
	byID map[string]RequestPolicy
}

func (m *memPolicyRepo) ListByOrg(context.Context, string) ([]RequestPolicy, error) {
	return nil, nil
}
func (m *memPolicyRepo) Insert(context.Context, RequestPolicy) error { return nil }
func (m *memPolicyRepo) Get(context.Context, string) (*RequestPolicy, error) {
	return nil, kernel.ErrNotFound
}
func (m *memPolicyRepo) Update(context.Context, RequestPolicy) (*RequestPolicy, error) {
	return nil, kernel.ErrNotFound
}
func (m *memPolicyRepo) UpdatePriorities(_ context.Context, orgID string, items []PolicyPriorityUpdate) error {
	for _, item := range items {
		p, ok := m.byID[item.ID]
		if !ok || p.OrganizationID != orgID {
			return kernel.ErrNotFound
		}
		p.Priority = item.Priority
		m.byID[item.ID] = p
	}
	return nil
}
func (m *memPolicyRepo) Delete(context.Context, string) error { return kernel.ErrNotFound }
func (m *memPolicyRepo) OrgID(context.Context, string) (string, error) {
	return "", kernel.ErrNotFound
}

func TestReorderPolicies(t *testing.T) {
	t.Parallel()
	repo := &memPolicyRepo{byID: map[string]RequestPolicy{
		"pol_a": {ID: "pol_a", OrganizationID: "org_1", Priority: 10},
		"pol_b": {ID: "pol_b", OrganizationID: "org_1", Priority: 20},
	}}
	err := ReorderPolicies(context.Background(), repo, "org_1", []PolicyPriorityUpdate{
		{ID: "pol_a", Priority: 30},
		{ID: "pol_b", Priority: 20},
	})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if repo.byID["pol_a"].Priority != 30 || repo.byID["pol_b"].Priority != 20 {
		t.Fatalf("priorities=%+v", repo.byID)
	}
}

func TestReorderPoliciesRejectsEmpty(t *testing.T) {
	t.Parallel()
	err := ReorderPolicies(context.Background(), &memPolicyRepo{}, "org_1", nil)
	if !errors.Is(err, kernel.ErrInvalidRequest) {
		t.Fatalf("err=%v", err)
	}
}

func TestReorderPoliciesRejectsDuplicates(t *testing.T) {
	t.Parallel()
	err := ReorderPolicies(context.Background(), &memPolicyRepo{}, "org_1", []PolicyPriorityUpdate{
		{ID: "pol_a", Priority: 1},
		{ID: "pol_a", Priority: 2},
	})
	if !errors.Is(err, kernel.ErrInvalidRequest) {
		t.Fatalf("err=%v", err)
	}
}

func TestReorderPoliciesRejectsForeignPolicy(t *testing.T) {
	t.Parallel()
	repo := &memPolicyRepo{byID: map[string]RequestPolicy{
		"pol_a": {ID: "pol_a", OrganizationID: "org_other", Priority: 10},
	}}
	err := ReorderPolicies(context.Background(), repo, "org_1", []PolicyPriorityUpdate{
		{ID: "pol_a", Priority: 30},
	})
	if !errors.Is(err, kernel.ErrNotFound) {
		t.Fatalf("err=%v", err)
	}
}
