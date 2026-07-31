package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/adapters/objectstore"
	"github.com/curefatih/afi/internal/federation"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/regions"
	"github.com/curefatih/afi/internal/snapshot"
)

func federationNow() time.Time { return time.Now().UTC() }

func (s *Store) federationRepo() *FederationStore {
	return NewFederationStore(s.pool)
}

// FederationRepo exposes the federation repository for the regional puller.
func (s *Store) FederationRepo() *FederationStore {
	return s.federationRepo()
}

func (s *Store) ListFederationPeers(ctx context.Context) ([]federation.ControlPlanePeer, error) {
	return s.federationRepo().ListPeers(ctx)
}

func (s *Store) GetFederationPeer(ctx context.Context, peerID string) (*federation.ControlPlanePeer, error) {
	return s.federationRepo().GetPeer(ctx, peerID)
}

func (s *Store) RegisterFederationPeer(ctx context.Context, name, regionID, baseURL string) (*federation.PeerWithToken, error) {
	return federation.RegisterPeer(ctx, s.federationRepo(), s, newPlatformID("peer"), name, regionID, baseURL)
}

func (s *Store) UpdateFederationPeer(ctx context.Context, peerID, name, baseURL, status string) (*federation.ControlPlanePeer, error) {
	return federation.UpdatePeer(ctx, s.federationRepo(), peerID, name, baseURL, status)
}

func (s *Store) RotateFederationPeerJoinToken(ctx context.Context, peerID string) (*federation.PeerWithToken, error) {
	return federation.RotatePeerJoinToken(ctx, s.federationRepo(), peerID)
}

func (s *Store) AuthenticateFederationPeerToken(ctx context.Context, rawToken string) (*federation.ControlPlanePeer, error) {
	return federation.AuthenticatePeerToken(ctx, s.federationRepo(), rawToken)
}

func (s *Store) JoinFederationPeer(ctx context.Context, rawToken string) (*federation.ControlPlanePeer, error) {
	return federation.Join(ctx, s.federationRepo(), rawToken)
}

func (s *Store) BumpFederationRevision(ctx context.Context) (int64, error) {
	return s.federationRepo().BumpRevision(ctx)
}

func (s *Store) GetFederationRevision(ctx context.Context) (int64, error) {
	return s.federationRepo().GetRevision(ctx)
}

func (s *Store) GetFederationSyncState(ctx context.Context, regionSlug string) (*federation.SyncState, error) {
	return s.federationRepo().GetSyncState(ctx, regionSlug)
}

func (s *Store) UpsertFederationSyncState(ctx context.Context, st federation.SyncState) error {
	return s.federationRepo().UpsertSyncState(ctx, st)
}

func (s *Store) RecordFederationPeerSync(ctx context.Context, peerID string, cursor int64, syncErr string) error {
	return s.federationRepo().RecordPeerSync(ctx, peerID, cursor, federationNow(), syncErr)
}

func (s *Store) ExportFederationRegion(ctx context.Context, slug string, since int64, objectPrefix string) (*federation.RegionExport, error) {
	src := &federationExportSource{store: s, mirror: s.federationMirror}
	return federation.BuildRegionExport(ctx, s.federationRepo(), src, slug, since, objectPrefix)
}

type federationExportSource struct {
	store  *Store
	mirror *objectstore.SnapshotStore
}

func (e *federationExportSource) GetRegionBySlug(ctx context.Context, slug string) (*regions.Region, error) {
	return e.store.GetRegionBySlug(ctx, slug)
}

func (e *federationExportSource) ListMembershipsByRegion(ctx context.Context, regionID string) ([]regions.OrgRegionMembership, error) {
	return e.store.regionsRepo().ListMembershipsByRegion(ctx, regionID)
}

func (e *federationExportSource) ListOverlaysByRegion(ctx context.Context, regionID string) ([]regions.RegionConfigOverlay, error) {
	return e.store.regionsRepo().ListOverlaysByRegion(ctx, regionID)
}

func (e *federationExportSource) LatestRegionalSnapshot(ctx context.Context, regionSlug string) (*snapshot.Snapshot, error) {
	if e.mirror != nil {
		snap, err := e.mirror.ForRegion(regionSlug).Latest(ctx)
		if err == nil {
			return snap, nil
		}
		if err != nil && !errors.Is(err, kernel.ErrNotFound) {
			return nil, err
		}
	}
	return e.store.compileRegionalSnapshot(ctx, regionSlug)
}

