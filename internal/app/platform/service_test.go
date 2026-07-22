package platform_test

import (
	"context"
	"testing"
	"time"

	"github.com/curefatih/afi/internal/access"
	"github.com/curefatih/afi/internal/app/platform"
	"github.com/curefatih/afi/internal/credentials"
	"github.com/curefatih/afi/internal/gatewayconfig"
	"github.com/curefatih/afi/internal/identity"
	"github.com/curefatih/afi/internal/snapshot"
	"github.com/curefatih/afi/internal/tenancy"
	"github.com/curefatih/afi/internal/usage"
)

type memSnap struct{ n int }

func (m *memSnap) PublishSnapshot(context.Context) error { m.n++; return nil }

type memEvents struct{ names []platform.EventName }

func (m *memEvents) Record(_ context.Context, e platform.Event) {
	m.names = append(m.names, e.Name)
}

type memAPI struct {
	created int
	keys    []access.APIKey
	invite  *tenancy.InviteOutcome
	team    *tenancy.Team
}

func (m *memAPI) ListOrganizationsForUser(context.Context, string) ([]tenancy.Organization, error) {
	return nil, nil
}
func (m *memAPI) CreateOrganization(_ context.Context, name, _ string) (*tenancy.Organization, error) {
	return &tenancy.Organization{ID: "org_1", Name: name}, nil
}
func (m *memAPI) ListOrgMembers(context.Context, string) ([]tenancy.OrgMember, error) {
	return nil, nil
}
func (m *memAPI) UpdateOrgMemberRole(context.Context, string, string, string, string) (*tenancy.OrgMember, error) {
	return &tenancy.OrgMember{UserID: "u1", Role: "admin"}, nil
}
func (m *memAPI) InviteOrgMember(context.Context, string, string, string) (*tenancy.InviteOutcome, string, error) {
	if m.invite != nil {
		return m.invite, "raw-token", nil
	}
	return &tenancy.InviteOutcome{
		Status: "invited",
		Invite: &tenancy.OrgInvite{ID: "inv_1", OrganizationID: "org_1"},
	}, "raw-token", nil
}
func (m *memAPI) ListOrgInvites(context.Context, string) ([]tenancy.OrgInvite, error) {
	return nil, nil
}
func (m *memAPI) RevokeOrgInvite(context.Context, string, string) error { return nil }
func (m *memAPI) ResendOrgInvite(context.Context, string, string) (*tenancy.OrgInvite, string, error) {
	return &tenancy.OrgInvite{ID: "inv_1"}, "raw-token", nil
}
func (m *memAPI) PreviewOrgInvite(context.Context, string) (*tenancy.InvitePreview, error) {
	return &tenancy.InvitePreview{OrganizationID: "org_1", Email: "a@b.com"}, nil
}
func (m *memAPI) AcceptOrgInvite(context.Context, string, string, string) (*tenancy.OrgMember, *identity.User, error) {
	return &tenancy.OrgMember{UserID: "u1"}, &identity.User{ID: "u1", Email: "a@b.com"}, nil
}
func (m *memAPI) ListTeams(context.Context, string, string) ([]tenancy.Team, error) { return nil, nil }
func (m *memAPI) CreateTeam(_ context.Context, orgID, name, _ string) (*tenancy.Team, error) {
	return &tenancy.Team{ID: "team_1", OrganizationID: orgID, Name: name}, nil
}
func (m *memAPI) GetTeam(_ context.Context, teamID string) (*tenancy.Team, error) {
	if m.team != nil {
		return m.team, nil
	}
	return &tenancy.Team{ID: teamID, OrganizationID: "org_1"}, nil
}
func (m *memAPI) ListTeamMembers(context.Context, string) ([]tenancy.TeamMember, error) {
	return nil, nil
}
func (m *memAPI) AddTeamMember(context.Context, string, string) (*tenancy.TeamMember, error) {
	return &tenancy.TeamMember{UserID: "u2", Role: "member"}, nil
}
func (m *memAPI) UpdateTeamMemberRole(context.Context, string, string, string, string) (*tenancy.TeamMember, error) {
	return &tenancy.TeamMember{UserID: "u2", Role: "admin"}, nil
}
func (m *memAPI) RemoveTeamMember(context.Context, string, string) error { return nil }
func (m *memAPI) ListProjects(context.Context, string, string) ([]tenancy.Project, error) {
	return nil, nil
}
func (m *memAPI) CreateProject(context.Context, string, string, string) (*tenancy.Project, error) {
	panic("unused")
}
func (m *memAPI) ListAPIKeys(context.Context, string) ([]access.APIKey, error) { return nil, nil }
func (m *memAPI) ListOrgAPIKeys(context.Context, string) ([]access.APIKey, error) {
	return m.keys, nil
}
func (m *memAPI) CreateAPIKey(context.Context, string, string, string, string, string, string) (*access.APIKey, error) {
	panic("unused")
}
func (m *memAPI) DeleteAPIKey(context.Context, string) error { panic("unused") }
func (m *memAPI) ListProviders(context.Context, string) ([]gatewayconfig.Provider, error) {
	return nil, nil
}
func (m *memAPI) ListProviderHealth(context.Context, string, time.Time, time.Time) ([]usage.ProviderHealth, error) {
	return nil, nil
}
func (m *memAPI) CreateProvider(context.Context, string, string, string, string, string, snapshot.ProviderCapabilities) (*gatewayconfig.Provider, error) {
	panic("unused")
}
func (m *memAPI) UpdateProvider(context.Context, string, string, string, string) (*gatewayconfig.Provider, error) {
	panic("unused")
}
func (m *memAPI) DeleteProvider(context.Context, string) error                      { panic("unused") }
func (m *memAPI) ListRoutes(context.Context, string) ([]gatewayconfig.Route, error) { return nil, nil }
func (m *memAPI) CreateRoute(context.Context, string, string, string, string, []gatewayconfig.RouteFallback, *gatewayconfig.RetryConfig) (*gatewayconfig.Route, error) {
	panic("unused")
}
func (m *memAPI) UpdateRoute(context.Context, string, string, string, string, []gatewayconfig.RouteFallback, *gatewayconfig.RetryConfig) (*gatewayconfig.Route, error) {
	panic("unused")
}
func (m *memAPI) DeleteRoute(context.Context, string) error { panic("unused") }
func (m *memAPI) ListUsage(context.Context, string, usage.Filter) ([]usage.Record, error) {
	return nil, nil
}
func (m *memAPI) SummarizeUsage(context.Context, string, usage.Filter) ([]usage.SummaryBucket, error) {
	return nil, nil
}
func (m *memAPI) ListQuotas(context.Context, string) ([]gatewayconfig.Quota, error) { return nil, nil }
func (m *memAPI) CreateQuota(_ context.Context, orgID, scopeType, scopeID, metric string, limitValue int64, window string) (*gatewayconfig.Quota, error) {
	m.created++
	return &gatewayconfig.Quota{
		ID: "quota_1", OrganizationID: orgID, ScopeType: scopeType, ScopeID: scopeID,
		Metric: metric, LimitValue: limitValue, Window: window,
	}, nil
}
func (m *memAPI) UpdateQuota(context.Context, string, int64) (*gatewayconfig.Quota, error) {
	panic("unused")
}
func (m *memAPI) DeleteQuota(context.Context, string) error { panic("unused") }
func (m *memAPI) ListPolicies(context.Context, string) ([]gatewayconfig.RequestPolicy, error) {
	return nil, nil
}
func (m *memAPI) CreatePolicy(context.Context, string, string, string, []gatewayconfig.PolicyAction, bool, int) (*gatewayconfig.RequestPolicy, error) {
	panic("unused")
}
func (m *memAPI) UpdatePolicy(context.Context, string, *string, *string, []gatewayconfig.PolicyAction, *bool, *int) (*gatewayconfig.RequestPolicy, error) {
	panic("unused")
}
func (m *memAPI) ReorderPolicies(context.Context, string, []gatewayconfig.PolicyPriorityUpdate) error {
	panic("unused")
}
func (m *memAPI) DeletePolicy(context.Context, string) error { panic("unused") }
func (m *memAPI) ListWasmHooks(context.Context, string) ([]gatewayconfig.WasmHook, error) {
	return nil, nil
}
func (m *memAPI) CreateWasmHook(context.Context, string, string, string, string, string, bool, int, []byte) (*gatewayconfig.WasmHook, error) {
	panic("unused")
}
func (m *memAPI) UpdateWasmHook(context.Context, string, *string, *string, *string, *string, *bool, *int, []byte) (*gatewayconfig.WasmHook, error) {
	panic("unused")
}
func (m *memAPI) DeleteWasmHook(context.Context, string) error { panic("unused") }
func (m *memAPI) ListCredentials(context.Context, string) ([]credentials.Credential, error) {
	return nil, nil
}
func (m *memAPI) CreateCredential(context.Context, string, string, string, string, string, string) (*credentials.Credential, error) {
	panic("unused")
}
func (m *memAPI) UpdateCredential(context.Context, string, string, string) (*credentials.Credential, error) {
	panic("unused")
}
func (m *memAPI) RotateCredential(context.Context, string, string, string) (*credentials.Credential, error) {
	panic("unused")
}
func (m *memAPI) DeleteCredential(context.Context, string) error { panic("unused") }
func (m *memAPI) ListCredentialAssignments(context.Context, string) ([]credentials.Assignment, error) {
	return nil, nil
}
func (m *memAPI) AssignCredential(context.Context, string, string, string, string) (*credentials.Assignment, error) {
	panic("unused")
}
func (m *memAPI) DeleteCredentialAssignment(context.Context, string) error { panic("unused") }

