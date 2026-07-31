package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/curefatih/afi/internal/federation"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FederationStore implements federation.Repository.
type FederationStore struct {
	Pool *pgxpool.Pool
}

func NewFederationStore(pool *pgxpool.Pool) *FederationStore {
	return &FederationStore{Pool: pool}
}

func (s *FederationStore) CreatePeer(ctx context.Context, p federation.ControlPlanePeer) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO federation_peers (
			id, name, region_id, base_url, status, join_token_hash,
			last_sync_cursor, last_sync_error, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
	`, p.ID, p.Name, p.RegionID, p.BaseURL, p.Status, p.JoinTokenHash,
		p.LastSyncCursor, p.LastSyncError, p.CreatedAt, p.UpdatedAt)
	return err
}

func (s *FederationStore) scanPeer(row pgx.Row) (*federation.ControlPlanePeer, error) {
	var p federation.ControlPlanePeer
	var lastSeen *time.Time
	err := row.Scan(
		&p.ID, &p.Name, &p.RegionID, &p.BaseURL, &p.Status, &p.JoinTokenHash,
		&lastSeen, &p.LastSyncCursor, &p.LastSyncError, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	p.LastSyncAt = lastSeen
	return &p, nil
}

func (s *FederationStore) GetPeer(ctx context.Context, peerID string) (*federation.ControlPlanePeer, error) {
	return s.scanPeer(s.Pool.QueryRow(ctx, `
		SELECT id, name, region_id, base_url, status, join_token_hash,
			last_sync_at, last_sync_cursor, last_sync_error, created_at, updated_at
		FROM federation_peers WHERE id = $1
	`, peerID))
}

func (s *FederationStore) GetPeerByJoinTokenHash(ctx context.Context, hash string) (*federation.ControlPlanePeer, error) {
	return s.scanPeer(s.Pool.QueryRow(ctx, `
		SELECT id, name, region_id, base_url, status, join_token_hash,
			last_sync_at, last_sync_cursor, last_sync_error, created_at, updated_at
		FROM federation_peers WHERE join_token_hash = $1
	`, hash))
}

func (s *FederationStore) ListPeers(ctx context.Context) ([]federation.ControlPlanePeer, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, name, region_id, base_url, status, join_token_hash,
			last_sync_at, last_sync_cursor, last_sync_error, created_at, updated_at
		FROM federation_peers ORDER BY created_at
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []federation.ControlPlanePeer
	for rows.Next() {
		var p federation.ControlPlanePeer
		var lastSeen *time.Time
		if err := rows.Scan(
			&p.ID, &p.Name, &p.RegionID, &p.BaseURL, &p.Status, &p.JoinTokenHash,
			&lastSeen, &p.LastSyncCursor, &p.LastSyncError, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		p.LastSyncAt = lastSeen
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *FederationStore) UpdatePeer(ctx context.Context, peerID, name, baseURL, status string) (*federation.ControlPlanePeer, error) {
	_, err := s.Pool.Exec(ctx, `
		UPDATE federation_peers SET name=$2, base_url=$3, status=$4, updated_at=NOW()
		WHERE id=$1
	`, peerID, name, baseURL, status)
	if err != nil {
		return nil, err
	}
	return s.GetPeer(ctx, peerID)
}

func (s *FederationStore) UpdatePeerJoinTokenHash(ctx context.Context, peerID, hash string) error {
	tag, err := s.Pool.Exec(ctx, `
		UPDATE federation_peers SET join_token_hash=$2, status='pending', updated_at=NOW()
		WHERE id=$1
	`, peerID, hash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return kernel.ErrNotFound
	}
	return nil
}

func (s *FederationStore) RecordPeerSync(ctx context.Context, peerID string, cursor int64, at time.Time, syncErr string) error {
	tag, err := s.Pool.Exec(ctx, `
		UPDATE federation_peers
		SET last_sync_at=$2, last_sync_cursor=$3, last_sync_error=$4, updated_at=NOW()
		WHERE id=$1
	`, peerID, at, cursor, syncErr)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return kernel.ErrNotFound
	}
	return nil
}

func (s *FederationStore) GetRevision(ctx context.Context) (int64, error) {
	var rev int64
	err := s.Pool.QueryRow(ctx, `SELECT revision FROM federation_meta WHERE id = 1`).Scan(&rev)
	if errors.Is(err, pgx.ErrNoRows) {
		_, _ = s.Pool.Exec(ctx, `INSERT INTO federation_meta (id, revision) VALUES (1, 0) ON CONFLICT DO NOTHING`)
		return 0, nil
	}
	return rev, err
}

func (s *FederationStore) BumpRevision(ctx context.Context) (int64, error) {
	var rev int64
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO federation_meta (id, revision) VALUES (1, 1)
		ON CONFLICT (id) DO UPDATE SET revision = federation_meta.revision + 1
		RETURNING revision
	`).Scan(&rev)
	return rev, err
}

func (s *FederationStore) GetSyncState(ctx context.Context, regionSlug string) (*federation.SyncState, error) {
	var st federation.SyncState
	var last *time.Time
	err := s.Pool.QueryRow(ctx, `
		SELECT region_slug, cursor, last_sync_at, last_sync_error, updated_at
		FROM federation_sync_state WHERE region_slug = $1
	`, regionSlug).Scan(&st.RegionSlug, &st.Cursor, &last, &st.LastSyncError, &st.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	st.LastSyncAt = last
	return &st, nil
}

func (s *FederationStore) UpsertSyncState(ctx context.Context, st federation.SyncState) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO federation_sync_state (region_slug, cursor, last_sync_at, last_sync_error, updated_at)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (region_slug) DO UPDATE
		SET cursor = EXCLUDED.cursor, last_sync_at = EXCLUDED.last_sync_at,
			last_sync_error = EXCLUDED.last_sync_error, updated_at = EXCLUDED.updated_at
	`, st.RegionSlug, st.Cursor, st.LastSyncAt, st.LastSyncError, st.UpdatedAt)
	return err
}
