package snapshot

import (
	"encoding/json"
	"strings"
	"time"
)

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

	CredentialStorageEnv         = "env"
	CredentialStorageEncryptedDB = "encrypted_db"
	CredentialStorageVault       = "vault"
	CredentialStatusActive       = "active"
	CredentialStatusDisabled     = "disabled"
)

// RequestWindows are checked on every request (rate limits + lifetime).
var RequestWindows = []string{WindowMinute, WindowHour, WindowDay, WindowTotal}

type Snapshot struct {
	Version        int64                   `json:"version"`
	CreatedAt      time.Time               `json:"created_at"`
	APIKeys        map[string]APIKey       `json:"api_keys"`    // keyed by key hash
	Providers      map[string]Provider     `json:"providers"`   // keyed by provider id
	Routes         map[string]Route        `json:"routes"`      // keyed by orgID + "::" + model
	Credentials    map[string]Credential   `json:"credentials"` // keyed by credential id
	Assignments    map[string]string       `json:"assignments"` // providerType::scopeType::scopeID → credential id
	Quotas         []Quota                 `json:"quotas"`
	Policies       []Policy                `json:"policies"`
	WasmHooks      []WasmHook              `json:"wasm_hooks"`
	MCPBackends    map[string]MCPBackend   `json:"mcp_backends,omitempty"` // keyed by backend id
	MCPRoutes      map[string]string       `json:"mcp_routes,omitempty"`   // orgID::alias → backend id
	DefaultRetries map[string]*RetryConfig `json:"default_retries,omitempty"` // orgID → default retry
}

// MCPBackend is a compiled MCP Streamable HTTP upstream.
type MCPBackend struct {
	ID              string   `json:"id"`
	OrganizationID  string   `json:"organization_id"`
	Alias           string   `json:"alias"`
	Name            string   `json:"name"`
	BaseURL         string   `json:"base_url"`
	APIKeyEnv       string   `json:"api_key_env"`
	MethodAllowlist []string `json:"method_allowlist,omitempty"`
	Enabled         bool     `json:"enabled"`
	// InlineAPIKey is request-scoped; set by the gateway after secret resolution.
	InlineAPIKey string `json:"-"`
}

// WasmHook is a compiled sandboxed lifecycle binding for the gateway.
type WasmHook struct {
	ID             string          `json:"id"`
	OrganizationID string          `json:"organization_id"`
	Name           string          `json:"name"`
	Phase          string          `json:"phase"` // before_call | before_chat | after_call
	ModuleURI      string          `json:"module_uri"`
	Digest         string          `json:"digest,omitempty"` // sha256 hex; empty skips verify
	Enabled        bool            `json:"enabled"`
	Priority       int             `json:"priority"`
	Config         json.RawMessage `json:"config,omitempty"`
}

// Wasm hook phases in the compiled snapshot.
const (
	WasmPhaseBeforeCall = "before_call"
	WasmPhaseBeforeChat = "before_chat"
	WasmPhaseAfterCall  = "after_call"
)

// Credential is a compiled upstream secret reference (never plaintext for encrypted_db).
type Credential struct {
	ID               string `json:"id"`
	OrganizationID   string `json:"organization_id,omitempty"`
	Name             string `json:"name,omitempty"`
	ProviderType     string `json:"provider_type"`
	StorageKind      string `json:"storage_kind"`
	SecretRef        string `json:"secret_ref,omitempty"`
	EncryptedPayload []byte `json:"encrypted_payload,omitempty"`
	KeyVersion       int    `json:"key_version,omitempty"`
	Status           string `json:"status"`
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
	// InlineAPIKey is request-scoped; set by the gateway after credential resolution.
	// Never persisted in snapshots (json:"-").
	InlineAPIKey string `json:"-"`
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
	Retry          *RetryConfig  `json:"retry,omitempty"`
}

func routeKey(orgID, model string) string {
	return orgID + "::" + model
}

