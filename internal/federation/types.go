package federation

import (
	"fmt"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/regions"
	"github.com/curefatih/afi/internal/snapshot"
)

const (
	PeerStatusPending  = "pending"
	PeerStatusActive   = "active"
	PeerStatusDisabled = "disabled"
)

// ControlPlanePeer is a registered regional control plane trusted by the hub.
type ControlPlanePeer struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	RegionID       string     `json:"region_id"`
	BaseURL        string     `json:"base_url,omitempty"`
	Status         string     `json:"status"`
	JoinTokenHash  string     `json:"-"`
	LastSyncAt     *time.Time `json:"last_sync_at,omitempty"`
	LastSyncCursor int64      `json:"last_sync_cursor"`
	LastSyncError  string     `json:"last_sync_error,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// PeerWithToken is returned once when a join token is minted.
type PeerWithToken struct {
	Peer      ControlPlanePeer `json:"peer"`
	JoinToken string           `json:"join_token"`
}

// SyncState is the regional CP's last applied hub revision.
type SyncState struct {
	RegionSlug    string     `json:"region_slug"`
	Cursor        int64      `json:"cursor"`
	LastSyncAt    *time.Time `json:"last_sync_at,omitempty"`
	LastSyncError string     `json:"last_sync_error,omitempty"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// SnapshotMeta locates the compiled regional snapshot on the hub.
type SnapshotMeta struct {
	Version      int64  `json:"version"`
	ObjectPrefix string `json:"object_prefix,omitempty"`
}

// RegionExport is the pull-sync document for one region.
type RegionExport struct {
	Revision     int64                          `json:"revision"`
	RegionSlug   string                         `json:"region_slug"`
	RegionID     string                         `json:"region_id"`
	Memberships  []regions.OrgRegionMembership  `json:"memberships"`
	Overlays     []regions.RegionConfigOverlay  `json:"overlays"`
	SnapshotMeta SnapshotMeta                   `json:"snapshot_meta"`
	// Snapshot is the compiled regional snapshot when available (hub embed for regional put).
	Snapshot *snapshot.Snapshot `json:"snapshot,omitempty"`
}

// ValidatePeerStatus checks peer status.
func ValidatePeerStatus(status string) error {
	switch status {
	case PeerStatusPending, PeerStatusActive, PeerStatusDisabled:
		return nil
	default:
		return fmt.Errorf("%w: status must be pending, active, or disabled", kernel.ErrInvalidRequest)
	}
}

// NewControlPlanePeer builds a validated peer.
func NewControlPlanePeer(id, name, regionID, baseURL, joinTokenHash string, now time.Time) (*ControlPlanePeer, error) {
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	regionID = strings.TrimSpace(regionID)
	baseURL = strings.TrimSpace(baseURL)
	joinTokenHash = strings.TrimSpace(joinTokenHash)
	if id == "" || name == "" || regionID == "" {
		return nil, fmt.Errorf("%w: id, name, and region_id required", kernel.ErrInvalidRequest)
	}
	if joinTokenHash == "" {
		return nil, fmt.Errorf("%w: join token hash required", kernel.ErrInvalidRequest)
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	return &ControlPlanePeer{
		ID: id, Name: name, RegionID: regionID, BaseURL: baseURL,
		Status: PeerStatusPending, JoinTokenHash: joinTokenHash,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}
