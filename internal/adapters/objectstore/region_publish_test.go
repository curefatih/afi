package objectstore_test

import (
	"context"
	"path"
	"testing"

	"github.com/curefatih/afi/internal/adapters/objectstore"
	"github.com/curefatih/afi/internal/snapshot"
)

// Targeted publish should only write listed region prefixes (simulates PublishRegionSnapshots filter).
func TestForRegionIndependentPrefixes(t *testing.T) {
	blob := &memBlob{}
	root := objectstore.NewSnapshotStore(blob, "snapshots")
	eu := root.ForRegion("eu-west")
	us := root.ForRegion("us-east")

	euSnap := &snapshot.Snapshot{Version: 1, AllowedOrganizationIDs: []string{"org_a"}}
	usSnap := &snapshot.Snapshot{Version: 1, AllowedOrganizationIDs: []string{"org_b"}}
	if _, err := eu.Put(context.Background(), euSnap); err != nil {
		t.Fatal(err)
	}
	if _, err := us.Put(context.Background(), usSnap); err != nil {
		t.Fatal(err)
	}

	euKey := path.Join("snapshots", "eu-west", "1.json")
	usKey := path.Join("snapshots", "us-east", "1.json")
	if _, ok := blob.objects[euKey]; !ok {
		t.Fatalf("missing %s keys=%v", euKey, blob.objects)
	}
	if _, ok := blob.objects[usKey]; !ok {
		t.Fatalf("missing %s", usKey)
	}

	// Updating only EU leaves US blob intact when Put is scoped.
	eu2 := &snapshot.Snapshot{Version: 2, AllowedOrganizationIDs: []string{"org_a", "org_c"}}
	if _, err := eu.Put(context.Background(), eu2); err != nil {
		t.Fatal(err)
	}
	gotUS, err := us.Latest(context.Background())
	if err != nil || gotUS.Version != 1 || len(gotUS.AllowedOrganizationIDs) != 1 {
		t.Fatalf("us should stay at v1: %+v err=%v", gotUS, err)
	}
	gotEU, err := eu.Latest(context.Background())
	if err != nil || gotEU.Version != 2 {
		t.Fatalf("eu=%+v err=%v", gotEU, err)
	}
}
