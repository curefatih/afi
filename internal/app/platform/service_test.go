package platform_test

import (
	"context"
	"testing"

	"github.com/curefatih/afi/internal/access"
	"github.com/curefatih/afi/internal/app/platform"
	"github.com/curefatih/afi/internal/gatewayconfig"
	"github.com/curefatih/afi/internal/snapshot"
	"github.com/curefatih/afi/internal/tenancy"
)

type memSnap struct{ n int }

func (m *memSnap) PublishSnapshot(context.Context) error { m.n++; return nil }

type memAPI struct {
	created int
}

func (m *memAPI) CreateProject(context.Context, string, string, string) (*tenancy.Project, error) {
	panic("unused")
}
func (m *memAPI) CreateAPIKey(context.Context, string, string, string, string, string, string) (*access.APIKey, error) {
	panic("unused")
}
func (m *memAPI) DeleteAPIKey(context.Context, string) error { panic("unused") }
func (m *memAPI) CreateProvider(context.Context, string, string, string, string, string, snapshot.ProviderCapabilities) (*gatewayconfig.Provider, error) {
	panic("unused")
}
func (m *memAPI) UpdateProvider(context.Context, string, string, string, string) (*gatewayconfig.Provider, error) {
	panic("unused")
}
func (m *memAPI) DeleteProvider(context.Context, string) error { panic("unused") }
func (m *memAPI) CreateRoute(context.Context, string, string, string, string, []gatewayconfig.RouteFallback) (*gatewayconfig.Route, error) {
	panic("unused")
}
func (m *memAPI) UpdateRoute(context.Context, string, string, string, string, []gatewayconfig.RouteFallback) (*gatewayconfig.Route, error) {
	panic("unused")
}
func (m *memAPI) DeleteRoute(context.Context, string) error { panic("unused") }
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
func (m *memAPI) CreatePolicy(context.Context, string, string, string, bool, int) (*gatewayconfig.RequestPolicy, error) {
	panic("unused")
}
func (m *memAPI) UpdatePolicy(context.Context, string, *string, *string, *bool, *int) (*gatewayconfig.RequestPolicy, error) {
	panic("unused")
}
func (m *memAPI) DeletePolicy(context.Context, string) error { panic("unused") }

func TestServiceCreateQuotaPublishes(t *testing.T) {
	t.Parallel()
	api := &memAPI{}
	snap := &memSnap{}
	svc := platform.New(api, snap)
	q, err := svc.CreateQuota(context.Background(), "org", snapshot.ScopeOrganization, "org", snapshot.MetricRequests, 10, snapshot.WindowTotal)
	if err != nil {
		t.Fatal(err)
	}
	if q == nil || q.ID != "quota_1" || api.created != 1 || snap.n != 1 {
		t.Fatalf("q=%+v created=%d snap=%d", q, api.created, snap.n)
	}
}