func mcpRouteKey(orgID, alias string) string {
	return orgID + "::" + alias
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

// LookupMCPBackend resolves an org-scoped MCP alias to a compiled backend.
func (s *Snapshot) LookupMCPBackend(orgID, alias string) (MCPBackend, bool) {
	if s == nil || s.MCPRoutes == nil || s.MCPBackends == nil {
		return MCPBackend{}, false
	}
	id, ok := s.MCPRoutes[mcpRouteKey(orgID, alias)]
	if !ok {
		return MCPBackend{}, false
	}
	b, ok := s.MCPBackends[id]
	if !ok || !b.Enabled {
		return MCPBackend{}, false
	}
	return b, true
}

// AssignmentKey builds the snapshot assignment index key.
func AssignmentKey(providerType, scopeType, scopeID string) string {
	return providerType + "::" + scopeType + "::" + scopeID
}

// LookupCredentialByName finds an active credential in the org by display name + provider type.
// Names are matched case-sensitively after trim (same uniqueness as provider_credentials).
func (s *Snapshot) LookupCredentialByName(orgID, providerType, name string) (Credential, bool) {
	if s == nil || s.Credentials == nil {
		return Credential{}, false
	}
	orgID = strings.TrimSpace(orgID)
	providerType = strings.TrimSpace(providerType)
	name = strings.TrimSpace(name)
	if orgID == "" || providerType == "" || name == "" {
		return Credential{}, false
	}
	for _, c := range s.Credentials {
		if c.Status != CredentialStatusActive {
			continue
		}
		if c.OrganizationID == orgID && c.ProviderType == providerType && c.Name == name {
			return c, true
		}
	}
	return Credential{}, false
}

// ResolveCredential picks the most specific active credential for a provider type
// (api_key → project → organization). Returns false when no assignment matches.
func (s *Snapshot) ResolveCredential(providerType string, key APIKey) (Credential, bool) {
	c, ok, _ := s.ResolveCredentialForCall(providerType, key, "")
	return c, ok
}

// ResolveCredentialForCall resolves BYOK for a call.
// When overrideName is non-empty (from a matching select-credential policy), that name
// selects a credential in the org for the provider type. Otherwise falls back to
// assignment scopes: api_key → project → organization.
//
// If overrideName is set but no matching credential exists, ok=false and
// missingOverride=true so the gateway can fail closed instead of using the platform key.
func (s *Snapshot) ResolveCredentialForCall(providerType string, key APIKey, overrideName string) (cred Credential, ok bool, missingOverride bool) {
	if s == nil || s.Credentials == nil {
		return Credential{}, false, false
	}
	if name := strings.TrimSpace(overrideName); name != "" {
		c, found := s.LookupCredentialByName(key.OrganizationID, providerType, name)
		if !found {
			return Credential{}, false, true
		}
		return c, true, false
	}
	if s.Assignments == nil {
		return Credential{}, false, false
	}
	if key.ID != "" {
		if id, hit := s.Assignments[AssignmentKey(providerType, ScopeAPIKey, key.ID)]; hit {
			if c, hit := s.Credentials[id]; hit && c.Status == CredentialStatusActive {
				return c, true, false
			}
		}
	}
	if key.ProjectID != "" {
		if id, hit := s.Assignments[AssignmentKey(providerType, ScopeProject, key.ProjectID)]; hit {
			if c, hit := s.Credentials[id]; hit && c.Status == CredentialStatusActive {
				return c, true, false
			}
		}
	}
	if key.OrganizationID != "" {
		if id, hit := s.Assignments[AssignmentKey(providerType, ScopeOrganization, key.OrganizationID)]; hit {
			if c, hit := s.Credentials[id]; hit && c.Status == CredentialStatusActive {
				return c, true, false
			}
		}
	}
	return Credential{}, false, false
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

// ResolveRetry returns the effective retry for a route: route-specific, else org default, else nil.
func (s *Snapshot) ResolveRetry(route Route) *RetryConfig {
	if route.Retry != nil {
		return route.Retry
	}
	if s == nil || s.DefaultRetries == nil {
		return nil
	}
	return s.DefaultRetries[route.OrganizationID]
}
