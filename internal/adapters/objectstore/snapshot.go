package objectstore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/snapshot"
)

// SnapshotStore implements snapshot.Store using an S3-compatible object store.
// Layout: {prefix}/{version}.json and {prefix}/latest (plain version number).
type SnapshotStore struct {
	Blob   Store
	Prefix string
}

func NewSnapshotStore(blob Store, prefix string) *SnapshotStore {
	prefix = strings.Trim(strings.TrimSpace(prefix), "/")
	if prefix == "" {
		prefix = "snapshots"
	}
	return &SnapshotStore{Blob: blob, Prefix: prefix}
}

// ForRegion returns a store rooted at {prefix}/{regionSlug}.
func (s *SnapshotStore) ForRegion(regionSlug string) *SnapshotStore {
	slug := strings.Trim(strings.TrimSpace(regionSlug), "/")
	if slug == "" {
		return s
	}
	return NewSnapshotStore(s.Blob, path.Join(s.Prefix, slug))
}

func (s *SnapshotStore) versionKey(version int64) string {
	return path.Join(s.Prefix, fmt.Sprintf("%d.json", version))
}

func (s *SnapshotStore) latestKey() string {
	return path.Join(s.Prefix, "latest")
}

// Put writes the snapshot blob and updates the latest pointer.
func (s *SnapshotStore) Put(ctx context.Context, snap *snapshot.Snapshot) (int64, error) {
	if s == nil || s.Blob == nil {
		return 0, fmt.Errorf("objectstore snapshot: not configured")
	}
	if snap == nil {
		return 0, fmt.Errorf("objectstore snapshot: nil snapshot")
	}
	if snap.Version <= 0 {
		return 0, fmt.Errorf("objectstore snapshot: version required")
	}
	payload, err := json.Marshal(snap)
	if err != nil {
		return 0, err
	}
	if err := s.Blob.Put(ctx, s.versionKey(snap.Version), bytes.NewReader(payload), int64(len(payload)), PutOptions{
		ContentType: "application/json",
	}); err != nil {
		return 0, err
	}
	ver := []byte(strconv.FormatInt(snap.Version, 10))
	if err := s.Blob.Put(ctx, s.latestKey(), bytes.NewReader(ver), int64(len(ver)), PutOptions{
		ContentType: "text/plain",
	}); err != nil {
		return 0, err
	}
	return snap.Version, nil
}

// Latest loads the snapshot pointed at by the latest pointer.
func (s *SnapshotStore) Latest(ctx context.Context) (*snapshot.Snapshot, error) {
	if s == nil || s.Blob == nil {
		return nil, fmt.Errorf("objectstore snapshot: not configured")
	}
	verBytes, err := s.Blob.Get(ctx, s.latestKey())
	if err != nil {
		return nil, mapNotFound(err)
	}
	version, err := strconv.ParseInt(strings.TrimSpace(string(verBytes)), 10, 64)
	if err != nil || version <= 0 {
		return nil, fmt.Errorf("objectstore snapshot: invalid latest pointer")
	}
	payload, err := s.Blob.Get(ctx, s.versionKey(version))
	if err != nil {
		return nil, mapNotFound(err)
	}
	var snap snapshot.Snapshot
	if err := json.Unmarshal(payload, &snap); err != nil {
		return nil, err
	}
	snap.Version = version
	return &snap, nil
}

// Watch polls for newer snapshots (no push notifications).
func (s *SnapshotStore) Watch(ctx context.Context, pollInterval time.Duration, onUpdate func(*snapshot.Snapshot)) error {
	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}
	current, err := s.Latest(ctx)
	if err != nil && !errors.Is(err, kernel.ErrNotFound) {
		return err
	}
	var currentVersion int64
	if current != nil {
		currentVersion = current.Version
		onUpdate(current)
	}
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			latest, err := s.Latest(ctx)
			if err != nil {
				if errors.Is(err, kernel.ErrNotFound) {
					continue
				}
				return err
			}
			if latest.Version > currentVersion {
				currentVersion = latest.Version
				onUpdate(latest)
			}
		}
	}
}

func mapNotFound(err error) error {
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "nosuchkey") || strings.Contains(msg, "not found") || strings.Contains(msg, "404") {
		return kernel.ErrNotFound
	}
	return err
}

// FanoutStore writes to a primary store then mirrors to an object-store snapshot store.
type FanoutStore struct {
	Primary snapshot.Store
	Mirror  *SnapshotStore
}

func (f *FanoutStore) Put(ctx context.Context, snap *snapshot.Snapshot) (int64, error) {
	version, err := f.Primary.Put(ctx, snap)
	if err != nil {
		return 0, err
	}
	snap.Version = version
	if f.Mirror != nil {
		if _, err := f.Mirror.Put(ctx, snap); err != nil {
			return version, fmt.Errorf("snapshot mirror: %w", err)
		}
	}
	return version, nil
}

func (f *FanoutStore) Latest(ctx context.Context) (*snapshot.Snapshot, error) {
	return f.Primary.Latest(ctx)
}

func (f *FanoutStore) Watch(ctx context.Context, pollInterval time.Duration, onUpdate func(*snapshot.Snapshot)) error {
	return f.Primary.Watch(ctx, pollInterval, onUpdate)
}
