package snapshot

import "testing"

func TestAllowsOrganization(t *testing.T) {
	global := &Snapshot{} // nil allowlist
	if !global.AllowsOrganization("org_a") {
		t.Fatal("global should allow")
	}
	scoped := &Snapshot{AllowedOrganizationIDs: []string{"org_a"}}
	if !scoped.AllowsOrganization("org_a") {
		t.Fatal("should allow org_a")
	}
	if scoped.AllowsOrganization("org_b") {
		t.Fatal("should deny org_b")
	}
	empty := &Snapshot{AllowedOrganizationIDs: []string{}}
	if empty.AllowsOrganization("org_a") {
		t.Fatal("empty allowlist should deny")
	}
}

func TestCompileRegionSetsAllowlist(t *testing.T) {
	src := Source{
		APIKeys: []APIKey{{KeyHash: "h", OrganizationID: "o1", Name: "k"}},
	}
	s := CompileRegion(src, []string{"o1"})
	if s.AllowedOrganizationIDs == nil || len(s.AllowedOrganizationIDs) != 1 {
		t.Fatalf("%v", s.AllowedOrganizationIDs)
	}
}
