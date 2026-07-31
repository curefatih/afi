package regions

import (
	"github.com/curefatih/afi/internal/snapshot"
)

// OrgSlice is one organization's contribution to a snapshot.Source.
type OrgSlice struct {
	APIKeys      []snapshot.APIKey
	SigningKeys  []snapshot.SigningKey
	Providers    []snapshot.Provider // may be empty; providers resolved globally after merge
	Routes       []snapshot.Route
	Quotas       []snapshot.Quota
	Policies     []snapshot.Policy
	WasmHooks    []snapshot.WasmHook
	MCPBackends  []snapshot.MCPBackend
	A2AAgents    []snapshot.A2AAgent
	Credentials  []snapshot.Credential
	Assignments  []snapshot.CredentialAssignment
	DefaultRetry *snapshot.RetryConfig
	ObjectStore  *snapshot.ObjectStoreConfig
}

// IndexBaseByOrg splits a loaded base Source into per-org slices plus a global provider pool.
func IndexBaseByOrg(base snapshot.Source) (map[string]OrgSlice, []snapshot.Provider) {
	out := map[string]OrgSlice{}
	ensure := func(orgID string) OrgSlice {
		s, ok := out[orgID]
		if !ok {
			s = OrgSlice{}
		}
		return s
	}
	for _, k := range base.APIKeys {
		if k.OrganizationID == "" {
			continue
		}
		s := ensure(k.OrganizationID)
		s.APIKeys = append(s.APIKeys, k)
		out[k.OrganizationID] = s
	}
	for _, k := range base.SigningKeys {
		if k.OrganizationID == "" {
			continue
		}
		s := ensure(k.OrganizationID)
		s.SigningKeys = append(s.SigningKeys, k)
		out[k.OrganizationID] = s
	}
	for _, r := range base.Routes {
		if r.OrganizationID == "" {
			continue
		}
		s := ensure(r.OrganizationID)
		s.Routes = append(s.Routes, r)
		out[r.OrganizationID] = s
	}
	for _, q := range base.Quotas {
		if q.OrganizationID == "" {
			continue
		}
		s := ensure(q.OrganizationID)
		s.Quotas = append(s.Quotas, q)
		out[q.OrganizationID] = s
	}
	for _, p := range base.Policies {
		if p.OrganizationID == "" {
			continue
		}
		s := ensure(p.OrganizationID)
		s.Policies = append(s.Policies, p)
		out[p.OrganizationID] = s
	}
	for _, h := range base.WasmHooks {
		if h.OrganizationID == "" {
			continue
		}
		s := ensure(h.OrganizationID)
		s.WasmHooks = append(s.WasmHooks, h)
		out[h.OrganizationID] = s
	}
	for _, b := range base.MCPBackends {
		if b.OrganizationID == "" {
			continue
		}
		s := ensure(b.OrganizationID)
		s.MCPBackends = append(s.MCPBackends, b)
		out[b.OrganizationID] = s
	}
	for _, a := range base.A2AAgents {
		if a.OrganizationID == "" {
			continue
		}
		s := ensure(a.OrganizationID)
		s.A2AAgents = append(s.A2AAgents, a)
		out[a.OrganizationID] = s
	}
	for _, c := range base.Credentials {
		if c.OrganizationID == "" {
			continue
		}
		s := ensure(c.OrganizationID)
		s.Credentials = append(s.Credentials, c)
		out[c.OrganizationID] = s
	}
	credOrg := map[string]string{}
	for _, c := range base.Credentials {
		credOrg[c.ID] = c.OrganizationID
	}
	for _, a := range base.Assignments {
		orgID := credOrg[a.CredentialID]
		if orgID == "" {
			continue
		}
		s := ensure(orgID)
		s.Assignments = append(s.Assignments, a)
		out[orgID] = s
	}
	for orgID, retry := range base.DefaultRetries {
		if orgID == "" || retry == nil {
			continue
		}
		s := ensure(orgID)
		cp := *retry
		s.DefaultRetry = &cp
		out[orgID] = s
	}
	for orgID, store := range base.ObjectStores {
		if orgID == "" || store == nil {
			continue
		}
		s := ensure(orgID)
		cp := *store
		s.ObjectStore = &cp
		out[orgID] = s
	}
	return out, append([]snapshot.Provider(nil), base.Providers...)
}

