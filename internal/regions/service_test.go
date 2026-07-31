package regions

import (
	"context"
	"testing"
	"time"

	"github.com/curefatih/afi/internal/identity"
	"github.com/curefatih/afi/internal/kernel"
)

type memRepo struct {
	regions     map[string]*Region
	bySlug      map[string]string
	deployments map[string]*GatewayDeployment
	byToken     map[string]string
	memberships map[string]*OrgRegionMembership // regionID|orgID
	overlays    map[string]*RegionConfigOverlay
}

func newMemRepo() *memRepo {
	return &memRepo{
		regions:     map[string]*Region{},
		bySlug:      map[string]string{},
		deployments: map[string]*GatewayDeployment{},
		byToken:     map[string]string{},
		memberships: map[string]*OrgRegionMembership{},
		overlays:    map[string]*RegionConfigOverlay{},
	}
}

func memKey(regionID, orgID string) string { return regionID + "|" + orgID }

func (m *memRepo) CreateRegion(_ context.Context, r Region) error {
	cp := r
	m.regions[r.ID] = &cp
	m.bySlug[r.Slug] = r.ID
	return nil
}

func (m *memRepo) GetRegion(_ context.Context, regionID string) (*Region, error) {
	r, ok := m.regions[regionID]
	if !ok {
		return nil, kernel.ErrNotFound
	}
	cp := *r
	return &cp, nil
}

func (m *memRepo) GetRegionBySlug(_ context.Context, slug string) (*Region, error) {
	id, ok := m.bySlug[slug]
	if !ok {
		return nil, kernel.ErrNotFound
	}
	return m.GetRegion(context.Background(), id)
}

func (m *memRepo) ListRegions(_ context.Context) ([]Region, error) {
	out := make([]Region, 0, len(m.regions))
	for _, r := range m.regions {
		out = append(out, *r)
	}
	return out, nil
}

func (m *memRepo) UpdateRegion(_ context.Context, regionID, name, status string) (*Region, error) {
	r, ok := m.regions[regionID]
	if !ok {
		return nil, kernel.ErrNotFound
	}
	r.Name = name
	r.Status = status
	r.UpdatedAt = time.Now().UTC()
	cp := *r
	return &cp, nil
}

func (m *memRepo) CountDeployments(_ context.Context, regionID string) (int, error) {
	n := 0
	for _, d := range m.deployments {
		if d.RegionID == regionID {
			n++
		}
	}
	return n, nil
}

func (m *memRepo) InsertDeployment(_ context.Context, d GatewayDeployment) error {
	cp := d
	m.deployments[d.ID] = &cp
	m.byToken[d.JoinTokenHash] = d.ID
	return nil
}

func (m *memRepo) GetDeployment(_ context.Context, deploymentID string) (*GatewayDeployment, error) {
	d, ok := m.deployments[deploymentID]
	if !ok {
		return nil, kernel.ErrNotFound
	}
	cp := *d
	return &cp, nil
}

func (m *memRepo) ListDeploymentsByRegion(_ context.Context, regionID string) ([]GatewayDeployment, error) {
	var out []GatewayDeployment
	for _, d := range m.deployments {
		if d.RegionID == regionID {
			out = append(out, *d)
		}
	}
	return out, nil
}

func (m *memRepo) UpdateJoinTokenHash(_ context.Context, deploymentID, joinTokenHash string) error {
	d, ok := m.deployments[deploymentID]
	if !ok {
		return kernel.ErrNotFound
	}
	delete(m.byToken, d.JoinTokenHash)
	d.JoinTokenHash = joinTokenHash
	d.UpdatedAt = time.Now().UTC()
	m.byToken[joinTokenHash] = deploymentID
	return nil
}

func (m *memRepo) RecordHeartbeat(_ context.Context, deploymentID string, snapVersion int64, build string, at time.Time) error {
	d, ok := m.deployments[deploymentID]
	if !ok {
		return kernel.ErrNotFound
	}
	d.Status = DeploymentStatusHealthy
	d.LastSeenAt = &at
	d.ReportedSnapshotVersion = snapVersion
	d.ReportedBuild = build
	d.UpdatedAt = at
	return nil
}

func (m *memRepo) GetDeploymentByJoinTokenHash(_ context.Context, tokenHash string) (*GatewayDeployment, error) {
	id, ok := m.byToken[tokenHash]
	if !ok {
		return nil, kernel.ErrNotFound
	}
	return m.GetDeployment(context.Background(), id)
}

func (m *memRepo) UpdateDeploymentStatus(_ context.Context, deploymentID, status string) error {
	d, ok := m.deployments[deploymentID]
	if !ok {
		return kernel.ErrNotFound
	}
	d.Status = status
	d.UpdatedAt = time.Now().UTC()
	return nil
}

func (m *memRepo) ListMembershipsByRegion(_ context.Context, regionID string) ([]OrgRegionMembership, error) {
	var out []OrgRegionMembership
	for _, x := range m.memberships {
		if x.RegionID == regionID {
			out = append(out, *x)
		}
	}
	return out, nil
}

func (m *memRepo) ListMembershipsByOrg(_ context.Context, orgID string) ([]OrgRegionMembership, error) {
	var out []OrgRegionMembership
	for _, x := range m.memberships {
		if x.OrganizationID == orgID {
			out = append(out, *x)
		}
	}
	return out, nil
}

