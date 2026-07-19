package snapshot

import "time"

// Source is the control-plane view used to compile a snapshot.
type Source struct {
	APIKeys   []APIKey
	Providers []Provider
	Routes    []Route
	Quotas    []Quota
	Policies  []Policy
}

// Compile builds an immutable snapshot payload (version assigned by store on Put).
func Compile(src Source) *Snapshot {
	s := &Snapshot{
		CreatedAt: time.Now().UTC(),
		APIKeys:   make(map[string]APIKey, len(src.APIKeys)),
		Providers: make(map[string]Provider, len(src.Providers)),
		Routes:    make(map[string]Route, len(src.Routes)),
		Quotas:    append([]Quota(nil), src.Quotas...),
		Policies:  append([]Policy(nil), src.Policies...),
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
	return s
}
