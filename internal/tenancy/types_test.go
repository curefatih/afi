package tenancy

import (
	"errors"
	"testing"

	"github.com/curefatih/afi/internal/kernel"
)

func TestValidateOrgRole(t *testing.T) {
	t.Parallel()
	if err := ValidateOrgRole("boss"); !errors.Is(err, kernel.ErrInvalidRequest) {
		t.Fatalf("err=%v", err)
	}
}

func TestAssertSoleOwnerSafe(t *testing.T) {
	t.Parallel()
	err := AssertSoleOwnerSafe("u1", "u1", OrgRoleOwner, OrgRoleMember, 1)
	if !errors.Is(err, kernel.ErrInvalidRequest) {
		t.Fatalf("err=%v", err)
	}
	if err := AssertSoleOwnerSafe("u1", "u2", OrgRoleOwner, OrgRoleMember, 2); err != nil {
		t.Fatal(err)
	}
}

func TestShouldDemoteActorOnOwnerTransfer(t *testing.T) {
	t.Parallel()
	if !ShouldDemoteActorOnOwnerTransfer("a", "b", OrgRoleOwner) {
		t.Fatal("expected demote")
	}
	if ShouldDemoteActorOnOwnerTransfer("a", "a", OrgRoleOwner) {
		t.Fatal("self promote should not demote")
	}
}
