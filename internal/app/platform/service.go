package platform

import (
	"context"
	"fmt"

	"github.com/curefatih/afi/internal/access"
	"github.com/curefatih/afi/internal/gatewayconfig"
	"github.com/curefatih/afi/internal/snapshot"
	"github.com/curefatih/afi/internal/tenancy"
)

// SnapshotPublisher publishes compiled gateway snapshots after config changes.
type SnapshotPublisher interface {
	PublishSnapshot(ctx context.Context) error
}

// ConfigAPI is the write/query surface needed for gateway-config mutations.
type ConfigAPI interface {
	CreateProject(ctx context.Context, orgID, teamID, name string) (*tenancy.Project, error)
	CreateAPIKey(ctx context.Context, orgID, kind, ownerUserID, projectID, name, rawKey string) (*access.APIKey, error)
	DeleteAPIKey(ctx context.Context, keyID string) error
	CreateProvider(ctx context.Context, orgID, name, typ, baseURL, apiKeyEnv string, caps snapshot.ProviderCapabilities) (*gatewayconfig.Provider, error)
	UpdateProvider(ctx context.Context, providerID, name, baseURL, apiKeyEnv string) (*gatewayconfig.Provider, error)
	DeleteProvider(ctx context.Context, providerID string) error
	CreateRoute(ctx context.Context, orgID, model, providerID, targetModel string, fallbacks []gatewayconfig.RouteFallback) (*gatewayconfig.Route, error)
	UpdateRoute(ctx context.Context, routeID, model, providerID, targetModel string, fallbacks []gatewayconfig.RouteFallback) (*gatewayconfig.Route, error)
	DeleteRoute(ctx context.Context, routeID string) error
	CreateQuota(ctx context.Context, orgID, scopeType, scopeID, metric string, limitValue int64, window string) (*gatewayconfig.Quota, error)
	UpdateQuota(ctx context.Context, quotaID string, limitValue int64) (*gatewayconfig.Quota, error)
	DeleteQuota(ctx context.Context, quotaID string) error
	CreatePolicy(ctx context.Context, orgID, name, expression string, enabled bool, priority int) (*gatewayconfig.RequestPolicy, error)
	UpdatePolicy(ctx context.Context, policyID string, name, expression *string, enabled *bool, priority *int) (*gatewayconfig.RequestPolicy, error)
	DeletePolicy(ctx context.Context, policyID string) error
}

// Service orchestrates platform commands that must republish the gateway snapshot.
type Service struct {
	API  ConfigAPI
	Snap SnapshotPublisher
}

func New(api ConfigAPI, snap SnapshotPublisher) *Service {
	return &Service{API: api, Snap: snap}
}

func (s *Service) publish(ctx context.Context, action string) error {
	if s.Snap == nil {
		return fmt.Errorf("%s but snapshot publisher unavailable", action)
	}
	if err := s.Snap.PublishSnapshot(ctx); err != nil {
		return fmt.Errorf("%s but snapshot publish failed: %w", action, err)
	}
	return nil
}

func (s *Service) CreateProject(ctx context.Context, orgID, teamID, name string) (*tenancy.Project, error) {
	p, err := s.API.CreateProject(ctx, orgID, teamID, name)
	if err != nil {
		return nil, err
	}
	return p, s.publish(ctx, "created")
}

func (s *Service) CreateAPIKey(ctx context.Context, orgID, kind, ownerUserID, projectID, name, rawKey string) (*access.APIKey, error) {
	k, err := s.API.CreateAPIKey(ctx, orgID, kind, ownerUserID, projectID, name, rawKey)
	if err != nil {
		return nil, err
	}
	return k, s.publish(ctx, "created")
}

func (s *Service) DeleteAPIKey(ctx context.Context, keyID string) error {
	if err := s.API.DeleteAPIKey(ctx, keyID); err != nil {
		return err
	}
	return s.publish(ctx, "deleted")
}

