package gatewayconfig

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/snapshot"
)

// ProviderRepository persists write-model providers.
type ProviderRepository interface {
	ListByOrg(ctx context.Context, orgID string) ([]Provider, error)
	Insert(ctx context.Context, p Provider) error
	Update(ctx context.Context, providerID, name, baseURL, apiKeyEnv string, config *json.RawMessage) (*Provider, error)
	Delete(ctx context.Context, providerID string) error
	OrgID(ctx context.Context, providerID string) (string, error)
}

// RouteRepository persists write-model routes.
type RouteRepository interface {
	ListByOrg(ctx context.Context, orgID string) ([]Route, error)
	Insert(ctx context.Context, r Route) error
	Update(ctx context.Context, routeID, model, providerID, targetModel string, fallbacks []RouteFallback, retry *RetryConfig, strategy string, weight int) (*Route, error)
	Delete(ctx context.Context, routeID string) error
	OrgID(ctx context.Context, routeID string) (string, error)
}

// CreateProvider validates and persists a provider.
func CreateProvider(
	ctx context.Context,
	repo ProviderRepository,
	id, orgID, name, typ, baseURL, apiKeyEnv string,
	caps snapshot.ProviderCapabilities,
	config json.RawMessage,
) (*Provider, error) {
	p, err := NewProvider(id, orgID, name, typ, baseURL, apiKeyEnv, caps, config, timeNowUTC())
	if err != nil {
		return nil, err
	}
	if err := repo.Insert(ctx, *p); err != nil {
		return nil, err
	}
	return p, nil
}

// UpdateProvider validates and persists provider field updates.
// When config is nil, the stored config JSON is left unchanged.
func UpdateProvider(
	ctx context.Context,
	repo ProviderRepository,
	providerID, name, baseURL, apiKeyEnv string,
	config *json.RawMessage,
) (*Provider, error) {
	if config != nil {
		cfg, err := NormalizeProviderConfig(*config)
		if err != nil {
			return nil, err
		}
		c := cfg
		config = &c
	}
	return repo.Update(ctx, providerID, name, baseURL, apiKeyEnv, config)
}

// AssertProviderInOrg ensures providerID belongs to orgID.
func AssertProviderInOrg(ctx context.Context, orgID, providerID string, providers ProviderRepository) error {
	provOrg, err := providers.OrgID(ctx, providerID)
	if err != nil {
		return err
	}
	if provOrg != orgID {
		return fmt.Errorf("%w: provider %s is not in organization", kernel.ErrInvalidRequest, providerID)
	}
	return nil
}

func assertRouteProviders(ctx context.Context, orgID, providerID string, fallbacks []RouteFallback, providers ProviderRepository) error {
	if err := AssertProviderInOrg(ctx, orgID, providerID, providers); err != nil {
		return err
	}
	for _, fb := range fallbacks {
		if err := AssertProviderInOrg(ctx, orgID, fb.ProviderID, providers); err != nil {
			return err
		}
	}
	return nil
}

// CreateRoute validates and persists a route.
func CreateRoute(
	ctx context.Context,
	repo RouteRepository,
	providers ProviderRepository,
	id, orgID, model, providerID, targetModel string,
	fallbacks []RouteFallback,
	retry *RetryConfig,
	strategy string,
	weight int,
) (*Route, error) {
	r, err := NewRoute(id, orgID, model, providerID, targetModel, fallbacks, retry, strategy, weight, timeNowUTC())
	if err != nil {
		return nil, err
	}
	if err := assertRouteProviders(ctx, orgID, providerID, fallbacks, providers); err != nil {
		return nil, err
	}
	if err := repo.Insert(ctx, *r); err != nil {
		return nil, err
	}
	return r, nil
}

// UpdateRoute validates provider ownership then persists.
func UpdateRoute(
	ctx context.Context,
	repo RouteRepository,
	providers ProviderRepository,
	routeID, orgID, model, providerID, targetModel string,
	fallbacks []RouteFallback,
	retry *RetryConfig,
	strategy string,
	weight int,
) (*Route, error) {
	strategy, weight, fallbacks, err := NormalizeRouteRouting(strategy, weight, fallbacks)
	if err != nil {
		return nil, err
	}
	retry, err = NormalizeRetry(retry)
	if err != nil {
		return nil, err
	}
	if err := assertRouteProviders(ctx, orgID, providerID, fallbacks, providers); err != nil {
		return nil, err
	}
	return repo.Update(ctx, routeID, model, providerID, targetModel, fallbacks, retry, strategy, weight)
}
