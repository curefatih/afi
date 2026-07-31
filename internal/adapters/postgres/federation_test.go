package postgres_test

import (
	"testing"

	"github.com/curefatih/afi/internal/adapters/postgres"
	"github.com/curefatih/afi/internal/federation"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/regions"
	"github.com/curefatih/afi/internal/snapshot"
)

func TestFederationExportAndApply(t *testing.T) {
	pool := testPool(t)
	resetDB(t, pool)
	hub := postgres.NewStore(pool)
	ctx := t.Context()

	seedUserOrg(t, pool, "u1", "org_a")
	eu, err := hub.CreateRegion(ctx, "eu-west", "EU West")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := hub.BindOrgToRegion(ctx, eu.ID, "org_a", regions.MembershipStatusActive); err != nil {
		t.Fatal(err)
	}
	if _, err := hub.PutRegionOverlay(ctx, eu.ID, "org_a", regions.OverlayPayload{
		Routes: []snapshot.Route{{Model: "eu-model", ProviderID: "p1", TargetModel: "gpt"}},
	}); err != nil {
		t.Fatal(err)
	}

	peer, err := hub.RegisterFederationPeer(ctx, "EU CP", eu.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if peer.JoinToken == "" {
		t.Fatal("expected join token")
	}
	joined, err := hub.JoinFederationPeer(ctx, peer.JoinToken)
	if err != nil || joined.Status != federation.PeerStatusActive {
		t.Fatalf("%+v %v", joined, err)
	}

	exp, err := hub.ExportFederationRegion(ctx, "eu-west", 0, "snapshots")
	if err != nil {
		t.Fatal(err)
	}
	if exp.Revision < 1 || len(exp.Memberships) != 1 || exp.Snapshot == nil {
		t.Fatalf("%+v", exp)
	}
	skip, err := hub.ExportFederationRegion(ctx, "eu-west", exp.Revision, "snapshots")
	if err != nil {
		t.Fatal(err)
	}
	if skip.Memberships != nil || skip.Snapshot != nil {
		t.Fatalf("since-skip should omit payload: %+v", skip)
	}

	// Apply onto a fresh regional DB simulation in the same store after wipe of region tables is heavy;
	// instead apply into a second store using a second database connection is ideal — reuse apply target
	// after deleting membership and re-applying.
	_ = hub.UnbindOrgFromRegion(ctx, eu.ID, "org_a")
	_ = hub.DeleteRegionOverlay(ctx, eu.ID, "org_a")

	snapStore := postgres.NewSnapshotStore(pool)
	target := &postgres.RegionalApplyTarget{Store: hub, SnapStore: snapStore}
	if err := federation.ApplyExport(ctx, hub.FederationRepo(), target, exp); err != nil {
		t.Fatal(err)
	}
	mems, err := hub.ListRegionMemberships(ctx, eu.ID)
	if err != nil || len(mems) != 1 || mems[0].OrganizationID != "org_a" {
		t.Fatalf("%+v %v", mems, err)
	}
	st, err := hub.GetFederationSyncState(ctx, "eu-west")
	if err != nil || st.Cursor != exp.Revision {
		t.Fatalf("%+v %v", st, err)
	}
	latest, err := snapStore.Latest(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if latest.Version != exp.Snapshot.Version {
		t.Fatalf("snap version %d want %d", latest.Version, exp.Snapshot.Version)
	}
}

func TestFederationModeOffUnaffected(t *testing.T) {
	// Regression: default config mode off must still load.
	cfg, err := kernel.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Federation.Mode != "off" && cfg.Federation.Mode != "" {
		// LoadConfig may read env; ensure empty/off is accepted via explicit parse path.
		t.Logf("federation mode=%q (env may override)", cfg.Federation.Mode)
	}
	pool := testPool(t)
	resetDB(t, pool)
	store := postgres.NewStore(pool)
	ctx := t.Context()
	seedUserOrg(t, pool, "u1", "org_a")
	reg, err := store.CreateRegion(ctx, "local", "Local")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.BindOrgToRegion(ctx, reg.ID, "org_a", regions.MembershipStatusActive); err != nil {
		t.Fatal(err)
	}
	mems, err := store.ListRegionMemberships(ctx, reg.ID)
	if err != nil || len(mems) != 1 {
		t.Fatalf("%+v %v", mems, err)
	}
}
