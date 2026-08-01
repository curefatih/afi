package regions_test

import (
	"testing"

	"github.com/curefatih/afi/internal/regions"
	"github.com/curefatih/afi/internal/snapshot"
)

// Success criteria: org A bound only to eu → us regional source has no A keys/routes.
func TestRegionIsolationOrgAOnlyInEU(t *testing.T) {
	t.Parallel()
	base := snapshot.Source{
		APIKeys: []snapshot.APIKey{
			{ID: "ka", KeyHash: "ha", OrganizationID: "org_a", Name: "a"},
			{ID: "kb", KeyHash: "hb", OrganizationID: "org_b", Name: "b"},
		},
		Routes: []snapshot.Route{
			{OrganizationID: "org_a", Model: "m-a", ProviderID: "prov_a", TargetModel: "m-a"},
			{OrganizationID: "org_b", Model: "m-b", ProviderID: "prov_b", TargetModel: "m-b"},
		},
		Providers: []snapshot.Provider{
			{ID: "prov_a", Type: "openai", Name: "A"},
			{ID: "prov_b", Type: "openai", Name: "B"},
		},
	}
	euMem := []regions.OrgRegionMembership{
		{OrganizationID: "org_a", RegionID: "eu", Status: regions.MembershipStatusActive},
	}
	usMem := []regions.OrgRegionMembership{
		{OrganizationID: "org_b", RegionID: "us", Status: regions.MembershipStatusActive},
	}
	euSrc, euAllow := regions.BuildRegionSource(base, euMem, nil)
	usSrc, usAllow := regions.BuildRegionSource(base, usMem, nil)
	euSnap := snapshot.CompileRegion(euSrc, euAllow)
	usSnap := snapshot.CompileRegion(usSrc, usAllow)

	if !euSnap.AllowsOrganization("org_a") || euSnap.AllowsOrganization("org_b") {
		t.Fatalf("eu allowlist=%v", euSnap.AllowedOrganizationIDs)
	}
	if usSnap.AllowsOrganization("org_a") || !usSnap.AllowsOrganization("org_b") {
		t.Fatalf("us allowlist=%v", usSnap.AllowedOrganizationIDs)
	}
	if _, ok := euSnap.APIKeys["ha"]; !ok {
		t.Fatal("eu missing org_a key")
	}
	if _, ok := usSnap.APIKeys["ha"]; ok {
		t.Fatal("us should not have org_a key")
	}
	if _, _, ok := usSnap.LookupRoute("org_a", "m-a"); ok {
		t.Fatal("us should not have org_a route")
	}
}

// Success criteria: overlay replaces routes in EU; US inherit unaffected.
func TestOverlayReplaceEUOnly(t *testing.T) {
	t.Parallel()
	base := snapshot.Source{
		APIKeys: []snapshot.APIKey{
			{ID: "ka", KeyHash: "ha", OrganizationID: "org_a", Name: "a"},
		},
		Routes: []snapshot.Route{
			{OrganizationID: "org_a", Model: "base-model", ProviderID: "prov", TargetModel: "base"},
		},
		Providers: []snapshot.Provider{{ID: "prov", Type: "openai", Name: "P"}},
	}
	mem := []regions.OrgRegionMembership{
		{OrganizationID: "org_a", RegionID: "eu", Status: regions.MembershipStatusActive},
	}
	ov := []regions.RegionConfigOverlay{{
		OrganizationID: "org_a", RegionID: "eu",
		Payload: regions.OverlayPayload{
			Routes: []snapshot.Route{
				{OrganizationID: "org_a", Model: "eu-only", ProviderID: "prov", TargetModel: "eu"},
			},
			Providers: []snapshot.Provider{{ID: "prov", Type: "openai", Name: "P"}},
		},
	}}
	euSrc, euAllow := regions.BuildRegionSource(base, mem, ov)
	inheritSrc, inheritAllow := regions.BuildRegionSource(base, mem, nil)
	euSnap := snapshot.CompileRegion(euSrc, euAllow)
	inheritSnap := snapshot.CompileRegion(inheritSrc, inheritAllow)

	if _, _, ok := euSnap.LookupRoute("org_a", "eu-only"); !ok {
		t.Fatal("overlay route missing")
	}
	if _, _, ok := euSnap.LookupRoute("org_a", "base-model"); ok {
		t.Fatal("base route should be replaced away")
	}
	if _, _, ok := inheritSnap.LookupRoute("org_a", "base-model"); !ok {
		t.Fatal("inherit should keep base route")
	}
	if _, _, ok := inheritSnap.LookupRoute("org_a", "eu-only"); ok {
		t.Fatal("inherit should not see overlay route")
	}
}

// Regression: global Compile has nil allowlist → unrestricted.
func TestGlobalCompileUnrestrictedAllowlist(t *testing.T) {
	t.Parallel()
	src := snapshot.Source{
		APIKeys: []snapshot.APIKey{
			{ID: "k", KeyHash: "h", OrganizationID: "org_x", Name: "x"},
		},
	}
	s := snapshot.Compile(src)
	if s.AllowedOrganizationIDs != nil {
		t.Fatalf("global allowlist should be nil, got %v", s.AllowedOrganizationIDs)
	}
	if !s.AllowsOrganization("org_x") || !s.AllowsOrganization("any") {
		t.Fatal("global snapshot must allow all orgs")
	}
}
