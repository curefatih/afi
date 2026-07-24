// Package provider documents the AFI gateway ChatProvider contract for
// multi-model extensibility.
//
// Built-in adapters live in github.com/curefatih/afi/internal/dataplane and
// register into a Registry used by the gateway pipeline. Out-of-tree adapters
// implement ChatProvider in this package (or a module that imports it), then
// register at process startup via dataplane.Registry.RegisterSDK — see
// github.com/curefatih/afi/extensions/echo for a working example.
//
// Prefer ChatIRProvider (sdk/chatir) when the adapter can speak typed chat IR
// directly; otherwise the gateway bridges through OpenAI-shaped Chat bytes.
//
// Process-isolated gRPC providers are supported via
// github.com/curefatih/afi/internal/adapters/grpcprovider and
// proto/afi/extension/v1 (see extensions/grpcecho). WASM is for lifecycle hooks
// (docs/hooks/wasm.md), not ChatProvider adapters. See docs/development/providers.md.
package provider
