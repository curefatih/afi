package providercatalog

import (
	"sort"
	"strings"
	"sync"
)

// AuthMode controls whether an empty api_key_env is allowed at call time.
type AuthMode string

const (
	// AuthAPIKey requires api_key_env or a BYOK credential.
	AuthAPIKey AuthMode = "api_key"
	// AuthOptional allows empty api_key_env (e.g. AWS default credential chain).
	AuthOptional AuthMode = "optional"
)

// Capabilities describes default modality support for a provider type.
type Capabilities struct {
	Chat      bool `json:"chat"`
	Stream    bool `json:"stream"`
	TTS       bool `json:"tts"`
	STT       bool `json:"stt"`
	Embedding bool `json:"embedding"`
	Image     bool `json:"image"`
}

// SeedRoute is an optional demo route created with a seeded provider.
type SeedRoute struct {
	ID          string
	Model       string
	TargetModel string
}

// Spec is the single registration record for a provider type.
type Spec struct {
	Type             string
	DisplayName      string
	DefaultBaseURL   string
	DefaultAPIKeyEnv string
	Capabilities     Capabilities
	AuthMode         AuthMode
	UIVisible        bool
	Seed             bool
	SeedID           string // defaults to prov_<type>
	SeedRoute        *SeedRoute
	CatalogAlias     string // modelcatalog fallback type
}

var (
	mu    sync.RWMutex
	byType = map[string]Spec{}
)

// RegisterSpec stores or replaces a provider type Spec.
func RegisterSpec(spec Spec) {
	typ := strings.ToLower(strings.TrimSpace(spec.Type))
	if typ == "" {
		return
	}
	spec.Type = typ
	if spec.AuthMode == "" {
		spec.AuthMode = AuthAPIKey
	}
	mu.Lock()
	defer mu.Unlock()
	byType[typ] = spec
}

// Lookup returns the Spec for a provider type.
func Lookup(typ string) (Spec, bool) {
	mu.RLock()
	defer mu.RUnlock()
	s, ok := byType[strings.ToLower(strings.TrimSpace(typ))]
	return s, ok
}

// Must returns the Spec or a zero Spec with AuthAPIKey for unknown types.
func Must(typ string) Spec {
	if s, ok := Lookup(typ); ok {
		return s
	}
	return Spec{Type: strings.ToLower(strings.TrimSpace(typ)), AuthMode: AuthAPIKey}
}

// All returns registered Specs sorted by type.
func All() []Spec {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]Spec, 0, len(byType))
	for _, s := range byType {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Type < out[j].Type })
	return out
}

// UIVisible returns Specs marked for the create-provider UI.
func UIVisible() []Spec {
	all := All()
	out := make([]Spec, 0, len(all))
	for _, s := range all {
		if s.UIVisible {
			out = append(out, s)
		}
	}
	return out
}

// Seedable returns Specs that should appear in local-dev seed.
func Seedable() []Spec {
	all := All()
	out := make([]Spec, 0, len(all))
	for _, s := range all {
		if s.Seed {
			out = append(out, s)
		}
	}
	return out
}

// SeedProviderID returns the stable seed row id for a Spec.
func (s Spec) SeedProviderID() string {
	if id := strings.TrimSpace(s.SeedID); id != "" {
		return id
	}
	return "prov_" + s.Type
}

// AllowsEmptyAPIKey reports whether empty api_key_env is valid for this type.
func AllowsEmptyAPIKey(typ string) bool {
	s, ok := Lookup(typ)
	if !ok {
		return false
	}
	return s.AuthMode == AuthOptional
}

// CatalogAliases returns modelcatalog fallback types for a provider type.
func CatalogAliases(typ string) []string {
	s, ok := Lookup(typ)
	if !ok || strings.TrimSpace(s.CatalogAlias) == "" {
		return nil
	}
	return []string{strings.ToLower(strings.TrimSpace(s.CatalogAlias))}
}
