package access

import (
	"errors"
	"testing"

	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/snapshot"
)

func TestValidateKindRulesPersonal(t *testing.T) {
	t.Parallel()
	if err := ValidateKindRules(snapshot.KeyKindPersonal, "", ""); !errors.Is(err, kernel.ErrInvalidRequest) {
		t.Fatalf("err=%v", err)
	}
	if err := ValidateKindRules(snapshot.KeyKindPersonal, "u1", "p1"); !errors.Is(err, kernel.ErrInvalidRequest) {
		t.Fatalf("err=%v", err)
	}
	if err := ValidateKindRules(snapshot.KeyKindPersonal, "u1", ""); err != nil {
		t.Fatal(err)
	}
}

func TestValidateKindRulesServiceAccount(t *testing.T) {
	t.Parallel()
	if err := ValidateKindRules(snapshot.KeyKindServiceAccount, "u1", ""); !errors.Is(err, kernel.ErrInvalidRequest) {
		t.Fatalf("err=%v", err)
	}
	if err := ValidateKindRules(snapshot.KeyKindServiceAccount, "", "p1"); err != nil {
		t.Fatal(err)
	}
}

func TestNewAPIKeyDefaultsKind(t *testing.T) {
	t.Parallel()
	k, err := NewAPIKey("key_1", "org_1", "", "", "", "n", "sk-test-abcdefgh", timeNowUTC())
	if err != nil {
		t.Fatal(err)
	}
	if k.Kind != snapshot.KeyKindServiceAccount {
		t.Fatalf("kind=%q", k.Kind)
	}
}
