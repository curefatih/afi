package tenancy

import (
	"errors"
	"testing"
	"time"

	"github.com/curefatih/afi/internal/kernel"
)

func TestNewTeam(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	team, err := NewTeam("team_1", "org_1", "  Platform  ", now)
	if err != nil {
		t.Fatal(err)
	}
	if team.ID != "team_1" || team.TeamID != "team_1" || team.OrganizationID != "org_1" || team.Name != "Platform" {
		t.Fatalf("%+v", team)
	}
	if _, err := NewTeam("", "org_1", "Platform", now); !errors.Is(err, kernel.ErrInvalidRequest) {
		t.Fatalf("err=%v", err)
	}
	if _, err := NewTeam("team_1", "org_1", "  ", now); !errors.Is(err, kernel.ErrInvalidRequest) {
		t.Fatalf("err=%v", err)
	}
}

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
