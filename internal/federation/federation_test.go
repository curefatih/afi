package federation

import (
	"context"
	"testing"
	"time"

	"github.com/curefatih/afi/internal/identity"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/regions"
	"github.com/curefatih/afi/internal/snapshot"
)

type memRepo struct {
	peers     map[string]*ControlPlanePeer
	byHash    map[string]string
	revision  int64
	syncState map[string]*SyncState
}

func newMemRepo() *memRepo {
	return &memRepo{
		peers:     map[string]*ControlPlanePeer{},
		byHash:    map[string]string{},
		syncState: map[string]*SyncState{},
	}
}

func (m *memRepo) CreatePeer(_ context.Context, p ControlPlanePeer) error {
	cp := p
	m.peers[p.ID] = &cp
	m.byHash[p.JoinTokenHash] = p.ID
	return nil
}
func (m *memRepo) GetPeer(_ context.Context, peerID string) (*ControlPlanePeer, error) {
	p, ok := m.peers[peerID]
	if !ok {
		return nil, kernel.ErrNotFound
	}
	cp := *p
	return &cp, nil
}
func (m *memRepo) GetPeerByJoinTokenHash(_ context.Context, hash string) (*ControlPlanePeer, error) {
	id, ok := m.byHash[hash]
	if !ok {
		return nil, kernel.ErrNotFound
	}
	return m.GetPeer(context.Background(), id)
}
func (m *memRepo) ListPeers(context.Context) ([]ControlPlanePeer, error) {
	out := make([]ControlPlanePeer, 0, len(m.peers))
	for _, p := range m.peers {
		out = append(out, *p)
	}
	return out, nil
}
func (m *memRepo) UpdatePeer(_ context.Context, peerID, name, baseURL, status string) (*ControlPlanePeer, error) {
	p, err := m.GetPeer(context.Background(), peerID)
	if err != nil {
		return nil, err
	}
	p.Name, p.BaseURL, p.Status = name, baseURL, status
	p.UpdatedAt = time.Now().UTC()
	m.peers[peerID] = p
	return p, nil
}
func (m *memRepo) UpdatePeerJoinTokenHash(_ context.Context, peerID, hash string) error {
	p, err := m.GetPeer(context.Background(), peerID)
	if err != nil {
		return err
	}
	delete(m.byHash, p.JoinTokenHash)
	p.JoinTokenHash = hash
	m.byHash[hash] = peerID
	m.peers[peerID] = p
	return nil
}
func (m *memRepo) UpdatePeerJoinTokenEnc(_ context.Context, peerID string, enc []byte) error {
	p, err := m.GetPeer(context.Background(), peerID)
	if err != nil {
		return err
	}
	p.JoinTokenEnc = append([]byte(nil), enc...)
	m.peers[peerID] = p
	return nil
}
func (m *memRepo) RecordPeerSync(_ context.Context, peerID string, cursor int64, at time.Time, syncErr string) error {
	p, err := m.GetPeer(context.Background(), peerID)
	if err != nil {
		return err
	}
	p.LastSyncAt = &at
	p.LastSyncCursor = cursor
	p.LastSyncError = syncErr
	m.peers[peerID] = p
	return nil
}
func (m *memRepo) GetRevision(context.Context) (int64, error) { return m.revision, nil }
func (m *memRepo) BumpRevision(context.Context) (int64, error) {
	m.revision++
	return m.revision, nil
}
func (m *memRepo) GetSyncState(_ context.Context, slug string) (*SyncState, error) {
	s, ok := m.syncState[slug]
	if !ok {
		return nil, kernel.ErrNotFound
	}
	cp := *s
	return &cp, nil
}
func (m *memRepo) UpsertSyncState(_ context.Context, st SyncState) error {
	cp := st
	m.syncState[st.RegionSlug] = &cp
	return nil
}

type memRegions struct {
	byID   map[string]*regions.Region
	bySlug map[string]*regions.Region
}

func (m *memRegions) GetRegion(_ context.Context, id string) (*regions.Region, error) {
	r, ok := m.byID[id]
	if !ok {
		return nil, kernel.ErrNotFound
	}
	cp := *r
	return &cp, nil
}
func (m *memRegions) GetRegionBySlug(_ context.Context, slug string) (*regions.Region, error) {
	r, ok := m.bySlug[slug]
	if !ok {
		return nil, kernel.ErrNotFound
	}
	cp := *r
	return &cp, nil
}

func TestRegisterAndJoinPeer(t *testing.T) {
	repo := newMemRepo()
	regs := &memRegions{
		byID:   map[string]*regions.Region{},
		bySlug: map[string]*regions.Region{},
	}
	r := &regions.Region{ID: "reg_1", Slug: "eu-west", Name: "EU", Status: regions.RegionStatusActive}
	regs.byID[r.ID] = r
	regs.bySlug[r.Slug] = r

	out, err := RegisterPeer(context.Background(), repo, regs, "peer_1", "EU CP", "reg_1", "")
	if err != nil {
		t.Fatal(err)
	}
	if out.JoinToken == "" || out.Peer.Status != PeerStatusPending {
		t.Fatalf("%+v", out)
	}
	p, err := Join(context.Background(), repo, out.JoinToken)
	if err != nil {
		t.Fatal(err)
	}
	if p.Status != PeerStatusActive {
		t.Fatalf("status=%s", p.Status)
	}
	if _, err := Join(context.Background(), repo, "bad"); err != kernel.ErrUnauthorized {
		t.Fatalf("got %v", err)
	}
	_ = identity.HashOpaqueToken(out.JoinToken)
}

