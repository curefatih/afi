package federation

import (
	"context"
	"time"
)

// Repository persists federation peers and sync metadata on the hub (and regional sync state).
type Repository interface {
	CreatePeer(ctx context.Context, p ControlPlanePeer) error
	GetPeer(ctx context.Context, peerID string) (*ControlPlanePeer, error)
	GetPeerByJoinTokenHash(ctx context.Context, hash string) (*ControlPlanePeer, error)
	ListPeers(ctx context.Context) ([]ControlPlanePeer, error)
	UpdatePeer(ctx context.Context, peerID, name, baseURL, status string) (*ControlPlanePeer, error)
	UpdatePeerJoinTokenHash(ctx context.Context, peerID, hash string) error
	// UpdatePeerJoinTokenEnc stores the hub-sealed join token used for usage report pulls.
	UpdatePeerJoinTokenEnc(ctx context.Context, peerID string, enc []byte) error
	RecordPeerSync(ctx context.Context, peerID string, cursor int64, at time.Time, syncErr string) error

	GetRevision(ctx context.Context) (int64, error)
	BumpRevision(ctx context.Context) (int64, error)

	GetSyncState(ctx context.Context, regionSlug string) (*SyncState, error)
	UpsertSyncState(ctx context.Context, st SyncState) error
}
