package federation

import (
	"context"
	"errors"
	"fmt"

	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/regions"
	"github.com/curefatih/afi/internal/snapshot"
)

// ExportSource loads region-scoped data for a federation export.
type ExportSource interface {
	GetRegionBySlug(ctx context.Context, slug string) (*regions.Region, error)
	ListMembershipsByRegion(ctx context.Context, regionID string) ([]regions.OrgRegionMembership, error)
	ListOverlaysByRegion(ctx context.Context, regionID string) ([]regions.RegionConfigOverlay, error)
	LatestRegionalSnapshot(ctx context.Context, regionSlug string) (*snapshot.Snapshot, error)
}

// BuildRegionExport assembles a pull document for the given region slug.
// When since >= current revision (and since > 0), memberships/overlays/snapshot are omitted.
func BuildRegionExport(ctx context.Context, repo Repository, src ExportSource, regionSlug string, since int64, objectPrefix string) (*RegionExport, error) {
	regionSlug = regions.NormalizeRegionSlug(regionSlug)
	if regionSlug == "" {
		return nil, fmt.Errorf("%w: region slug required", kernel.ErrInvalidRequest)
	}
	reg, err := src.GetRegionBySlug(ctx, regionSlug)
	if err != nil {
		return nil, err
	}
	rev, err := repo.GetRevision(ctx)
	if err != nil {
		return nil, err
	}
	prefix := stringsTrimPrefix(objectPrefix)
	out := &RegionExport{
		Revision:   rev,
		RegionSlug: reg.Slug,
		RegionID:   reg.ID,
		SnapshotMeta: SnapshotMeta{
			ObjectPrefix: prefix + "/" + reg.Slug,
		},
	}
	if since > 0 && since >= rev {
		// Nil memberships/overlays/snapshot = unchanged (do not wipe regional tables).
		return out, nil
	}
	mems, err := src.ListMembershipsByRegion(ctx, reg.ID)
	if err != nil {
		return nil, err
	}
	ovs, err := src.ListOverlaysByRegion(ctx, reg.ID)
	if err != nil {
		return nil, err
	}
	if mems == nil {
		mems = []regions.OrgRegionMembership{}
	}
	if ovs == nil {
		ovs = []regions.RegionConfigOverlay{}
	}
	out.Memberships = mems
	out.Overlays = ovs
	snap, err := src.LatestRegionalSnapshot(ctx, reg.Slug)
	if err != nil && !errors.Is(err, kernel.ErrNotFound) {
		return nil, err
	}
	if snap != nil {
		out.Snapshot = snap
		out.SnapshotMeta.Version = snap.Version
	}
	return out, nil
}

func stringsTrimPrefix(prefix string) string {
	for len(prefix) > 0 && (prefix[0] == '/' || prefix[len(prefix)-1] == '/') {
		if prefix[0] == '/' {
			prefix = prefix[1:]
		}
		if len(prefix) > 0 && prefix[len(prefix)-1] == '/' {
			prefix = prefix[:len(prefix)-1]
		}
	}
	if prefix == "" {
		return "snapshots"
	}
	return prefix
}
