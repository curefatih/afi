package platform_test

import (
	"context"
	"testing"
	"time"

	"github.com/curefatih/afi/internal/access"
	"github.com/curefatih/afi/internal/app/platform"
	"github.com/curefatih/afi/internal/gatewayconfig"
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
}

func (m *memAPI) ListOrganizationsForUser(context.Context, string) ([]tenancy.Organization, error) {
	return nil, nil
}
func (m *memAPI) ListOrgMembers(context.Context, string) ([]tenancy.OrgMember, error) { return nil, nil }
func (m *memAPI) ListTeams(context.Context, string) ([]tenancy.Team, error)           { return nil, nil }
func (m *memAPI) GetTeam(context.Context, string) (*tenancy.Team, error)              { return nil, nil }
func (m *memAPI) ListTeamMembers(context.Context, string) ([]tenancy.TeamMember, error) {
	return nil, nil
}
func (m *memAPI) ListProjects(context.Context, string) ([]tenancy.Project, error) { return nil, nil }
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
func (m *memAPI) DeleteProvider(context.Context, string) error { panic("unused") }
func (m *memAPI) ListRoutes(context.Context, string) ([]gatewayconfig.Route, error) { return nil, nil }
func (m *memAPI) CreateRoute(context.Context, string, string, string, string, []gatewayconfig.RouteFallback) (*gatewayconfig.Route, error) {
	panic("unused")
}
func (m *memAPI) UpdateRoute(context.Context, string, string, string, string, []gatewayconfig.RouteFallback) (*gatewayconfig.Route, error) {
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
func (m *memAPI) CreatePolicy(context.Context, string, string, string, bool, int) (*gatewayconfig.RequestPolicy, error) {
	panic("unused")
}
func (m *memAPI) UpdatePolicy(context.Context, string, *string, *string, *bool, *int) (*gatewayconfig.RequestPolicy, error) {
	panic("unused")
}
func (m *memAPI) DeletePolicy(context.Context, string) error { panic("unused") }

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
