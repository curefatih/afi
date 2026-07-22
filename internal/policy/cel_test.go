package policy_test

import (
	"encoding/json"
	"testing"

	"github.com/curefatih/afi/internal/policy"
	"github.com/curefatih/afi/internal/snapshot"
)

func TestApplyDeny(t *testing.T) {
	ev, err := policy.NewEvaluator()
	if err != nil {
		t.Fatal(err)
	}
	key := snapshot.APIKey{ID: "k1", OrganizationID: "o1"}
	policies := []snapshot.Policy{{
		OrganizationID: "o1", Name: "block-model",
		Expression: `request.model == "blocked"`,
		Actions:    []snapshot.PolicyAction{{Type: snapshot.PolicyActionDeny}},
		Enabled:    true, Priority: 10,
	}}
	d, err := ev.Apply(policies, key, policy.Request{Model: "echo-demo"}, policy.Credential{})
	if err != nil || !d.Allowed {
		t.Fatalf("allow: %+v err=%v", d, err)
	}
	d, err = ev.Apply(policies, key, policy.Request{Model: "blocked"}, policy.Credential{})
	if err != nil || d.Allowed || d.DeniedBy != "block-model" {
		t.Fatalf("deny: %+v err=%v", d, err)
	}
}

func TestApplyMultipleThenActions(t *testing.T) {
	ev, err := policy.NewEvaluator()
	if err != nil {
		t.Fatal(err)
	}
	key := snapshot.APIKey{OrganizationID: "o1"}
	hdrCfg, _ := json.Marshal(map[string]string{"header": "X-Partner", "value_expr": `request.headers["x-tenant-id"]`})
	credCfg, _ := json.Marshal(map[string]string{"credential_name_expr": `request.headers["x-tenant-id"]`})
	policies := []snapshot.Policy{{
		OrganizationID: "o1", Name: "partner",
		Expression: `("x-tenant-id" in request.headers) && request.headers["x-tenant-id"] != ""`,
		Actions: []snapshot.PolicyAction{
			{Type: snapshot.PolicyActionUseCredential, Config: credCfg},
			{Type: snapshot.PolicyActionSetHeader, Config: hdrCfg},
		},
		Enabled: true, Priority: 10,
	}}
	d, err := ev.Apply(policies, key, policy.Request{
		Model:   "m",
		Headers: map[string]string{"x-tenant-id": "acme"},
	}, policy.Credential{})
	if err != nil || !d.Allowed {
		t.Fatalf("%+v err=%v", d, err)
	}
	if d.CredentialName != "acme" {
		t.Fatalf("credential=%q", d.CredentialName)
	}
	if d.RequestHeaders["X-Partner"] != "acme" {
		t.Fatalf("headers=%v", d.RequestHeaders)
	}
}

func TestApplyAllowShortCircuit(t *testing.T) {
	ev, err := policy.NewEvaluator()
	if err != nil {
		t.Fatal(err)
	}
	key := snapshot.APIKey{OrganizationID: "o1"}
	policies := []snapshot.Policy{
		{
			OrganizationID: "o1", Name: "vip-allow",
			Expression: `("x-vip" in request.headers) && request.headers["x-vip"] == "1"`,
			Actions:    []snapshot.PolicyAction{{Type: snapshot.PolicyActionAllow}},
			Enabled:    true, Priority: 20,
		},
		{
			OrganizationID: "o1", Name: "deny-all",
			Expression: "true",
			Actions:    []snapshot.PolicyAction{{Type: snapshot.PolicyActionDeny}},
			Enabled:    true, Priority: 10,
		},
	}
	d, err := ev.Apply(policies, key, policy.Request{
		Model:   "m",
		Headers: map[string]string{"x-vip": "1"},
	}, policy.Credential{})
	if err != nil || !d.Allowed || d.MatchedAllowName != "vip-allow" {
		t.Fatalf("vip: %+v err=%v", d, err)
	}
	d, err = ev.Apply(policies, key, policy.Request{Model: "m"}, policy.Credential{})
	if err != nil || d.Allowed || d.DeniedBy != "deny-all" {
		t.Fatalf("deny: %+v err=%v", d, err)
	}
}

func TestDisabledPolicySkipped(t *testing.T) {
	ev, err := policy.NewEvaluator()
	if err != nil {
		t.Fatal(err)
	}
	key := snapshot.APIKey{OrganizationID: "o1"}
	policies := []snapshot.Policy{{
		OrganizationID: "o1", Name: "deny-all", Expression: "true",
		Actions: []snapshot.PolicyAction{{Type: snapshot.PolicyActionDeny}},
		Enabled: false, Priority: 1,
	}}
	d, err := ev.Apply(policies, key, policy.Request{Model: "x"}, policy.Credential{})
	if err != nil || !d.Allowed {
		t.Fatalf("%+v err=%v", d, err)
	}
}
