package postgres_test

import (
	"errors"
	"testing"

	"github.com/curefatih/afi/internal/adapters/postgres"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/regions"
	"github.com/curefatih/afi/internal/snapshot"
)

func TestRegionOverlayChangesCompiledSource(t *testing.T) {
	pool := testPool(t)
	resetDB(t, pool)
	store := postgres.NewStore(pool)
	ctx := t.Context()

	seedUserOrg(t, pool, "u1", "org_a")
	seedUserOrg(t, pool, "u2", "org_b")

	prov, err := store.CreateProvider(ctx, "org_a", "OpenAI", "openai", "https://api.openai.com/v1", "OPENAI_API_KEY", snapshot.ProviderCapabilities{Chat: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateRoute(ctx, "org_a", "base-model", prov.ID, "gpt-4o-mini", nil, nil, "", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateAPIKey(ctx, "org_a", "service_account", "", "", "", "key-a", "sk-org-a-test-key"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateAPIKey(ctx, "org_b", "service_account", "", "", "", "key-b", "sk-org-b-test-key"); err != nil {
		t.Fatal(err)
	}

	eu, err := store.CreateRegion(ctx, "eu-west", "EU West")
	if err != nil {
		t.Fatal(err)
	}
	us, err := store.CreateRegion(ctx, "us-east", "US East")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.BindOrgToRegion(ctx, eu.ID, "org_a", regions.MembershipStatusActive); err != nil {
		t.Fatal(err)
	}
	if _, err := store.BindOrgToRegion(ctx, us.ID, "org_b", regions.MembershipStatusActive); err != nil {
		t.Fatal(err)
	}
	if _, err := store.PutRegionOverlay(ctx, eu.ID, "org_a", regions.OverlayPayload{
		Providers: []snapshot.Provider{{
			ID: prov.ID, Name: "OpenAI", Type: "openai", BaseURL: "https://api.openai.com/v1",
		}},
		Routes: []snapshot.Route{{
			Model: "eu-only", ProviderID: prov.ID, TargetModel: "gpt-4o-mini",
		}},
	}); err != nil {
		t.Fatal(err)
	}

	base, err := store.LoadSnapshotSource(ctx)
	if err != nil {
		t.Fatal(err)
	}
	euMem, err := store.ListRegionMemberships(ctx, eu.ID)
	if err != nil {
		t.Fatal(err)
	}
	usMem, err := store.ListRegionMemberships(ctx, us.ID)
	if err != nil {
		t.Fatal(err)
	}
	ov, err := store.GetRegionOverlay(ctx, eu.ID, "org_a")
	if err != nil {
		t.Fatal(err)
	}
	euOv := []regions.RegionConfigOverlay{*ov}
	if _, err := store.GetRegionOverlay(ctx, us.ID, "org_b"); err != nil && !errors.Is(err, kernel.ErrNotFound) {
		t.Fatal(err)
	}

	euSrc, euAllow := regions.BuildRegionSource(base, euMem, euOv)
	usSrc, usAllow := regions.BuildRegionSource(base, usMem, nil)
	euSnap := snapshot.CompileRegion(euSrc, euAllow)
	usSnap := snapshot.CompileRegion(usSrc, usAllow)

	if !euSnap.AllowsOrganization("org_a") || euSnap.AllowsOrganization("org_b") {
		t.Fatalf("eu allow=%v", euSnap.AllowedOrganizationIDs)
	}
	if usSnap.AllowsOrganization("org_a") || !usSnap.AllowsOrganization("org_b") {
		t.Fatalf("us allow=%v", usSnap.AllowedOrganizationIDs)
	}
	if _, _, ok := euSnap.LookupRoute("org_a", "eu-only"); !ok {
		t.Fatal("eu overlay route missing")
	}
	if _, _, ok := euSnap.LookupRoute("org_a", "base-model"); ok {
		t.Fatal("eu should not keep base route after overlay")
	}
	if _, ok := usSnap.APIKeys[snapshot.HashKey("sk-org-a-test-key")]; ok {
		t.Fatal("us should not include org_a key")
	}
}
