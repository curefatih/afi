package gatewayconfig

import (
	"context"

	"github.com/curefatih/afi/internal/snapshot"
)

// ProviderRepository persists write-model providers.
type ProviderRepository interface {
	ListByOrg(ctx context.Context, orgID string) ([]Provider, error)
	Insert(ctx context.Context, p Provider) error
	Update(ctx context.Context, providerID, name, baseURL, apiKeyEnv string) (*Provider, error)
	Delete(ctx context.Context, providerID string) error
	OrgID(ctx context.Context, providerID string) (string, error)
}

// RouteRepository persists write-model routes.
type RouteRepository interface {
	ListByOrg(ctx context.Context, orgID string) ([]Route, error)
	Insert(ctx context.Context, r Route) error
	Update(ctx context.Context, routeID, model, providerID, targetModel string, fallbacks []RouteFallback) (*Route, error)
	Delete(ctx context.Context, routeID string) error
	OrgID(ctx context.Context, routeID string) (string, error)
}

// CreateProvider validates and persists a provider.
func CreateProvider(
	ctx context.Context,
	repo ProviderRepository,
	id, orgID, name, typ, baseURL, apiKeyEnv string,
	caps snapshot.ProviderCapabilities,
) (*Provider, error) {
	p, err := NewProvider(id, orgID, name, typ, baseURL, apiKeyEnv, caps, timeNowUTC())
	if err != nil {
		return nil, err
	}
	if err := repo.Insert(ctx, *p); err != nil {
		return nil, err
	}
	return p, nil
}

// CreateRoute validates and persists a route.
func CreateRoute(
	ctx context.Context,
	repo RouteRepository,
	id, orgID, model, providerID, targetModel string,
	fallbacks []RouteFallback,
) (*Route, error) {
	r, err := NewRoute(id, orgID, model, providerID, targetModel, fallbacks, timeNowUTC())
	if err != nil {
		return nil, err
	}
	if err := repo.Insert(ctx, *r); err != nil {
		return nil, err
	}
	return r, nil
}
