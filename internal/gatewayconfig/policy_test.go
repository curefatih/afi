package gatewayconfig

import (
	"context"
	"encoding/json"
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
	p, err := NewRequestPolicy("pol_1", "org_1", "allow", "true", []PolicyAction{{Type: ActionDeny}}, true, 10, timeNowUTC(), okValidator{})
	if err != nil || p.Name != "allow" || len(p.Actions) != 1 || p.Actions[0].Type != ActionDeny {
		t.Fatalf("p=%+v err=%v", p, err)
	}
}

func TestNewRequestPolicyMultiActions(t *testing.T) {
	t.Parallel()
	cfg, _ := json.Marshal(map[string]string{"header": "X-A", "value": "1"})
	p, err := NewRequestPolicy("pol_1", "org_1", "multi", "true", []PolicyAction{
		{Type: ActionUseCredential, Config: json.RawMessage(`{"credential_name":"acme"}`)},
		{Type: ActionSetHeader, Config: cfg},
	}, true, 1, timeNowUTC(), okValidator{})
	if err != nil || len(p.Actions) != 2 {
		t.Fatalf("p=%+v err=%v", p, err)
	}
}

func TestNewRequestPolicyRejectsBadCEL(t *testing.T) {
	t.Parallel()
	_, err := NewRequestPolicy("pol_1", "org_1", "allow", "true", []PolicyAction{{Type: ActionDeny}}, true, 1, timeNowUTC(), badValidator{})
	if !errors.Is(err, kernel.ErrInvalidRequest) {
		t.Fatalf("err=%v", err)
	}
}

func TestNewRequestPolicyRequiresAction(t *testing.T) {
	t.Parallel()
	_, err := NewRequestPolicy("pol_1", "org_1", "allow", "true", nil, true, 1, timeNowUTC(), okValidator{})
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
