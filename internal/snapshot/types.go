package snapshot

import "time"

// Snapshot is an immutable compiled gateway configuration.
const (
	ScopeOrganization = "organization"
	ScopeProject      = "project"
	ScopeUser         = "user"
	ScopeAPIKey       = "api_key"

	MetricRequests = "requests"
	MetricTokens   = "tokens"

	WindowTotal  = "total"
	WindowMinute = "minute"
	WindowHour   = "hour"
	WindowDay    = "day"

	KeyKindPersonal       = "personal"
	KeyKindServiceAccount = "service_account"
)

// RequestWindows are checked on every request (rate limits + lifetime).
var RequestWindows = []string{WindowMinute, WindowHour, WindowDay, WindowTotal}

type Snapshot struct {
	Version   int64               `json:"version"`
	CreatedAt time.Time           `json:"created_at"`
	APIKeys   map[string]APIKey   `json:"api_keys"`  // keyed by key hash
	Providers map[string]Provider `json:"providers"` // keyed by provider id
	Routes    map[string]Route    `json:"routes"`    // keyed by orgID + "::" + model
	Quotas    []Quota             `json:"quotas"`
	Policies  []Policy            `json:"policies"`
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

// Policy is a CEL allow-expression compiled into the gateway snapshot.
type Policy struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
	Expression     string `json:"expression"`
	Enabled        bool   `json:"enabled"`
	Priority       int    `json:"priority"`
}

type APIKey struct {
	KeyHash        string `json:"key_hash"`
	KeyPrefix      string `json:"key_prefix"`
	ProjectID      string `json:"project_id,omitempty"`
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
	ID             string `json:"id,omitempty"`
	Kind           string `json:"kind,omitempty"`
	OwnerUserID    string `json:"owner_user_id,omitempty"`
}

type Provider struct {
	ID           string               `json:"id"`
	Type         string               `json:"type"` // openai | anthropic | gemini | openai_compatible | …
	BaseURL      string               `json:"base_url"`
	APIKeyEnv    string               `json:"api_key_env"`
	Name         string               `json:"name"`
	Capabilities ProviderCapabilities `json:"capabilities"`
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

// ValidQuotaWindow reports whether window is supported.
func ValidQuotaWindow(window string) bool {
	switch window {
	case WindowTotal, WindowMinute, WindowHour, WindowDay:
		return true
	default:
		return false
	}
}

// ResolveQuota picks the most specific matching quota for the metric and window
// (api_key > user > project > organization).
func (s *Snapshot) ResolveQuota(key APIKey, metric, window string) (Quota, bool) {
	if s == nil {
		return Quota{}, false
	}
	if window == "" {
		window = WindowTotal
	}
	var found Quota
	best := 0
	for _, q := range s.Quotas {
		if q.Metric != metric || q.Window != window {
			continue
		}
		rank := 0
		switch q.ScopeType {
		case ScopeAPIKey:
			if q.ScopeID == key.ID {
				rank = 4
			}
		case ScopeUser:
			if key.OwnerUserID != "" && q.ScopeID == key.OwnerUserID {
				rank = 3
			}
		case ScopeProject:
			if key.ProjectID != "" && q.ScopeID == key.ProjectID {
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