func (s *Service) CreateProvider(ctx context.Context, orgID, name, typ, baseURL, apiKeyEnv string, caps snapshot.ProviderCapabilities) (*gatewayconfig.Provider, error) {
	p, err := s.API.CreateProvider(ctx, orgID, name, typ, baseURL, apiKeyEnv, caps)
	if err != nil {
		return nil, err
	}
	return p, s.publish(ctx, "created")
}

func (s *Service) UpdateProvider(ctx context.Context, providerID, name, baseURL, apiKeyEnv string) (*gatewayconfig.Provider, error) {
	p, err := s.API.UpdateProvider(ctx, providerID, name, baseURL, apiKeyEnv)
	if err != nil {
		return nil, err
	}
	return p, s.publish(ctx, "updated")
}

func (s *Service) DeleteProvider(ctx context.Context, providerID string) error {
	if err := s.API.DeleteProvider(ctx, providerID); err != nil {
		return err
	}
	return s.publish(ctx, "deleted")
}

func (s *Service) CreateRoute(ctx context.Context, orgID, model, providerID, targetModel string, fallbacks []gatewayconfig.RouteFallback) (*gatewayconfig.Route, error) {
	r, err := s.API.CreateRoute(ctx, orgID, model, providerID, targetModel, fallbacks)
	if err != nil {
		return nil, err
	}
	return r, s.publish(ctx, "created")
}

func (s *Service) UpdateRoute(ctx context.Context, routeID, model, providerID, targetModel string, fallbacks []gatewayconfig.RouteFallback) (*gatewayconfig.Route, error) {
	r, err := s.API.UpdateRoute(ctx, routeID, model, providerID, targetModel, fallbacks)
	if err != nil {
		return nil, err
	}
	return r, s.publish(ctx, "updated")
}

func (s *Service) DeleteRoute(ctx context.Context, routeID string) error {
	if err := s.API.DeleteRoute(ctx, routeID); err != nil {
		return err
	}
	return s.publish(ctx, "deleted")
}

func (s *Service) CreateQuota(ctx context.Context, orgID, scopeType, scopeID, metric string, limitValue int64, window string) (*gatewayconfig.Quota, error) {
	q, err := s.API.CreateQuota(ctx, orgID, scopeType, scopeID, metric, limitValue, window)
	if err != nil {
		return nil, err
	}
	return q, s.publish(ctx, "created")
}

func (s *Service) UpdateQuota(ctx context.Context, quotaID string, limitValue int64) (*gatewayconfig.Quota, error) {
	q, err := s.API.UpdateQuota(ctx, quotaID, limitValue)
	if err != nil {
		return nil, err
	}
	return q, s.publish(ctx, "updated")
}

func (s *Service) DeleteQuota(ctx context.Context, quotaID string) error {
	if err := s.API.DeleteQuota(ctx, quotaID); err != nil {
		return err
	}
	return s.publish(ctx, "deleted")
}

func (s *Service) CreatePolicy(ctx context.Context, orgID, name, expression string, enabled bool, priority int) (*gatewayconfig.RequestPolicy, error) {
	p, err := s.API.CreatePolicy(ctx, orgID, name, expression, enabled, priority)
	if err != nil {
		return nil, err
	}
	return p, s.publish(ctx, "created")
}

func (s *Service) UpdatePolicy(ctx context.Context, policyID string, name, expression *string, enabled *bool, priority *int) (*gatewayconfig.RequestPolicy, error) {
	p, err := s.API.UpdatePolicy(ctx, policyID, name, expression, enabled, priority)
	if err != nil {
		return nil, err
	}
	return p, s.publish(ctx, "updated")
}

func (s *Service) DeletePolicy(ctx context.Context, policyID string) error {
	if err := s.API.DeletePolicy(ctx, policyID); err != nil {
		return err
	}
	return s.publish(ctx, "deleted")
}
