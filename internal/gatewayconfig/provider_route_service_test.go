package gatewayconfig

import (
	"context"
	"errors"
	"testing"

	"github.com/curefatih/afi/internal/kernel"
)

type memProviders struct {
	orgByID map[string]string
}

func (m *memProviders) ListByOrg(context.Context, string) ([]Provider, error) { return nil, nil }
func (m *memProviders) Insert(context.Context, Provider) error                { return nil }
func (m *memProviders) Update(context.Context, string, string, string, string) (*Provider, error) {
	return nil, nil
}
func (m *memProviders) Delete(context.Context, string) error { return nil }
func (m *memProviders) OrgID(_ context.Context, providerID string) (string, error) {
	if org, ok := m.orgByID[providerID]; ok {
		return org, nil
	}
	return "", kernel.ErrNotFound
}

type memRoutes struct {
	inserted *Route
}

func (m *memRoutes) ListByOrg(context.Context, string) ([]Route, error) { return nil, nil }
func (m *memRoutes) Insert(_ context.Context, r Route) error {
	cp := r
	m.inserted = &cp
	return nil
}
func (m *memRoutes) Update(context.Context, string, string, string, string, []RouteFallback, *RetryConfig) (*Route, error) {
	return nil, nil
}
func (m *memRoutes) Delete(context.Context, string) error { return nil }
func (m *memRoutes) OrgID(context.Context, string) (string, error) {
	return "", kernel.ErrNotFound
}

func TestCreateRouteRejectsCrossOrgProvider(t *testing.T) {
	t.Parallel()
	providers := &memProviders{orgByID: map[string]string{"prov_other": "org_b"}}
	routes := &memRoutes{}
	_, err := CreateRoute(context.Background(), routes, providers, "route_1", "org_a", "gpt-4o", "prov_other", "gpt-4o", nil, nil)
	if !errors.Is(err, kernel.ErrInvalidRequest) {
		t.Fatalf("err=%v", err)
	}
	if routes.inserted != nil {
		t.Fatal("expected no insert")
	}
}

func TestCreateRouteAcceptsSameOrgProvider(t *testing.T) {
	t.Parallel()
	providers := &memProviders{orgByID: map[string]string{"prov_ok": "org_a"}}
	routes := &memRoutes{}
	r, err := CreateRoute(context.Background(), routes, providers, "route_1", "org_a", "gpt-4o", "prov_ok", "gpt-4o", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r == nil || routes.inserted == nil {
		t.Fatal("expected insert")
	}
}