func TestBuildRegionExportSince(t *testing.T) {
	repo := newMemRepo()
	repo.revision = 5
	regs := &memRegions{
		byID:   map[string]*regions.Region{},
		bySlug: map[string]*regions.Region{},
	}
	r := &regions.Region{ID: "reg_1", Slug: "eu-west", Name: "EU", Status: regions.RegionStatusActive}
	regs.byID[r.ID] = r
	regs.bySlug[r.Slug] = r

	src := &memExport{
		regs: regs,
		mems: []regions.OrgRegionMembership{{OrganizationID: "o1", RegionID: "reg_1", Status: regions.MembershipStatusActive}},
		snap: &snapshot.Snapshot{Version: 9, AllowedOrganizationIDs: []string{"o1"}},
	}
	full, err := BuildRegionExport(context.Background(), repo, src, "eu-west", 0, "snapshots")
	if err != nil {
		t.Fatal(err)
	}
	if full.Revision != 5 || len(full.Memberships) != 1 || full.Snapshot == nil || full.Snapshot.Version != 9 {
		t.Fatalf("%+v", full)
	}
	skip, err := BuildRegionExport(context.Background(), repo, src, "eu-west", 5, "snapshots")
	if err != nil {
		t.Fatal(err)
	}
	if skip.Snapshot != nil || skip.Memberships != nil || skip.Overlays != nil {
		t.Fatalf("expected unchanged skip export: %+v", skip)
	}
}

type memExport struct {
	regs *memRegions
	mems []regions.OrgRegionMembership
	ovs  []regions.RegionConfigOverlay
	snap *snapshot.Snapshot
}

func (m *memExport) GetRegionBySlug(ctx context.Context, slug string) (*regions.Region, error) {
	return m.regs.GetRegionBySlug(ctx, slug)
}
func (m *memExport) ListMembershipsByRegion(context.Context, string) ([]regions.OrgRegionMembership, error) {
	return m.mems, nil
}
func (m *memExport) ListOverlaysByRegion(context.Context, string) ([]regions.RegionConfigOverlay, error) {
	return m.ovs, nil
}
func (m *memExport) LatestRegionalSnapshot(context.Context, string) (*snapshot.Snapshot, error) {
	if m.snap == nil {
		return nil, kernel.ErrNotFound
	}
	return m.snap, nil
}

func TestApplyExport(t *testing.T) {
	repo := newMemRepo()
	target := &memApply{}
	exp := &RegionExport{
		Revision: 3, RegionID: "reg_1", RegionSlug: "eu-west",
		Memberships: []regions.OrgRegionMembership{{OrganizationID: "o1", RegionID: "reg_1", Status: "active"}},
		Overlays:    []regions.RegionConfigOverlay{},
		Snapshot:    &snapshot.Snapshot{Version: 2},
	}
	if err := ApplyExport(context.Background(), repo, target, exp); err != nil {
		t.Fatal(err)
	}
	if !target.ensured || target.memN != 1 || target.snapN != 1 {
		t.Fatalf("%+v", target)
	}
	st, err := repo.GetSyncState(context.Background(), "eu-west")
	if err != nil || st.Cursor != 3 {
		t.Fatalf("%+v %v", st, err)
	}
}

func TestApplyExportSinceSkipDoesNotWipe(t *testing.T) {
	repo := newMemRepo()
	target := &memApply{}
	// Seed cursor via full apply first.
	_ = ApplyExport(context.Background(), repo, target, &RegionExport{
		Revision: 1, RegionID: "reg_1", RegionSlug: "eu-west",
		Memberships: []regions.OrgRegionMembership{{OrganizationID: "o1", RegionID: "reg_1", Status: "active"}},
		Overlays:    []regions.RegionConfigOverlay{},
	})
	target.memN = 0
	skip := &RegionExport{Revision: 1, RegionID: "reg_1", RegionSlug: "eu-west"}
	if err := ApplyExport(context.Background(), repo, target, skip); err != nil {
		t.Fatal(err)
	}
	if target.memN != 0 {
		t.Fatalf("since-skip should not replace memberships, memN=%d", target.memN)
	}
}

type memApply struct {
	ensured bool
	memN    int
	ovN     int
	snapN   int
}

func (m *memApply) EnsureRegion(context.Context, string, string, string) error {
	m.ensured = true
	return nil
}
func (m *memApply) ReplaceMemberships(_ context.Context, _ string, mems []regions.OrgRegionMembership) error {
	m.memN = len(mems)
	return nil
}
func (m *memApply) ReplaceOverlays(_ context.Context, _ string, ovs []regions.RegionConfigOverlay) error {
	m.ovN = len(ovs)
	return nil
}
func (m *memApply) PutSnapshot(context.Context, *snapshot.Snapshot) (int64, error) {
	m.snapN++
	return 1, nil
}