func (s *Store) compileRegionalSnapshot(ctx context.Context, regionSlug string) (*snapshot.Snapshot, error) {
	reg, err := s.GetRegionBySlug(ctx, regionSlug)
	if err != nil {
		return nil, err
	}
	base, err := s.LoadSnapshotSource(ctx)
	if err != nil {
		return nil, err
	}
	mems, err := s.regionsRepo().ListMembershipsByRegion(ctx, reg.ID)
	if err != nil {
		return nil, err
	}
	ovs, err := s.regionsRepo().ListOverlaysByRegion(ctx, reg.ID)
	if err != nil {
		return nil, err
	}
	src, allowed := regions.BuildRegionSource(base, mems, ovs)
	return snapshot.CompileRegion(src, allowed), nil
}

// RegionalApplyTarget applies federation exports onto local Phase 2 tables + snapshot store.
type RegionalApplyTarget struct {
	Store     *Store
	SnapStore snapshot.Store
}

func (t *RegionalApplyTarget) EnsureRegion(ctx context.Context, id, slug, name string) error {
	if _, err := t.Store.GetRegion(ctx, id); err == nil {
		return nil
	}
	if _, err := t.Store.GetRegionBySlug(ctx, slug); err == nil {
		return nil
	}
	// Insert region directly so IDs match the hub.
	return t.Store.regionsRepo().CreateRegion(ctx, regions.Region{
		ID: id, Slug: slug, Name: name, Status: regions.RegionStatusActive,
		CreatedAt: federationNow(), UpdatedAt: federationNow(),
	})
}

func (t *RegionalApplyTarget) ReplaceMemberships(ctx context.Context, regionID string, mems []regions.OrgRegionMembership) error {
	repo := t.Store.regionsRepo()
	existing, err := repo.ListMembershipsByRegion(ctx, regionID)
	if err != nil {
		return err
	}
	keep := map[string]struct{}{}
	for _, m := range mems {
		if err := t.ensureOrgStub(ctx, m.OrganizationID); err != nil {
			return err
		}
		keep[m.OrganizationID] = struct{}{}
		if err := repo.UpsertMembership(ctx, m); err != nil {
			return err
		}
	}
	for _, m := range existing {
		if _, ok := keep[m.OrganizationID]; !ok {
			_ = repo.DeleteOverlay(ctx, regionID, m.OrganizationID)
			_ = repo.DeleteMembership(ctx, regionID, m.OrganizationID)
		}
	}
	return nil
}

func (t *RegionalApplyTarget) ReplaceOverlays(ctx context.Context, regionID string, overlays []regions.RegionConfigOverlay) error {
	repo := t.Store.regionsRepo()
	existing, err := repo.ListOverlaysByRegion(ctx, regionID)
	if err != nil {
		return err
	}
	keep := map[string]struct{}{}
	for _, o := range overlays {
		if err := t.ensureOrgStub(ctx, o.OrganizationID); err != nil {
			return err
		}
		keep[o.OrganizationID] = struct{}{}
		if err := repo.UpsertOverlay(ctx, o); err != nil {
			return err
		}
	}
	for _, o := range existing {
		if _, ok := keep[o.OrganizationID]; !ok {
			_ = repo.DeleteOverlay(ctx, regionID, o.OrganizationID)
		}
	}
	return nil
}

func (t *RegionalApplyTarget) ensureOrgStub(ctx context.Context, orgID string) error {
	orgID = strings.TrimSpace(orgID)
	if orgID == "" {
		return fmt.Errorf("%w: organization_id required", kernel.ErrInvalidRequest)
	}
	if _, err := t.Store.GetOrganization(ctx, orgID); err == nil {
		return nil
	}
	_, err := t.Store.pool.Exec(ctx, `
		INSERT INTO organizations (id, name, created_at) VALUES ($1, $2, NOW())
		ON CONFLICT (id) DO NOTHING
	`, orgID, "federated:"+orgID)
	return err
}

func (t *RegionalApplyTarget) PutSnapshot(ctx context.Context, snap *snapshot.Snapshot) (int64, error) {
	if t.SnapStore == nil {
		return 0, fmt.Errorf("snapshot store not configured")
	}
	return t.SnapStore.Put(ctx, snap)
}
