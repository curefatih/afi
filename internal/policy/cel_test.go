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
	if err := policy.Validate(`request.model != "blocked"`); err != nil {
		t.Fatal(err)
	}
	key := snapshot.APIKey{ID: "k1", OrganizationID: "o1"}
	policies := []snapshot.Policy{{
		OrganizationID: "o1", Name: "block-model",
		Expression: `request.model == "blocked"`,
		Action:     snapshot.PolicyActionDeny,
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
			Action:     snapshot.PolicyActionAllow,
			Enabled:    true, Priority: 20,
		},
		{
			OrganizationID: "o1", Name: "deny-all",
			Expression: "true",
			Action:     snapshot.PolicyActionDeny,
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

func TestApplySetHeaderAndCredential(t *testing.T) {
	ev, err := policy.NewEvaluator()
	if err != nil {
		t.Fatal(err)
	}
	key := snapshot.APIKey{OrganizationID: "o1"}
	hdrCfg, _ := json.Marshal(map[string]string{"header": "X-Partner", "value": "acme"})
	credCfg, _ := json.Marshal(map[string]string{"credential_name": "partner-acme"})
	policies := []snapshot.Policy{
		{
			OrganizationID: "o1", Name: "use-acme",
			Expression:   `("x-tenant-id" in request.headers) && request.headers["x-tenant-id"] == "acme"`,
			Action:       snapshot.PolicyActionUseCredential,
			ActionConfig: credCfg,
			Enabled:      true, Priority: 20,
		},
		{
			OrganizationID: "o1", Name: "tag-partner",
			Expression:   `("x-tenant-id" in request.headers) && request.headers["x-tenant-id"] == "acme"`,
			Action:       snapshot.PolicyActionSetHeader,
			ActionConfig: hdrCfg,
			Enabled:      true, Priority: 10,
		},
	}
	d, err := ev.Apply(policies, key, policy.Request{
		Model:   "m",
		Headers: map[string]string{"x-tenant-id": "acme"},
	}, policy.Credential{})
	if err != nil || !d.Allowed {
		t.Fatalf("apply: %+v err=%v", d, err)
	}
	if d.CredentialName != "partner-acme" {
		t.Fatalf("credential=%q", d.CredentialName)
	}
	if d.RequestHeaders["X-Partner"] != "acme" {
		t.Fatalf("headers=%v", d.RequestHeaders)
	}
}

func TestApplyDynamicCredentialFromHeader(t *testing.T) {
	ev, err := policy.NewEvaluator()
	if err != nil {
		t.Fatal(err)
	}
	key := snapshot.APIKey{OrganizationID: "o1"}
	credCfg, _ := json.Marshal(map[string]string{
		"credential_name_expr": `request.headers["x-tenant-id"]`,
	})
	policies := []snapshot.Policy{{
		OrganizationID: "o1", Name: "by-header",
		Expression:   `("x-tenant-id" in request.headers) && request.headers["x-tenant-id"] != ""`,
		Action:       snapshot.PolicyActionUseCredential,
		ActionConfig: credCfg,
		Enabled:      true, Priority: 10,
	}}
	d, err := ev.Apply(policies, key, policy.Request{
		Model:   "m",
		Headers: map[string]string{"x-tenant-id": "somecompany"},
	}, policy.Credential{})
	if err != nil || !d.Allowed || d.CredentialName != "somecompany" {
		t.Fatalf("got %+v err=%v", d, err)
	}
}

func TestValidateString(t *testing.T) {
	if err := policy.ValidateString(`request.headers["x-tenant-id"]`); err != nil {
		t.Fatal(err)
	}
	if err := policy.ValidateString(`"literal"`); err != nil {
		t.Fatal(err)
	}
	if err := policy.ValidateString(`1 + 1`); err == nil {
		t.Fatal("expected non-string reject")
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
		Action: snapshot.PolicyActionDeny, Enabled: false, Priority: 1,
	}}
	d, err := ev.Apply(policies, key, policy.Request{Model: "x"}, policy.Credential{})
	if err != nil || !d.Allowed {
		t.Fatalf("%+v err=%v", d, err)
	}
}