func TestServiceCreateQuotaPublishesAndEmits(t *testing.T) {
	t.Parallel()
	api := &memAPI{}
	snap := &memSnap{}
	ev := &memEvents{}
	svc := platform.New(api, snap)
	svc.Events = ev
	q, err := svc.CreateQuota(context.Background(), "org", snapshot.ScopeOrganization, "org", snapshot.MetricRequests, 10, snapshot.WindowTotal)
	if err != nil {
		t.Fatal(err)
	}
	if q == nil || q.ID != "quota_1" || api.created != 1 || snap.n != 1 {
		t.Fatalf("q=%+v created=%d snap=%d", q, api.created, snap.n)
	}
	if len(ev.names) != 2 || ev.names[0] != platform.EventSnapshotPublish || ev.names[1] != platform.EventQuotaCreated {
		t.Fatalf("events=%v", ev.names)
	}
}

func TestServiceEmitsViaBus(t *testing.T) {
	t.Parallel()
	api := &memAPI{}
	snap := &memSnap{}
	bus := platform.NewBus()
	mem := platform.NewMemoryRecorder(32)
	bus.SubscribeAll(func(ctx context.Context, e platform.Event) { mem.Record(ctx, e) })

	svc := platform.New(api, snap)
	svc.Events = bus
	q, err := svc.CreateQuota(context.Background(), "org", snapshot.ScopeOrganization, "org", snapshot.MetricRequests, 10, snapshot.WindowTotal)
	if err != nil || q == nil {
		t.Fatalf("err=%v q=%v", err, q)
	}
	events := mem.Snapshot()
	if len(events) != 2 {
		t.Fatalf("events=%+v", events)
	}
	if events[0].Name != platform.EventSnapshotPublish || events[1].Name != platform.EventQuotaCreated {
		t.Fatalf("order=%v %v", events[0].Name, events[1].Name)
	}
	if events[1].ResourceID != "quota_1" || events[1].OrganizationID != "org" {
		t.Fatalf("quota event=%+v", events[1])
	}
	if events[0].ID == "" || events[1].ID == "" || events[1].At.IsZero() {
		t.Fatalf("missing id/at: %+v", events)
	}
}

