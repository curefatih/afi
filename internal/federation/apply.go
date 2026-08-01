package federation

import (
	"context"
	"fmt"
	"time"

	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/regions"
	"github.com/curefatih/afi/internal/snapshot"
)

// ApplyTarget applies a pulled export onto a regional control plane.
type ApplyTarget interface {
	EnsureRegion(ctx context.Context, id, slug, name string) error
	ReplaceMemberships(ctx context.Context, regionID string, mems []regions.OrgRegionMembership) error
	ReplaceOverlays(ctx context.Context, regionID string, overlays []regions.RegionConfigOverlay) error
	PutSnapshot(ctx context.Context, snap *snapshot.Snapshot) (int64, error)
}

// ApplyExport upserts region topology + memberships/overlays and stores the embedded snapshot.
func ApplyExport(ctx context.Context, repo Repository, target ApplyTarget, exp *RegionExport) error {
	if exp == nil {
		return fmt.Errorf("%w: nil export", kernel.ErrInvalidRequest)
	}
	if exp.RegionID == "" || exp.RegionSlug == "" {
		return fmt.Errorf("%w: region_id and region_slug required", kernel.ErrInvalidRequest)
	}
	if err := target.EnsureRegion(ctx, exp.RegionID, exp.RegionSlug, exp.RegionSlug); err != nil {
		return err
	}
	// Nil slices mean unchanged (since-skip); non-nil (including empty) means full replace.
	if exp.Memberships != nil {
		if err := target.ReplaceMemberships(ctx, exp.RegionID, exp.Memberships); err != nil {
			return err
		}
	}
	if exp.Overlays != nil {
		if err := target.ReplaceOverlays(ctx, exp.RegionID, exp.Overlays); err != nil {
			return err
		}
	}
	if exp.Snapshot != nil {
		if _, err := target.PutSnapshot(ctx, exp.Snapshot); err != nil {
			return err
		}
	}
	now := time.Now().UTC()
	return repo.UpsertSyncState(ctx, SyncState{
		RegionSlug: exp.RegionSlug,
		Cursor:     exp.Revision,
		LastSyncAt: &now,
		UpdatedAt:  now,
	})
}
