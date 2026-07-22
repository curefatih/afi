package gatewayconfig

import (
	"fmt"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/snapshot"
)

// Provider is the write-model upstream provider configuration.
type Provider struct {
	ID             string                        `json:"id"`
	OrganizationID string                        `json:"organization_id"`
	Name           string                        `json:"name"`
	Type           string                        `json:"type"`
	BaseURL        string                        `json:"base_url"`
	APIKeyEnv      string                        `json:"api_key_env"`
	Capabilities   snapshot.ProviderCapabilities `json:"capabilities"`
	CreatedAt      time.Time                     `json:"created_at"`
}

// RouteFallback is a secondary provider target for a model route.
type RouteFallback struct {
	ProviderID  string `json:"provider_id"`
	TargetModel string `json:"target_model"`
}

// Route maps a virtual model to a provider target (+ optional fallbacks / retry).
type Route struct {
	ID             string          `json:"id"`
	OrganizationID string          `json:"organization_id"`
	Model          string          `json:"model"`
	ProviderID     string          `json:"provider_id"`
	TargetModel    string          `json:"target_model"`
	Fallbacks      []RouteFallback `json:"fallbacks"`
	Retry          *RetryConfig    `json:"retry,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}

// NewProvider builds a validated provider entity.
func NewProvider(id, orgID, name, typ, baseURL, apiKeyEnv string, caps snapshot.ProviderCapabilities, now time.Time) (*Provider, error) {
	name = strings.TrimSpace(name)
	typ = strings.TrimSpace(typ)
	if id == "" || orgID == "" {
		return nil, fmt.Errorf("%w: id and organization_id required", kernel.ErrInvalidRequest)
	}
	if name == "" || typ == "" {
		return nil, fmt.Errorf("%w: name and type required", kernel.ErrInvalidRequest)
	}
	caps = snapshot.NormalizeCapabilities(typ, caps)
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return &Provider{
		ID:             id,
		OrganizationID: orgID,
		Name:           name,
		Type:           typ,
		BaseURL:        baseURL,
		APIKeyEnv:      apiKeyEnv,
		Capabilities:   caps,
		CreatedAt:      now.UTC(),
	}, nil
}

// NewRoute builds a validated route entity.
func NewRoute(id, orgID, model, providerID, targetModel string, fallbacks []RouteFallback, retry *RetryConfig, now time.Time) (*Route, error) {
	model = strings.TrimSpace(model)
	providerID = strings.TrimSpace(providerID)
	if id == "" || orgID == "" {
		return nil, fmt.Errorf("%w: id and organization_id required", kernel.ErrInvalidRequest)
	}
	if model == "" || providerID == "" {
		return nil, fmt.Errorf("%w: model and provider_id required", kernel.ErrInvalidRequest)
	}
	if fallbacks == nil {
		fallbacks = []RouteFallback{}
	}
	retry, err := NormalizeRetry(retry)
	if err != nil {
		return nil, err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return &Route{
		ID:             id,
		OrganizationID: orgID,
		Model:          model,
		ProviderID:     providerID,
		TargetModel:    targetModel,
		Fallbacks:      fallbacks,
		Retry:          retry,
		CreatedAt:      now.UTC(),
	}, nil
}
