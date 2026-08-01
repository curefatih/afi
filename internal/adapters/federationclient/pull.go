package federationclient

import (
	"context"
	"errors"
	"time"

	"github.com/curefatih/afi/internal/federation"
	"github.com/curefatih/afi/internal/kernel"
)

// PullOnce joins (best-effort), loads cursor, exports, and applies.
func PullOnce(ctx context.Context, c *Client, repo federation.Repository, target federation.ApplyTarget, regionSlug string) (*federation.RegionExport, error) {
	if _, err := c.Join(ctx); err != nil {
		// Join soft-fails after first success; export still authenticates with token.
		_ = err
	}
	var since int64
	if st, err := repo.GetSyncState(ctx, regionSlug); err == nil && st != nil {
		since = st.Cursor
	} else if err != nil && !errors.Is(err, kernel.ErrNotFound) {
		return nil, err
	}
	exp, err := c.Export(ctx, regionSlug, since)
	if err != nil {
		return nil, err
	}
	if err := federation.ApplyExport(ctx, repo, target, exp); err != nil {
		now := time.Now().UTC()
		_ = repo.UpsertSyncState(ctx, federation.SyncState{
			RegionSlug:    regionSlug,
			Cursor:        since,
			LastSyncAt:    &now,
			LastSyncError: err.Error(),
			UpdatedAt:     now,
		})
		return exp, err
	}
	return exp, nil
}

// RunPullLoop polls the hub until ctx is cancelled.
func RunPullLoop(ctx context.Context, c *Client, repo federation.Repository, target federation.ApplyTarget, regionSlug string, interval time.Duration, onErr func(error)) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	run := func() {
		if _, err := PullOnce(ctx, c, repo, target, regionSlug); err != nil && onErr != nil {
			onErr(err)
		}
	}
	run()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}
