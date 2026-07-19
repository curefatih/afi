package policy_test

import (
	"testing"

	"github.com/curefatih/afi/internal/policy"
	"github.com/curefatih/afi/internal/snapshot"
)

func TestValidateAndEvaluate(t *testing.T) {
	ev, err := policy.NewEvaluator()
	if err != nil {
		t.Fatal(err)
	}
	if err := policy.Validate(`request.model != "blocked"`); err != nil {
		t.Fatal(err)
	}
	if err := policy.Validate(`1 + 1`); err == nil {
		t.Fatal("expected non-bool reject")
	}

	key := snapshot.APIKey{ID: "k1", OrganizationID: "o1", ProjectID: "p1", Kind: snapshot.KeyKindServiceAccount}
	policies := []snapshot.Policy{{
		ID: "1", OrganizationID: "o1", Name: "block-model", Expression: `request.model != "blocked"`,
		Enabled: true, Priority: 10,
	}}
	ok, name, err := ev.Evaluate(policies, key, policy.Request{Model: "echo-demo", Path: "/v1/chat/completions"})
	if err != nil || !ok || name != "" {
		t.Fatalf("allow: ok=%v name=%q err=%v", ok, name, err)
	}
	ok, name, err = ev.Evaluate(policies, key, policy.Request{Model: "blocked", Path: "/v1/chat/completions"})
	if err != nil || ok || name != "block-model" {
		t.Fatalf("deny: ok=%v name=%q err=%v", ok, name, err)
	}
}

func TestDisabledPolicySkipped(t *testing.T) {
	ev, err := policy.NewEvaluator()
	if err != nil {
		t.Fatal(err)
	}
	key := snapshot.APIKey{OrganizationID: "o1"}
	policies := []snapshot.Policy{{
		OrganizationID: "o1", Name: "deny-all", Expression: "false", Enabled: false, Priority: 1,
	}}
	ok, _, err := ev.Evaluate(policies, key, policy.Request{Model: "x"})
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}
