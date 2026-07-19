package snapshot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/curefatih/afi/internal/kernel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const channelName = "afi_snapshot"

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Put(ctx context.Context, snap *Snapshot) (int64, error) {
	payload, err := json.Marshal(snap)
	if err != nil {
		return 0, fmt.Errorf("marshal snapshot: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	var version int64
	err = tx.QueryRow(ctx, `
		INSERT INTO gateway_snapshots (payload)
		VALUES ($1)
		RETURNING version
	`, payload).Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("insert snapshot: %w", err)
	}

	if _, err := tx.Exec(ctx, `SELECT pg_notify($1, $2)`, channelName, fmt.Sprintf("%d", version)); err != nil {
		return 0, fmt.Errorf("notify: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}

	snap.Version = version
	return version, nil
}

func (s *Store) Latest(ctx context.Context) (*Snapshot, error) {
	var version int64
	var payload []byte
	var createdAt time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT version, payload, created_at
		FROM gateway_snapshots
		ORDER BY version DESC
		LIMIT 1
	`).Scan(&version, &payload, &createdAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	var snap Snapshot
	if err := json.Unmarshal(payload, &snap); err != nil {
		return nil, fmt.Errorf("unmarshal snapshot: %w", err)
	}
	snap.Version = version
	snap.CreatedAt = createdAt
	return &snap, nil
}

// Watch calls onUpdate whenever a newer snapshot appears.
// Polls at pollInterval and also wakes on Postgres LISTEN/NOTIFY.
func (s *Store) Watch(ctx context.Context, pollInterval time.Duration, onUpdate func(*Snapshot)) error {
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

	notifyCh := make(chan struct{}, 1)
	go s.listenLoop(ctx, notifyCh)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	check := func() error {
		latest, err := s.Latest(ctx)
		if err != nil {
			if errors.Is(err, kernel.ErrNotFound) {
				return nil
			}
			return err
		}
		if latest.Version > currentVersion {
			currentVersion = latest.Version
			onUpdate(latest)
		}
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := check(); err != nil {
				return err
			}
		case <-notifyCh:
			if err := check(); err != nil {
				return err
			}
		}
	}
}

func (s *Store) listenLoop(ctx context.Context, notifyCh chan<- struct{}) {
	for {
		if ctx.Err() != nil {
			return
		}
		conn, err := s.pool.Acquire(ctx)
		if err != nil {
			return
		}
		if _, err := conn.Exec(ctx, "LISTEN "+channelName); err != nil {
			conn.Release()
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
				continue
			}
		}
		for {
			_, err := conn.Conn().WaitForNotification(ctx)
			if err != nil {
				conn.Release()
				break
			}
			select {
			case notifyCh <- struct{}{}:
			default:
			}
		}
	}
}
