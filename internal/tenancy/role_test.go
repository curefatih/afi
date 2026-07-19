package tenancy

import (
	"errors"
	"testing"

	"github.com/curefatih/afi/internal/kernel"
)

func TestParseOrgRole(t *testing.T) {
	t.Parallel()
	r, err := ParseOrgRole(OrgRoleOwner)
	if err != nil || !r.CanChangeRoles() || !r.IsAdmin() {
		t.Fatalf("owner: %q err=%v", r, err)
	}
	r, err = ParseOrgRole(OrgRoleAdmin)
	if err != nil || r.CanChangeRoles() || !r.IsAdmin() {
		t.Fatalf("admin: %q err=%v", r, err)
	}
	if _, err := ParseOrgRole("boss"); !errors.Is(err, kernel.ErrInvalidRequest) {
		t.Fatalf("err=%v", err)
	}
}
