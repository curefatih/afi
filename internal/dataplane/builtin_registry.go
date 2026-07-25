package dataplane

import (
	"sync"

	"github.com/curefatih/afi/internal/adapters/secrets"
	"github.com/curefatih/afi/internal/providercatalog"
)

// BuiltinFactory builds a ChatProvider from a secret resolver.
type BuiltinFactory func(sec secrets.Resolver) ChatProvider

type builtinEntry struct {
	typ     string
	factory BuiltinFactory
}

var (
	builtinMu      sync.Mutex
	builtinEntries []builtinEntry
)

// registerBuiltin registers an in-tree adapter factory. Spec metadata must already
// exist in providercatalog (see providercatalog/builtins.go).
func registerBuiltin(typ string, factory BuiltinFactory) {
	typ = normalizeProviderType(typ)
	if typ == "" || factory == nil {
		return
	}
	builtinMu.Lock()
	defer builtinMu.Unlock()
	for i, e := range builtinEntries {
		if e.typ == typ {
			builtinEntries[i].factory = factory
			return
		}
	}
	builtinEntries = append(builtinEntries, builtinEntry{typ: typ, factory: factory})
}

func normalizeProviderType(typ string) string {
	if s, ok := providercatalog.Lookup(typ); ok {
		return s.Type
	}
	return typ
}

func providerCapsFromSpec(typ string) ProviderCaps {
	s := providercatalog.Must(typ)
	return ProviderCaps{
		Chat: s.Capabilities.Chat, Stream: s.Capabilities.Stream,
		TTS: s.Capabilities.TTS, STT: s.Capabilities.STT,
		Embedding: s.Capabilities.Embedding, Image: s.Capabilities.Image,
	}
}

func buildBuiltinRegistry(sec secrets.Resolver) *Registry {
	if sec == nil {
		sec = secrets.Default()
	}
	builtinMu.Lock()
	entries := append([]builtinEntry(nil), builtinEntries...)
	builtinMu.Unlock()

	reg := NewRegistry()
	for _, e := range entries {
		reg.Register(e.factory(sec))
	}
	return reg
}
