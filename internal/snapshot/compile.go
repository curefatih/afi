package snapshot

import (
	"time"
)

// Source is the control-plane view used to compile a snapshot.
type Source struct {
	APIKeys        []APIKey
	Providers      []Provider
	Routes         []Route
	Credentials    []Credential
	Assignments    []CredentialAssignment
	Quotas         []Quota
	Policies       []Policy
	WasmHooks      []WasmHook
	MCPBackends    []MCPBackend
	DefaultRetries map[string]*RetryConfig // orgID → default
}

// CredentialAssignment is a compile-time binding of credential → scope.
type CredentialAssignment struct {
	CredentialID string
	ProviderType string
	ScopeType    string
	ScopeID      string
}

// Compile builds an immutable snapshot payload (version assigned by store on Put).
func Compile(src Source) *Snapshot {
	s := &Snapshot{
		CreatedAt:      time.Now().UTC(),
		APIKeys:        make(map[string]APIKey, len(src.APIKeys)),
		Providers:      make(map[string]Provider, len(src.Providers)),
		Routes:         make(map[string]Route, len(src.Routes)),
		Credentials:    make(map[string]Credential, len(src.Credentials)),
		Assignments:    make(map[string]string, len(src.Assignments)),
		Quotas:         append([]Quota(nil), src.Quotas...),
		Policies:       append([]Policy(nil), src.Policies...),
		WasmHooks:      append([]WasmHook(nil), src.WasmHooks...),
		MCPBackends:    make(map[string]MCPBackend, len(src.MCPBackends)),
		MCPRoutes:      make(map[string]string, len(src.MCPBackends)),
		DefaultRetries: make(map[string]*RetryConfig, len(src.DefaultRetries)),
	}
	for _, k := range src.APIKeys {
		if k.KeyHash == "" {
			continue
		}
		s.APIKeys[k.KeyHash] = k
	}
	for _, p := range src.Providers {
		p.Capabilities = NormalizeCapabilities(p.Type, p.Capabilities)
		s.Providers[p.ID] = p
	}
	for _, r := range src.Routes {
		s.Routes[routeKey(r.OrganizationID, r.Model)] = r
	}
	for _, c := range src.Credentials {
		if c.ID == "" {
			continue
		}
		s.Credentials[c.ID] = c
	}
	for _, a := range src.Assignments {
		if a.CredentialID == "" || a.ProviderType == "" || a.ScopeType == "" || a.ScopeID == "" {
			continue
		}
		s.Assignments[AssignmentKey(a.ProviderType, a.ScopeType, a.ScopeID)] = a.CredentialID
	}
	for _, b := range src.MCPBackends {
		if b.ID == "" || b.OrganizationID == "" || b.Alias == "" {
			continue
		}
		s.MCPBackends[b.ID] = b
		if b.Enabled {
			s.MCPRoutes[mcpRouteKey(b.OrganizationID, b.Alias)] = b.ID
		}
	}
	if len(s.MCPBackends) == 0 {
		s.MCPBackends = nil
		s.MCPRoutes = nil
	}
	for orgID, retry := range src.DefaultRetries {
		if orgID == "" || retry == nil {
			continue
		}
		cp := *retry
		s.DefaultRetries[orgID] = &cp
	}
	if len(s.DefaultRetries) == 0 {
		s.DefaultRetries = nil
	}
	return s
}