func (m *memRepo) GetMembership(_ context.Context, regionID, orgID string) (*OrgRegionMembership, error) {
	x, ok := m.memberships[memKey(regionID, orgID)]
	if !ok {
		return nil, kernel.ErrNotFound
	}
	cp := *x
	return &cp, nil
}

func (m *memRepo) UpsertMembership(_ context.Context, mem OrgRegionMembership) error {
	cp := mem
	m.memberships[memKey(mem.RegionID, mem.OrganizationID)] = &cp
	return nil
}

func (m *memRepo) DeleteMembership(_ context.Context, regionID, orgID string) error {
	k := memKey(regionID, orgID)
	if _, ok := m.memberships[k]; !ok {
		return kernel.ErrNotFound
	}
	delete(m.memberships, k)
	return nil
}

func (m *memRepo) GetOverlay(_ context.Context, regionID, orgID string) (*RegionConfigOverlay, error) {
	x, ok := m.overlays[memKey(regionID, orgID)]
	if !ok {
		return nil, kernel.ErrNotFound
	}
	cp := *x
	return &cp, nil
}

func (m *memRepo) ListOverlaysByRegion(_ context.Context, regionID string) ([]RegionConfigOverlay, error) {
	var out []RegionConfigOverlay
	for _, x := range m.overlays {
		if x.RegionID == regionID {
			out = append(out, *x)
		}
	}
	return out, nil
}

func (m *memRepo) UpsertOverlay(_ context.Context, o RegionConfigOverlay) error {
	cp := o
	m.overlays[memKey(o.RegionID, o.OrganizationID)] = &cp
	return nil
}

func (m *memRepo) DeleteOverlay(_ context.Context, regionID, orgID string) error {
	delete(m.overlays, memKey(regionID, orgID))
	return nil
}

func TestCreateRegionUniqueSlug(t *testing.T) {
	repo := newMemRepo()
	if _, err := CreateRegion(context.Background(), repo, "reg_1", "eu-west", "EU West"); err != nil {
		t.Fatal(err)
	}
	_, err := CreateRegion(context.Background(), repo, "reg_2", "EU-WEST", "EU West 2")
	if err == nil {
		t.Fatal("expected duplicate slug error")
	}
}

func TestRegisterAndHeartbeat(t *testing.T) {
	repo := newMemRepo()
	if _, err := CreateRegion(context.Background(), repo, "reg_1", "us-east", "US East"); err != nil {
		t.Fatal(err)
	}
	out, err := RegisterDeployment(context.Background(), repo, "dep_1", "reg_1", "gw-a", "https://gw.example")
	if err != nil {
		t.Fatal(err)
	}
	if out.JoinToken == "" || out.Deployment.Status != DeploymentStatusPending {
		t.Fatalf("unexpected: %+v", out)
	}
	hash := identity.HashOpaqueToken(out.JoinToken)
	d, err := RecordHeartbeat(context.Background(), repo, "dep_1", hash, 42, "dev")
	if err != nil {
		t.Fatal(err)
	}
	if d.Status != DeploymentStatusHealthy || d.ReportedSnapshotVersion != 42 {
		t.Fatalf("heartbeat: %+v", d)
	}
	_, err = RecordHeartbeat(context.Background(), repo, "dep_1", "bad", 43, "")
	if err != kernel.ErrUnauthorized {
		t.Fatalf("want unauthorized, got %v", err)
	}
}

func TestEffectiveStatusStale(t *testing.T) {
	past := time.Now().UTC().Add(-5 * time.Minute)
	d := GatewayDeployment{Status: DeploymentStatusHealthy, LastSeenAt: &past}
	if got := d.EffectiveStatus(time.Now().UTC(), DefaultHeartbeatTTL); got != DeploymentStatusStale {
		t.Fatalf("got %s", got)
	}
	recent := time.Now().UTC().Add(-30 * time.Second)
	d.LastSeenAt = &recent
	if got := d.EffectiveStatus(time.Now().UTC(), DefaultHeartbeatTTL); got != DeploymentStatusHealthy {
		t.Fatalf("got %s", got)
	}
}

func TestBindAllOrganizations(t *testing.T) {
	repo := newMemRepo()
	r, err := CreateRegion(context.Background(), repo, "reg_1", "default", "Default")
	if err != nil {
		t.Fatal(err)
	}
	n, err := BindAllOrganizations(context.Background(), repo, r.ID, []string{"org_a", "org_b", " org_a "})
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("bound=%d", n)
	}
	mems, err := repo.ListMembershipsByRegion(context.Background(), r.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(mems) != 2 {
		t.Fatalf("want 2 unique memberships, got %d", len(mems))
	}
}

func TestDisabledRegionRejectsRegister(t *testing.T) {
	repo := newMemRepo()
	r, err := CreateRegion(context.Background(), repo, "reg_1", "ap-south", "AP South")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := UpdateRegion(context.Background(), repo, r.ID, "", RegionStatusDisabled); err != nil {
		t.Fatal(err)
	}
	_, err = RegisterDeployment(context.Background(), repo, "dep_1", r.ID, "gw", "")
	if err == nil {
		t.Fatal("expected error")
	}
}
