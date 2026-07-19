package snapshot

import "time"

// Snapshot is an immutable compiled gateway configuration.
type Snapshot struct {
	Version   int64               `json:"version"`
	CreatedAt time.Time           `json:"created_at"`
	APIKeys   map[string]APIKey   `json:"api_keys"`  // keyed by key hash
	Providers map[string]Provider `json:"providers"` // keyed by provider id
	Routes    map[string]Route    `json:"routes"`    // keyed by orgID + "::" + model
}

type APIKey struct {
	KeyHash        string `json:"key_hash"`
	KeyPrefix      string `json:"key_prefix"`
	ProjectID      string `json:"project_id"`
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
	ID             string `json:"id,omitempty"`
}

type Provider struct {
	ID        string `json:"id"`
	Type      string `json:"type"` // openai
	BaseURL   string `json:"base_url"`
	APIKeyEnv string `json:"api_key_env"`
	Name      string `json:"name"`
}

type Route struct {
	OrganizationID string `json:"organization_id"`
	Model          string `json:"model"`
	ProviderID     string `json:"provider_id"`
	TargetModel    string `json:"target_model"`
}

func routeKey(orgID, model string) string {
	return orgID + "::" + model
}

func (s *Snapshot) LookupKey(raw string) (APIKey, bool) {
	if s == nil || s.APIKeys == nil {
		return APIKey{}, false
	}
	k, ok := s.APIKeys[HashKey(raw)]
	return k, ok
}

func (s *Snapshot) LookupRoute(orgID, model string) (Route, Provider, bool) {
	if s == nil || s.Routes == nil {
		return Route{}, Provider{}, false
	}
	r, ok := s.Routes[routeKey(orgID, model)]
	if !ok {
		return Route{}, Provider{}, false
	}
	p, ok := s.Providers[r.ProviderID]
	if !ok {
		return Route{}, Provider{}, false
	}
	return r, p, true
}
