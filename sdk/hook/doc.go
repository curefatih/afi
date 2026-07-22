// Package hook is the stable lifecycle API for gateway extensions.
//
// In-process Go hooks register on the dataplane HookChain from cmd/gateway.
// Sandboxed WASM guests use the same semantics via internal/adapters/wasm
// (see docs/hooks/wasm.md); they do not import this package.
//
// BeforeCall receives a mutable CallContext (principal, route, tags, metadata, body)
// and may allow, enrich, or deny. AfterCall runs after the provider attempt.
package hook
