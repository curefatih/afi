package federation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/identity"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/regions"
)

var timeNowUTC = func() time.Time { return time.Now().UTC() }

// RegionLookup resolves regions for peer registration and export.
type RegionLookup interface {
	GetRegion(ctx context.Context, regionID string) (*regions.Region, error)
	GetRegionBySlug(ctx context.Context, slug string) (*regions.Region, error)
}

// RegisterPeer creates a regional CP peer and returns the join token once.
func RegisterPeer(ctx context.Context, repo Repository, lookup RegionLookup, id, name, regionID, baseURL string) (*PeerWithToken, error) {
	reg, err := lookup.GetRegion(ctx, regionID)
	if err != nil {
		return nil, err
	}
	if reg.Status == regions.RegionStatusDisabled {
		return nil, fmt.Errorf("%w: region is disabled", kernel.ErrInvalidRequest)
	}
	raw, hash, err := identity.NewOpaqueToken()
	if err != nil {
		return nil, err
	}
	p, err := NewControlPlanePeer(id, name, regionID, baseURL, hash, timeNowUTC())
	if err != nil {
		return nil, err
	}
	if err := repo.CreatePeer(ctx, *p); err != nil {
		return nil, err
	}
	return &PeerWithToken{Peer: *p, JoinToken: raw}, nil
}

// RotatePeerJoinToken issues a new join token.
func RotatePeerJoinToken(ctx context.Context, repo Repository, peerID string) (*PeerWithToken, error) {
	p, err := repo.GetPeer(ctx, peerID)
	if err != nil {
		return nil, err
	}
	raw, hash, err := identity.NewOpaqueToken()
	if err != nil {
		return nil, err
	}
	if err := repo.UpdatePeerJoinTokenHash(ctx, peerID, hash); err != nil {
		return nil, err
	}
	p.JoinTokenHash = hash
	p.UpdatedAt = timeNowUTC()
	p.Status = PeerStatusPending
	return &PeerWithToken{Peer: *p, JoinToken: raw}, nil
}

// UpdatePeer patches name, base URL, and/or status.
func UpdatePeer(ctx context.Context, repo Repository, peerID, name, baseURL, status string) (*ControlPlanePeer, error) {
	cur, err := repo.GetPeer(ctx, peerID)
	if err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	baseURL = strings.TrimSpace(baseURL)
	status = strings.TrimSpace(status)
	if name == "" {
		name = cur.Name
	}
	if baseURL == "" {
		baseURL = cur.BaseURL
	}
	if status == "" {
		status = cur.Status
	}
	if err := ValidatePeerStatus(status); err != nil {
		return nil, err
	}
	return repo.UpdatePeer(ctx, peerID, name, baseURL, status)
}

// AuthenticatePeerToken resolves a peer by raw join token and marks it active on first use.
func AuthenticatePeerToken(ctx context.Context, repo Repository, rawToken string) (*ControlPlanePeer, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return nil, kernel.ErrUnauthorized
	}
	hash := identity.HashOpaqueToken(rawToken)
	p, err := repo.GetPeerByJoinTokenHash(ctx, hash)
	if err != nil {
		if errors.Is(err, kernel.ErrNotFound) {
			return nil, kernel.ErrUnauthorized
		}
		return nil, err
	}
	if p.Status == PeerStatusDisabled {
		return nil, kernel.ErrUnauthorized
	}
	if p.Status == PeerStatusPending {
		updated, err := repo.UpdatePeer(ctx, p.ID, p.Name, p.BaseURL, PeerStatusActive)
		if err != nil {
			return nil, err
		}
		return updated, nil
	}
	return p, nil
}

// Join exchanges a join token for peer identity (activates pending peers).
func Join(ctx context.Context, repo Repository, rawToken string) (*ControlPlanePeer, error) {
	return AuthenticatePeerToken(ctx, repo, rawToken)
}

// BumpRevision advances the hub federation revision (call after membership/overlay/publish).
func BumpRevision(ctx context.Context, repo Repository) (int64, error) {
	return repo.BumpRevision(ctx)
}
