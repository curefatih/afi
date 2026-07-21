// Package provider documents the AFI gateway ChatProvider contract for
// multi-model extensibility.
//
// Built-in adapters live in github.com/curefatih/afi/internal/dataplane and
// register into a Registry used by the gateway pipeline. Out-of-tree adapters
// implement ChatProvider in this package (or a module that imports it), then
// register at process startup via dataplane.Registry.RegisterSDK — see
// github.com/curefatih/afi/extensions/echo for a working example.
//
// gRPC provider runtimes are not shipped yet. WASM is for lifecycle hooks
// (docs/hooks/wasm.md), not ChatProvider adapters. See docs/development/providers.md.
package provider
