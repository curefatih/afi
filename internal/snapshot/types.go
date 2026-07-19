package snapshot

import "time"

// Snapshot is an immutable compiled gateway configuration.
const (
	ScopeOrganization = "organization"
	ScopeProject      = "project"
	ScopeAPIKey       = "api_key"

	MetricRequests = "requests"
	MetricTokens   = "tokens"

	WindowTotal = "total"
)

type Snapshot struct {
	Version   int64               `json:"version"`
	CreatedAt time.Time           `json:"created_at"`
	APIKeys   map[string]APIKey   `json:"api_keys"`  // keyed by key hash
	Providers map[string]Provider `json:"providers"` // keyed by provider id
	Routes    map[string]Route    `json:"routes"`    // keyed by orgID + "::" + model
	Quotas    []Quota             `json:"quotas"`
}

type Quota struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	ScopeType      string `json:"scope_type"`
	ScopeID        string `json:"scope_id"`
	Metric         string `json:"metric"`
	LimitValue     int64  `json:"limit_value"`
	Window         string `json:"window"`
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
	Type      string `json:"type"` // openai | anthropic
	BaseURL   string `json:"base_url"`
	APIKeyEnv string `json:"api_key_env"`
	Name      string `json:"name"`
}

// RouteTarget is a provider + model pair used for primary routing or failover.
type RouteTarget struct {
	ProviderID  string `json:"provider_id"`
	TargetModel string `json:"target_model"`
}

type Route struct {
	OrganizationID string        `json:"organization_id"`
	Model          string        `json:"model"`
	ProviderID     string        `json:"provider_id"`
	TargetModel    string        `json:"target_model"`
	Fallbacks      []RouteTarget `json:"fallbacks,omitempty"`
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

// ResolveQuota picks the most specific matching quota for the metric (api_key > project > organization).
func (s *Snapshot) ResolveQuota(key APIKey, metric string) (Quota, bool) {
	if s == nil {
		return Quota{}, false
	}
	var found Quota
	best := 0
	for _, q := range s.Quotas {
		if q.Metric != metric || q.Window != WindowTotal {
			continue
		}
		rank := 0
		switch q.ScopeType {
		case ScopeAPIKey:
			if q.ScopeID == key.ID {
				rank = 3
			}
		case ScopeProject:
			if q.ScopeID == key.ProjectID {
				rank = 2
			}
		case ScopeOrganization:
			if q.ScopeID == key.OrganizationID {
				rank = 1
			}
		}
		if rank > best {
			best = rank
			found = q
		}
	}
	return found, best > 0
}