func TestListVisibleOrgAPIKeysFilters(t *testing.T) {
	t.Parallel()
	api := &memAPI{keys: []access.APIKey{
		{ID: "k1", Kind: snapshot.KeyKindServiceAccount},
		{ID: "k2", Kind: snapshot.KeyKindPersonal, OwnerUserID: "u1"},
		{ID: "k3", Kind: snapshot.KeyKindPersonal, OwnerUserID: "u2"},
	}}
	svc := platform.New(api, &memSnap{})
	got, err := svc.ListVisibleOrgAPIKeys(context.Background(), "org", "u1", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "k1" || got[1].ID != "k2" {
		t.Fatalf("got=%+v", got)
	}
	all, err := svc.ListVisibleOrgAPIKeys(context.Background(), "org", "u1", true)
	if err != nil || len(all) != 3 {
		t.Fatalf("admin got=%d err=%v", len(all), err)
	}
}

func TestTenancyCommandsEmitWithoutPublish(t *testing.T) {
	t.Parallel()
	api := &memAPI{}
	snap := &memSnap{}
	ev := &memEvents{}
	svc := platform.New(api, snap)
	svc.Events = ev

	org, err := svc.CreateOrganization(context.Background(), "Acme", "u1")
	if err != nil || org.ID != "org_1" {
		t.Fatalf("org=%+v err=%v", org, err)
	}
	team, err := svc.CreateTeam(context.Background(), "org_1", "Eng", "u1")
	if err != nil || team.ID != "team_1" {
		t.Fatalf("team=%+v err=%v", team, err)
	}
	outcome, _, err := svc.InviteOrgMember(context.Background(), "org_1", "a@b.com", "u1")
	if err != nil || outcome.Status != "invited" {
		t.Fatalf("invite=%+v err=%v", outcome, err)
	}
	if snap.n != 0 {
		t.Fatalf("expected no snapshot publish, got %d", snap.n)
	}
	want := []platform.EventName{
		platform.EventOrgCreated,
		platform.EventTeamCreated,
		platform.EventInviteCreated,
	}
	if len(ev.names) != len(want) {
		t.Fatalf("events=%v want=%v", ev.names, want)
	}
	for i := range want {
		if ev.names[i] != want[i] {
			t.Fatalf("events=%v want=%v", ev.names, want)
		}
	}
}

func TestInviteOrgMemberEmitsMemberAdded(t *testing.T) {
	t.Parallel()
	api := &memAPI{invite: &tenancy.InviteOutcome{
		Status: "added",
		Member: &tenancy.OrgMember{UserID: "u2"},
	}}
	snap := &memSnap{}
	ev := &memEvents{}
	svc := platform.New(api, snap)
	svc.Events = ev
	_, _, err := svc.InviteOrgMember(context.Background(), "org_1", "a@b.com", "u1")
	if err != nil {
		t.Fatal(err)
	}
	if snap.n != 0 || len(ev.names) != 1 || ev.names[0] != platform.EventMemberAdded {
		t.Fatalf("snap=%d events=%v", snap.n, ev.names)
	}
}

func TestPublishSnapshotEmitsEvent(t *testing.T) {
	t.Parallel()
	snap := &memSnap{}
	ev := &memEvents{}
	svc := platform.New(&memAPI{}, snap)
	svc.Events = ev
	if err := svc.PublishSnapshot(context.Background()); err != nil {
		t.Fatal(err)
	}
	if snap.n != 1 || len(ev.names) != 1 || ev.names[0] != platform.EventSnapshotPublish {
		t.Fatalf("snap=%d events=%v", snap.n, ev.names)
	}
}

func TestUpdateTeamMemberRoleEmitsWithoutPublish(t *testing.T) {
	t.Parallel()
	snap := &memSnap{}
	ev := &memEvents{}
	svc := platform.New(&memAPI{}, snap)
	svc.Events = ev
	m, err := svc.UpdateTeamMemberRole(context.Background(), "team_1", "u1", "u2", "admin")
	if err != nil || m == nil || m.Role != "admin" {
		t.Fatalf("m=%+v err=%v", m, err)
	}
	if snap.n != 0 || len(ev.names) != 1 || ev.names[0] != platform.EventTeamMemberRoleUpdated {
		t.Fatalf("snap=%d events=%v", snap.n, ev.names)
	}
}