// OverlayToSlice converts an overlay payload into an OrgSlice (keys come from base).
func OverlayToSlice(orgID string, baseKeys OrgSlice, payload OverlayPayload) OrgSlice {
	s := OrgSlice{
		APIKeys:     append([]snapshot.APIKey(nil), baseKeys.APIKeys...),
		SigningKeys: append([]snapshot.SigningKey(nil), baseKeys.SigningKeys...),
		Providers:   append([]snapshot.Provider(nil), payload.Providers...),
		Routes:      append([]snapshot.Route(nil), payload.Routes...),
		Quotas:      append([]snapshot.Quota(nil), payload.Quotas...),
		Policies:    append([]snapshot.Policy(nil), payload.Policies...),
		WasmHooks:   append([]snapshot.WasmHook(nil), payload.WasmHooks...),
		MCPBackends: append([]snapshot.MCPBackend(nil), payload.MCPBackends...),
		A2AAgents:   append([]snapshot.A2AAgent(nil), payload.A2AAgents...),
		Credentials: append([]snapshot.Credential(nil), payload.Credentials...),
		Assignments: append([]snapshot.CredentialAssignment(nil), payload.Assignments...),
	}
	if payload.DefaultRetry != nil {
		cp := *payload.DefaultRetry
		s.DefaultRetry = &cp
	}
	if payload.ObjectStore != nil {
		cp := *payload.ObjectStore
		s.ObjectStore = &cp
	}
	for i := range s.Routes {
		s.Routes[i].OrganizationID = orgID
	}
	for i := range s.Quotas {
		s.Quotas[i].OrganizationID = orgID
	}
	for i := range s.Policies {
		s.Policies[i].OrganizationID = orgID
	}
	for i := range s.WasmHooks {
		s.WasmHooks[i].OrganizationID = orgID
	}
	for i := range s.MCPBackends {
		s.MCPBackends[i].OrganizationID = orgID
	}
	for i := range s.A2AAgents {
		s.A2AAgents[i].OrganizationID = orgID
	}
	for i := range s.Credentials {
		s.Credentials[i].OrganizationID = orgID
	}
	return s
}

// BuildRegionSource filters base config by active memberships and applies full overlays.
// Returns a Source ready for snapshot.Compile and the allowlist of org IDs (always non-nil).
func BuildRegionSource(
	base snapshot.Source,
	memberships []OrgRegionMembership,
	overlays []RegionConfigOverlay,
) (snapshot.Source, []string) {
	byOrg, providers := IndexBaseByOrg(base)
	overlayByOrg := map[string]RegionConfigOverlay{}
	for _, o := range overlays {
		overlayByOrg[o.OrganizationID] = o
	}

	allowed := make([]string, 0, len(memberships))
	allowedSet := map[string]struct{}{}
	for _, m := range memberships {
		if m.Status != MembershipStatusActive {
			continue
		}
		if _, ok := allowedSet[m.OrganizationID]; ok {
			continue
		}
		allowedSet[m.OrganizationID] = struct{}{}
		allowed = append(allowed, m.OrganizationID)
	}

	var src snapshot.Source
	src.DefaultRetries = map[string]*snapshot.RetryConfig{}
	src.ObjectStores = map[string]*snapshot.ObjectStoreConfig{}
	neededProviders := map[string]struct{}{}

	for _, orgID := range allowed {
		baseSlice := byOrg[orgID]
		var slice OrgSlice
		if ov, ok := overlayByOrg[orgID]; ok {
			slice = OverlayToSlice(orgID, baseSlice, ov.Payload)
		} else {
			slice = baseSlice
		}
		src.APIKeys = append(src.APIKeys, slice.APIKeys...)
		src.SigningKeys = append(src.SigningKeys, slice.SigningKeys...)
		src.Routes = append(src.Routes, slice.Routes...)
		src.Quotas = append(src.Quotas, slice.Quotas...)
		src.Policies = append(src.Policies, slice.Policies...)
		src.WasmHooks = append(src.WasmHooks, slice.WasmHooks...)
		src.MCPBackends = append(src.MCPBackends, slice.MCPBackends...)
		src.A2AAgents = append(src.A2AAgents, slice.A2AAgents...)
		src.Credentials = append(src.Credentials, slice.Credentials...)
		src.Assignments = append(src.Assignments, slice.Assignments...)
		if slice.DefaultRetry != nil {
			src.DefaultRetries[orgID] = slice.DefaultRetry
		}
		if slice.ObjectStore != nil {
			src.ObjectStores[orgID] = slice.ObjectStore
		}
		for _, r := range slice.Routes {
			neededProviders[r.ProviderID] = struct{}{}
			for _, f := range r.Fallbacks {
				neededProviders[f.ProviderID] = struct{}{}
			}
		}
		for _, p := range slice.Providers {
			neededProviders[p.ID] = struct{}{}
		}
	}

	provByID := map[string]snapshot.Provider{}
	for _, p := range providers {
		provByID[p.ID] = p
	}
	for _, o := range overlays {
		for _, p := range o.Payload.Providers {
			provByID[p.ID] = p
		}
	}
	for id := range neededProviders {
		if p, ok := provByID[id]; ok {
			src.Providers = append(src.Providers, p)
		}
	}
	if len(src.DefaultRetries) == 0 {
		src.DefaultRetries = nil
	}
	if len(src.ObjectStores) == 0 {
		src.ObjectStores = nil
	}
	return src, allowed
}
